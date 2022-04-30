package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"log"
	"net/http"
	"os"
)

var db = dynamodb.New(session.New(), aws.NewConfig().WithRegion("us-east-1"))

const TABLE_NAME = "user"

var errorLogger = log.New(os.Stderr, "ERROR ", log.Llongfile)

type User struct {
	UserId string `json:"userID"`
	Title  string `json:"title"`
	Author string `json:"author"`
}

func main() {
	lambda.Start(Handler)
}

func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Println("request.Path", request.Path)
	fmt.Println("request.Path", request)
	var user User
	var userID string

	err := json.Unmarshal([]byte(request.Body), &user)

	if err != nil {
		return response("Couldn't unmarshal json into user struct", http.StatusBadRequest), nil
	}

	fmt.Println("request method", request.HTTPMethod)
	switch request.HTTPMethod {
	case "GET":
		return GetUserDetails(db, userID)
	case "POST":
		return SaveUser(db, &user)
	case "PUT":
		return UpdateUser(db, &user)
	}

	return response("", http.StatusMethodNotAllowed), nil
}

func response(body string, status int) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Body:       body,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
	}
}

func SaveUser(db *dynamodb.DynamoDB, user *User) (events.APIGatewayProxyResponse, error) {
	userMap, err := dynamodbattribute.MarshalMap(&user)

	if err != nil {
		fmt.Println("Failed to marshal to dynamo map")
		return response(err.Error(), http.StatusBadRequest), nil
	}

	input := &dynamodb.PutItemInput{
		Item:      userMap,
		TableName: aws.String(TABLE_NAME),
	}

	_, err = db.PutItem(input)

	if err != nil {
		fmt.Println("Failed to write to dynamo")
		return response(err.Error(), http.StatusInternalServerError), nil
	}

	return response("success", http.StatusOK), nil
}

func UpdateUser(db *dynamodb.DynamoDB, user *User) (events.APIGatewayProxyResponse, error) {
	userMap, err := dynamodbattribute.MarshalMap(&user)

	if err != nil {
		fmt.Println("Failed to marshal to dynamo map")
		return response(err.Error(), http.StatusBadRequest), nil
	}

	input := &dynamodb.PutItemInput{
		Item:      userMap,
		TableName: aws.String(TABLE_NAME),
	}

	_, err = db.PutItem(input)

	if err != nil {
		fmt.Println("Failed to write to dynamo")
		return response(err.Error(), http.StatusInternalServerError), nil
	}

	return response("success", http.StatusOK), nil
}

func GetUserDetails(db *dynamodb.DynamoDB, userID string) (events.APIGatewayProxyResponse, error) {
	result, err := db.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(TABLE_NAME),
		Key: map[string]*dynamodb.AttributeValue{
			"CoordinateBucket": {
				S: aws.String(userID),
			},
		},
	})
	if err != nil {
		return response(err.Error(), http.StatusBadRequest), nil
	}

	user := &User{}
	err = dynamodbattribute.UnmarshalMap(result.Item, user)

	body, err := json.Marshal(user)
	if err != nil {
		return response(err.Error(), http.StatusInternalServerError), nil
	}

	return response(string(body), http.StatusOK), nil
}
