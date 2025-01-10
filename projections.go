package eventsourcing

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hallgren/eventsourcing/core"
)

type fetchFunc func() (core.Iterator, error)
type callbackFunc func(e Event) error

type ProjectionRepository struct {
	er    *EventRepository
	count int
}

// ErrProjectionAlreadyRunning is returned if Run is called on an already running projection
var ErrProjectionAlreadyRunning = errors.New("projection is already running")

// NewProjectionRepository returns a ProjectionHandler
func NewProjectionRepository(er *EventRepository) *ProjectionRepository {
	return &ProjectionRepository{
		er: er,
	}
}

type Projection struct {
	running   atomic.Bool
	fetchF    fetchFunc
	callbackF callbackFunc
	handler   *ProjectionRepository
	trigger   chan func()
	Strict    bool // Strict indicate if the projection should return error if the event it fetches is not found in the register
	Name      string
}

// Group runs projections concurrently
type Group struct {
	Pace        time.Duration // Pace is used when a projection is running and it reaches the end of the event stream
	handler     *ProjectionRepository
	projections []*Projection
	cancelF     context.CancelFunc
	wg          sync.WaitGroup
	ErrChan     chan error
}

// ProjectionResult is the return type for a Group and Race
type ProjectionResult struct {
	Error            error
	Name             string
	LastHandledEvent Event
}

// Projection creates a projection that will run down an event stream
func (pr *ProjectionRepository) Projection(fetchF fetchFunc, callbackF callbackFunc) *Projection {
	projection := Projection{
		fetchF:    fetchF,
		callbackF: callbackF,
		handler:   pr,
		trigger:   make(chan func()),
		Strict:    true,                        // Default strict is active
		Name:      fmt.Sprintf("%d", pr.count), // Default the name to it's creation index
	}
	pr.count++
	return &projection
}

// TriggerAsync force a running projection to run immediately independent on the pace
// It will return immediately after triggering the prjection to run.
// If the trigger channel is already filled it will return without inserting any value.
func (p *Projection) TriggerAsync() {
	if !p.running.Load() {
		return
	}
	select {
	case p.trigger <- func() {}:
	default:
	}
}

// TriggerSync force a running projection to run immediately independent on the pace
// It will wait for the projection to finish running to its current end before returning.
func (p *Projection) TriggerSync() {
	if !p.running.Load() {
		return
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	f := func() {
		wg.Done()
	}
	p.trigger <- f
	wg.Wait()
}

// Run runs the projection forever until the context is cancelled. When there are no more events to consume it
// waits for a trigger or context cancel.
func (p *Projection) Run(ctx context.Context, pace time.Duration) error {
	if p.running.Load() {
		return ErrProjectionAlreadyRunning
	}
	p.running.Store(true)
	defer func() {
		p.running.Store(false)
	}()

	var noopFunc = func() {}
	var f = noopFunc
	triggerFunc := func() {
		f()
		// reset the f to the noop func
		f = noopFunc
	}
	for {
		result := p.RunToEnd(ctx)
		// if triggered by a sync trigger the triggerFunc callback that it's finished
		// if not triggered by a sync trigger the triggerFunc will call an no ops function
		triggerFunc()
		if result.Error != nil {
			return result.Error
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pace):
		case f = <-p.trigger:
		}
	}
}

// RunToEnd runs until the projection reaches the end of the event stream
func (p *Projection) RunToEnd(ctx context.Context) ProjectionResult {
	var result ProjectionResult
	var lastHandledEvent Event

	for {
		select {
		case <-ctx.Done():
			return ProjectionResult{Error: ctx.Err(), Name: result.Name, LastHandledEvent: result.LastHandledEvent}
		default:
			ran, result := p.RunOnce()
			// if the first event returned error or if it did not run at all
			if result.LastHandledEvent.GlobalVersion() == 0 {
				result.LastHandledEvent = lastHandledEvent
			}
			if result.Error != nil {
				return result
			}
			// hit the end of the event stream
			if !ran {
				return result
			}
			lastHandledEvent = result.LastHandledEvent
		}
	}
}

// RunOnce runs the fetch method one time
func (p *Projection) RunOnce() (bool, ProjectionResult) {
	// ran indicate if there were events to fetch
	var ran bool
	var lastHandledEvent Event

	coreIterator, err := p.fetchF()
	if err != nil {
		return false, ProjectionResult{Error: err, Name: p.Name, LastHandledEvent: lastHandledEvent}
	}
	iterator := &Iterator{
		iterator: coreIterator,
		er:       p.handler.er,
	}
	defer iterator.Close()

	for iterator.Next() {
		ran = true
		event, err := iterator.Value()
		if err != nil {
			if errors.Is(err, ErrEventNotRegistered) {
				if p.Strict {
					err = fmt.Errorf("event not registered aggregate type: %s, reason: %s, global version: %d, %w", event.AggregateType(), event.Reason(), event.GlobalVersion(), ErrEventNotRegistered)
					return false, ProjectionResult{Error: err, Name: p.Name, LastHandledEvent: lastHandledEvent}
				}
				continue
			}
			return false, ProjectionResult{Error: err, Name: p.Name, LastHandledEvent: lastHandledEvent}
		}

		err = p.callbackF(event)
		if err != nil {
			return false, ProjectionResult{Error: err, Name: p.Name, LastHandledEvent: lastHandledEvent}
		}
		// keep a reference to the last successfully handled event
		lastHandledEvent = event
	}
	return ran, ProjectionResult{Error: nil, Name: p.Name, LastHandledEvent: lastHandledEvent}
}

// Group runs a group of projections concurrently
func (ph *ProjectionRepository) Group(projections ...*Projection) *Group {
	return &Group{
		handler:     ph,
		projections: projections,
		cancelF:     func() {},
		Pace:        time.Second * 10, // Default pace 10 seconds
	}
}

// Start starts all projectinos in the group, an error channel i created on the group to notify
// if a result containing an error is returned from a projection
func (g *Group) Start() {
	g.ErrChan = make(chan error)
	ctx, cancel := context.WithCancel(context.Background())
	g.cancelF = cancel

	g.wg.Add(len(g.projections))
	for _, projection := range g.projections {
		go func(p *Projection) {
			defer g.wg.Done()
			err := p.Run(ctx, g.Pace)
			if !errors.Is(err, context.Canceled) {
				g.ErrChan <- err
			}
		}(projection)
	}
}

// TriggerAsync force all projections to run not waiting for them to finish
func (g *Group) TriggerAsync() {
	for _, projection := range g.projections {
		projection.TriggerAsync()
	}
}

// TriggerSync force all projections to run and wait for them to finish
func (g *Group) TriggerSync() {
	wg := sync.WaitGroup{}
	for _, projection := range g.projections {
		wg.Add(1)
		go func(p *Projection) {
			p.TriggerSync()
			wg.Done()
		}(projection)
	}
	wg.Wait()
}

// Stop halts all projections in the group
func (g *Group) Stop() {
	if g.ErrChan == nil {
		return
	}
	g.cancelF()

	// return when all projections has stopped
	g.wg.Wait()

	// close the error channel
	close(g.ErrChan)

	g.ErrChan = nil
}

// Race runs the projections to the end of the events streams.
// Can be used on a stale event stream with no more events coming in or when you want to know when all projections are done.
func (p *ProjectionRepository) Race(cancelOnError bool, projections ...*Projection) ([]ProjectionResult, error) {
	var lock sync.Mutex
	var causingErr error

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(len(projections))

	results := make([]ProjectionResult, len(projections))
	for i, projection := range projections {
		go func(pr *Projection, index int) {
			defer wg.Done()
			result := pr.RunToEnd(ctx)
			if result.Error != nil {
				if !errors.Is(result.Error, context.Canceled) && cancelOnError {
					cancel()

					lock.Lock()
					causingErr = result.Error
					lock.Unlock()
				}
			}
			lock.Lock()
			results[index] = result
			lock.Unlock()
		}(projection, i)
	}
	wg.Wait()
	return results, causingErr
}
