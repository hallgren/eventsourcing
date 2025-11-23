package core

// FetchFunc is the event fetch function used by the projections
type FetchFunc func() (Iterator, error)
