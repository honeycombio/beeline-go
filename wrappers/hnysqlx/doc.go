// Package hnysqlx wraps `jmoiron/sqlx` to emit one Honeycomb event per DB call.
//
// After opening a DB connection, replace the *sqlx.DB object with a *hnysqlx.DB
// object. The *hnysqlx.DB struct implements all the same functions as the
// normal *sqlx.DB struct, and emits an event to Honeycomb with details about
// the SQL event made.
//
// If you're using transactions, named statements, and so on, there will be a
// similar swap of `*sqlx` to `*hnysqlx` for each of the additional types you're
// using.
//
// Additionally, if hnysqlx is used in conjunction with one of the Honeycomb
// HTTP wrappers *and* you're using the context-aware versions of the SQL calls,
// the trace ID picked up in the HTTP event will appear in the SQL event. This
// will ensure you can track any SQL call back to the HTTP event that triggered
// it.
//
// It is strongly suggested that you use the context-aware version of all calls
// whenever possible; doing so not only lets you cancel your database calls, but
// dramatically increases the value of the SQL isntrumentation by letting you
// tie it back to individual HTTP requests.
//
// If you need to differentiate multiple DB connections, there is a
// *libhoney.Builder associated with the *hnysqlx.DB (as well as with
// transactions and statements). Adding fields to this builder will add those
// fields to all events generated from that DB connection.
//
package hnysqlx
