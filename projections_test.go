package eventsourcing_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/hallgren/eventsourcing"
	"github.com/hallgren/eventsourcing/core"
	"github.com/hallgren/eventsourcing/eventstore/memory"
)

func createPersonEvent(es *memory.Memory, name string, age int) error {
	person, err := CreatePerson(name)
	if err != nil {
		return err
	}

	for i := 0; i < age; i++ {
		person.GrowOlder()
	}

	events := make([]core.Event, 0)
	for _, e := range person.Events() {
		data, err := json.Marshal(e.Data())
		if err != nil {
			return err
		}

		events = append(events, core.Event{
			AggregateID:   e.AggregateID(),
			Reason:        e.Reason(),
			AggregateType: e.AggregateType(),
			Version:       core.Version(e.Version()),
			GlobalVersion: core.Version(e.GlobalVersion()),
			Timestamp:     e.Timestamp(),
			Data:          data,
		})
	}
	return es.Save(events)
}

func TestRunOnce(t *testing.T) {
	// setup
	es := memory.Create()
	eventsourcing.AggregateRegister(&Person{})

	projectedName := ""

	err := createPersonEvent(es, "kalle", 0)
	if err != nil {
		t.Fatal(err)
	}

	err = createPersonEvent(es, "anka", 0)
	if err != nil {
		t.Fatal(err)
	}

	// run projection one event at each run
	proj := eventsourcing.NewProjection(es.All(0, 1), func(event eventsourcing.Event) error {
		switch e := event.Data().(type) {
		case *Born:
			projectedName = e.Name
		}
		return nil
	})

	// should set projectedName to kalle
	work, result := proj.RunOnce()
	if result.Error != nil {
		t.Fatal(result)
	}

	if !work {
		t.Fatal("there was no work to do")
	}
	if projectedName != "kalle" {
		t.Fatalf("expected %q was %q", "kalle", projectedName)
	}

	// should set the projected name to anka
	work, result = proj.RunOnce()
	if result.Error != nil {
		t.Fatal(result.Error)
	}

	if !work {
		t.Fatal("there was no work to do")
	}
	if projectedName != "anka" {
		t.Fatalf("expected %q was %q", "anka", projectedName)
	}
}

func TestRun(t *testing.T) {
	es := memory.Create()
	eventsourcing.AggregateRegister(&Person{})

	projectedName := ""
	sourceName := "kalle"

	err := createPersonEvent(es, sourceName, 1)
	if err != nil {
		t.Fatal(err)
	}

	// run projection
	proj := eventsourcing.NewProjection(es.All(0, 1), func(event eventsourcing.Event) error {
		switch e := event.Data().(type) {
		case *Born:
			projectedName = e.Name
		}
		return nil
	})

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
	defer cancel()

	// will run once then sleep for 10 seconds
	err = proj.Run(ctx, time.Second*10)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}

	if projectedName != sourceName {
		t.Fatalf("expected %q was %q", sourceName, projectedName)
	}
}

func TestRunSameProjectionConcurrently(t *testing.T) {
	// setup
	es := memory.Create()
	eventsourcing.AggregateRegister(&Person{})

	sourceName := "kalle"

	err := createPersonEvent(es, sourceName, 0)
	if err != nil {
		t.Fatal(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	// run projection
	proj := eventsourcing.NewProjection(es.All(0, 1), func(event eventsourcing.Event) error {
		wg.Done()
		return nil
	})

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
	defer cancel()

	// Run the projection
	go func() {
		proj.Run(ctx, time.Second*10)
	}()

	// wait to make sure the projection is already running
	wg.Wait()

	err = proj.Run(ctx, time.Second*10)
	if !errors.Is(err, eventsourcing.ErrProjectionAlreadyRunning) {
		t.Fatal(err)
	}
}

func TestTriggerSync(t *testing.T) {
	// setup
	es := memory.Create()
	eventsourcing.AggregateRegister(&Person{})

	projectedName := ""
	sourceName := "kalle"

	// run projection
	proj := eventsourcing.NewProjection(es.All(0, 1), func(event eventsourcing.Event) error {
		switch e := event.Data().(type) {
		case *Born:
			projectedName = e.Name
		}
		return nil
	})

	group := eventsourcing.NewProjectionGroup(proj)
	group.Start()
	defer group.Stop()

	// make sure the projection has finished it's first round
	time.Sleep(time.Millisecond * 10)

	// create the event after the projection is started as the projection would have consume it.
	err := createPersonEvent(es, sourceName, 1)
	if err != nil {
		t.Fatal(err)
	}

	// check projection is not updated before trigger
	if projectedName == sourceName {
		t.Fatalf("expected projected name to differ: %q was %q", sourceName, projectedName)
	}

	// trigger the projection
	group.TriggerSync()

	// check that the projected value is updated
	if projectedName != sourceName {
		t.Fatalf("expected projected name: %q was %q", sourceName, projectedName)
	}
}

func TestTriggerAsync(t *testing.T) {
	// setup
	es := memory.Create()
	eventsourcing.AggregateRegister(&Person{})

	projectedName := ""
	sourceName := "kalle"

	wg := sync.WaitGroup{}
	wg.Add(1)

	// run projection
	proj := eventsourcing.NewProjection(es.All(0, 1), func(event eventsourcing.Event) error {
		switch e := event.Data().(type) {
		case *Born:
			projectedName = e.Name
		}
		wg.Done()
		return nil
	})

	group := eventsourcing.NewProjectionGroup(proj)
	group.Start()
	defer group.Stop()

	// make sure the projection has finished it's first round
	time.Sleep(time.Millisecond * 10)

	// create the event after the projection is started as the projection would have consume it.
	err := createPersonEvent(es, sourceName, 0)
	if err != nil {
		t.Fatal(err)
	}

	// check projection is not updated before trigger
	if projectedName == sourceName {
		t.Fatalf("expected projected name to differ: %q was %q", sourceName, projectedName)
	}

	// trigger the projection
	group.TriggerAsync()
	group.TriggerAsync()
	group.TriggerAsync()

	// wait until the async trigger has finished
	wg.Wait()

	// check that the projected value is updated
	if projectedName != sourceName {
		t.Fatalf("expected projected name: %q was %q", sourceName, projectedName)
	}
}

func TestCloseEmptyGroup(t *testing.T) {
	g := eventsourcing.NewProjectionGroup()
	g.Stop()
	g.Start()
	g.Stop()
	g.Stop()
}

func TestStartMultipleProjections(t *testing.T) {
	// setup
	es := memory.Create()
	eventsourcing.AggregateRegister(&Person{})

	// callback that handles the events
	callbackF := func(event eventsourcing.Event) error {
		return nil
	}

	r1 := eventsourcing.NewProjection(es.All(0, 1), callbackF)
	r2 := eventsourcing.NewProjection(es.All(0, 1), callbackF)
	r3 := eventsourcing.NewProjection(es.All(0, 1), callbackF)

	g := eventsourcing.NewProjectionGroup(r1, r2, r3)
	g.Start()
	g.Stop()
}

func TestErrorFromCallback(t *testing.T) {
	// setup
	es := memory.Create()
	eventsourcing.AggregateRegister(&Person{})

	err := createPersonEvent(es, "kalle", 1)
	if err != nil {
		t.Fatal(err)
	}

	// define application error that can be returned from the callback function
	var ErrApplication = errors.New("application error")

	// callback that handles the events
	callbackF := func(event eventsourcing.Event) error {
		return ErrApplication
	}

	r := eventsourcing.NewProjection(es.All(0, 1), callbackF)

	g := eventsourcing.NewProjectionGroup(r)

	g.Start()
	defer g.Stop()

	select {
	case err = <-g.ErrChan:
	case <-time.After(time.Second):
		t.Fatal("test timed out")
	}

	if !errors.Is(err, ErrApplication) {
		if err != nil {
			t.Fatalf("expected application error but got %s", err.Error())
		}
		t.Fatal("got none error expected ErrApplication")
	}
}

func TestStrict(t *testing.T) {
	// setup
	es := memory.Create()
	eventsourcing.ResetRegsiter()

	// We do not register the Person aggregate with the Born event attached
	err := createPersonEvent(es, "kalle", 1)
	if err != nil {
		t.Fatal(err)
	}

	proj := eventsourcing.NewProjection(es.All(0, 1), func(event eventsourcing.Event) error {
		return nil
	})

	_, result := proj.RunOnce()
	if !errors.Is(result.Error, eventsourcing.ErrEventNotRegistered) {
		t.Fatalf("expected ErrEventNotRegistered got %q", err.Error())
	}
}

func TestRace(t *testing.T) {
	// setup
	es := memory.Create()
	eventsourcing.AggregateRegister(&Person{})

	err := createPersonEvent(es, "kalle", 50)
	if err != nil {
		t.Fatal(err)
	}

	// callback that handles the events
	callbackF := func(event eventsourcing.Event) error {
		time.Sleep(time.Millisecond * 2)
		return nil
	}

	applicationErr := errors.New("an error")

	r1 := eventsourcing.NewProjection(es.All(0, 1), callbackF)
	r2 := eventsourcing.NewProjection(es.All(0, 1), func(e eventsourcing.Event) error {
		time.Sleep(time.Millisecond)
		if e.GlobalVersion() == 31 {
			return applicationErr
		}
		return nil
	})

	result, err := eventsourcing.Race(true, r1, r2)

	// causing err should be applicationErr
	if !errors.Is(err, applicationErr) {
		t.Fatalf("expected causing error to be applicationErr got %v", err)
	}

	// projection 0 should have a context.Canceled error
	if !errors.Is(result[0].Error, context.Canceled) {
		t.Fatalf("expected projection %q to have err 'context.Canceled' got %v", result[0].Name, result[0].Error)
	}

	// projection 1 should have a applicationErr error
	if !errors.Is(result[1].Error, applicationErr) {
		t.Fatalf("expected projection %q to have err 'applicationErr' got %v", result[1].Name, result[1].Error)
	}

	// projection 1 should have halted on event with GlobalVersion 30
	if result[1].LastHandledEvent.GlobalVersion() != 30 {
		t.Fatalf("expected projection 1 Event.GlobalVersion() to be 30 but was %d", result[1].LastHandledEvent.GlobalVersion())
	}
}

func TestKeepStartPosition(t *testing.T) {
	// setup
	es := memory.Create()
	eventsourcing.AggregateRegister(&Person{})

	err := createPersonEvent(es, "kalle", 5)
	if err != nil {
		t.Fatal(err)
	}

	start := core.Version(0)
	counter := 0

	// callback that handles the events
	callbackF := func(event eventsourcing.Event) error {
		switch event.Data().(type) {
		case *AgedOneYear:
			counter++
		}
		start = core.Version(event.GlobalVersion() + 1)
		return nil
	}

	r := eventsourcing.NewProjection(es.All(0, 1), callbackF)

	_, err = eventsourcing.Race(true, r)
	if err != nil {
		t.Fatal(err)
	}

	err = createPersonEvent(es, "anka", 5)
	if err != nil {
		t.Fatal(err)
	}

	_, err = eventsourcing.Race(true, r)
	if err != nil {
		t.Fatal(err)
	}

	// Born 2 + AgedOnYear 5 + 5 = 12 + Next Event 1 = 13
	if start != 13 {
		t.Fatalf("expected start to be 13 was %d", start)
	}

	if counter != 10 {
		t.Fatalf("expected counter to be 10 was %d", counter)
	}
}
