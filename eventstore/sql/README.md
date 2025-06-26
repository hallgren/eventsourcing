The sql eventstore is a module containing multiple sql based event stores.

* sqlite
  * NewSQLite(db *sql.DB) (*SQLite, error) 
  * NewSQLiteSingelWriter(db *sql.DB) (*SQLite, error)
* postgres
  * NewPostgres(db *sql.DB) (*Postgres, error)
