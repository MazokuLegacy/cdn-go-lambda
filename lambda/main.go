package main

import (
	"context"
	"fmt"
	"log"
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
	if pathArr[0] != "cards" {
		return storeAndReturnTransformedMedia(fetchedObject, s3Client, key, operationString, sourceContentType)
	}
	operationsMap := getOperationsMap(operationString)
	requestedFormat := operationsMap["format"]
	requestedFrame, hasFrame := operationsMap["frame"]
	var frameObject []byte
	if hasFrame {
		frameType := ".png"
		if _, ok := map[string]string{"gif": "", "webm": "", "mp4": ""}[requestedFormat]; ok {
			frameType = ".webm"
		}
		framekey := "frames/" + requestedFrame + "/frame" + frameType
		log.Println(framekey)
		frameObject, _, err = fetchS3Object("frames/"+requestedFrame+"/frame"+frameType, s3Client)
		if err != nil {
			handleFatalError(err, "failed to fetch frame")
			return internalServerError("failed to fetch frame")
		}
	}
	width, ok := operationsMap["width"]
	if !ok {
		width = "750"
	}
	requestedWidth, err := strconv.Atoi(width)
	if handleFatalError(err, "width is not a valid number") {
		return internalServerError("width is not a valid number")
	}
	if requestedWidth > 750 {
		requestedWidth = 750
	}
	var output []byte
	contentType := sourceContentType
	if sourceContentType == "video/webm" {
		switch requestedFormat {
		case "gif":
			var modifiedOutput []byte
			if hasFrame {
				modifiedOutput, err = framedWebm(fetchedObject, frameObject, requestedWidth)
				if handleFatalError(err, "failed to add frame") {
					return internalServerError("failed to add frame")
				}
			} else {
				modifiedOutput = fetchedObject
			}
			output, err = webmToGif(modifiedOutput, requestedWidth)
			if handleFatalError(err, "failed to convert to gif") {
				return internalServerError("failed to convert to gif")
			}
			contentType = "image/" + requestedFormat
		case "webp":
			contentType = "image/" + requestedFormat
			thumb, err := webmToWebp(fetchedObject, requestedWidth)
			if hasFrame {
				frameThumb, err := pngToWebp(frameObject, requestedWidth)
				if handleFatalError(err, "failed to convert to webp") {
					return internalServerError("failed to convert to webp")
				}
				output, err = framedWebp(thumb, frameThumb, requestedWidth)
			} else {
				output = thumb
			}
			if handleFatalError(err, "failed to convert to webp") {
				return internalServerError("failed to convert to webp")
			}
		case "mp4":
			output, err = convertWebmToMP4(fetchedObject, requestedWidth)
			if handleFatalError(err, "failed to convert to mp4") {
				return internalServerError("failed to convert to mp4")
			}
			contentType = "video/" + requestedFormat
		default:
			if hasFrame {
				output, err = framedWebm(fetchedObject, frameObject, requestedWidth)
			} else {
				if requestedWidth != 750 {
					output, err = scaleWebm(fetchedObject, requestedWidth)
					if handleFatalError(err, "failed to convert to mp4") {
						return internalServerError("failed to convert to mp4")
					}
					contentType = "video/webm"
				}
			}

		}
	} else {
		output, err = pngToWebp(fetchedObject, int(requestedWidth))
		if handleFatalError(err, "failed to resize and convert to webp") {
			return internalServerError("failed to resize and convert to webp")
		}
		contentType = "image/webp"
	}
	return storeAndReturnTransformedMedia(output, s3Client, key, operationString, contentType)
}

func main() {
	lambda.Start(LambdaHandler)
}
