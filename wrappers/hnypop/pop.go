package hnypop

import (
	"database/sql"
	"math/rand"

	"github.com/gobuffalo/pop"
	"github.com/honeycombio/beeline-go/wrappers/hnysqlx"
	"github.com/jmoiron/sqlx"
)

type DB struct {
	DB *hnysqlx.DB
	tx *pop.Tx
}

func (m *DB) Select(dest interface{}, query string, args ...interface{}) error {
	return m.DB.Select(dest, query, args...)
}
func (m *DB) Get(dest interface{}, query string, args ...interface{}) error {
	return m.DB.Get(dest, query, args...)
}
func (m *DB) NamedExec(query string, arg interface{}) (sql.Result, error) {
	return m.DB.NamedExec(query, arg)
}
func (m *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return m.DB.Exec(query, args...)
}
func (m *DB) PrepareNamed(query string) (*sqlx.NamedStmt, error) {
	stmt, err := m.DB.PrepareNamed(query)
	return stmt.GetWrappedNamedStmt(), err
}
func (m *DB) Transaction() (*pop.Tx, error) {
	t := &pop.Tx{
		ID: rand.Int(),
	}
	tx, err := m.DB.Beginx()
	t.Tx = tx.GetWrappedTx()
	m.tx = t
	return t, err
}
func (m *DB) Rollback() error {
	return m.tx.Rollback()
}
func (m *DB) Commit() error {
	return m.tx.Commit()
}
func (m *DB) Close() error {
	return m.Close()
}
