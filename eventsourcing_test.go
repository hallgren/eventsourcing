package eventsourcing_test

/*
func TestGetWithContextCancel(t *testing.T) {
	es := memory.Create()
	aggregate.AggregateRegister(&Person{})

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	err = aggregate.AggregateSave(es, person)
	if err != nil {
		t.Fatal("could not save aggregate")
	}

	ctx, cancel := context.WithCancel(context.Background())

	// cancel the context
	cancel()
	err = eventsourcing.AggregateLoad(ctx, es, person.ID(), person)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected error context.Canceled but was %v", err)
	}
}

func TestSaveWhenAggregateNotRegistered(t *testing.T) {
	es := memory.Create()
	eventsourcing.ResetRegsiter()

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	err = eventsourcing.AggregateSave(es, person)
	if !errors.Is(err, eventsourcing.ErrAggregateNotRegistered) {
		t.Fatalf("could save aggregate that was not registered, err: %v", err)
	}
}

func TestMultipleSave(t *testing.T) {
	es := memory.Create()
	eventsourcing.AggregateRegister(&Person{})

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	err = eventsourcing.AggregateSave(es, person)
	if err != nil {
		t.Fatalf("could not save aggregate, err: %v", err)
	}

	version := person.Version()

	err = eventsourcing.AggregateSave(es, person)
	if err != nil {
		t.Fatalf("save should be a nop, err: %v", err)
	}

	if version != person.Version() {
		t.Fatalf("the nop save should not change the aggregate version exp:%d, actual:%d", version, person.Version())
	}
}

func TestConcurrentRead(t *testing.T) {
	es := memory.Create()
	eventsourcing.AggregateRegister(&Person{})

	person, err := CreatePerson("kalle")
	if err != nil {
		t.Fatal(err)
	}
	err = eventsourcing.AggregateSave(es, person)
	if err != nil {
		t.Fatal("could not save aggregate")
	}
	person2, err := CreatePerson("anka")
	if err != nil {
		t.Fatal(err)
	}
	err = eventsourcing.AggregateSave(es, person2)
	if err != nil {
		t.Fatal("could not save aggregate")
	}

	for i := 1; i <= 1000; i++ {
		p1 := Person{}
		p2 := Person{}
		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			eventsourcing.AggregateLoad(context.Background(), es, person.ID(), &p1)
			wg.Done()
		}()
		go func() {
			eventsourcing.AggregateLoad(context.Background(), es, person2.ID(), &p2)
			wg.Done()
		}()
		wg.Wait()
		if p1.Name == p2.Name {
			t.Fatal("name should differ")
		}
	}
}
*/
