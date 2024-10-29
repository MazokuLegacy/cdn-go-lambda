package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func getScale(width int) string {
	return "scale=" + strconv.Itoa(width) + ":-1"
}

func processFilesParallel(inputs [][]byte) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(inputs))

	for i, fileData := range inputs {
		wg.Add(1)
		go func(i int, fileData []byte) {
			defer wg.Done()
			inPath := "/tmp/pack/card" + strconv.Itoa(i) + ".png"
			inFile, err := os.Create(inPath)
			if err != nil {
				errCh <- err
				return
			}
			_, writeErr := inFile.Write(fileData)
			if writeErr != nil {
				errCh <- writeErr
			}
			inFile.Close()
		}(i, fileData)
	}
	go func() {
		wg.Wait()
		close(errCh)
	}()
	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

func fetchS3ObjectsParallel(keys []string, s3Client *s3.Client) ([][]byte, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([][]byte, len(keys))
	var anyerr error
	wg.Add(len(keys))
	for i, key := range keys {
		go func(k string, i int) {
			defer wg.Done()
			data, sourceContentType, err := fetchS3Object(key, s3Client)
			if err != nil {
				anyerr = err
				return
			}
			var result []byte
			if sourceContentType == "video/webm" {
				result, err = webmToPng(data, 1500)
				if err != nil {
					anyerr = err
					return
				}
			} else {
				result = data
			}
			mu.Lock()
			results[i] = result
			mu.Unlock()
		}(key, i)
	}

	wg.Wait()
	return results, anyerr
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
	go func() {
		transformedBucket := os.Getenv("transformedImageBucketName")
		_, _ = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket:      aws.String(transformedBucket),
			Key:         aws.String(key + "/" + operations),
			Body:        bytes.NewReader(object),
			ContentType: aws.String(contentType),
		})
	}()
	if len(object) > 6291456 {
		redirectUrl := "https://cdn.mazoku.cc/" + key + "?" + strings.ReplaceAll(operations, ",", "&")
		return events.LambdaFunctionURLResponse{
			StatusCode: 302,
			Headers: map[string]string{
				"Location":      redirectUrl,
				"Cache-Control": "no-cache, no-store, must-revalidate",
				"Pragma":        "no-cache",
				"Expires":       "0",
			},
		}, nil
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
		StatusCode: 200,
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
