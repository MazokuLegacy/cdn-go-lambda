package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/modfy/fluent-ffmpeg"
)

func LambdaHandler(ctx context.Context, event events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {

	ffmpegPath := fmt.Sprintf("%s/bin", os.Getenv("LAMBDA_TASK_ROOT"))
	os.Setenv("PATH", os.Getenv("PATH")+":"+ffmpegPath)
	pathArr := strings.Split(event.RawPath, "/")[1:]
	if pathArr[0] == "frames" {
		return events.LambdaFunctionURLResponse{
			StatusCode: 401,
			Body:       "Unauthorized",
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
		}, nil
	}
	lastIndex := len(pathArr) - 1
	key := strings.Join(pathArr[:lastIndex], "/")
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
	if handleFatalError(err, "failed to load config") {
		return internalServerError("failed to load config")
	}
	s3Client := s3.NewFromConfig(cfg)
	log.Println(key)
	fetchedObject, sourceContentType, err := fetchS3Object(key, s3Client)
	if handleFatalError(err, "failed to fetch original image") {
		return internalServerError("failed to fetch original image")
	}
	operationString := pathArr[lastIndex]
	operationsMap := getOperationsMap(operationString)
	requestedFormat := operationsMap["format"]
	log.Println(requestedFormat)
	if pathArr[0] != "cards" {
		return storeAndReturnTransformedMedia(fetchedObject, s3Client, key, operationString, sourceContentType)
	}
	if sourceContentType == "video/webm" {
		output := bytes.Clone(fetchedObject)
		var err error
		contentType := sourceContentType
		switch requestedFormat {
		case "webm":
			return storeAndReturnTransformedMedia(output, s3Client, key, operationString, "image/webp")
		case "mp4":
			output, err = convertWebMToMP4(fetchedObject)
			if handleFatalError(err, "failed to convert to mp4") {
				return internalServerError("failed to convert to mp4")
			}
			contentType = "video/" + requestedFormat
		default:
			output, err = getWebpFromWebm(fetchedObject)
			if handleFatalError(err, "failed to convert to webp") {
				return internalServerError("failed to convert to webp")
			}
			contentType = "image/" + requestedFormat

		}
		return storeAndReturnTransformedMedia(output, s3Client, key, operationString, contentType)
	}
	return events.LambdaFunctionURLResponse{
		StatusCode: 200,
		Body:       "nice",
		Headers: map[string]string{
			"Content-Type": "text/plain",
		},
	}, nil
}

func getWebpFromWebm(input []byte) ([]byte, error) {
	inPath := "/tmp/input.webm"
	outPath := "/tmp/output.webp"
	inFile, err := os.Create(inPath)
	if err != nil {
		return nil, err
	}
	defer inFile.Close()
	defer os.Remove(inPath)
	inFile.Write(input)
	outFile, err := os.Create(outPath)
	if err != nil {
		return nil, err
	}
	defer outFile.Close()
	defer os.Remove(outPath)
	cmd := exec.Command("ffmpeg", "-codec:v", "libvpx-vp9", "-y", "-i", inPath, "-vframes", "1", "-ss", "0", outPath)
	err = cmd.Start()
	if err != nil {
		fmt.Println("Error starting command:", err)
		return nil, err
	}
	err = cmd.Wait()
	if err != nil {
		fmt.Println("Error waiting for command:", err)
		return nil, err
	}
	log.Println("completed")
	output, err := io.ReadAll(outFile)
	return output, nil
}

func convertWebMToMP4(input []byte) ([]byte, error) {
	inputReader := bytes.NewReader(input)
	filePath := "/tmp/output.mp4"
	file, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	defer os.Remove(filePath)
	err = fluentffmpeg.NewCommand("").VideoCodec("libvpx-vp9").PipeInput(inputReader).OutputFormat("mp4").OutputPath(filePath).Overwrite(true).Run()

	if err != nil {
		return nil, err
	}
	output, err := io.ReadAll(file)
	return output, nil
}

func fetchS3Object(key string, s3Client *s3.Client) ([]byte, string, error) {
	output, err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(os.Getenv("originalImageBucketName")),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", err
	}
	defer output.Body.Close()

	body, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, "", err
	}
	contentType := *output.ContentType
	return body, contentType, nil
}

func storeAndReturnTransformedMedia(object []byte, s3Client *s3.Client, key string, operations string, contentType string) (events.LambdaFunctionURLResponse, error) {
	_, err := s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(os.Getenv("transformedImageBucketName")),
		Key:         aws.String(key + "/" + operations),
		Body:        bytes.NewReader(object),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return internalServerError("saving image to bucket failed")
	}
	encodedObject := base64.StdEncoding.EncodeToString(object)
	return events.LambdaFunctionURLResponse{
		StatusCode: 200,
		Body:       encodedObject,
		Headers: map[string]string{
			"Content-Type": contentType,
		},
		IsBase64Encoded: true,
	}, nil
}

func getOperationsMap(operationString string) (operations map[string]string) {
	result := make(map[string]string)
	pairs := strings.Split(operationString, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			key := kv[0]
			value := kv[1]
			result[key] = value
		}
	}
	return result
}

func internalServerError(message string) (events.LambdaFunctionURLResponse, error) {
	return events.LambdaFunctionURLResponse{
		StatusCode: 500,
		Body:       message,
		Headers: map[string]string{
			"Content-Type": "text/plain",
		},
	}, nil
}

func handleFatalError(err error, message string) bool {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
		return true
	}
	return false
}

func main() {
	lambda.Start(LambdaHandler)
}
