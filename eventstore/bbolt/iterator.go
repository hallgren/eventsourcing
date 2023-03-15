package bbolt

import (
	"errors"
	"fmt"

	"github.com/hallgren/eventsourcing"
	"go.etcd.io/bbolt"
)

type iterator[T any] struct {
	tx              *bbolt.Tx
	bucketName      string
	firstEventIndex uint64
	cursor          *bbolt.Cursor
	serializer      eventsourcing.Serializer[T]
}

// Close closes the iterator
func (i *iterator[T]) Close() {
	i.tx.Rollback()
}

// Next return the next event
func (i *iterator[T]) Next() (eventsourcing.Event[T], error) {
	var k, obj []byte
	if i.cursor == nil {
		bucket := i.tx.Bucket([]byte(i.bucketName))
		if bucket == nil {
			return eventsourcing.Event[T]{}, eventsourcing.ErrNoMoreEvents
		}
		i.cursor = bucket.Cursor()
		k, obj = i.cursor.Seek(itob(i.firstEventIndex))
		if k == nil {
			return eventsourcing.Event[T]{}, eventsourcing.ErrNoMoreEvents
		}
	} else {
		k, obj = i.cursor.Next()
	}
	if k == nil {
		return eventsourcing.Event[T]{}, eventsourcing.ErrNoMoreEvents
	}
	bEvent := boltEvent{}
	err := i.serializer.Unmarshal(obj, &bEvent)
	if err != nil {
		return eventsourcing.Event[T]{}, errors.New(fmt.Sprintf("could not deserialize event, %v", err))
	}
	f, ok := i.serializer.Type(bEvent.AggregateType, bEvent.Reason)
	if !ok {
		// if the typ/reason is not register jump over the event
		return i.Next()
	}
	eventData := f()
	err = i.serializer.Unmarshal(bEvent.Data, &eventData)
	if err != nil {
		return eventsourcing.Event[T]{}, errors.New(fmt.Sprintf("could not deserialize event data, %v", err))
	}
	event := eventsourcing.Event[T]{
		AggregateID:   bEvent.AggregateID,
		AggregateType: bEvent.AggregateType,
		Version:       eventsourcing.Version(bEvent.Version),
		GlobalVersion: eventsourcing.Version(bEvent.GlobalVersion),
		Timestamp:     bEvent.Timestamp,
		Metadata:      bEvent.Metadata,
		Data:          eventData,
	}
	return event, nil
}
