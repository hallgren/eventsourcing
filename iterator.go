package eventsourcing

import (
	"github.com/hallgren/eventsourcing/core"
)

// Iterator to stream events to reduce memory foot print
type Iterator struct {
	iterator core.Iterator
	er       *EventRepository
}

// Close the underlaying iterator
func (i *Iterator) Close() {
	i.iterator.Close()
}

func (i *Iterator) Next() bool {
	return i.iterator.Next()
}

func (i *Iterator) Value() (Event, error) {
	event, err := i.iterator.Value()
	if err != nil {
		return Event{}, err
	}
	// apply the event to the aggregate
	f, found := i.er.register.EventRegistered(event)
	if !found {
		return Event{}, ErrEventNotRegistered
	}
	data := f()
	err = i.er.encoder.Deserialize(event.Data, &data)
	if err != nil {
		return Event{}, err
	}
	metadata := make(map[string]interface{})
	if event.Metadata != nil {
		err = i.er.encoder.Deserialize(event.Metadata, &metadata)
		if err != nil {
			return Event{}, err
		}
	}
	return NewEvent(event, data, metadata), nil
}
