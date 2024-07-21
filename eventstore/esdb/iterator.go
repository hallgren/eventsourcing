package esdb

import (
	"strings"

	"github.com/EventStore/EventStore-Client-Go/v4/esdb"
	"github.com/hallgren/eventsourcing/core"
)

type iterator struct {
	stream *esdb.ReadStream
	event  *esdb.ResolvedEvent
}

// Close closes the stream
func (i *iterator) Close() {
	i.stream.Close()
}

// Next steps to the next event in the stream
func (i *iterator) Next() bool {
	eventESDB, err := i.stream.Recv()
	if err != nil {
		return false
	}
	i.event = eventESDB
	return true
}

// Value returns the event from the stream
func (i *iterator) Value() (core.Event, error) {
	stream := strings.Split(i.event.Event.StreamID, streamSeparator)

	event := core.Event{
		AggregateID:   stream[1],
		Version:       core.Version(i.event.Event.EventNumber) + 1, // +1 as the eventsourcing Version starts on 1 but the esdb event version starts on 0
		AggregateType: stream[0],
		Timestamp:     i.event.Event.CreatedDate,
		Data:          i.event.Event.Data,
		Metadata:      i.event.Event.UserMetadata,
		Reason:        i.event.Event.EventType,
		// Can't get the global version when using the ReadStream method
		//GlobalVersion: core.Version(event.Event.Position.Commit),
	}
	return event, nil
}
