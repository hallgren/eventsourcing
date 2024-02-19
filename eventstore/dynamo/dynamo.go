package dynamo

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/hallgren/eventsourcing/core"
)

type DynamoEventStore struct {
	db        *dynamodb.Client
	tableName string
}

func New(cfg aws.Config, tableName string) *DynamoEventStore {
	return &DynamoEventStore{
		db:        dynamodb.NewFromConfig(cfg),
		tableName: tableName,
	}
}

func (store *DynamoEventStore) Save(events []core.Event) error {
	if len(events) == 0 {
		return nil
	}

	ctx := context.Background()
	lastVersion, err := store.getLastVersion(ctx, events[0].AggregateID)
	if err != nil {
		return err
	}

	if core.Version(lastVersion)+1 != events[0].Version {
		return core.ErrConcurrency
	}

	transactItems := make([]types.TransactWriteItem, len(events))

	for i, event := range events {
		globalVersion, err := store.getAndUpdateGlobalVersion(ctx)
		if err != nil {
			return err
		}

		events[i].GlobalVersion = globalVersion

		av, err := attributevalue.MarshalMap(event)
		if err != nil {
			return err
		}

		transactItems[i] = types.TransactWriteItem{
			Put: &types.Put{
				TableName: aws.String(store.tableName),
				Item:      av,
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":version": &types.AttributeValueMemberN{
						Value: fmt.Sprintf("%d", event.Version-1),
					},
				},
				ConditionExpression: aws.String(
					"attribute_not_exists(AggregateID) OR Version = :version",
				),
			},
		}
	}

	_, err = store.db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: transactItems,
	})

	if err != nil {
		var cce *types.TransactionCanceledException
		if ok := errors.As(err, &cce); ok {
			return core.ErrConcurrency
		}
		return err
	}

	return nil
}

func (store *DynamoEventStore) Get(
	ctx context.Context,
	id string,
	aggregateType string,
	afterVersion core.Version,
) (core.Iterator, error) {
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String(store.tableName),
		KeyConditionExpression: aws.String("AggregateID = :id AND Version > :version"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":id": &types.AttributeValueMemberS{Value: id},
			":version": &types.AttributeValueMemberN{
				Value: strconv.FormatInt(int64(afterVersion), 10),
			},
		},
	}

	result, err := store.db.Query(ctx, queryInput)
	if err != nil {
		return nil, err
	}

	return &iterator{
		items:   result.Items,
		current: -1, // Start before the first element
	}, nil
}

func (store *DynamoEventStore) getAndUpdateGlobalVersion(
	ctx context.Context,
) (core.Version, error) {
	key := map[string]types.AttributeValue{
		"AggregateID": &types.AttributeValueMemberS{Value: "GlobalVersionCounter"},
		"Version":     &types.AttributeValueMemberN{Value: "0"},
	}

	update := "SET GlobalVersion = if_not_exists(GlobalVersion, :start) + :inc"
	exprAttrValues := map[string]types.AttributeValue{
		":inc":   &types.AttributeValueMemberN{Value: "1"},
		":start": &types.AttributeValueMemberN{Value: "0"},
	}

	input := &dynamodb.UpdateItemInput{
		TableName:                 aws.String(store.tableName),
		Key:                       key,
		UpdateExpression:          aws.String(update),
		ExpressionAttributeValues: exprAttrValues,
		ReturnValues:              types.ReturnValueUpdatedNew,
	}

	result, err := store.db.UpdateItem(ctx, input)
	if err != nil {
		return 0, err
	}

	newVersionStr := result.Attributes["GlobalVersion"].(*types.AttributeValueMemberN).Value
	newVersion, err := strconv.ParseUint(newVersionStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing GlobalVersion: %w", err)
	}

	return core.Version(newVersion), nil
}

func (store *DynamoEventStore) getLastVersion(
	ctx context.Context,
	aggregateID string,
) (core.Version, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(store.tableName),
		KeyConditionExpression: aws.String("AggregateID = :aggregateID"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":aggregateID": &types.AttributeValueMemberS{Value: aggregateID},
		},
		ScanIndexForward: aws.Bool(false),
		Limit:            aws.Int32(1),
	}

	result, err := store.db.Query(ctx, input)
	if err != nil {
		return 0, err
	}

	if len(result.Items) == 0 {
		return 0, nil
	}

	var event core.Event
	err = attributevalue.UnmarshalMap(result.Items[0], &event)
	if err != nil {
		return 0, err
	}

	return event.Version, nil
}
