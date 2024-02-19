package dynamo_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/hallgren/eventsourcing/core"
	"github.com/hallgren/eventsourcing/core/testsuite"
	"github.com/hallgren/eventsourcing/eventstore/dynamo"
)

func TestDynamoDBEventStore(t *testing.T) {
	f := func() (core.EventStore, func(), error) {
		// AWS SDK v2 configuration
		cfg, err := config.LoadDefaultConfig(
			context.TODO(),
			config.WithRegion("us-east-1"), // Replace with your AWS region
			// Configuration for local DynamoDB or custom endpoint
			config.WithEndpointResolverWithOptions(
				aws.EndpointResolverWithOptionsFunc(
					func(service, region string, options ...interface{}) (aws.Endpoint, error) {
						return aws.Endpoint{
							URL:           "http://localhost:8000", // Local DynamoDB endpoint
							SigningRegion: "us-east-1",
						}, nil
					},
				),
			),
			config.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider("dummy", "dummy", ""),
			), // Static credentials
		)
		if err != nil {
			return nil, nil, err
		}

		// Ensure you have the DynamoDB table set up properly
		tableName := "EventStoreTable" // Replace with the name of your table

		es := dynamo.New(cfg, tableName)

		db := dynamodb.NewFromConfig(cfg)

		input := &dynamodb.CreateTableInput{
			TableName: aws.String("EventStoreTable"),
			KeySchema: []types.KeySchemaElement{
				{
					AttributeName: aws.String("AggregateID"),
					KeyType:       types.KeyTypeHash, // Partition key
				},
				{
					AttributeName: aws.String("Version"),
					KeyType:       types.KeyTypeRange, // Sort key
				},
			},
			AttributeDefinitions: []types.AttributeDefinition{
				{
					AttributeName: aws.String("AggregateID"),
					AttributeType: types.ScalarAttributeTypeS, // String
				},
				{
					AttributeName: aws.String("Version"),
					AttributeType: types.ScalarAttributeTypeN, // Number
				},
				{
					AttributeName: aws.String(
						"GlobalVersion",
					), // Ensure this is defined as a Number for the GSI
					AttributeType: types.ScalarAttributeTypeN, // Number
				},
			},
			GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
				{
					IndexName: aws.String("GlobalVersionIndex"), // Unique name for the GSI
					KeySchema: []types.KeySchemaElement{
						{
							AttributeName: aws.String(
								"GlobalVersion",
							), // Partition key for the GSI, defined as a Number
							KeyType: types.KeyTypeHash,
						},
						// Optional: If you need a sort key, define it here
					},
					Projection: &types.Projection{
						ProjectionType: types.ProjectionTypeAll, // Adjust based on your needs (ALL, KEYS_ONLY, INCLUDE)
					},
					ProvisionedThroughput: &types.ProvisionedThroughput{
						ReadCapacityUnits:  aws.Int64(1), // Adjust based on your expected workload
						WriteCapacityUnits: aws.Int64(1),
					},
				},
			},
			ProvisionedThroughput: &types.ProvisionedThroughput{
				ReadCapacityUnits:  aws.Int64(5), // Main table throughput settings
				WriteCapacityUnits: aws.Int64(5),
			},
		}

		_, err = db.CreateTable(context.TODO(), input)
		if err != nil {
			return nil, nil, err
		}

		// Closure function to clean up resources, if necessary
		closeFunc := func() {
			_, err := db.DeleteTable(context.TODO(), &dynamodb.DeleteTableInput{
				TableName: aws.String("EventStoreTable"), // Replace with your actual table name
			})

			if err != nil {
				t.Fatalf("Failed to delete test table: %v", err)
			}
		}

		return es, closeFunc, nil
	}

	testsuite.Test(t, f)
}
