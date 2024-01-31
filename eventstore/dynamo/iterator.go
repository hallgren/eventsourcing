package dynamo

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/hallgren/eventsourcing/core"
)

type iterator struct {
	items   []map[string]*dynamodb.AttributeValue
	current int
}

func (it *iterator) Next() bool {
	it.current++
	if it.current >= len(it.items) {
		return false
	}

	return true
}

func (it *iterator) Value() (core.Event, error) {
	var event core.Event
	err := dynamodbattribute.UnmarshalMap(it.items[it.current], &event)
	if err != nil {
		return core.Event{}, err
	}

	return event, nil
}

func (it *iterator) Close() {
	// No resources to close in this example, but in case of network connections or file handles, close them here.
}
