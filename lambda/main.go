package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
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
	operations := pathArr[lastIndex]
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
	if handleFatalError(err, "failed to load config") {
		return internalServerError("failed to load config")
	}
	s3Client := s3.NewFromConfig(cfg)
	fetchedObject, sourceContentType, err := fetchS3Object(key, s3Client)
	if handleFatalError(err, "failed to fetch original image") {
		return internalServerError("failed to fetch original image")
	}
	if pathArr[0] != "cards" {
		return storeAndReturnTransformedMedia(fetchedObject, s3Client, key, operations, sourceContentType)
	}

	if sourceContentType == "video/webm" {
		mp4, err := convertWebMToMP4(fetchedObject)
		if handleFatalError(err, "failed to convert to mp4") {
			return internalServerError("failed to convert to mp4")
		}
		storeAndReturnTransformedMedia(mp4, s3Client, key, operations, sourceContentType)
	}
	return events.LambdaFunctionURLResponse{
		StatusCode: 200,
		Body:       "nice",
		Headers: map[string]string{
			"Content-Type": "text/plain",
		},
	}, nil
}

func convertWebMToMP4(input []byte) ([]byte, error) {
	inputReader := bytes.NewReader(input)
	buf := &bytes.Buffer{}
	tempFile, err := os.CreateTemp("", "output.mp4")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	err = fluentffmpeg.NewCommand("").PipeInput(inputReader).OutputFormat("mp4").OutputPath("output.mp4").Overwrite(true).OutputLogs(buf).Run()
	out, _ := io.ReadAll(buf)
	fmt.Println(string(out))
	if err != nil {
		return nil, err
	}
	output, err := io.ReadAll(tempFile)
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
