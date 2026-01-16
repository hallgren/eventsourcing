GENERIC REFACTOR PLAN (v2)

This file describes the stepwise plan for the generics v2 refactor being implemented on branch `generics-v2`.

Scope
- Full, breaking refactor to use Go generics across the public API.
- Module path will be bumped to github.com/hallgren/eventsourcing/v2 and the repo will target Go 1.20.

Goals
1. Provide compile-time typed events for aggregates using Go generics (Event[T], Root[Ev], etc.).
2. Keep persistence representation `core.Event` unchanged (Data/Metadata []byte) and convert between typed events and core.Event at the eventstore boundary.
3. Provide migration docs and examples and publish as v2.

Planned commits (small, reviewable):
1. Add this plan file (you are here).
2. Update go.mod (module path -> /v2, go 1.20).
3. Introduce eventsourcing generic Event[T] and helpers (NewEvent, DataAs, etc.).
4. Convert aggregate.Root to generic Root[Ev] and TrackChange helpers.
5. PoC: update memory eventstore package to convert typed events to core.Event and back. Update tests/examples.
6. Convert remaining eventstores: sql, bbolt, esdb, kurrent.
7. Update registry to support typed deserialization runtime checks.
8. Update examples, tests, docs, CI to Go 1.20.

Validation strategy
- After each commit run `go test ./...` and fix compile errors.
- Smoke test examples in example/.

Rollback plan
- If issues appear we can revert the branch and iterate on the PoC approach.

Notes
- This file is the first commit on branch generics-v2. Subsequent commits will be small and focused to keep reviews easy.
