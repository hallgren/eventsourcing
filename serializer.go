package eventsourcing

import (
	"errors"
	"reflect"
	"strings"
)

type eventFunc[T any] func() T
type MarshalSnapshotFunc func(v any) ([]byte, error)
type UnmarshalSnapshotFunc func(data []byte, v any) error

// Serializer for json serializes
type Serializer[T any] struct {
	eventRegister map[string]eventFunc[T]
	marshal       MarshalSnapshotFunc
	unmarshal     UnmarshalSnapshotFunc
}

// NewSerializer returns a json Handle
func NewSerializer[T any](marshalF MarshalSnapshotFunc, unmarshalF UnmarshalSnapshotFunc) *Serializer[T] {
	return &Serializer[T]{
		eventRegister: make(map[string]eventFunc[T]),
		marshal:       marshalF,
		unmarshal:     unmarshalF,
	}
}

var (
	// ErrAggregateNameMissing return if aggregate name is missing
	ErrAggregateNameMissing = errors.New("missing aggregate name")

	// ErrNoEventsToRegister return if no events to register
	ErrNoEventsToRegister = errors.New("no events to register")

	// ErrEventNameMissing return if Event name is missing
	ErrEventNameMissing = errors.New("missing event name")
)

func event[T any](event T) eventFunc[T] {
	return func() T { return event }
}

// Events is a helper function to make the event type registration simpler
func (h *Serializer[T]) Events(events ...T) []eventFunc[T] {
	res := []eventFunc[T]{}
	for _, e := range events {
		res = append(res, event(e))
	}
	return res
}

// Register will hold a map of aggregate_event to be able to set the currect type when
// the data is unmarhaled.
func (h *Serializer[T]) Register(aggregate Aggregate[T], events []eventFunc[T]) error {
	typ := reflect.TypeOf(aggregate).Elem().Name()
	if typ == "" {
		return ErrAggregateNameMissing
	}
	if i := strings.Index(typ, "["); i != -1 {
		typ = typ[:i]
	}

	if len(events) == 0 {
		return ErrNoEventsToRegister
	}

	for _, f := range events {
		event := f()
		reason := reflect.TypeOf(event).Elem().Name()
		if reason == "" {
			return ErrEventNameMissing
		}
		h.eventRegister[typ+"_"+reason] = f
	}
	return nil
}

// RegisterTypes events aggregate
func (h *Serializer[T]) RegisterTypes(aggregate Aggregate[T], events ...eventFunc[T]) error {
	return h.Register(aggregate, events)
}

// Type return a struct from the registry
func (h *Serializer[T]) Type(typ, reason string) (eventFunc[T], bool) {
	d, ok := h.eventRegister[typ+"_"+reason]
	return d, ok
}

// Marshal pass the request to the under laying Marshal method
func (h *Serializer[T]) Marshal(v any) ([]byte, error) {
	return h.marshal(v)
}

// Unmarshal pass the request to the under laying Unmarshal method
func (h *Serializer[T]) Unmarshal(data []byte, v any) error {
	return h.unmarshal(data, v)
}
