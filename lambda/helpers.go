package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func getScale(width int) string {
	return "scale=" + strconv.Itoa(width) + ":-1"
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
	transformedBucket := os.Getenv("transformedImageBucketName")
	_, err := s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(transformedBucket),
		Key:         aws.String(key + "/" + operations),
		Body:        bytes.NewReader(object),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return internalServerError("saving image to bucket failed")
	}
	s3Url := "https://d1j2iknz72s0sm.cloudfront.net/" + key + "?" + operations
	return events.LambdaFunctionURLResponse{
		StatusCode: 302,
		Headers: map[string]string{
			"Location": s3Url,
		},
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
