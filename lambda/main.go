package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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
	fetchedObject, sourceContentType, err := fetchS3Object(key, s3Client)
	if handleFatalError(err, "failed to fetch original image") {
		return internalServerError("failed to fetch original image")
	}
	operationString := pathArr[lastIndex]
	operationsMap := getOperationsMap(operationString)
	requestedFormat := operationsMap["format"]
	if pathArr[0] != "cards" {
		return storeAndReturnTransformedMedia(fetchedObject, s3Client, key, operationString, sourceContentType)
	}
	width := operationsMap["width"]
	if width == "" {
		width = "750"
	}
	requestedWidth, err := strconv.Atoi(operationsMap["width"])
	if handleFatalError(err, "width is not a valid number") {
		return internalServerError("width is not a valid number")
	}
	if requestedWidth > 750 {
		requestedWidth = 750
	}
	if sourceContentType == "video/webm" {
		output := bytes.Clone(fetchedObject)
		var err error
		contentType := sourceContentType
		switch requestedFormat {
		case "gif":
			output, err = webmToGif(fetchedObject, requestedWidth)
			if handleFatalError(err, "failed to convert to png") {
				return internalServerError("failed to convert to png")
			}
			contentType = "image/" + requestedFormat
		case "webp":
			output, err = webmToWebp(fetchedObject, requestedWidth)
			if handleFatalError(err, "failed to convert to webp") {
				return internalServerError("failed to convert to webp")
			}
			contentType = "image/" + requestedFormat
		case "mp4":
			output, err = convertWebmToMP4(fetchedObject, requestedWidth)
			if handleFatalError(err, "failed to convert to mp4") {
				return internalServerError("failed to convert to mp4")
			}
			contentType = "video/" + requestedFormat
		default:
			if requestedWidth != 750 {
				output, err = scaleWebm(fetchedObject, requestedWidth)
				if handleFatalError(err, "failed to convert to mp4") {
					return internalServerError("failed to convert to mp4")
				}
				contentType = "video/webm"
			}
		}
		return storeAndReturnTransformedMedia(output, s3Client, key, operationString, contentType)
	}
	output, err := pngToWebp(fetchedObject, int(requestedWidth))
	if handleFatalError(err, "failed to resize and convert to webp") {
		return internalServerError("failed to resize and convert to webp")
	}
	return storeAndReturnTransformedMedia(output, s3Client, key, operationString, "image/webp")
}

func main() {
	lambda.Start(LambdaHandler)
}
