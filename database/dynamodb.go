package database

import (
	"errors"
	"log"
	"os"

	"github.com/Real-Dev-Squad/feature-flag-backend/models"
	"github.com/Real-Dev-Squad/feature-flag-backend/utils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

var db *dynamodb.DynamoDB
var marshalMapFunction = dynamodbattribute.MarshalMap
var unmarshalMapFunction = dynamodbattribute.UnmarshalMap

// Initializes DynamoDB connection
func CreateDynamoDB() *dynamodb.DynamoDB {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Error is \n %v", err)
		}
	}()

	// Fetch the region from the environment variable
	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		log.Println("AWS_REGION environment variable is not set")
		utils.ServerError(errors.New("AWS_REGION is not set, please configure it"))
	}

	// Create a new DynamoDB session
	if db == nil {
		sess, err := session.NewSession(&aws.Config{
			Region: aws.String(awsRegion),
		})

		if err != nil {
			log.Printf("Error creating the dynamodb session: \n %v", err)
			utils.ServerError(errors.New("Error creating DynamoDB session"))
		}

		db = dynamodb.New(sess)
	}

	return db
}

// MarshalMap converts struct to DynamoDB map
func MarshalMap(input interface{}) (map[string]*dynamodb.AttributeValue, error) {
	item, err := marshalMapFunction(input)
	if err != nil {
		return nil, err
	}
	return item, nil
}

// UnmarshalMap converts DynamoDB map to struct
func UnmarshalMap(input map[string]*dynamodb.AttributeValue, targetStruct interface{}) error {
	err := unmarshalMapFunction(input, &targetStruct)
	if err != nil {
		return err
	}
	return nil
}

// Function to process feature flag retrieval by hash key
func ProcessGetFeatureFlagByHashKey(attributeName string, attributeValue string) (*utils.FeatureFlagResponse, error) {
	db := CreateDynamoDB()

	input := &dynamodb.GetItemInput{
		TableName: aws.String(utils.FEATURE_FLAG_TABLE_NAME),
		Key: map[string]*dynamodb.AttributeValue{
			attributeName: {
				S: aws.String(attributeValue),
			},
		},
	}

	result, err := db.GetItem(input)
	if err != nil {
		utils.DdbError(err)
		return nil, err
	}

	if len(result.Item) == 0 {
		return nil, nil
	}

	featureFlagResponse := new(utils.FeatureFlagResponse)
	err = UnmarshalMap(result.Item, &featureFlagResponse)

	if err != nil {
		log.Println(err, " is the error while converting to ddb object")
		return nil, err
	}
	return featureFlagResponse, nil
}

// Function to add user-feature flag mappings
func AddUserFeatureFlagMapping(featureFlagUserMappings []models.FeatureFlagUserMapping) ([]models.FeatureFlagUserMapping, error) {
	db := CreateDynamoDB()

	for _, featureFlagUserMapping := range featureFlagUserMappings {
		item, err := MarshalMap(featureFlagUserMapping)
		if err != nil {
			return nil, err
		}

		input := &dynamodb.PutItemInput{
			TableName:           aws.String(utils.FEATURE_FLAG_USER_MAPPING_TABLE_NAME),
			Item:                item,
			ConditionExpression: aws.String("attribute_not_exists(userId)"),
		}

		_, err = db.PutItem(input)
		if err != nil {
			utils.DdbError(err)
			return nil, err
		}
	}
	return featureFlagUserMappings, nil
}
