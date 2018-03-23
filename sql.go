package honeycomb

import (
	"database/sql"

	libhoney "github.com/honeycombio/libhoney-go"
)

type DB struct {
	*sql.DB
	builder *libhoney.Builder
	// events will be a map of in-flight events for transactions, but that's not implemented yet.
	// events  map[int]*libhoney.Event
}

func WrapDB(b *libhoney.Builder, s *sql.DB) *DB {
	return &DB{
		DB:      s,
		builder: b,
	}
}

// func (db *DB) Begin() (*Tx, error)                                       {}
// func (db *DB) BeginTx(ctx context.Context, opts *TxOptions) (*Tx, error) {}
// func (db *DB) Close() error                                              {}
// func (db *DB) Conn(ctx context.Context) (*Conn, error)                   {}
// func (db *DB) Driver() driver.Driver {}

func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	ev := db.builder.NewEvent()
	defer ev.Send()
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	res, err := db.DB.Exec(query, args...)

	if err != nil {
		ev.AddField("error", err)
	} else {
		id, lierr := res.LastInsertId()
		if lierr == nil {
			ev.AddField("last_insert_id", id)
		}
		numrows, nrerr := res.RowsAffected()
		if nrerr == nil {
			ev.AddField("rows_affected", numrows)
		}
	}
	return res, err
}

// func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (Result, error) {}
// func (db *DB) Ping() error                                                                        {}
// func (db *DB) PingContext(ctx context.Context) error                                              {}
// func (db *DB) Prepare(query string) (*Stmt, error)                                                {}
// func (db *DB) PrepareContext(ctx context.Context, query string) (*Stmt, error)                    {}
// func (db *DB) Query(query string, args ...interface{}) (*Rows, error)                             {}
// func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*Rows, error) {}
// func (db *DB) QueryRow(query string, args ...interface{}) *Row                                    {}
// func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *Row        {}
// func (db *DB) SetConnMaxLifetime(d time.Duration)                                                 {}
// func (db *DB) SetMaxIdleConns(n int)                                                              {}
// func (db *DB) SetMaxOpenConns(n int)                                                              {}
// func (db *DB) Stats() DBStats                                                                     {}
