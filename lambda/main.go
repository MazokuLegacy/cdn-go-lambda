package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"log"
)

func LambdHandler(ctx context.Context, event events.LambdaFunctionURLRequest) (string, error) {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("failed to marshal event: %v", err)
	}
	log.Printf("Received event: %s", eventJSON)
	return "Event processed successfully", nil
}

func main() {
	lambda.Start(LambdHandler)
}
