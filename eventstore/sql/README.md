The sql eventstore is a module containing multiple sql based event stores.

* SQLite
  * NewSQLite(db *sql.DB) (*SQLite, error) 
  * NewSQLiteSingelWriter(db *sql.DB) (*SQLite, error)
* Postgres
  * NewPostgres(db *sql.DB) (*Postgres, error)
