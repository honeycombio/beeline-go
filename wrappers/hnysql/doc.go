// Package hnysql wraps `database.sql` to emit one Honeycomb event per DB call.
//
// After opening a DB connection, replace the *sql.DB object with a *hnysql.DB
// object. The *hnysql.DB struct implements all the same functions as the normal
// *sql.DB struct, and emits an event to Honeycomb with details about the SQL
// event made.
//
// Additionally, if hnysql is used in conjunction with one of the Honeycomb HTTP
// wrappers *and* you use the context-aware version of the DB calls, the trace
// ID picked up in the HTTP event will appear in the SQL event to allow easy
// identification of what HTTP call triggers which SQL calls.
//
// It is strongly suggested that you use the context-aware version of all calls
// whenever possible; doing so not only lets you cancel your database calls, but
// dramatically increases the value of the SQL isntrumentation by letting you
// tie it back to individual HTTP requests.
package hnysql
