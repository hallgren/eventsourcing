module github.com/hallgren/eventsourcing/eventstore/bbolt

go 1.23

toolchain go1.23.6

require (
	github.com/hallgren/eventsourcing/core v0.5.1
	go.etcd.io/bbolt v1.4.3
)

require golang.org/x/sys v0.29.0 // indirect

replace github.com/hallgren/eventsourcing/core => ../../core
