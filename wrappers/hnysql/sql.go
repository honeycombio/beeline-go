package hnysql

import (
	"context"
	"database/sql"

	"github.com/satori/go.uuid"

	honeycomb "github.com/honeycombio/honeycomb-go-magic"
	libhoney "github.com/honeycombio/libhoney-go"
)

type DB struct {
	*sql.DB
	builder *libhoney.Builder
	// events will be a map of in-flight events for transactions, but that's not implemented yet.
	// events  map[int]*libhoney.Event
}

func WrapDB(b *libhoney.Builder, s *sql.DB) *DB {
	db := &DB{
		DB:      s,
		builder: b,
	}
	addConns := func() interface{} {
		stats := s.Stats()
		return stats.OpenConnections
	}
	b.AddDynamicField("open_conns", addConns)
	b.AddField("meta.type", "sql")
	return db
}

func (db *DB) Begin() (*Tx, error) {
	ev := db.builder.NewEvent()
	defer ev.Send()
	bld := db.builder.Clone()
	txid := uuid.NewV4().String()
	wrapTx := &Tx{
		builder: bld,
		txid:    txid,
	}
	ev.AddField("txId", txid)
	ev.AddField("call", "Begin")

	// do DB call
	timer := honeycomb.StartTimer()
	tx, err := db.DB.Begin()
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	wrapTx.Tx = tx

	return wrapTx, err
}

func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	// TODO if ctx.Cancel is called, the transaction is rolled back. We should
	// submit an event indicating the rollback.
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
	ev.AddField("call", "BeginTx")

	ev.AddField("options", opts)
	bld.AddField("options", opts)

	// do DB call
	timer := honeycomb.StartTimer()
	tx, err := db.DB.BeginTx(ctx, opts)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	wrapTx.Tx = tx

	return wrapTx, err
}

func (db *DB) Conn(ctx context.Context) (*Conn, error) {
	ev := db.builder.NewEvent()
	defer ev.Send()
	addTraceID(ctx, ev)
	bld := db.builder.Clone()
	addTraceIDBuilder(ctx, bld)
	connid := uuid.NewV4().String()
	wrapConn := &Conn{
		builder: bld,
		connid:  connid,
	}
	bld.AddField("connId", connid)
	ev.AddField("connId", connid)
	ev.AddField("call", "Conn")

	// do DB call
	timer := honeycomb.StartTimer()
	conn, err := db.DB.Conn(ctx)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	wrapConn.Conn = conn

	return wrapConn, err

}

func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	ev := db.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "Exec")
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	res, err := db.DB.Exec(query, args...)
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

func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	ev := db.builder.NewEvent()
	defer ev.Send()
	addTraceID(ctx, ev)
	ev.AddField("call", "ExecContext")
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	res, err := db.DB.ExecContext(ctx, query, args...)
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

func (db *DB) Prepare(query string) (*Stmt, error) {
	bld := db.builder.Clone()
	stmtid := uuid.NewV4().String()
	wrapStmt := &Stmt{
		builder: bld,
	}
	bld.AddField("stmtId", stmtid)
	bld.AddField("query", query)
	ev := bld.NewEvent()
	defer ev.Send()
	ev.AddField("call", "Prepare")

	// do DB call
	timer := honeycomb.StartTimer()
	stmt, err := db.DB.Prepare(query)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	wrapStmt.Stmt = stmt

	return wrapStmt, err
}

func (db *DB) PrepareContext(ctx context.Context, query string) (*Stmt, error) {
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
	ev.AddField("call", "PrepareContext")

	// do DB call
	timer := honeycomb.StartTimer()
	stmt, err := db.DB.PrepareContext(ctx, query)
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

// not implemented in the wrapper - should just fall through to the superclass
// func (db *DB) Close() error                        {}
// func (db *DB) Driver() driver.Driver               {}
// func (db *DB) SetConnMaxLifetime(d time.Duration)  {}
// func (db *DB) SetMaxIdleConns(n int)               {}
// func (db *DB) SetMaxOpenConns(n int)               {}
// func (db *DB) Stats() DBStats                      {}

type Conn struct {
	*sql.Conn
	builder *libhoney.Builder
	connid  string
}

func (c *Conn) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	// TODO if ctx.Cancel is called, the transaction is rolled back. We should
	// submit an event indicating the rollback.
	ev := c.builder.NewEvent()
	defer ev.Send()
	bld := c.builder.Clone()
	txid := uuid.NewV4().String()
	wrapTx := &Tx{
		builder: bld,
		txid:    txid,
	}
	bld.AddField("txId", txid)
	ev.AddField("txId", txid)
	ev.AddField("call", "BeginTx")

	ev.AddField("options", opts)
	bld.AddField("options", opts)

	// do DB call
	timer := honeycomb.StartTimer()
	tx, err := c.Conn.BeginTx(ctx, opts)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	wrapTx.Tx = tx

	return wrapTx, err
}

func (c *Conn) Close() error {
	ev := c.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "Close")

	// do DB call
	timer := honeycomb.StartTimer()
	err := c.Conn.Close()
	duration := timer.Finish()
	ev.AddField("durationMs", duration)
	if err != nil {
		ev.AddField("error", err)
	}
	return err
}

func (c *Conn) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	ev := c.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "ExecContext")
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	res, err := c.Conn.ExecContext(ctx, query, args...)
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

func (c *Conn) PingContext(ctx context.Context) error {
	ev := c.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "Ping")

	timer := honeycomb.StartTimer()
	err := c.Conn.PingContext(ctx)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	return err
}

func (c *Conn) PrepareContext(ctx context.Context, query string) (*Stmt, error) {
	bld := c.builder.Clone()
	stmtid := uuid.NewV4().String()
	wrapStmt := &Stmt{
		builder: bld,
	}
	bld.AddField("stmtId", stmtid)
	bld.AddField("query", query)
	ev := bld.NewEvent()
	defer ev.Send()
	ev.AddField("call", "PrepareContext")

	// do DB call
	timer := honeycomb.StartTimer()
	stmt, err := c.Conn.PrepareContext(ctx, query)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	wrapStmt.Stmt = stmt

	return wrapStmt, err
}

func (c *Conn) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	ev := c.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "QueryContext")
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	rows, err := c.Conn.QueryContext(ctx, query, args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)
	return rows, err
}

func (c *Conn) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	ev := c.builder.NewEvent()
	defer ev.Send()
	ev.AddField("call", "QueryRowContext")
	ev.AddField("query", query)
	ev.AddField("query_args", args)

	// do DB call
	timer := honeycomb.StartTimer()
	row := c.Conn.QueryRowContext(ctx, query, args...)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)
	return row
}

type Stmt struct {
	*sql.Stmt
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
	*sql.Tx
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
func (tx *Tx) Prepare(query string) (*Stmt, error) {
	bld := tx.builder.Clone()
	stmtid := uuid.NewV4().String()
	wrapStmt := &Stmt{
		builder: bld,
	}
	bld.AddField("stmtId", stmtid)
	bld.AddField("query", query)
	ev := bld.NewEvent()
	defer ev.Send()
	ev.AddField("call", "Prepare")

	// do DB call
	timer := honeycomb.StartTimer()
	stmt, err := tx.Tx.Prepare(query)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	wrapStmt.Stmt = stmt

	return wrapStmt, err
}
func (tx *Tx) PrepareContext(ctx context.Context, query string) (*Stmt, error) {
	bld := tx.builder.Clone()
	stmtid := uuid.NewV4().String()
	wrapStmt := &Stmt{
		builder: bld,
	}
	bld.AddField("stmtId", stmtid)
	bld.AddField("query", query)
	ev := bld.NewEvent()
	defer ev.Send()
	ev.AddField("call", "PrepareContext")

	// do DB call
	timer := honeycomb.StartTimer()
	stmt, err := tx.Tx.PrepareContext(ctx, query)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	wrapStmt.Stmt = stmt

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
func (tx *Tx) Stmt(stmt *Stmt) *Stmt {
	bld := tx.builder.Clone()
	stmtid := uuid.NewV4().String()
	wrapStmt := &Stmt{
		builder: bld,
	}
	bld.AddField("stmtId", stmtid)
	ev := bld.NewEvent()
	defer ev.Send()
	ev.AddField("call", "Stmt")

	// do DB call
	timer := honeycomb.StartTimer()
	newStmt := tx.Tx.Stmt(stmt.Stmt)
	duration := timer.Finish()
	ev.AddField("durationMs", duration)

	wrapStmt.Stmt = newStmt

	return wrapStmt
}
func (tx *Tx) StmtContext(ctx context.Context, stmt *Stmt) *Stmt {
	bld := tx.builder.Clone()
	stmtid := uuid.NewV4().String()
	wrapStmt := &Stmt{
		builder: bld,
	}
	bld.AddField("stmtId", stmtid)
	ev := bld.NewEvent()
	defer ev.Send()
	ev.AddField("call", "StmtContext")

	// do DB call
	timer := honeycomb.StartTimer()
	newStmt := tx.Tx.StmtContext(ctx, stmt.Stmt)
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
