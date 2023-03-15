module github.com/hallgren/eventsourcing/eventstore/bbolt

go 1.18

require (
	github.com/hallgren/eventsourcing v0.0.20
	go.etcd.io/bbolt v1.3.6
)

require golang.org/x/sys v0.3.0 // indirect

//replace github.com/hallgren/eventsourcing => ../..
