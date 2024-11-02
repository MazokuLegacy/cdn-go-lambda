package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

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
	operationString := pathArr[lastIndex]
	operationsMap := getOperationsMap(operationString)
	s3Client := s3.NewFromConfig(cfg)
	width, ok := operationsMap["width"]
	if !ok {
		width = "750"
	}
	requestedWidth, err := strconv.Atoi(width)
	if err != nil {
		requestedWidth = 750
		delete(operationsMap, "width")
	}
	if requestedWidth > 750 {
		requestedWidth = 750
	}
	if pathArr[0] == "packs" {
		cardIds := pathArr[1:lastIndex]
		cardKeys := cardIds[0:0]
		for _, id := range cardIds {
			cardKeys = append(cardKeys, "cards/"+id+"/card")
		}
		var strtfetch = time.Now()
		cardObjects, err := fetchS3ObjectsParallel(cardKeys, s3Client)
		fmt.Println("fetch time ", time.Since(strtfetch))
		if handleFatalError(err, "failed to fetch card images") {
			return internalServerError("failed to fetch card images")
		}
		var strtpack = time.Now()
		output, err := packWebp(cardObjects, requestedWidth)
		fmt.Println("pack time ", time.Since(strtpack))
		if handleFatalError(err, "failed to make webp") {
			return internalServerError("failed to make webp")
		}
		contentType := "image/webp"
		return storeAndReturnTransformedMedia(output, s3Client, key, operationString, contentType)
	}
	fetchedObject, sourceContentType, err := fetchS3Object(key, s3Client)
	if handleFatalError(err, "failed to fetch original image") {
		return internalServerError("failed to fetch original image")
	}
	if pathArr[0] != "cards" || operationString == "original" {
		return storeAndReturnTransformedMedia(fetchedObject, s3Client, key, operationString, sourceContentType)
	}
	requestedFormat := operationsMap["format"]
	requestedFrame, hasFrame := operationsMap["frame"]
	var frameObject []byte
	if hasFrame {
		framekey := "frames/" + requestedFrame + "/frame"
		log.Println(framekey)
		var frameContentType string
		frameObject, frameContentType, err = fetchS3Object(framekey, s3Client)
		if err != nil {
			handleFatalError(err, "failed to fetch frame")
			return internalServerError("failed to fetch frame")
		}
		if frameContentType != "video/webm" {
			requestedFormat = "webp"
		}
	}
	mask := operationsMap["mask"]
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
				if mask == "true" {
					output, err = maskifyWebm(fetchedObject, requestedWidth)
					if handleFatalError(err, "failed to maskify webm") {
						return internalServerError("failed to maskify webm")
					}
				} else {
					if requestedWidth != 750 {
						output, err = scaleWebm(fetchedObject, requestedWidth)
						if handleFatalError(err, "failed to scale webm") {
							return internalServerError("failed to scale webm")
						}
					} else {
						output = fetchedObject
					}
				}
			}
			contentType = "video/webm"
		}
	} else {
		thumb, err := pngToWebp(fetchedObject, requestedWidth)
		if handleFatalError(err, "failed to resize and convert to webp") {
			return internalServerError("failed to resize and convert to webp")
		}
		if hasFrame {
			frameThumb, err := pngToWebp(frameObject, requestedWidth)
			if handleFatalError(err, "failed to convert to webp") {
				return internalServerError("failed to convert to webp")
			}
			output, err = framedWebp(thumb, frameThumb, requestedWidth)
		} else {
			output = thumb
		}
		contentType = "image/webp"
	}
	return storeAndReturnTransformedMedia(output, s3Client, key, operationString, contentType)
}

func main() {
	lambda.Start(LambdaHandler)
}
