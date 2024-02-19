package dynamo

import (
	"errors"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/hallgren/eventsourcing/core"
)

type iterator struct {
	items   []map[string]types.AttributeValue
	current int
}

func (it *iterator) Next() bool {
	it.current++
	return it.current < len(it.items)
}

func (it *iterator) Value() (core.Event, error) {
	if it.current < 0 || it.current >= len(it.items) {
		return core.Event{}, errors.New("iterator out of bounds")
	}
	var event core.Event
	err := attributevalue.UnmarshalMap(it.items[it.current], &event)
	if err != nil {
		return core.Event{}, err
	}
	return event, nil
}

func (it *iterator) Close() {
	// No resources to close in this example, but in case of network connections or file handles, close them here.
}
