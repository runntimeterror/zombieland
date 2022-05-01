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
	"strings"
)

const TABLE_NAME = "user"

var errorLogger = log.New(os.Stderr, "ERROR ", log.Llongfile)

type User struct {
	UserId    string              `json:"userId" `
	FirstName string              `json:"firstname"`
	LastName  string              `json:"lastname"`
	Email     string              `json:"email"`
	Steps     int64               `json:"steps"`
	Level     int                 `json:"level"`
	Rewards   map[string][]string `json:"rewards"`
}

func main() {
	lambda.Start(Handler)
}

func Handler(request events.APIGatewayV2HTTPRequest) (events.APIGatewayProxyResponse, error) {
	var user User
	//var userID string

	var db = dynamodb.New(session.New(), aws.NewConfig().WithRegion("us-west-2"))

	path := request.RawPath

	if strings.Contains(path, "getuser") {
		path = "/getuser"
	}

	switch path {
	case "/getuser":
		return GetUserDetails(db, request.PathParameters["userId"])
	case "/saveuser":
		err := json.Unmarshal([]byte(request.Body), &user)

		if err != nil {
			return response("Couldn't unmarshal json into user struct", http.StatusBadRequest), nil
		}
		return SaveUser(db, &user)
	case "/updateuser":
		err := json.Unmarshal([]byte(request.Body), &user)

		if err != nil {
			return response("Couldn't unmarshal json into user struct", http.StatusBadRequest), nil
		}
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

func GetUserDetails(db *dynamodb.DynamoDB, userID string) (events.APIGatewayProxyResponse, error) {
	result, err := db.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String("user"),
		Key: map[string]*dynamodb.AttributeValue{
			"userId": {
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

func SaveUser(db *dynamodb.DynamoDB, user *User) (events.APIGatewayProxyResponse, error) {
	userMap, err := dynamodbattribute.MarshalMap(&user)

	if err != nil {
		fmt.Println("Failed to marshal to dynamo map")
		return response(err.Error(), http.StatusBadRequest), nil
	}

	input := &dynamodb.PutItemInput{
		Item:      userMap,
		TableName: aws.String("user"),
	}

	op, err := db.PutItem(input)
	fmt.Println(op)
	if err != nil {
		fmt.Println("Failed to write to dynamo", err)
		return response(err.Error(), http.StatusInternalServerError), nil
	}

	return response("success", http.StatusOK), nil
}

func UpdateUser(db *dynamodb.DynamoDB, user *User) (events.APIGatewayProxyResponse, error) {
	//userMap, err := dynamodbattribute.MarshalMap(&user)
	//
	//if err != nil {
	//	fmt.Println("Failed to marshal to dynamo map")
	//	return response(err.Error(), http.StatusBadRequest), nil
	//}

	type updateInfo struct {
		Steps   int64               `json:"steps"`
		Rewards map[string][]string `json:"rewards"`
	}
	update, err := dynamodbattribute.MarshalMap(updateInfo{
		Steps:   user.Steps,
		Rewards: user.Rewards,
	})

	input := &dynamodb.UpdateItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"userId": {
				S: aws.String(user.UserId),
			},
		},
		ExpressionAttributeValues: update,
		UpdateExpression:          aws.String("SET steps =:steps,  rewards = :rewards"),
		TableName:                 aws.String(TABLE_NAME),
	}

	_, err = db.UpdateItem(input)

	if err != nil {
		fmt.Println("Failed to write to dynamo")
		return response(err.Error(), http.StatusInternalServerError), nil
	}

	return response("success", http.StatusOK), nil
}
