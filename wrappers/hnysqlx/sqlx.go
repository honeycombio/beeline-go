package hnysqlx

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/satori/go.uuid"

	honeycomb "github.com/honeycombio/honeycomb-go-magic"
	libhoney "github.com/honeycombio/libhoney-go"
)

type DB struct {
	*sqlx.DB
	builder *libhoney.Builder
	// events will be a map of in-flight events for transactions, but that's not implemented yet.
	// events  map[int]*libhoney.Event
}

func WrapDB(b *libhoney.Builder, s *sqlx.DB) *DB {
	db := &DB{
		DB:      s,
		builder: b,
	}
	addConns := func() interface{} {
		stats := s.DB.Stats()
		return stats.OpenConnections
	}
	b.AddDynamicField("open_conns", addConns)
	b.AddField("meta.type", "sqlx")
	return db
}

func (db *DB) BeginTxx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	ev := db.builder.NewEvent()
	defer ev.Send()
	addTraceID(ctx, ev)
	bld := db.builder.Clone()
	addTraceIDBuilder(ctx, bld)
	txid := uuid.NewV4().String()
	wrapTx := &Tx{
		builder: bld,
		txid:    txid,
	}
	bld.AddField("txId", txid)
	ev.AddField("txId", txid)
	ev.AddField("call", "BeginTxx")

	ev.AddField("options", opts)
	bld.AddField("options", opts)

	// do DB call
	timer := honeycomb.StartTimer()
	tx, err := db.DB.BeginTxx(ctx, opts)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	wrapTx.Tx = tx

	return wrapTx, err
}

func (db *DB) Beginx() (*Tx, error) {
	ev := db.builder.NewEvent()
	defer ev.Send()
	bld := db.builder.Clone()
	txid := uuid.NewV4().String()
	wrapTx := &Tx{
		builder: bld,
		txid:    txid,
	}
	ev.AddField("txId", txid)
	ev.AddField("call", "Beginx")

	// do DB call
	timer := honeycomb.StartTimer()
	tx, err := db.DB.Beginx()
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	wrapTx.Tx = tx

	return wrapTx, err
}
func (db *DB) Get(dest interface{}, query string, args ...interface{}) error {
	ev := db.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "Get")
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	err := db.DB.Get(dest, query, args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	// capture results
	if err != nil {
		ev.AddField("error", err)
	}
	return err
}
func (db *DB) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	ev := db.builder.NewEvent()
	defer ev.Send()
	addTraceID(ctx, ev)
	ev.AddField("call", "GetContext")
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	err := db.DB.GetContext(ctx, dest, query, args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	// capture results
	if err != nil {
		ev.AddField("error", err)
	}
	return err
}
func (db *DB) MustBegin() *Tx {
	ev := db.builder.NewEvent()
	defer ev.Send()
	bld := db.builder.Clone()
	txid := uuid.NewV4().String()
	wrapTx := &Tx{
		builder: bld,
		txid:    txid,
	}
	ev.AddField("txId", txid)
	ev.AddField("call", "MustBegin")

	// do DB call
	timer := honeycomb.StartTimer()
	tx, err := db.DB.Beginx()
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	wrapTx.Tx = tx

	if err != nil {
		ev.AddField("panic", err)
		panic(err)
	}
	return wrapTx
}

func (db *DB) MustBeginTx(ctx context.Context, opts *sql.TxOptions) *Tx {
	ev := db.builder.NewEvent()
	defer ev.Send()
	bld := db.builder.Clone()
	txid := uuid.NewV4().String()
	wrapTx := &Tx{
		builder: bld,
		txid:    txid,
	}
	ev.AddField("txId", txid)
	ev.AddField("call", "MustBegin")

	// do DB call
	timer := honeycomb.StartTimer()
	tx, err := db.DB.BeginTxx(ctx, opts)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	wrapTx.Tx = tx

	if err != nil {
		ev.AddField("panic", err)
		panic(err)
	}
	return wrapTx
}

func (db *DB) MustExec(query string, args ...interface{}) sql.Result {
	ev := db.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "MustExec")
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	res := db.DB.MustExec(query, args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	// capture results
	id, lierr := res.LastInsertId()
	if lierr == nil {
		ev.AddField("last_insert_id", id)
	}
	numrows, nrerr := res.RowsAffected()
	if nrerr == nil {
		ev.AddField("rows_affected", numrows)
	}

	return res
}

func (db *DB) MustExecContext(ctx context.Context, query string, args ...interface{}) sql.Result {
	ev := db.builder.NewEvent()
	defer ev.Send()
	addTraceID(ctx, ev)
	ev.AddField("call", "MustExecContext")
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	res := db.DB.MustExecContext(ctx, query, args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	id, lierr := res.LastInsertId()
	if lierr == nil {
		ev.AddField("last_insert_id", id)
	}
	numrows, nrerr := res.RowsAffected()
	if nrerr == nil {
		ev.AddField("rows_affected", numrows)
	}

	return res

}
func (db *DB) NamedExec(query string, arg interface{}) (sql.Result, error) {
	ev := db.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "NamedExec")
	ev.AddField("query", query)
	ev.AddField("query_arg", arg)

	// do DB call
	timer := honeycomb.StartTimer()
	res, err := db.DB.NamedExec(query, arg)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	// capture results
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
func (db *DB) NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	ev := db.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "NamedExecContext")
	ev.AddField("query", query)
	ev.AddField("query_arg", arg)

	// do DB call
	timer := honeycomb.StartTimer()
	res, err := db.DB.NamedExecContext(ctx, query, arg)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	// capture results
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
func (db *DB) NamedQuery(query string, arg interface{}) (*sqlx.Rows, error) {
	ev := db.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "NamedQuery")
	ev.AddField("query", query)
	ev.AddField("query_arg", arg)

	// do DB call
	timer := honeycomb.StartTimer()
	rows, err := db.DB.NamedQuery(query, arg)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	return rows, err
}
func (db *DB) NamedQueryContext(ctx context.Context, query string, arg interface{}) (*sqlx.Rows, error) {
	ev := db.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "NamedQueryContext")
	ev.AddField("query", query)
	ev.AddField("query_arg", arg)

	// do DB call
	timer := honeycomb.StartTimer()
	rows, err := db.DB.NamedQueryContext(ctx, query, arg)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	return rows, err
}

func (db *DB) Ping() error {
	ev := db.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "Ping")

	timer := honeycomb.StartTimer()
	err := db.DB.Ping()
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	return err
}

func (db *DB) PingContext(ctx context.Context) error {
	ev := db.builder.NewEvent()
	defer ev.Send()
	addTraceID(ctx, ev)
	ev.AddField("call", "Ping")

	timer := honeycomb.StartTimer()
	err := db.DB.Ping()
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	return err
}

func (db *DB) PrepareNamed(query string) (*NamedStmt, error) {
	bld := db.builder.Clone()
	stmtid := uuid.NewV4().String()
	wrapStmt := &NamedStmt{
		builder: bld,
	}
	bld.AddField("stmtId", stmtid)
	bld.AddField("query", query)
	ev := bld.NewEvent()
	defer ev.Send()
	ev.AddField("call", "PrepareNamed")

	// do DB call
	timer := honeycomb.StartTimer()
	stmt, err := db.DB.PrepareNamed(query)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	wrapStmt.NamedStmt = stmt

	return wrapStmt, err
}

func (db *DB) PrepareNamedContext(ctx context.Context, query string) (*NamedStmt, error) {
	bld := db.builder.Clone()
	stmtid := uuid.NewV4().String()
	wrapStmt := &NamedStmt{
		builder: bld,
	}
	bld.AddField("stmtId", stmtid)
	bld.AddField("query", query)
	ev := bld.NewEvent()
	defer ev.Send()
	addTraceID(ctx, ev)
	ev.AddField("call", "PrepareNamedContext")

	// do DB call
	timer := honeycomb.StartTimer()
	stmt, err := db.DB.PrepareNamedContext(ctx, query)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	wrapStmt.NamedStmt = stmt

	return wrapStmt, err
}

func (db *DB) Preparex(query string) (*Stmt, error) {
	bld := db.builder.Clone()
	stmtid := uuid.NewV4().String()
	wrapStmt := &Stmt{
		builder: bld,
	}
	bld.AddField("stmtId", stmtid)
	bld.AddField("query", query)
	ev := bld.NewEvent()
	defer ev.Send()
	ev.AddField("call", "Preparex")

	// do DB call
	timer := honeycomb.StartTimer()
	stmt, err := db.DB.Preparex(query)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	wrapStmt.Stmt = stmt

	return wrapStmt, err
}

func (db *DB) PreparexContext(ctx context.Context, query string) (*Stmt, error) {
	bld := db.builder.Clone()
	stmtid := uuid.NewV4().String()
	wrapStmt := &Stmt{
		builder: bld,
	}
	bld.AddField("stmtId", stmtid)
	bld.AddField("query", query)
	ev := bld.NewEvent()
	defer ev.Send()
	addTraceID(ctx, ev)
	ev.AddField("call", "PreparexContext")

	// do DB call
	timer := honeycomb.StartTimer()
	stmt, err := db.DB.PreparexContext(ctx, query)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	wrapStmt.Stmt = stmt

	return wrapStmt, err
}

func (db *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	ev := db.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "Query")
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	rows, err := db.DB.Query(query, args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)
	return rows, err
}
func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	ev := db.builder.NewEvent()
	defer ev.Send()
	addTraceID(ctx, ev)
	ev.AddField("call", "QueryContext")
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	rows, err := db.DB.QueryContext(ctx, query, args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)
	return rows, err

}
func (db *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	ev := db.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "QueryRow")
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	row := db.DB.QueryRow(query, args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)
	return row
}
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	ev := db.builder.NewEvent()
	defer ev.Send()
	addTraceID(ctx, ev)
	ev.AddField("call", "QueryRowContext")
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	row := db.DB.QueryRowContext(ctx, query, args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)
	return row
}
func (db *DB) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	ev := db.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "Queryx")
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	rows, err := db.DB.Queryx(query, args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)
	return rows, err
}
func (db *DB) QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	ev := db.builder.NewEvent()
	defer ev.Send()
	addTraceID(ctx, ev)
	ev.AddField("call", "QueryxContext")
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	rows, err := db.DB.QueryxContext(ctx, query, args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)
	return rows, err

}
func (db *DB) QueryRowx(query string, args ...interface{}) *sqlx.Row {
	ev := db.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "QueryRowx")
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	row := db.DB.QueryRowx(query, args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)
	return row
}
func (db *DB) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	ev := db.builder.NewEvent()
	defer ev.Send()
	addTraceID(ctx, ev)
	ev.AddField("call", "QueryRowxContext")
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	row := db.DB.QueryRowxContext(ctx, query, args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)
	return row
}
func (db *DB) Select(dest interface{}, query string, args ...interface{}) error {
	ev := db.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "Select")
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	err := db.DB.Select(dest, query, args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)
	return err
}
func (db *DB) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	ev := db.builder.NewEvent()
	defer ev.Send()
	addTraceID(ctx, ev)
	ev.AddField("call", "SelectContext")
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	err := db.DB.SelectContext(ctx, dest, query, args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)
	return err
}

// not implemented in the wrapper - should just fall through to the superclass
// func (db *DB) Close() error                        {}
// func (db *DB) Driver() driver.Driver               {}
// func (db *DB) SetConnMaxLifetime(d time.Duration)  {}
// func (db *DB) SetMaxIdleConns(n int)               {}
// func (db *DB) SetMaxOpenConns(n int)               {}
// func (db *DB) Stats() DBStats                      {}

type NamedStmt struct {
	*sqlx.NamedStmt
	builder *libhoney.Builder
}

type Stmt struct {
	*sqlx.Stmt
	builder *libhoney.Builder
}

func (s *Stmt) Close() error {
	ev := s.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "Close")

	// do DB call
	timer := honeycomb.StartTimer()
	err := s.Stmt.Close()
	duration := timer.Finish()
	ev.AddField("durationMs", duration)
	if err != nil {
		ev.AddField("error", err)
	}
	return err
}
func (s *Stmt) Exec(args ...interface{}) (sql.Result, error) {
	ev := s.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "Exec")
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	res, err := s.Stmt.Exec(args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	// capture results
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
func (s *Stmt) ExecContext(ctx context.Context, args ...interface{}) (sql.Result, error) {
	ev := s.builder.NewEvent()
	defer ev.Send()
	addTraceID(ctx, ev)
	ev.AddField("call", "ExecContext")
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	res, err := s.Stmt.ExecContext(ctx, args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	// capture results
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
func (s *Stmt) Query(args ...interface{}) (*sql.Rows, error) {
	ev := s.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "Query")
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	rows, err := s.Stmt.Query(args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)
	return rows, err
}
func (s *Stmt) QueryContext(ctx context.Context, args ...interface{}) (*sql.Rows, error) {
	ev := s.builder.NewEvent()
	defer ev.Send()
	addTraceID(ctx, ev)
	ev.AddField("call", "QueryContext")
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	rows, err := s.Stmt.QueryContext(ctx, args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)
	return rows, err
}
func (s *Stmt) QueryRow(args ...interface{}) *sql.Row {
	ev := s.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "QueryRow")
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	row := s.Stmt.QueryRow(args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)
	return row
}
func (s *Stmt) QueryRowContext(ctx context.Context, args ...interface{}) *sql.Row {
	ev := s.builder.NewEvent()
	defer ev.Send()
	addTraceID(ctx, ev)
	ev.AddField("call", "QueryRowContext")
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	row := s.Stmt.QueryRowContext(ctx, args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)
	return row
}

type Tx struct {
	*sqlx.Tx
	builder *libhoney.Builder
	txid    string
}

func (tx *Tx) Commit() error {
	ev := tx.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "Commit")

	// do DB call
	timer := honeycomb.StartTimer()
	err := tx.Tx.Commit()
	duration := timer.Finish()
	ev.AddField("durationMs", duration)
	if err != nil {
		ev.AddField("error", err)
	}
	return err
}
func (tx *Tx) Exec(query string, args ...interface{}) (sql.Result, error) {
	ev := tx.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "Exec")
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	res, err := tx.Tx.Exec(query, args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	// capture results
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
func (tx *Tx) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	ev := tx.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "ExecContext")
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	res, err := tx.Tx.ExecContext(ctx, query, args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	// capture results
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
func (tx *Tx) Preparex(query string) (*Stmt, error) {
	bld := tx.builder.Clone()
	stmtid := uuid.NewV4().String()
	wrapStmt := &Stmt{
		builder: bld,
	}
	bld.AddField("stmtId", stmtid)
	bld.AddField("query", query)
	ev := bld.NewEvent()
	defer ev.Send()
	ev.AddField("call", "Preparex")

	// do DB call
	timer := honeycomb.StartTimer()
	stmt, err := tx.Tx.Preparex(query)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	wrapStmt.Stmt = stmt

	return wrapStmt, err
}
func (tx *Tx) PreparexContext(ctx context.Context, query string) (*Stmt, error) {
	bld := tx.builder.Clone()
	stmtid := uuid.NewV4().String()
	wrapStmt := &Stmt{
		builder: bld,
	}
	bld.AddField("stmtId", stmtid)
	bld.AddField("query", query)
	ev := bld.NewEvent()
	defer ev.Send()
	ev.AddField("call", "PreparexContext")

	// do DB call
	timer := honeycomb.StartTimer()
	stmt, err := tx.Tx.PreparexContext(ctx, query)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	wrapStmt.Stmt = stmt

	return wrapStmt, err
}

func (tx *Tx) PrepareNamed(query string) (*NamedStmt, error) {
	bld := tx.builder.Clone()
	stmtid := uuid.NewV4().String()
	wrapStmt := &NamedStmt{
		builder: bld,
	}
	bld.AddField("stmtId", stmtid)
	bld.AddField("query", query)
	ev := bld.NewEvent()
	defer ev.Send()
	ev.AddField("call", "PrepareNamed")

	// do DB call
	timer := honeycomb.StartTimer()
	stmt, err := tx.Tx.PrepareNamed(query)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	wrapStmt.NamedStmt = stmt

	return wrapStmt, err
}

func (tx *Tx) PrepareNamedContext(ctx context.Context, query string) (*NamedStmt, error) {
	bld := tx.builder.Clone()
	stmtid := uuid.NewV4().String()
	wrapStmt := &NamedStmt{
		builder: bld,
	}
	bld.AddField("stmtId", stmtid)
	bld.AddField("query", query)
	ev := bld.NewEvent()
	defer ev.Send()
	addTraceID(ctx, ev)
	ev.AddField("call", "PrepareNamedContext")

	// do DB call
	timer := honeycomb.StartTimer()
	stmt, err := tx.Tx.PrepareNamedContext(ctx, query)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	wrapStmt.NamedStmt = stmt

	return wrapStmt, err
}

func (tx *Tx) Query(query string, args ...interface{}) (*sql.Rows, error) {
	ev := tx.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "Query")
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	rows, err := tx.Tx.Query(query, args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)
	return rows, err
}
func (tx *Tx) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	ev := tx.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "QueryContext")
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	rows, err := tx.Tx.QueryContext(ctx, query, args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)
	return rows, err
}
func (tx *Tx) QueryRow(query string, args ...interface{}) *sql.Row {
	ev := tx.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "QueryRow")
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	row := tx.Tx.QueryRow(query, args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)
	return row
}
func (tx *Tx) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	ev := tx.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "QueryRowContext")
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	row := tx.Tx.QueryRowContext(ctx, query, args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)
	return row
}
func (tx *Tx) Rollback() error {
	ev := tx.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "Rollback")

	// do DB call
	timer := honeycomb.StartTimer()
	err := tx.Tx.Rollback()
	duration := timer.Finish()
	ev.AddField("durationMs", duration)
	if err != nil {
		ev.AddField("error", err)
	}
	return err
}
func (tx *Tx) Stmtx(stmt *Stmt) *Stmt {
	bld := tx.builder.Clone()
	stmtid := uuid.NewV4().String()
	wrapStmt := &Stmt{
		builder: bld,
	}
	bld.AddField("stmtId", stmtid)
	ev := bld.NewEvent()
	defer ev.Send()
	ev.AddField("call", "Stmtx")

	// do DB call
	timer := honeycomb.StartTimer()
	newStmt := tx.Tx.Stmtx(stmt.Stmt)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	wrapStmt.Stmt = newStmt

	return wrapStmt
}
func (tx *Tx) StmtxContext(ctx context.Context, stmt *Stmt) *Stmt {
	bld := tx.builder.Clone()
	stmtid := uuid.NewV4().String()
	wrapStmt := &Stmt{
		builder: bld,
	}
	bld.AddField("stmtId", stmtid)
	ev := bld.NewEvent()
	defer ev.Send()
	ev.AddField("call", "StmtxContext")

	// do DB call
	timer := honeycomb.StartTimer()
	newStmt := tx.Tx.StmtxContext(ctx, stmt.Stmt)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	wrapStmt.Stmt = newStmt

	return wrapStmt
}

// additional helper functions

func addTraceID(ctx context.Context, ev *libhoney.Event) {
	// get a transaction ID from the request's event, if it's sitting in context
	if parentEv := honeycomb.ContextEvent(ctx); parentEv != nil {
		if id, ok := parentEv.Fields()["Trace.TraceId"]; ok {
			ev.AddField("Trace.TraceId", id)
		}
	}
}
func addTraceIDBuilder(ctx context.Context, bld *libhoney.Builder) {
	// get a transaction ID from the request's event, if it's sitting in context
	if parentEv := honeycomb.ContextEvent(ctx); parentEv != nil {
		if id, ok := parentEv.Fields()["Trace.TraceId"]; ok {
			bld.AddField("Trace.TraceId", id)
		}
	}
}
