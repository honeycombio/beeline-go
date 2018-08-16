package hnysqlx

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"

	"github.com/honeycombio/beeline-go/internal"
	"github.com/honeycombio/libhoney-go"
)

type DB struct {
	// wdb is the wrapped sql db. It is not embedded because it's better to fail
	// compilation if some methods are missing than it is to silently not
	// instrument those methods. If you believe that this wraps all methods, it
	// would be reasonable to think that calls that don't show up in Honeycomb
	// aren't happening when they are - they just fell through to the underlying
	// *sql.DB. So all methods present on *sql.DB are recreated here, but as the
	// wrapped package changes, we will fail to compile against apps using those
	// new features and need a patch.
	wdb *sqlx.DB
	// Builder is available in case you wish to add fields to every SQL event
	// that will be created.
	Builder *libhoney.Builder

	Mapper *reflectx.Mapper
}

func WrapDB(s *sqlx.DB) *DB {
	b := libhoney.NewBuilder()
	db := &DB{
		wdb:     s,
		Builder: b,
	}
	addConns := func() interface{} {
		stats := s.DB.Stats()
		return stats.OpenConnections
	}
	b.AddDynamicField("db.open_conns", addConns)
	b.AddField("meta.type", "sqlx")
	return db
}

func (db *DB) Beginx() (*Tx, error) {
	var err error
	ev, sender := internal.BuildDBEvent(db.Builder, "")
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	bld := db.Builder.Clone()
	newid, _ := uuid.NewRandom()
	txid := newid.String()
	wrapTx := &Tx{
		Builder: bld,
	}
	bld.AddField("db.tx_id", txid)
	ev.AddField("db.tx_id", txid)

	// do DB call
	tx, err := db.wdb.Beginx()
	wrapTx.wtx = tx
	return wrapTx, err
}

func (db *DB) BeginTxx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	var err error
	sender := internal.BuildDBSpan(ctx, db.Builder, "")
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	bld := db.Builder.Clone()
	wrapTx := &Tx{
		Builder: bld,
	}
	newid, _ := uuid.NewRandom()
	txid := newid.String()
	bld.AddField("db.tx_id", txid)
	internal.AddField(ctx, "db.tx_id", txid)

	bld.AddField("db.options", opts)
	internal.AddField(ctx, "db.options", opts)

	// do DB call
	tx, err := db.wdb.BeginTxx(ctx, opts)
	wrapTx.wtx = tx
	return wrapTx, err
}

func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	var err error
	ev, sender := internal.BuildDBEvent(db.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	// do DB call
	res, err := db.wdb.Exec(query, args...)

	// capture results
	if err == nil {
		id, lierr := res.LastInsertId()
		if lierr == nil {
			ev.AddField("db.last_insert_id", id)
		}
		numrows, nrerr := res.RowsAffected()
		if nrerr == nil {
			ev.AddField("db.rows_affected", numrows)
		}
	}
	return res, err
}

func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	var err error
	sender := internal.BuildDBSpan(ctx, db.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	// do DB call
	res, err := db.wdb.ExecContext(ctx, query, args...)

	// capture results
	if err == nil {
		id, lierr := res.LastInsertId()
		if lierr == nil {
			internal.AddField(ctx, "db.last_insert_id", id)
		}
		numrows, nrerr := res.RowsAffected()
		if nrerr == nil {
			internal.AddField(ctx, "db.rows_affected", numrows)
		}
	}
	return res, err
}

func (db *DB) Get(dest interface{}, query string, args ...interface{}) error {
	var err error
	ev, sender := internal.BuildDBEvent(db.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	// add the type of the objec being populated
	ev.AddField("db.dest_type", typeof(dest))

	// do DB call
	err = db.wdb.Get(dest, query, args...)
	return err
}

func (db *DB) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	var err error
	sender := internal.BuildDBSpan(ctx, db.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	// add the type of the objec being populated
	internal.AddField(ctx, "db.dest_type", typeof(dest))

	// do DB call
	err = db.wdb.GetContext(ctx, dest, query, args...)
	return err
}

func (db *DB) MapperFunc(mf func(string) string) {
	var err error
	_, sender := internal.BuildDBEvent(db.Builder, "")
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	// update the wrapped DB's Mapper
	db.wdb.MapperFunc(mf)
	// and copy it back here
	if db.wdb.Mapper != nil {
		db.Mapper = db.wdb.Mapper
	}
}

func (db *DB) MustBegin() *Tx {
	var err error
	ev, sender := internal.BuildDBEvent(db.Builder, "")
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	bld := db.Builder.Clone()
	wrapTx := &Tx{
		Builder: bld,
	}
	newid, _ := uuid.NewRandom()
	txid := newid.String()
	bld.AddField("db.tx_id", txid)
	ev.AddField("db.tx_id", txid)

	// do DB call
	tx, err := db.wdb.Beginx()

	wrapTx.wtx = tx

	if err != nil {
		ev.AddField("db.panic", err)
		panic(err)
	}
	return wrapTx
}

func (db *DB) MustBeginTx(ctx context.Context, opts *sql.TxOptions) *Tx {
	var err error
	sender := internal.BuildDBSpan(ctx, db.Builder, "")
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	bld := db.Builder.Clone()
	wrapTx := &Tx{
		Builder: bld,
	}
	newid, _ := uuid.NewRandom()
	txid := newid.String()
	bld.AddField("db.tx_id", txid)
	internal.AddField(ctx, "db.tx_id", txid)

	bld.AddField("db.options", opts)
	internal.AddField(ctx, "db.options", opts)

	// do DB call
	tx, err := db.wdb.BeginTxx(ctx, opts)

	wrapTx.wtx = tx

	// manually wrap the panic in order to report it
	if err != nil {
		internal.AddField(ctx, "db.panic", err)
		panic(err)
	}
	return wrapTx
}

func (db *DB) MustExec(query string, args ...interface{}) sql.Result {
	var err error
	ev, sender := internal.BuildDBEvent(db.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	// do DB call
	res, err := db.wdb.Exec(query, args...)

	// manually wrap the panic in order to report it
	if err != nil {
		ev.AddField("db.panic", err)
		panic(err)
	}

	// capture results
	id, lierr := res.LastInsertId()
	if lierr == nil {
		ev.AddField("db.last_insert_id", id)
	}
	numrows, nrerr := res.RowsAffected()
	if nrerr == nil {
		ev.AddField("db.rows_affected", numrows)
	}

	return res
}

func (db *DB) MustExecContext(ctx context.Context, query string, args ...interface{}) sql.Result {
	var err error
	sender := internal.BuildDBSpan(ctx, db.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	// do DB call
	res, err := db.wdb.ExecContext(ctx, query, args...)

	// manually wrap the panic in order to report it
	if err != nil {
		internal.AddField(ctx, "db.panic", err)
		panic(err)
	}

	id, lierr := res.LastInsertId()
	if lierr == nil {
		internal.AddField(ctx, "db.last_insert_id", id)
	}
	numrows, nrerr := res.RowsAffected()
	if nrerr == nil {
		internal.AddField(ctx, "db.rows_affected", numrows)
	}

	return res
}

func (db *DB) NamedExec(query string, arg interface{}) (sql.Result, error) {
	var err error
	ev, sender := internal.BuildDBEvent(db.Builder, query, arg)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	// do DB call
	res, err := db.wdb.NamedExec(query, arg)

	// capture results
	if err == nil {
		id, lierr := res.LastInsertId()
		if lierr == nil {
			ev.AddField("db.last_insert_id", id)
		}
		numrows, nrerr := res.RowsAffected()
		if nrerr == nil {
			ev.AddField("db.rows_affected", numrows)
		}
	}
	return res, err
}

func (db *DB) NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	var err error
	sender := internal.BuildDBSpan(ctx, db.Builder, query, arg)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	// do DB call
	res, err := db.wdb.NamedExecContext(ctx, query, arg)

	// capture results
	if err == nil {
		id, lierr := res.LastInsertId()
		if lierr == nil {
			internal.AddField(ctx, "db.last_insert_id", id)
		}
		numrows, nrerr := res.RowsAffected()
		if nrerr == nil {
			internal.AddField(ctx, "db.rows_affected", numrows)
		}
	}
	return res, err
}

func (db *DB) NamedQuery(query string, arg interface{}) (*sqlx.Rows, error) {
	var err error
	_, sender := internal.BuildDBEvent(db.Builder, query, arg)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	// do DB call
	rows, err := db.wdb.NamedQuery(query, arg)
	return rows, err
}

func (db *DB) NamedQueryContext(ctx context.Context, query string, arg interface{}) (*sqlx.Rows, error) {
	var err error
	sender := internal.BuildDBSpan(ctx, db.Builder, query, arg)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	// do DB call
	rows, err := db.wdb.NamedQueryContext(ctx, query, arg)
	return rows, err
}

func (db *DB) Ping() error {
	var err error
	_, sender := internal.BuildDBEvent(db.Builder, "")
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	err = db.wdb.Ping()
	return err
}

func (db *DB) PingContext(ctx context.Context) error {
	var err error
	sender := internal.BuildDBSpan(ctx, db.Builder, "")
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	err = db.wdb.Ping()
	return err
}

func (db *DB) PrepareNamed(query string) (*NamedStmt, error) {
	var err error
	ev, sender := internal.BuildDBEvent(db.Builder, query)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	bld := db.Builder.Clone()
	wrapStmt := &NamedStmt{
		Builder: bld,
	}
	newid, _ := uuid.NewRandom()
	stmtid := newid.String()
	bld.AddField("db.stmt_id", stmtid)
	ev.AddField("db.stmt_id", stmtid)

	// add the query to the statement's builder so all executions of this query
	// have it right there
	bld.AddField("db.query", query)

	// do DB call
	stmt, err := db.wdb.PrepareNamed(query)
	wrapStmt.wns = stmt
	return wrapStmt, err
}

func (db *DB) PrepareNamedContext(ctx context.Context, query string) (*NamedStmt, error) {
	var err error
	sender := internal.BuildDBSpan(ctx, db.Builder, query)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	bld := db.Builder.Clone()
	wrapStmt := &NamedStmt{
		Builder: bld,
	}
	newid, _ := uuid.NewRandom()
	stmtid := newid.String()
	bld.AddField("db.stmt_id", stmtid)
	internal.AddField(ctx, "db.stmt_id", stmtid)

	// do DB call
	stmt, err := db.wdb.PrepareNamedContext(ctx, query)
	wrapStmt.wns = stmt
	return wrapStmt, err
}

func (db *DB) Preparex(query string) (*Stmt, error) {
	var err error
	ev, sender := internal.BuildDBEvent(db.Builder, query)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	bld := db.Builder.Clone()
	wrapStmt := &Stmt{
		Builder: bld,
	}
	newid, _ := uuid.NewRandom()
	stmtid := newid.String()
	bld.AddField("db.stmt_id", stmtid)
	ev.AddField("db.stmt_id", stmtid)

	// do DB call
	stmt, err := db.wdb.Preparex(query)
	wrapStmt.wstmt = stmt
	return wrapStmt, err
}

func (db *DB) PreparexContext(ctx context.Context, query string) (*Stmt, error) {
	var err error
	sender := internal.BuildDBSpan(ctx, db.Builder, query)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	bld := db.Builder.Clone()
	wrapStmt := &Stmt{
		Builder: bld,
	}
	newid, _ := uuid.NewRandom()
	stmtid := newid.String()
	bld.AddField("db.stmt_id", stmtid)
	internal.AddField(ctx, "db.stmt_id", stmtid)

	// do DB call
	stmt, err := db.wdb.PreparexContext(ctx, query)
	wrapStmt.wstmt = stmt
	return wrapStmt, err
}

func (db *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	var err error
	_, sender := internal.BuildDBEvent(db.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	// do DB call
	rows, err := db.wdb.Query(query, args...)
	return rows, err
}

func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	var err error
	sender := internal.BuildDBSpan(ctx, db.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	// do DB call
	rows, err := db.wdb.QueryContext(ctx, query, args...)
	return rows, err

}

func (db *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	var err error
	_, sender := internal.BuildDBEvent(db.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	// do DB call
	row := db.wdb.QueryRow(query, args...)
	return row
}

func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	var err error
	sender := internal.BuildDBSpan(ctx, db.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	// do DB call
	row := db.wdb.QueryRowContext(ctx, query, args...)
	return row
}

func (db *DB) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	var err error
	_, sender := internal.BuildDBEvent(db.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	// do DB call
	rows, err := db.wdb.Queryx(query, args...)
	return rows, err
}

func (db *DB) QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	var err error
	sender := internal.BuildDBSpan(ctx, db.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	// do DB call
	rows, err := db.wdb.QueryxContext(ctx, query, args...)
	return rows, err
}

func (db *DB) QueryRowx(query string, args ...interface{}) *sqlx.Row {
	var err error
	_, sender := internal.BuildDBEvent(db.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	// do DB call
	row := db.wdb.QueryRowx(query, args...)
	return row
}

func (db *DB) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	var err error
	sender := internal.BuildDBSpan(ctx, db.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	// do DB call
	row := db.wdb.QueryRowxContext(ctx, query, args...)
	return row
}

func (db *DB) Rebind(query string) string {
	var err error
	_, sender := internal.BuildDBEvent(db.Builder, query)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	str := db.wdb.Rebind(query)
	return str
}

func (db *DB) Select(dest interface{}, query string, args ...interface{}) error {
	var err error
	ev, sender := internal.BuildDBEvent(db.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	ev.AddField("db.dest_type", typeof(dest))

	// do DB call
	err = db.wdb.Select(dest, query, args...)
	return err
}

func (db *DB) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	var err error
	sender := internal.BuildDBSpan(ctx, db.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	internal.AddField(ctx, "db.dest_type", typeof(dest))

	// do DB call
	err = db.wdb.SelectContext(ctx, dest, query, args...)
	return err
}

// not implemented in the wrapper - should just fall through to the superclass
func (db *DB) Close() error {
	var err error
	_, sender := internal.BuildDBEvent(db.Builder, "")
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	err = db.wdb.Close()
	return err

}

func (db *DB) Driver() driver.Driver {
	var err error
	_, sender := internal.BuildDBEvent(db.Builder, "")
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	return db.wdb.Driver()
}

func (db *DB) SetConnMaxLifetime(d time.Duration) {
	var err error
	_, sender := internal.BuildDBEvent(db.Builder, "")
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	db.wdb.SetConnMaxLifetime(d)
}

func (db *DB) SetMaxIdleConns(n int) {
	var err error
	_, sender := internal.BuildDBEvent(db.Builder, "")
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	db.wdb.SetMaxIdleConns(n)
}

func (db *DB) SetMaxOpenConns(n int) {
	var err error
	_, sender := internal.BuildDBEvent(db.Builder, "")
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	db.wdb.SetMaxOpenConns(n)
}

func (db *DB) Stats() sql.DBStats {
	var err error
	_, sender := internal.BuildDBEvent(db.Builder, "")
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if db.Mapper != nil {
		db.wdb.Mapper = db.Mapper
	}

	return db.wdb.Stats()
}

type NamedStmt struct {
	wns     *sqlx.NamedStmt
	Builder *libhoney.Builder
}

func (n *NamedStmt) Close() error {
	var err error
	_, sender := internal.BuildDBEvent(n.Builder, "")
	defer sender(err)

	err = n.wns.Close()
	return err
}

func (n *NamedStmt) Exec(arg interface{}) (sql.Result, error) {
	var err error
	ev, sender := internal.BuildDBEvent(n.Builder, "", arg)
	defer sender(err)

	res, err := n.wns.Exec(arg)

	// capture results
	if err == nil {
		id, lierr := res.LastInsertId()
		if lierr == nil {
			ev.AddField("db.last_insert_id", id)
		}
		numrows, nrerr := res.RowsAffected()
		if nrerr == nil {
			ev.AddField("db.rows_affected", numrows)
		}
	}
	return res, err
}

func (n *NamedStmt) ExecContext(ctx context.Context, arg interface{}) (sql.Result, error) {
	var err error
	sender := internal.BuildDBSpan(ctx, n.Builder, "", arg)
	defer sender(err)

	res, err := n.wns.ExecContext(ctx, arg)

	// capture results
	if err == nil {
		id, lierr := res.LastInsertId()
		if lierr == nil {
			internal.AddField(ctx, "db.last_insert_id", id)
		}
		numrows, nrerr := res.RowsAffected()
		if nrerr == nil {
			internal.AddField(ctx, "db.rows_affected", numrows)
		}
	}
	return res, err
}

func (n *NamedStmt) Get(dest interface{}, arg interface{}) error {
	var err error
	ev, sender := internal.BuildDBEvent(n.Builder, "", arg)
	defer sender(err)

	// add the type of the objec being populated
	ev.AddField("db.dest_type", typeof(dest))

	err = n.wns.Get(dest, arg)
	return err
}

func (n *NamedStmt) GetContext(ctx context.Context, dest interface{}, arg interface{}) error {
	var err error
	sender := internal.BuildDBSpan(ctx, n.Builder, "", arg)
	defer sender(err)

	// add the type of the objec being populated
	internal.AddField(ctx, "db.dest_type", typeof(dest))

	err = n.wns.GetContext(ctx, dest, arg)
	return err
}

func (n *NamedStmt) MustExec(arg interface{}) sql.Result {
	var err error
	ev, sender := internal.BuildDBEvent(n.Builder, "", arg)
	defer sender(err)

	// do DB call
	res, err := n.wns.Exec(arg)

	// manually wrap the panic in order to report it
	if err != nil {
		ev.AddField("db.panic", err)
		panic(err)
	}

	// capture results
	id, lierr := res.LastInsertId()
	if lierr == nil {
		ev.AddField("db.last_insert_id", id)
	}
	numrows, nrerr := res.RowsAffected()
	if nrerr == nil {
		ev.AddField("db.rows_affected", numrows)
	}
	return res
}

func (n *NamedStmt) MustExecContext(ctx context.Context, arg interface{}) sql.Result {
	var err error
	sender := internal.BuildDBSpan(ctx, n.Builder, "", arg)
	defer sender(err)

	// do DB call
	res, err := n.wns.ExecContext(ctx, arg)

	// manually wrap the panic in order to report it
	if err != nil {
		internal.AddField(ctx, "db.panic", err)
		panic(err)
	}

	// capture results
	id, lierr := res.LastInsertId()
	if lierr == nil {
		internal.AddField(ctx, "db.last_insert_id", id)
	}
	numrows, nrerr := res.RowsAffected()
	if nrerr == nil {
		internal.AddField(ctx, "db.rows_affected", numrows)
	}
	return res
}

func (n *NamedStmt) Query(arg interface{}) (*sql.Rows, error) {
	var err error
	_, sender := internal.BuildDBEvent(n.Builder, "", arg)
	defer sender(err)

	// do DB call
	rows, err := n.wns.Query(arg)
	return rows, err
}

func (n *NamedStmt) QueryContext(ctx context.Context, arg interface{}) (*sql.Rows, error) {
	var err error
	sender := internal.BuildDBSpan(ctx, n.Builder, "", arg)
	defer sender(err)

	// do DB call
	rows, err := n.wns.QueryContext(ctx, arg)
	return rows, err
}

func (n *NamedStmt) QueryRow(arg interface{}) *sqlx.Row {
	var err error
	_, sender := internal.BuildDBEvent(n.Builder, "", arg)
	defer sender(err)

	// do DB call
	row := n.wns.QueryRow(arg)
	return row
}

func (n *NamedStmt) QueryRowContext(ctx context.Context, arg interface{}) *sqlx.Row {
	var err error
	sender := internal.BuildDBSpan(ctx, n.Builder, "", arg)
	defer sender(err)

	// do DB call
	row := n.wns.QueryRowContext(ctx, arg)
	return row
}

func (n *NamedStmt) QueryRowx(arg interface{}) *sqlx.Row {
	var err error
	_, sender := internal.BuildDBEvent(n.Builder, "", arg)
	defer sender(err)

	// do DB call
	row := n.wns.QueryRowx(arg)
	return row
}

func (n *NamedStmt) QueryRowxContext(ctx context.Context, arg interface{}) *sqlx.Row {
	var err error
	sender := internal.BuildDBSpan(ctx, n.Builder, "", arg)
	defer sender(err)

	// do DB call
	row := n.wns.QueryRowxContext(ctx, arg)
	return row
}

func (n *NamedStmt) Queryx(arg interface{}) (*sqlx.Rows, error) {
	var err error
	_, sender := internal.BuildDBEvent(n.Builder, "", arg)
	defer sender(err)

	// do DB call
	rows, err := n.wns.Queryx(arg)
	return rows, err
}

func (n *NamedStmt) QueryxContext(ctx context.Context, arg interface{}) (*sqlx.Rows, error) {
	var err error
	sender := internal.BuildDBSpan(ctx, n.Builder, "", arg)
	defer sender(err)

	// do DB call
	rows, err := n.wns.QueryxContext(ctx, arg)
	return rows, err
}

func (n *NamedStmt) Select(dest interface{}, arg interface{}) error {
	var err error
	ev, sender := internal.BuildDBEvent(n.Builder, "", arg)
	defer sender(err)

	ev.AddField("db.dest_type", typeof(dest))

	// do DB call
	err = n.wns.Select(dest, arg)
	return err
}

func (n *NamedStmt) SelectContext(ctx context.Context, dest interface{}, arg interface{}) error {
	var err error
	sender := internal.BuildDBSpan(ctx, n.Builder, "", arg)
	defer sender(err)

	internal.AddField(ctx, "db.dest_type", typeof(dest))

	// do DB call
	err = n.wns.SelectContext(ctx, dest, arg)
	return err
}

func (n *NamedStmt) Unsafe() *NamedStmt {
	var err error
	_, sender := internal.BuildDBEvent(n.Builder, "")
	defer sender(err)

	newws := n.wns.Unsafe()
	n.wns = newws
	return n
}

type Stmt struct {
	wstmt   *sqlx.Stmt
	Builder *libhoney.Builder
	Mapper  *reflectx.Mapper
}

func (s *Stmt) Get(dest interface{}, args ...interface{}) error {
	var err error
	ev, sender := internal.BuildDBEvent(s.Builder, "", args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if s.Mapper != nil {
		s.wstmt.Mapper = s.Mapper
	}

	// add the type of the objec being populated
	ev.AddField("db.dest_type", typeof(dest))

	err = s.wstmt.Get(dest, args...)
	return err
}

func (s *Stmt) GetContext(ctx context.Context, dest interface{}, args ...interface{}) error {
	var err error
	sender := internal.BuildDBSpan(ctx, s.Builder, "", args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if s.Mapper != nil {
		s.wstmt.Mapper = s.Mapper
	}

	// add the type of the objec being populated
	internal.AddField(ctx, "db.dest_type", typeof(dest))

	err = s.wstmt.GetContext(ctx, dest, args...)
	return err
}

func (s *Stmt) MustExec(args ...interface{}) sql.Result {
	var err error
	ev, sender := internal.BuildDBEvent(s.Builder, "", args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if s.Mapper != nil {
		s.wstmt.Mapper = s.Mapper
	}

	// do DB call
	res, err := s.wstmt.Exec(args...)

	// manually wrap the panic in order to report it
	if err != nil {
		ev.AddField("db.panic", err)
		panic(err)
	}

	// capture results
	id, lierr := res.LastInsertId()
	if lierr == nil {
		ev.AddField("db.last_insert_id", id)
	}
	numrows, nrerr := res.RowsAffected()
	if nrerr == nil {
		ev.AddField("db.rows_affected", numrows)
	}
	return res
}

func (s *Stmt) MustExecContext(ctx context.Context, args ...interface{}) sql.Result {
	var err error
	sender := internal.BuildDBSpan(ctx, s.Builder, "", args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if s.Mapper != nil {
		s.wstmt.Mapper = s.Mapper
	}

	// do DB call
	res, err := s.wstmt.ExecContext(ctx, args...)

	// manually wrap the panic in order to report it
	if err != nil {
		internal.AddField(ctx, "db.panic", err)
		panic(err)
	}

	// capture results
	id, lierr := res.LastInsertId()
	if lierr == nil {
		internal.AddField(ctx, "db.last_insert_id", id)
	}
	numrows, nrerr := res.RowsAffected()
	if nrerr == nil {
		internal.AddField(ctx, "db.rows_affected", numrows)
	}
	return res
}

func (s *Stmt) QueryRowx(args ...interface{}) *sqlx.Row {
	var err error
	_, sender := internal.BuildDBEvent(s.Builder, "", args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if s.Mapper != nil {
		s.wstmt.Mapper = s.Mapper
	}

	// do DB call
	row := s.wstmt.QueryRowx(args...)
	return row
}

func (s *Stmt) QueryRowxContext(ctx context.Context, args ...interface{}) *sqlx.Row {
	var err error
	sender := internal.BuildDBSpan(ctx, s.Builder, "", args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if s.Mapper != nil {
		s.wstmt.Mapper = s.Mapper
	}

	// do DB call
	row := s.wstmt.QueryRowxContext(ctx, args...)
	return row
}

func (s *Stmt) Queryx(args ...interface{}) (*sqlx.Rows, error) {
	var err error
	_, sender := internal.BuildDBEvent(s.Builder, "", args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if s.Mapper != nil {
		s.wstmt.Mapper = s.Mapper
	}

	// do DB call
	rows, err := s.wstmt.Queryx(args...)
	return rows, err
}

func (s *Stmt) QueryxContext(ctx context.Context, args ...interface{}) (*sqlx.Rows, error) {
	var err error
	sender := internal.BuildDBSpan(ctx, s.Builder, "", args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if s.Mapper != nil {
		s.wstmt.Mapper = s.Mapper
	}

	// do DB call
	rows, err := s.wstmt.QueryxContext(ctx, args...)
	return rows, err
}

func (s *Stmt) Select(dest interface{}, args ...interface{}) error {
	var err error
	ev, sender := internal.BuildDBEvent(s.Builder, "", args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if s.Mapper != nil {
		s.wstmt.Mapper = s.Mapper
	}

	ev.AddField("db.dest_type", typeof(dest))

	// do DB call
	err = s.wstmt.Select(dest, args...)
	return err
}

func (s *Stmt) SelectContext(ctx context.Context, dest interface{}, args ...interface{}) error {
	var err error
	sender := internal.BuildDBSpan(ctx, s.Builder, "", args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if s.Mapper != nil {
		s.wstmt.Mapper = s.Mapper
	}

	internal.AddField(ctx, "db.dest_type", typeof(dest))

	// do DB call
	err = s.wstmt.SelectContext(ctx, dest, args...)
	return err
}

func (s *Stmt) Unsafe() *Stmt {
	var err error
	_, sender := internal.BuildDBEvent(s.Builder, "")
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if s.Mapper != nil {
		s.wstmt.Mapper = s.Mapper
	}

	newws := s.wstmt.Unsafe()
	s.wstmt = newws
	return s
}

type Tx struct {
	wtx     *sqlx.Tx
	Builder *libhoney.Builder
	Mapper  *reflectx.Mapper
}

func (tx *Tx) BindNamed(query string, arg interface{}) (string, []interface{}, error) {
	var err error
	_, sender := internal.BuildDBEvent(tx.Builder, query, arg)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	str, i, err := tx.wtx.BindNamed(query, arg)
	return str, i, err
}

func (tx *Tx) Commit() error {
	var err error
	_, sender := internal.BuildDBEvent(tx.Builder, "")
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	// do DB call
	err = tx.wtx.Commit()
	return err
}
func (tx *Tx) DriverName() string {
	var err error
	_, sender := internal.BuildDBEvent(tx.Builder, "")
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	name := tx.wtx.DriverName()
	return name
}

func (tx *Tx) Exec(query string, args ...interface{}) (sql.Result, error) {
	var err error
	ev, sender := internal.BuildDBEvent(tx.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	// do DB call
	res, err := tx.wtx.Exec(query, args...)

	// capture results
	if err == nil {
		id, lierr := res.LastInsertId()
		if lierr == nil {
			ev.AddField("db.last_insert_id", id)
		}
		numrows, nrerr := res.RowsAffected()
		if nrerr == nil {
			ev.AddField("db.rows_affected", numrows)
		}
	}
	return res, err
}

func (tx *Tx) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	var err error
	sender := internal.BuildDBSpan(ctx, tx.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	// do DB call
	res, err := tx.wtx.ExecContext(ctx, query, args...)

	// capture results
	if err == nil {
		id, lierr := res.LastInsertId()
		if lierr == nil {
			internal.AddField(ctx, "db.last_insert_id", id)
		}
		numrows, nrerr := res.RowsAffected()
		if nrerr == nil {
			internal.AddField(ctx, "db.rows_affected", numrows)
		}
	}
	return res, err
}

func (tx *Tx) Get(dest interface{}, query string, args ...interface{}) error {
	var err error
	ev, sender := internal.BuildDBEvent(tx.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	// add the type of the objec being populated
	ev.AddField("db.dest_type", typeof(dest))

	err = tx.wtx.Get(dest, query, args...)
	return err
}
func (tx *Tx) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	var err error
	sender := internal.BuildDBSpan(ctx, tx.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	// add the type of the objec being populated
	internal.AddField(ctx, "db.dest_type", typeof(dest))

	err = tx.wtx.GetContext(ctx, dest, query, args...)
	return err
}

func (tx *Tx) MustExec(query string, args ...interface{}) sql.Result {
	var err error
	ev, sender := internal.BuildDBEvent(tx.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	res, err := tx.wtx.Exec(query, args...)

	// manually wrap the panic in order to report it
	if err != nil {
		ev.AddField("db.panic", err)
		panic(err)
	}

	// capture results
	id, lierr := res.LastInsertId()
	if lierr == nil {
		ev.AddField("db.last_insert_id", id)
	}
	numrows, nrerr := res.RowsAffected()
	if nrerr == nil {
		ev.AddField("db.rows_affected", numrows)
	}

	return res
}

func (tx *Tx) MustExecContext(ctx context.Context, query string, args ...interface{}) sql.Result {
	var err error
	sender := internal.BuildDBSpan(ctx, tx.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	res, err := tx.wtx.Exec(query, args...)

	// manually wrap the panic in order to report it
	if err != nil {
		internal.AddField(ctx, "db.panic", err)
		panic(err)
	}

	// capture results
	id, lierr := res.LastInsertId()
	if lierr == nil {
		internal.AddField(ctx, "db.last_insert_id", id)
	}
	numrows, nrerr := res.RowsAffected()
	if nrerr == nil {
		internal.AddField(ctx, "db.rows_affected", numrows)
	}

	return res
}

func (tx *Tx) NamedExec(query string, arg interface{}) (sql.Result, error) {
	var err error
	ev, sender := internal.BuildDBEvent(tx.Builder, query, arg)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	// do DB call
	res, err := tx.wtx.NamedExec(query, arg)

	// capture results
	if err == nil {
		id, lierr := res.LastInsertId()
		if lierr == nil {
			ev.AddField("db.last_insert_id", id)
		}
		numrows, nrerr := res.RowsAffected()
		if nrerr == nil {
			ev.AddField("db.rows_affected", numrows)
		}
	}
	return res, err
}

func (tx *Tx) NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	var err error
	sender := internal.BuildDBSpan(ctx, tx.Builder, query, arg)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	// do DB call
	res, err := tx.wtx.NamedExecContext(ctx, query, arg)

	// capture results
	if err == nil {
		id, lierr := res.LastInsertId()
		if lierr == nil {
			internal.AddField(ctx, "db.last_insert_id", id)
		}
		numrows, nrerr := res.RowsAffected()
		if nrerr == nil {
			internal.AddField(ctx, "db.rows_affected", numrows)
		}
	}
	return res, err
}

func (tx *Tx) NamedQuery(query string, arg interface{}) (*sqlx.Rows, error) {
	var err error
	_, sender := internal.BuildDBEvent(tx.Builder, query, arg)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	// do DB call
	rows, err := tx.wtx.NamedQuery(query, arg)
	return rows, err
}

func (tx *Tx) NamedStmt(stmt *NamedStmt) *NamedStmt {
	var err error
	_, sender := internal.BuildDBEvent(tx.Builder, "")
	defer sender(err)

	bld := tx.Builder.Clone()
	wrapStmt := &NamedStmt{
		Builder: bld,
	}

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	ns := tx.wtx.NamedStmt(stmt.wns)
	wrapStmt.wns = ns
	return wrapStmt
}

func (tx *Tx) NamedStmtContext(ctx context.Context, stmt *NamedStmt) *NamedStmt {
	var err error
	sender := internal.BuildDBSpan(ctx, tx.Builder, "")
	defer sender(err)

	bld := tx.Builder.Clone()
	wrapStmt := &NamedStmt{
		Builder: bld,
	}

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	ns := tx.wtx.NamedStmtContext(ctx, stmt.wns)
	wrapStmt.wns = ns
	return wrapStmt
}

func (tx *Tx) PrepareNamed(query string) (*NamedStmt, error) {
	var err error
	ev, sender := internal.BuildDBEvent(tx.Builder, query)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	bld := tx.Builder.Clone()
	wrapStmt := &NamedStmt{
		Builder: bld,
	}
	newid, _ := uuid.NewRandom()
	stmtid := newid.String()
	bld.AddField("db.stmt_id", stmtid)
	ev.AddField("db.stmt_id", stmtid)
	bld.AddField("db.query", query)

	// do DB call
	stmt, err := tx.wtx.PrepareNamed(query)
	wrapStmt.wns = stmt
	return wrapStmt, err
}

func (tx *Tx) PrepareNamedContext(ctx context.Context, query string) (*NamedStmt, error) {
	var err error
	sender := internal.BuildDBSpan(ctx, tx.Builder, query)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	bld := tx.Builder.Clone()
	wrapStmt := &NamedStmt{
		Builder: bld,
	}
	newid, _ := uuid.NewRandom()
	stmtid := newid.String()
	bld.AddField("db.stmt_id", stmtid)
	internal.AddField(ctx, "db.stmt_id", stmtid)
	bld.AddField("db.query", query)

	// do DB call
	stmt, err := tx.wtx.PrepareNamedContext(ctx, query)
	wrapStmt.wns = stmt
	return wrapStmt, err
}

func (tx *Tx) Preparex(query string) (*Stmt, error) {
	var err error
	ev, sender := internal.BuildDBEvent(tx.Builder, query)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	bld := tx.Builder.Clone()
	wrapStmt := &Stmt{
		Builder: bld,
	}
	newid, _ := uuid.NewRandom()
	stmtid := newid.String()
	bld.AddField("db.stmt_id", stmtid)
	ev.AddField("db.stmt_id", stmtid)
	bld.AddField("db.query", query)

	// do DB call
	stmt, err := tx.wtx.Preparex(query)
	wrapStmt.wstmt = stmt
	return wrapStmt, err
}

func (tx *Tx) PreparexContext(ctx context.Context, query string) (*Stmt, error) {
	var err error
	sender := internal.BuildDBSpan(ctx, tx.Builder, query)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	bld := tx.Builder.Clone()
	wrapStmt := &Stmt{
		Builder: bld,
	}
	newid, _ := uuid.NewRandom()
	stmtid := newid.String()
	bld.AddField("db.stmt_id", stmtid)
	internal.AddField(ctx, "db.stmt_id", stmtid)
	bld.AddField("db.query", query)

	// do DB call
	stmt, err := tx.wtx.PreparexContext(ctx, query)
	wrapStmt.wstmt = stmt
	return wrapStmt, err
}

func (tx *Tx) Query(query string, args ...interface{}) (*sql.Rows, error) {
	var err error
	_, sender := internal.BuildDBEvent(tx.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	// do DB call
	rows, err := tx.wtx.Query(query, args...)
	return rows, err
}

func (tx *Tx) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	var err error
	sender := internal.BuildDBSpan(ctx, tx.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	// do DB call
	rows, err := tx.wtx.QueryContext(ctx, query, args...)
	return rows, err
}

func (tx *Tx) QueryRow(query string, args ...interface{}) *sql.Row {
	var err error
	_, sender := internal.BuildDBEvent(tx.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	// do DB call
	row := tx.wtx.QueryRow(query, args...)
	return row
}

func (tx *Tx) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	var err error
	sender := internal.BuildDBSpan(ctx, tx.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	// do DB call
	row := tx.wtx.QueryRowContext(ctx, query, args...)
	return row
}

func (tx *Tx) QueryRowx(query string, args ...interface{}) *sqlx.Row {
	var err error
	_, sender := internal.BuildDBEvent(tx.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	// do DB call
	row := tx.wtx.QueryRowx(query, args...)
	return row
}

func (tx *Tx) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	var err error
	_, sender := internal.BuildDBEvent(tx.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	// do DB call
	row := tx.wtx.QueryRowxContext(ctx, query, args...)
	return row
}

func (tx *Tx) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	var err error
	_, sender := internal.BuildDBEvent(tx.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	// do DB call
	rows, err := tx.wtx.Queryx(query, args...)
	return rows, err
}

func (tx *Tx) QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	var err error
	_, sender := internal.BuildDBEvent(tx.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	// do DB call
	rows, err := tx.wtx.QueryxContext(ctx, query, args...)
	return rows, err
}

func (tx *Tx) Rebind(query string) string {
	var err error
	_, sender := internal.BuildDBEvent(tx.Builder, query)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	str := tx.wtx.Rebind(query)
	return str
}

func (tx *Tx) Rollback() error {
	var err error
	_, sender := internal.BuildDBEvent(tx.Builder, "")
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	// do DB call
	err = tx.wtx.Rollback()
	return err
}

func (tx *Tx) Select(dest interface{}, query string, args ...interface{}) error {
	var err error
	ev, sender := internal.BuildDBEvent(tx.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	ev.AddField("db.dest_type", typeof(dest))

	err = tx.wtx.Select(dest, query, args...)
	return err
}

func (tx *Tx) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	var err error
	sender := internal.BuildDBSpan(ctx, tx.Builder, query, args...)
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	internal.AddField(ctx, "db.dest_type", typeof(dest))

	err = tx.wtx.SelectContext(ctx, dest, query, args...)
	return err
}

func (tx *Tx) Stmtx(stmt *Stmt) *Stmt {
	var err error
	ev, sender := internal.BuildDBEvent(tx.Builder, "")
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	bld := tx.Builder.Clone()
	wrapStmt := &Stmt{
		Builder: bld,
	}
	newid, _ := uuid.NewRandom()
	stmtid := newid.String()
	bld.AddField("db.stmt_id", stmtid)
	ev.AddField("db.stmt_id", stmtid)

	// do DB call
	newStmt := tx.wtx.Stmtx(stmt.wstmt)

	wrapStmt.wstmt = newStmt

	return wrapStmt
}

func (tx *Tx) StmtxContext(ctx context.Context, stmt *Stmt) *Stmt {
	var err error
	sender := internal.BuildDBSpan(ctx, tx.Builder, "")
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	bld := tx.Builder.Clone()
	wrapStmt := &Stmt{
		Builder: bld,
	}
	newid, _ := uuid.NewRandom()
	stmtid := newid.String()
	bld.AddField("db.stmt_id", stmtid)
	internal.AddField(ctx, "db.stmt_id", stmtid)

	// do DB call
	newStmt := tx.wtx.StmtxContext(ctx, stmt.wstmt)

	wrapStmt.wstmt = newStmt

	return wrapStmt
}

func (tx *Tx) Unsafe() *Tx {
	var err error
	_, sender := internal.BuildDBEvent(tx.Builder, "")
	defer sender(err)

	// ensure any changes to the Mapper get passed along
	if tx.Mapper != nil {
		tx.wtx.Mapper = tx.Mapper
	}

	newtx := tx.wtx.Unsafe()
	tx.wtx = newtx
	return tx
}

// additional helper functions
func typeof(i interface{}) string {
	t := reflect.TypeOf(i)
	if t != nil {
		return t.String()
	}
	return "nil"
}
