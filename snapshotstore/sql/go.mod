module github.com/hallgren/eventsourcing/snapshotstore/sql

go 1.13

require (
	github.com/go-sql-driver/mysql v1.7.0 // indirect
	github.com/hallgren/eventsourcing v0.0.20
	github.com/mattn/go-sqlite3 v1.14.16 // indirect
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/onsi/gomega v1.27.4 // indirect
	github.com/proullon/ramsql v0.0.0-20211120092837-c8d0a408b939
	github.com/ziutek/mymysql v1.5.4 // indirect
)

//replace github.com/hallgren/eventsourcing => ../..
