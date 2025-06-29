## BBolt Event Store

This event store supports the go.etcd.io/bbolt bolt driver.

### Constructor

```go
// New opens the event stream found in the given file. If the file is not found it will be created and
// initialized. Will return error if it has problems persisting the changes to the filesystem.
func New(dbFile string) (*BBolt, error) {
```

### Example of use

```go
import "github.com/hallgren/eventsourcing/eventstore/bbolt"

bboltEventStore, err := bbolt.New("bolt.db")
if err != nil {
	return nil, nil, err
}
```
