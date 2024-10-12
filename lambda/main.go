package main

import (
	"context"
	"encoding/base64"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"io"
	"log"
	"strings"
)

func LambdHandler(ctx context.Context, event events.LambdaFunctionURLRequest) events.LambdaFunctionURLResponse {

	pathArr := strings.Split(event.RawPath, "/")[1:]
	if pathArr[0] == "frames" {
		return events.LambdaFunctionURLResponse{
			StatusCode: 401,
			Body:       "Unauthorized",
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
		}
	}
	lastIndex := len(pathArr) - 1
	key := strings.Join(pathArr[:lastIndex], "/")
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if handleFatalError(err, err.Error()) {
		return internalServerError("failed to load config")
	}
	s3Client := s3.NewFromConfig(cfg)
	fetchedObject, sourceContentType, err := fetchS3Object(key, s3Client)
	if handleFatalError(err, err.Error()) {
		return internalServerError("failed to fetch original image")
	}
	if pathArr[0] != "cards" {
		return successfulResponse(fetchedObject, sourceContentType)
	}
	return events.LambdaFunctionURLResponse{
		StatusCode: 200,
		Body:       "nice",
		Headers: map[string]string{
			"Content-Type": "text/plain",
		},
	}
}

func main() {
	lambda.Start(LambdHandler)
}

func fetchS3Object(key string, s3Client *s3.Client) ([]byte, string, error) {
	output, err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String("mazoku-cdn"),
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
	contentType := output.ContentType
	return body, *contentType, nil
}

func successfulResponse(object []byte, contentType string) events.LambdaFunctionURLResponse {
	encodedObject := base64.StdEncoding.EncodeToString(object)
	return events.LambdaFunctionURLResponse{
		StatusCode: 200,
		Body:       encodedObject,
		Headers: map[string]string{
			"Content-Type": contentType,
		},
	}
}

func internalServerError(message string) events.LambdaFunctionURLResponse {
	return events.LambdaFunctionURLResponse{
		StatusCode: 500,
		Body:       "Unauthorized",
		Headers: map[string]string{
			"Content-Type": "text/plain",
		},
	}
}

func handleFatalError(err error, message string) bool {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
		return true
	}
	return false
}
