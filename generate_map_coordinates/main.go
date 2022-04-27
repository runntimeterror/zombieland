package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/url"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

// Response is of type APIGatewayProxyResponse since we're leveraging the
// AWS Lambda Proxy Request functionality (default behavior)
//
// https://serverless.com/framework/docs/providers/aws/events/apigateway/#lambda-proxy-integration
type Response events.APIGatewayProxyResponse

const RADIUS_IN_DEGREES = 0.0136             // equivalent to about 1500m radius
const GENERATION_RADIUS_IN_DEGREES = 0.00676 // equivalent to about 750m radius
const TABLE_NAME = "map_coordinates"

type CoordinateBucket struct {
	ZombieCoordinates  [][2]float64 `json:"ZombieCoordinates"`
	LootboxCoordinates [][2]float64 `json:"LootboxCoordinates"`
	Timestamp          time.Time    `json:"Timestamp"`
	CoordinateBucket   string       `json:"CoordinateBucket"`
}

// Handler is our lambda handler invoked by the `lambda.Start` function call
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (Response, error) {
	// TODO: change to using request body? because they are floats?
	currentTime := time.Now()
	longitudeRaw, found := request.PathParameters["longitude"]
	var bucketYCoord int
	if found {
		value, err := url.QueryUnescape(longitudeRaw)
		if err != nil {
			return Response{StatusCode: 500}, fmt.Errorf("failed to unescape longitudeRaw: %v, error: %v\n", longitudeRaw, err)
		}

		longitude, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return Response{StatusCode: 500}, fmt.Errorf("failed to convert longitude to float: %v, error: %v\n", value, err)
		}
		bucketYCoord = int(math.Floor(longitude / RADIUS_IN_DEGREES))
	}

	latitudeRaw, found := request.PathParameters["latitude"]
	var bucketXCoord int
	if found {
		value, err := url.QueryUnescape(latitudeRaw)
		if err != nil {
			return Response{StatusCode: 500}, fmt.Errorf("failed to unescape longitudeRaw: %v, error: %v\n", longitudeRaw, err)
		}

		latitude, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return Response{StatusCode: 500}, fmt.Errorf("failed to convert longitude to float: %v, error: %v\n", value, err)
		}
		bucketXCoord = int(math.Floor(latitude / RADIUS_IN_DEGREES))
	}

	// includes the current bucket
	coordinateBuckets := generateSurroundingBuckets([2]int{bucketXCoord, bucketYCoord})

	svc := dynamodb.New(session.New(),
		aws.NewConfig().WithRegion("us-west-2"),
	)

	/****************************************/
	/****************************************/
	/**** TRYING GETTING FOR THE BUCKETS ****/
	/****************************************/
	/****************************************/
	// GET RESULTS FROM BUCKET TO SEE IF THEY EXIST
	remainingcoordinates := [][2]int{}
	outputBucket := CoordinateBucket{
		ZombieCoordinates:  [][2]float64{},
		LootboxCoordinates: [][2]float64{},
		Timestamp:          currentTime,
		CoordinateBucket:   fmt.Sprintf("%v:%v", bucketXCoord, bucketYCoord),
	}
	for _, coordinateBucket := range coordinateBuckets {
		result, err := getCoordinateBucketFromDB(svc, fmt.Sprintf("%v:%v", coordinateBucket[0], coordinateBucket[1]))
		if err != nil {
			return Response{StatusCode: 500}, fmt.Errorf("failed to query dynaodb tableName: %v, error: %v\n", TABLE_NAME, err)
		}

		// compile found items into the results
		if result.Item != nil {
			cb := CoordinateBucket{}
			err := dynamodbattribute.UnmarshalMap(result.Item, &cb)
			if err != nil {
				return Response{StatusCode: 500}, fmt.Errorf("failed to unmarshal coordinate bucket: %v, error: %v\n", cb, err)
			}

			if currentTime.After(cb.Timestamp.Add(24 * time.Hour)) { // timestamp found in database is older than 24 hours
				remainingcoordinates = append(remainingcoordinates, coordinateBucket)
				continue
			}

			outputBucket.ZombieCoordinates = append(outputBucket.ZombieCoordinates, cb.ZombieCoordinates...)
			outputBucket.LootboxCoordinates = append(outputBucket.LootboxCoordinates, cb.LootboxCoordinates...)
			continue
		}

		// compile not found items into the remaining buckets
		remainingcoordinates = append(remainingcoordinates, coordinateBucket)
	}

	/********************************/
	/********************************/
	/**** WRITING TO THE BUCKETS ****/
	/********************************/
	/********************************/
	// generate coordinates for the remaining points and write into the database
	for _, coordinateBucket := range remainingcoordinates {
		latitude := float64(coordinateBucket[0]) * RADIUS_IN_DEGREES
		longitude := float64(coordinateBucket[1]) * RADIUS_IN_DEGREES
		zombieCoords := generateRandomCoordinates(latitude, longitude, GENERATION_RADIUS_IN_DEGREES, 5)
		lootboxCoords := generateRandomCoordinates(latitude, longitude, GENERATION_RADIUS_IN_DEGREES, 5)
		outputBucket.ZombieCoordinates = append(outputBucket.ZombieCoordinates, zombieCoords...)
		outputBucket.LootboxCoordinates = append(outputBucket.LootboxCoordinates, lootboxCoords...)

		cb := CoordinateBucket{
			ZombieCoordinates:  zombieCoords,
			LootboxCoordinates: lootboxCoords,
			Timestamp:          currentTime,
			CoordinateBucket:   fmt.Sprintf("%v:%v", coordinateBucket[0], coordinateBucket[1]),
		}

		putCoordinateIntoDB(svc, cb)
	}

	resp, err := createResponseOutput(outputBucket)
	return resp, err
	// return Response{}, nil
}

func generateRandomCoordinates(latitude float64, longitude float64, radiusInDegrees float64, count int) [][2]float64 {
	markers := [][2]float64{}
	for i := 0; i < count; i++ {
		w := float64(radiusInDegrees) * math.Sqrt(rand.Float64())
		t := 2 * math.Pi * rand.Float64()
		newLat := (w * math.Cos(t) / math.Cos(longitude)) + latitude
		newLong := (w * math.Sin(t)) + longitude
		markers = append(markers, [2]float64{newLat, newLong})
	}

	return markers
}

func getCoordinateBucketFromDB(svc *dynamodb.DynamoDB, coordinateBucket string) (*dynamodb.GetItemOutput, error) {
	result, err := svc.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(TABLE_NAME),
		Key: map[string]*dynamodb.AttributeValue{
			"CoordinateBucket": {
				S: aws.String(coordinateBucket),
			},
		},
	})
	return result, err
}

func putCoordinateIntoDB(svc *dynamodb.DynamoDB, bucket CoordinateBucket) error {
	dynamoItem, err := dynamodbattribute.MarshalMap(bucket)
	if err != nil {
		return fmt.Errorf("failed to marshal data into dynamoItem: %v, error: %v\n", bucket, err)
	}
	input := &dynamodb.PutItemInput{
		Item:      dynamoItem,
		TableName: aws.String("map_coordinates"),
	}
	_, err = svc.PutItem(input)
	if err != nil {
		return fmt.Errorf("failed to PutItem: %v, error: %v\n", input, err)
	}

	return nil
}

func createResponseOutput(coordinateBucket CoordinateBucket) (Response, error) {
	var buf bytes.Buffer

	body, err := json.Marshal(coordinateBucket)
	if err != nil {
		return Response{StatusCode: 404}, err
	}
	json.HTMLEscape(&buf, body)

	resp := Response{
		StatusCode:      200,
		IsBase64Encoded: false,
		Body:            buf.String(),
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
	}

	return resp, err
}

func generateSurroundingBuckets(bucket [2]int) [][2]int {
	buckets := [][2]int{bucket}
	for _, direction := range directions() {
		buckets = append(buckets, [2]int{bucket[0] + direction[0], bucket[1] + direction[1]})
	}

	return buckets
}

func directions() [][2]int {
	return [][2]int{{0, 1}, {1, 1}, {1, 0}, {1, -1}, {0, -1}, {-1, -1}, {-1, 0}, {-1, 1}}
}

func main() {
	lambda.Start(Handler)
}
