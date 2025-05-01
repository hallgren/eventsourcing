package kurrent

import (
	"context"

	"github.com/hallgren/eventsourcing/core"
	"github.com/kurrent-io/KurrentDB-Client-Go/kurrentdb"
)

const streamSeparator = "-"

// ESDB is the event store handler
type ESDB struct {
	client      *kurrentdb.Client
	contentType kurrentdb.ContentType
}

// Open binds the event store db client
func Open(client *kurrentdb.Client, jsonSerializer bool) *ESDB {
	// defaults to binary
	var contentType kurrentdb.ContentType
	if jsonSerializer {
		contentType = kurrentdb.ContentTypeJson
	}
	return &ESDB{
		client:      client,
		contentType: contentType,
	}
}

// Save persists events to the database
func (es *ESDB) Save(events []core.Event) error {
	// If no event return no error
	if len(events) == 0 {
		return nil
	}

	var streamOptions kurrentdb.AppendToStreamOptions
	aggregateID := events[0].AggregateID
	aggregateType := events[0].AggregateType
	version := events[0].Version
	stream := stream(aggregateType, aggregateID)
	esdbEvents := make([]kurrentdb.EventData, len(events))

	for i, event := range events {
		eventData := kurrentdb.EventData{
			ContentType: es.contentType,
			EventType:   event.Reason,
			Data:        event.Data,
			Metadata:    event.Metadata,
		}

		esdbEvents[i] = eventData
	}

	if version > 1 {
		// StreamRevision value -2 due to version in the eventsourcing pkg start on 1 but in esdb on 0
		// and also the AppendToStream streamOptions expected revision is one version before the first appended event.
		streamOptions.StreamState = kurrentdb.StreamRevision{Value: uint64(version) - 2}
	} else if version == 1 {
		streamOptions.StreamState = kurrentdb.NoStream{}
	}
	wr, err := es.client.AppendToStream(context.Background(), stream, streamOptions, esdbEvents...)
	if err != nil {
		if err, ok := kurrentdb.FromError(err); !ok {
			if err.Code() == kurrentdb.ErrorCodeWrongExpectedVersion {
				// return typed error if version is not the expected.
				return core.ErrConcurrency
			}
		}
		return err
	}
	for i := range events {
		// Set all events GlobalVersion to the last events commit position.
		events[i].GlobalVersion = core.Version(wr.CommitPosition)
	}
	return nil
}

func (es *ESDB) Get(ctx context.Context, id string, aggregateType string, afterVersion core.Version) (core.Iterator, error) {
	streamID := stream(aggregateType, id)

	from := kurrentdb.StreamRevision{Value: uint64(afterVersion)}
	stream, err := es.client.ReadStream(ctx, streamID, kurrentdb.ReadStreamOptions{From: from}, ^uint64(0))
	if err != nil {
		if err, ok := kurrentdb.FromError(err); !ok {
			if err.Code() == kurrentdb.ErrorCodeResourceNotFound {
				return core.ZeroIterator{}, nil
			}
		}
		return nil, err
	}
	return &iterator{stream: stream}, nil
}

func stream(aggregateType, aggregateID string) string {
	return aggregateType + streamSeparator + aggregateID
}
