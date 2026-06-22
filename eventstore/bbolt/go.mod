module github.com/hallgren/eventsourcing/eventstore/bbolt

go 1.25.0

require (
	github.com/hallgren/eventsourcing/core v0.5.2
	go.etcd.io/bbolt v1.5.0
)

require golang.org/x/sys v0.45.0 // indirect

// replace github.com/hallgren/eventsourcing/core => ../../core
