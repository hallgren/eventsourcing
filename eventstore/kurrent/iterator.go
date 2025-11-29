package kurrent

import (
	"strings"

	"github.com/hallgren/eventsourcing/core"
	"github.com/kurrent-io/KurrentDB-Client-Go/kurrentdb"
)

type Iterator struct {
	Stream *kurrentdb.ReadStream
	event  *kurrentdb.ResolvedEvent
}

// Close closes the stream
func (i *Iterator) Close() {
	i.Stream.Close()
}

// Next steps to the next event in the stream
func (i *Iterator) Next() bool {
	if i.Stream == nil {
		return false
	}
	event, err := i.Stream.Recv()
	if err != nil {
		return false
	}
	i.event = event
	return true
}

// Value returns the event from the stream
func (i *Iterator) Value() (core.Event, error) {
	stream := strings.Split(i.event.Event.StreamID, streamSeparator)

	event := core.Event{
		AggregateID:   stream[1],
		Version:       core.Version(i.event.Event.EventNumber) + 1, // +1 as the eventsourcing Version starts on 1 but the kurrent event version starts on 0
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
