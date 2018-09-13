package hnysql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"time"

	"github.com/google/uuid"

	"github.com/honeycombio/beeline-go/wrappers/common"
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
	wdb *sql.DB
	// Builder is available in case you wish to add fields to every SQL event
	// that will be created.
	Builder *libhoney.Builder
}

func WrapDB(s *sql.DB) *DB {
	b := libhoney.NewBuilder()
	db := &DB{
		wdb:     s,
		Builder: b,
	}
	addConns := func() interface{} {
		stats := s.Stats()
		return stats.OpenConnections
	}
	b.AddDynamicField("db.open_conns", addConns)
	b.AddField("meta.type", "sql")
	return db
}

func (db *DB) Begin() (*Tx, error) {
	var err error
	ev, sender := common.BuildDBEvent(db.Builder, "")
	defer sender(err)

	bld := db.Builder.Clone()
	wrapTx := &Tx{
		Builder: bld,
	}
	newid, _ := uuid.NewRandom()
	txid := newid.String()
	bld.AddField("db.txId", txid)
	ev.AddField("db.txId", txid)

	// do DB call
	tx, err := db.wdb.Begin()

	wrapTx.wtx = tx

	return wrapTx, err
}

func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	var err error
	ctx, span, sender := common.BuildDBSpan(ctx, db.Builder, "")
	defer sender(err)

	// TODO if ctx.Cancel is called, the transaction is rolled back. We should
	// submit an event indicating the rollback.
	bld := db.Builder.Clone()
	wrapTx := &Tx{
		Builder: bld,
	}
	newid, _ := uuid.NewRandom()
	txid := newid.String()
	bld.AddField("db.txId", txid)
	if span != nil {
		span.AddField("db.txId", txid)
	}

	bld.AddField("db.options", opts)
	if span != nil {
		span.AddField("db.options", opts)
	}

	// do DB call
	tx, err := db.wdb.BeginTx(ctx, opts)

	wrapTx.wtx = tx

	return wrapTx, err
}

func (db *DB) Conn(ctx context.Context) (*Conn, error) {
	var err error
	ctx, span, sender := common.BuildDBSpan(ctx, db.Builder, "")
	defer sender(err)
	bld := db.Builder.Clone()
	id, _ := uuid.NewRandom()
	connid := id.String()
	wrapConn := &Conn{
		Builder: bld,
	}
	bld.AddField("db.connId", connid)
	if span != nil {
		span.AddField("db.connId", connid)
	}

	// do DB call
	conn, err := db.wdb.Conn(ctx)

	wrapConn.wconn = conn

	return wrapConn, err
}

func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	var err error
	ev, sender := common.BuildDBEvent(db.Builder, query, args...)
	defer sender(err)

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
	ctx, span, sender := common.BuildDBSpan(ctx, db.Builder, query, args...)
	defer sender(err)

	// do DB call
	res, err := db.wdb.ExecContext(ctx, query, args...)

	// capture results
	if err == nil {
		id, lierr := res.LastInsertId()
		if lierr == nil {
			if span != nil {
				span.AddField("db.last_insert_id", id)
			}
		}
		numrows, nrerr := res.RowsAffected()
		if nrerr == nil {
			if span != nil {
				span.AddField("db.rows_affected", numrows)
			}
		}
	}
	return res, err
}

func (db *DB) Ping() error {
	var err error
	_, sender := common.BuildDBEvent(db.Builder, "")
	defer sender(err)
	err = db.wdb.Ping()
	return err
}

func (db *DB) PingContext(ctx context.Context) error {
	var err error
	ctx, _, sender := common.BuildDBSpan(ctx, db.Builder, "")
	defer sender(err)
	err = db.wdb.Ping()
	return err
}

func (db *DB) Prepare(query string) (*Stmt, error) {
	var err error
	ev, sender := common.BuildDBEvent(db.Builder, query)
	defer sender(err)

	bld := db.Builder.Clone()
	id, _ := uuid.NewRandom()
	stmtid := id.String()
	wrapStmt := &Stmt{
		Builder: bld,
	}
	bld.AddField("db.stmtId", stmtid)
	// add the query to the builder so all executions of this prepared statement
	// have the query right there
	bld.AddField("db.query", query)
	ev.AddField("db.stmtId", stmtid)

	// do DB call
	stmt, err := db.wdb.Prepare(query)
	wrapStmt.wstmt = stmt
	return wrapStmt, err
}

func (db *DB) PrepareContext(ctx context.Context, query string) (*Stmt, error) {
	var err error
	ctx, span, sender := common.BuildDBSpan(ctx, db.Builder, query)
	defer sender(err)

	bld := db.Builder.Clone()
	id, _ := uuid.NewRandom()
	stmtid := id.String()
	wrapStmt := &Stmt{
		Builder: bld,
	}
	bld.AddField("db.stmtId", stmtid)
	// add the query to the builder so all executions of this prepared statement
	// have the query right there
	bld.AddField("db.query", query)
	if span != nil {
		span.AddField("db.stmtId", stmtid)
	}

	// do DB call
	stmt, err := db.wdb.PrepareContext(ctx, query)
	wrapStmt.wstmt = stmt
	return wrapStmt, err
}

func (db *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	var err error
	_, sender := common.BuildDBEvent(db.Builder, query, args)
	defer sender(err)

	// do DB call
	rows, err := db.wdb.Query(query, args...)
	// TODO can we figure out the number of rows returned or anything like that?
	return rows, err
}

func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	var err error
	ctx, _, sender := common.BuildDBSpan(ctx, db.Builder, query, args)
	defer sender(err)

	// do DB call
	rows, err := db.wdb.QueryContext(ctx, query, args...)
	return rows, err
}

func (db *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	_, sender := common.BuildDBEvent(db.Builder, query, args)
	defer sender(nil)

	// do DB call
	row := db.wdb.QueryRow(query, args...)
	return row
}
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	ctx, _, sender := common.BuildDBSpan(ctx, db.Builder, query, args)
	defer sender(nil)

	// do DB call
	row := db.wdb.QueryRowContext(ctx, query, args...)
	return row
}

func (db *DB) Close() error {
	var err error
	_, sender := common.BuildDBEvent(db.Builder, "")
	defer sender(err)
	err = db.wdb.Close()
	return err
}

// these are not instrumented calls since they're more configuration-esque
func (db *DB) Driver() driver.Driver              { return db.wdb.Driver() }
func (db *DB) SetConnMaxLifetime(d time.Duration) { db.wdb.SetConnMaxLifetime(d) }
func (db *DB) SetMaxIdleConns(n int)              { db.wdb.SetMaxIdleConns(n) }
func (db *DB) SetMaxOpenConns(n int)              { db.wdb.SetMaxOpenConns(n) }
func (db *DB) Stats() sql.DBStats                 { return db.wdb.Stats() }

type Conn struct {
	wconn   *sql.Conn
	Builder *libhoney.Builder
}

func (c *Conn) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	var err error
	ctx, span, sender := common.BuildDBSpan(ctx, c.Builder, "")
	defer sender(err)
	// TODO if ctx.Cancel is called, the transaction is rolled back. We should
	// submit an event indicating the rollback.
	bld := c.Builder.Clone()
	id, _ := uuid.NewRandom()
	txid := id.String()
	wrapTx := &Tx{
		Builder: bld,
	}
	bld.AddField("db.txId", txid)
	if span != nil {
		span.AddField("db.txId", txid)
	}

	if span != nil {
		span.AddField("db.options", opts)
	}
	bld.AddField("db.options", opts)

	// do DB call
	tx, err := c.wconn.BeginTx(ctx, opts)

	wrapTx.wtx = tx

	return wrapTx, err
}

func (c *Conn) Close() error {
	var err error
	_, sender := common.BuildDBEvent(c.Builder, "")
	defer sender(err)

	// do DB call
	err = c.wconn.Close()
	return err
}

func (c *Conn) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	var err error
	ctx, span, sender := common.BuildDBSpan(ctx, c.Builder, query, args...)
	defer sender(err)

	// do DB call
	res, err := c.wconn.ExecContext(ctx, query, args...)

	// capture results
	if err == nil {
		id, lierr := res.LastInsertId()
		if lierr == nil {
			if span != nil {
				span.AddField("db.last_insert_id", id)
			}
		}
		numrows, nrerr := res.RowsAffected()
		if nrerr == nil {
			if span != nil {
				span.AddField("db.rows_affected", numrows)
			}
		}
	}
	return res, err
}

func (c *Conn) PingContext(ctx context.Context) error {
	var err error
	ctx, _, sender := common.BuildDBSpan(ctx, c.Builder, "")
	defer sender(err)
	err = c.wconn.PingContext(ctx)
	return err
}

func (c *Conn) PrepareContext(ctx context.Context, query string) (*Stmt, error) {
	var err error
	ctx, span, sender := common.BuildDBSpan(ctx, c.Builder, query)
	defer sender(err)

	bld := c.Builder.Clone()
	id, _ := uuid.NewRandom()
	stmtid := id.String()
	wrapStmt := &Stmt{
		Builder: bld,
	}
	bld.AddField("db.stmtId", stmtid)
	bld.AddField("db.query", query)
	if span != nil {
		span.AddField("db.stmtId", stmtid)
	}

	// do DB call
	stmt, err := c.wconn.PrepareContext(ctx, query)

	wrapStmt.wstmt = stmt

	return wrapStmt, err
}

func (c *Conn) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	var err error
	ctx, _, sender := common.BuildDBSpan(ctx, c.Builder, query, args)
	defer sender(err)

	// do DB call
	rows, err := c.wconn.QueryContext(ctx, query, args...)
	return rows, err
}

func (c *Conn) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	ctx, _, sender := common.BuildDBSpan(ctx, c.Builder, query, args)
	defer sender(nil)

	// do DB call
	row := c.wconn.QueryRowContext(ctx, query, args...)
	return row
}

type Stmt struct {
	wstmt   *sql.Stmt
	Builder *libhoney.Builder
}

func (s *Stmt) Close() error {
	var err error
	_, sender := common.BuildDBEvent(s.Builder, "")
	defer sender(err)
	err = s.wstmt.Close()
	return err
}

func (s *Stmt) Exec(args ...interface{}) (sql.Result, error) {
	var err error
	ev, sender := common.BuildDBEvent(s.Builder, "", args...)
	defer sender(err)

	// do DB call
	res, err := s.wstmt.Exec(args...)

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

func (s *Stmt) ExecContext(ctx context.Context, args ...interface{}) (sql.Result, error) {
	var err error
	ctx, span, sender := common.BuildDBSpan(ctx, s.Builder, "", args...)
	defer sender(err)

	// do DB call
	res, err := s.wstmt.ExecContext(ctx, args...)

	// capture results
	if err == nil {
		id, lierr := res.LastInsertId()
		if lierr == nil {
			if span != nil {
				span.AddField("db.last_insert_id", id)
			}
		}
		numrows, nrerr := res.RowsAffected()
		if nrerr == nil {
			if span != nil {
				span.AddField("db.rows_affected", numrows)
			}
		}
	}
	return res, err
}

func (s *Stmt) Query(args ...interface{}) (*sql.Rows, error) {
	var err error
	_, sender := common.BuildDBEvent(s.Builder, "", args)
	defer sender(err)

	// do DB call
	rows, err := s.wstmt.Query(args...)
	return rows, err
}

func (s *Stmt) QueryContext(ctx context.Context, args ...interface{}) (*sql.Rows, error) {
	var err error
	ctx, _, sender := common.BuildDBSpan(ctx, s.Builder, "", args)
	defer sender(err)

	// do DB call
	rows, err := s.wstmt.QueryContext(ctx, args...)
	return rows, err
}

func (s *Stmt) QueryRow(args ...interface{}) *sql.Row {
	_, sender := common.BuildDBEvent(s.Builder, "", args)
	defer sender(nil)

	// do DB call
	row := s.wstmt.QueryRow(args...)
	return row
}

func (s *Stmt) QueryRowContext(ctx context.Context, args ...interface{}) *sql.Row {
	ctx, _, sender := common.BuildDBSpan(ctx, s.Builder, "", args)
	defer sender(nil)

	// do DB call
	row := s.wstmt.QueryRowContext(ctx, args...)
	return row
}

type Tx struct {
	// wtx is the wrapped transaction
	wtx     *sql.Tx
	Builder *libhoney.Builder
}

func (tx *Tx) Commit() error {
	var err error
	_, sender := common.BuildDBEvent(tx.Builder, "")
	defer sender(err)

	// do DB call
	err = tx.wtx.Commit()
	return err
}

func (tx *Tx) Exec(query string, args ...interface{}) (sql.Result, error) {
	var err error
	ev, sender := common.BuildDBEvent(tx.Builder, query, args...)
	defer sender(err)

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
	ctx, span, sender := common.BuildDBSpan(ctx, tx.Builder, query, args...)
	defer sender(err)

	// do DB call
	res, err := tx.wtx.ExecContext(ctx, query, args...)

	// capture results
	if err == nil {
		id, lierr := res.LastInsertId()
		if lierr == nil {
			if span != nil {
				span.AddField("db.last_insert_id", id)
			}
		}
		numrows, nrerr := res.RowsAffected()
		if nrerr == nil {
			if span != nil {
				span.AddField("db.rows_affected", numrows)
			}
		}
	}
	return res, err
}

func (tx *Tx) Prepare(query string) (*Stmt, error) {
	var err error
	ev, sender := common.BuildDBEvent(tx.Builder, query)
	defer sender(err)

	bld := tx.Builder.Clone()
	id, _ := uuid.NewRandom()
	stmtid := id.String()
	wrapStmt := &Stmt{
		Builder: bld,
	}
	bld.AddField("db.stmtId", stmtid)
	ev.AddField("db.stmtId", stmtid)
	bld.AddField("db.query", query)

	// do DB call
	stmt, err := tx.wtx.Prepare(query)
	wrapStmt.wstmt = stmt
	return wrapStmt, err
}

func (tx *Tx) PrepareContext(ctx context.Context, query string) (*Stmt, error) {
	var err error
	ctx, span, sender := common.BuildDBSpan(ctx, tx.Builder, query)
	defer sender(err)

	bld := tx.Builder.Clone()
	id, _ := uuid.NewRandom()
	stmtid := id.String()
	wrapStmt := &Stmt{
		Builder: bld,
	}
	bld.AddField("db.stmtId", stmtid)
	if span != nil {
		span.AddField("db.stmtId", stmtid)
	}
	bld.AddField("db.query", query)

	// do DB call
	stmt, err := tx.wtx.PrepareContext(ctx, query)
	wrapStmt.wstmt = stmt
	return wrapStmt, err
}

func (tx *Tx) Query(query string, args ...interface{}) (*sql.Rows, error) {
	var err error
	_, sender := common.BuildDBEvent(tx.Builder, query, args)
	defer sender(err)

	// do DB call
	rows, err := tx.wtx.Query(query, args...)
	return rows, err
}

func (tx *Tx) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	var err error
	ctx, _, sender := common.BuildDBSpan(ctx, tx.Builder, query, args)
	defer sender(err)

	// do DB call
	rows, err := tx.wtx.QueryContext(ctx, query, args...)
	return rows, err
}

func (tx *Tx) QueryRow(query string, args ...interface{}) *sql.Row {
	_, sender := common.BuildDBEvent(tx.Builder, query, args)
	defer sender(nil)

	// do DB call
	row := tx.wtx.QueryRow(query, args...)
	return row
}

func (tx *Tx) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	ctx, _, sender := common.BuildDBSpan(ctx, tx.Builder, query, args)
	defer sender(nil)

	// do DB call
	row := tx.wtx.QueryRowContext(ctx, query, args...)
	return row
}

func (tx *Tx) Rollback() error {
	var err error
	_, sender := common.BuildDBEvent(tx.Builder, "")
	defer sender(err)

	// do DB call
	err = tx.wtx.Rollback()
	return err
}

func (tx *Tx) Stmt(stmt *Stmt) *Stmt {
	ev, sender := common.BuildDBEvent(tx.Builder, "")
	defer sender(nil)

	bld := stmt.Builder.Clone()
	wrapStmt := &Stmt{
		Builder: bld,
	}
	// add the transaction's ID to the statement so that when it gets executed
	// you get both
	bld.AddField("db.txid", tx.Builder.Fields()["db.txid"])
	ev.AddField("db.stmtid", stmt.Builder.Fields()["db.stmtid"])

	// do DB call
	newStmt := tx.wtx.Stmt(stmt.wstmt)
	wrapStmt.wstmt = newStmt
	return wrapStmt
}

func (tx *Tx) StmtContext(ctx context.Context, stmt *Stmt) *Stmt {
	ctx, span, sender := common.BuildDBSpan(ctx, tx.Builder, "")
	defer sender(nil)

	bld := stmt.Builder.Clone()
	wrapStmt := &Stmt{
		Builder: bld,
	}
	// add the transaction's ID to the statement so that when it gets executed
	// you get both
	bld.AddField("db.txid", tx.Builder.Fields()["db.txid"])
	if span != nil {
		span.AddField("db.stmtid", stmt.Builder.Fields()["db.stmtid"])
	}

	// do DB call
	newStmt := tx.wtx.StmtContext(ctx, stmt.wstmt)
	wrapStmt.wstmt = newStmt
	return wrapStmt
}
