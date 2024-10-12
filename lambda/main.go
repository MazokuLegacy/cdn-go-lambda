package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"strings"
)

func LambdHandler(ctx context.Context, event events.LambdaFunctionURLRequest) events.LambdaFunctionURLResponse {
	var key = event.RawPath
	var pathArr = strings.Split(key, "/")
	if pathArr[0] == "frames" {
		return events.LambdaFunctionURLResponse{
			StatusCode: 401,
			Body:       "Unautharized",
		}

	}
	return events.LambdaFunctionURLResponse{
		StatusCode: 200,
		Body:       "nice",
	}
}

func main() {
	lambda.Start(LambdHandler)
}
