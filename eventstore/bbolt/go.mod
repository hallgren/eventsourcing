module github.com/hallgren/eventsourcing/eventstore/bbolt

go 1.21

toolchain go1.22.5

require (
	github.com/hallgren/eventsourcing/core v0.4.0
	go.etcd.io/bbolt v1.3.10
)

require golang.org/x/sys v0.22.0 // indirect

// replace github.com/hallgren/eventsourcing/core => ../../core
