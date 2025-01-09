package eventsourcing

import (
	"reflect"

	"github.com/hallgren/eventsourcing/core"
)

type agg interface {
	Register(RegisterFunc)
}

type registerFunc = func() interface{}
type RegisterFunc = func(events ...interface{})

type Register struct {
	aggregateEvents map[string]registerFunc
	aggregates      map[string]struct{}
}

func NewRegister() *Register {
	return &Register{
		aggregateEvents: make(map[string]registerFunc),
		aggregates:      make(map[string]struct{}),
	}
}

// EventRegistered return the func to generate the correct event data type and true if it exists
// otherwise false.
func (r *Register) EventRegistered(event core.Event) (registerFunc, bool) {
	d, ok := r.aggregateEvents[event.AggregateType+"_"+event.Reason]
	return d, ok
}

// Register store the aggregate and calls the aggregate method Register to register the aggregate events.
func (r *Register) Register(a agg) {
	typ := reflect.TypeOf(a).Elem().Name()
	fu := r.registerAggregate(typ)
	a.Register(fu)
}

func (r *Register) registerAggregate(aggregateType string) RegisterFunc {
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
			r.aggregateEvents[aggregateType+"_"+reason] = f
		}
	}
}

func eventToFunc(event interface{}) registerFunc {
	return func() interface{} {
		// return a new instance of the event
		return reflect.New(reflect.TypeOf(event).Elem()).Interface()
	}
}
