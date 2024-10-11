package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"log"
)

func LambdHandler(ctx context.Context, event events.LambdaFunctionURLRequest) {
	log.Println(event)
	return
}

func main() {
	lambda.Start(LambdHandler)
}
