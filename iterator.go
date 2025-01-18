package eventsourcing

import (
	"github.com/hallgren/eventsourcing/core"
)

// iterator to stream events to reduce memory foot print
type iterator struct {
	iterator core.Iterator
}

// close the underlaying iterator
func (i *iterator) close() {
	i.iterator.Close()
}

func (i *iterator) next() bool {
	return i.iterator.Next()
}

func (i *iterator) value() (Event, error) {
	event, err := i.iterator.Value()
	if err != nil {
		return Event{}, err
	}
	// apply the event to the aggregate
	f, found := globalRegister.eventRegistered(event)
	if !found {
		return Event{}, ErrEventNotRegistered
	}
	data := f()
	err = encoder.Deserialize(event.Data, &data)
	if err != nil {
		return Event{}, err
	}
	metadata := make(map[string]interface{})
	if event.Metadata != nil {
		err = encoder.Deserialize(event.Metadata, &metadata)
		if err != nil {
			return Event{}, err
		}
	}
	return NewEvent(event, data, metadata), nil
}
