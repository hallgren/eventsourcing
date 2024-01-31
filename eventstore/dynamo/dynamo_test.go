package dynamo_test

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/hallgren/eventsourcing/core"
	"github.com/hallgren/eventsourcing/core/testsuite"
	"github.com/hallgren/eventsourcing/eventstore/dynamo"
)

func TestDynamoDBEventStore(t *testing.T) {
	f := func() (core.EventStore, func(), error) {
		// DynamoDB client configuration
		sess, err := session.NewSession(&aws.Config{
			Region: aws.String("us-east-1"), // Replace with your AWS region
			// Other configurations if necessary
			Endpoint:    aws.String("http://localhost:8000"),
			Credentials: credentials.NewStaticCredentials("dummy", "dummy", ""),
		})
		if err != nil {
			return nil, nil, err
		}

		// Ensure you have the DynamoDB table set up properly
		tableName := "EventStoreTable" // Replace with the name of your table

		es := dynamo.New(sess, tableName)

		db := es.DB()

		input := &dynamodb.CreateTableInput{
			TableName: aws.String("EventStoreTable"),
			KeySchema: []*dynamodb.KeySchemaElement{
				{
					AttributeName: aws.String("AggregateID"),
					KeyType:       aws.String("HASH"), // Partition key
				},
				{
					AttributeName: aws.String("Version"),
					KeyType:       aws.String("RANGE"), // Sort key
				},
			},
			AttributeDefinitions: []*dynamodb.AttributeDefinition{
				{
					AttributeName: aws.String("AggregateID"),
					AttributeType: aws.String("S"), // String
				},
				{
					AttributeName: aws.String("Version"),
					AttributeType: aws.String("N"), // Number
				},
				// Definition for GlobalVersionCounter if it's going to be used as a secondary index
				{
					AttributeName: aws.String("GlobalVersion"),
					AttributeType: aws.String("N"), // Number
				},
				// Add other attribute definitions here if necessary
			},
			ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
				ReadCapacityUnits:  aws.Int64(5), // Adjust according to your needs
				WriteCapacityUnits: aws.Int64(5),
			},
			// Add secondary index configurations here if necessary
		}

		db.CreateTable(input)

		// Closure function is not necessary for DynamoDB in this case,
		// but it's included to comply with the function signature.
		closeFunc := func() {}

		return es, closeFunc, nil
	}

	testsuite.Test(t, f)
}
