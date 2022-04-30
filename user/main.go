package user

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"net/http"
)

var db = dynamodb.New(session.New(), aws.NewConfig().WithRegion("us-east-1"))
const TABLE_NAME = "user"

type User struct {
	UserId   string `json:"userID"`
	Title  string `json:"title"`
	Author string `json:"author"`
}


func main() {
	lambda.Start(Handler)
}

func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse , error) {
	var user User

	err := json.Unmarshal([]byte(request.Body), &user)

	if err != nil {
		return response("Couldn't unmarshal json into user struct", http.StatusBadRequest), nil
	}

	err = SaveUser(db, &user)

	if err != nil {
		return response(err.Error(), http.StatusInternalServerError), nil
	}

	return response("Successfully wrote monster to log.", http.StatusOK), nil
}

func response(body string, statusCode int) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse {
		StatusCode: statusCode,
		Body: string(body),
		Headers: map[string]string {
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
	}
}

func SaveUser(db *dynamodb.DynamoDB, user *User) error{
	userMap, err := dynamodbattribute.MarshalMap(&user)

	if err != nil {
		fmt.Println("Failed to marshal to dynamo map")
		return err
	}


	input := &dynamodb.PutItemInput{
		Item:      userMap,
		TableName: aws.String(TABLE_NAME),
	}

	_, writeErr := db.PutItem(input)

	if writeErr != nil {
		fmt.Println("Failed to write to dynamo")
		return writeErr
	}

	return nil
}

func GetUserDetails(db *dynamodb.DynamoDB,userID string) (*User, error) {
	result, err := db.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(TABLE_NAME),
		Key: map[string]*dynamodb.AttributeValue{
			"CoordinateBucket": {
				S: aws.String(userID),
			},
		},
	})

	user := &User{}
	err = dynamodbattribute.UnmarshalMap(result.Item, user)
	return user, err
}
