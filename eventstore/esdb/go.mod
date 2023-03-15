module github.com/hallgren/eventsourcing/eventstore/esdb

go 1.18

require (
	github.com/EventStore/EventStore-Client-Go/v3 v3.0.0
	github.com/hallgren/eventsourcing v0.0.20
)

require (
	github.com/gofrs/uuid v4.2.0+incompatible // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	golang.org/x/net v0.0.0-20220425223048-2871e0cb64e4 // indirect
	golang.org/x/sys v0.0.0-20220503163025-988cb79eb6c6 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20220502173005-c8bf987b8c21 // indirect
	google.golang.org/grpc v1.46.0 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
)

//replace github.com/hallgren/eventsourcing => ../..
