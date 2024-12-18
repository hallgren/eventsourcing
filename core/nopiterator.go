package core

// ZeroIterator returns no data
func ZeroIterator() Iterator {
	return func(yield func(Event, error) bool) {}
}
