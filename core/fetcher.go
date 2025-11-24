package core

// Fetcher is the event fetch function concumed by projections
type Fetcher func() (Iterator, error)
