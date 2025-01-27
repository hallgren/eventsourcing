package internal

import (
	"reflect"

	"github.com/hallgren/eventsourcing/core"
)

type registerFunc = func() interface{}

type register struct {
	eventsF    map[string]registerFunc
	aggregates map[string]struct{}
}

// Aggregate interface to use the aggregate root specific methods
type aggregate interface {
	Register(func(events ...interface{}))
}

// GlobalRegister keeps track of registered aggregates and events
var GlobalRegister = newRegister()

// ResetRegister reset the event regsiter
func ResetRegister() {
	GlobalRegister = newRegister()
}

func newRegister() *register {
	return &register{
		eventsF:    make(map[string]registerFunc),
		aggregates: make(map[string]struct{}),
	}
}

// aggregateRegistered return true if the aggregate is registered
func (r *register) AggregateRegistered(a aggregate) bool {
	typ := aggregateType(a)
	_, ok := r.aggregates[typ]
	return ok
}

// EventRegistered return the func to generate the correct event data type and true if it exists
// otherwise false.
func (r *register) EventRegistered(event core.Event) (registerFunc, bool) {
	d, ok := r.eventsF[event.AggregateType+"_"+event.Reason]
	return d, ok
}

// Register store the aggregate and calls the aggregate method Register to Register the aggregate events.
func (r *register) Register(a aggregate) {
	typ := reflect.TypeOf(a).Elem().Name()
	fu := r.RegisterAggregate(typ)
	a.Register(fu)
}

func (r *register) RegisterAggregate(aggregateType string) func(events ...interface{}) {
	r.aggregates[aggregateType] = struct{}{}

	// fe is a helper function to make the event type registration simpler
	fe := func(events ...interface{}) []registerFunc {
		res := []registerFunc{}
		for _, e := range events {
			res = append(res, eventToFunc(e))
		}
		return res
	}

	return func(events ...interface{}) {
		eventsF := fe(events...)
		for _, f := range eventsF {
			event := f()
			reason := reflect.TypeOf(event).Elem().Name()
			r.eventsF[aggregateType+"_"+reason] = f
		}
	}
}

func eventToFunc(event interface{}) registerFunc {
	return func() interface{} {
		// return a new instance of the event
		return reflect.New(reflect.TypeOf(event).Elem()).Interface()
	}
}

func aggregateType(a aggregate) string {
	return reflect.TypeOf(a).Elem().Name()
}
