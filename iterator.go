package eventsourcing

import (
	"github.com/hallgren/eventsourcing/core"
	"github.com/hallgren/eventsourcing/internal"
)

// Iterator to stream events to reduce memory foot print
type Iterator struct {
	CoreIterator core.Iterator
}

// Close the underlaying iterator
func (i *Iterator) Close() {
	i.CoreIterator.Close()
}

func (i *Iterator) Next() bool {
	return i.CoreIterator.Next()
}

func (i *Iterator) Value() (Event, error) {
	event, err := i.CoreIterator.Value()
	if err != nil {
		return Event{}, err
	}
	// apply the event to the aggregate
	f, found := internal.GlobalRegister.EventRegistered(event)
	if !found {
		return Event{}, fmt.Errorf("event not registered, aggregate type: %s, reason: %s, global version: %d, %w", event.AggregateType(), event.event.Reason, event.GlobalVersion(), ErrEventNotRegistered)
	}
	data := f()
	err = internal.EventEncoder.Deserialize(event.Data, &data)
	if err != nil {
		return Event{}, err
	}
	metadata := make(map[string]interface{})
	if event.Metadata != nil {
		err = internal.EventEncoder.Deserialize(event.Metadata, &metadata)
		if err != nil {
			return Event{}, err
		}
	}
	return Event{
		event:    event,
		data:     data,
		metadata: metadata,
	}, nil
}
