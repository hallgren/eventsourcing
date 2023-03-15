package eventsourcing

import (
	"encoding/json"
	"errors"
	"reflect"
	"time"
)

// ErrNoEvents when there is no events to get
var ErrNoEvents = errors.New("no events")

// ErrNoMoreEvents when iterator has no more events to deliver
var ErrNoMoreEvents = errors.New("no more events")

// Event holding meta data and the application specific event in the Data property
type Event[T any] struct {
	AggregateID   string
	Version       Version
	GlobalVersion Version
	AggregateType string
	Timestamp     time.Time
	Data          T
	Metadata      map[string]interface{}
}

// Reason returns the name of the data struct
func (e Event[T]) Reason() string {
	if any(e.Data) == any(nil) {
		return ""
	}
	return reflect.TypeOf(e.Data).Elem().Name()
}

// DataAs convert the event.Data to the supplied type.
func (e Event[T]) DataAs(i interface{}) error {
	b, err := json.Marshal(e.Data)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, i)
}
