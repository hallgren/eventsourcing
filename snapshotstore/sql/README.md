The sql is a module containing multiple sql based snapshot stores.

* SQLite
  * NewSQLite(db *sql.DB) (*SQLite, error) 
* Postgres
  * NewPostgres(db *sql.DB) (*Postgres, error)
