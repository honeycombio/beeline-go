// package hnysql wraps `database.sql` to emit one Honeycomb event per DB call
//
// After opening a DB connection, replace the *sql.DB object with a *hnysql.DB
// object. The *hnysql.DB struct implements all the same functions as the normal
// *sql.DB struct, and emits an event to Honeycomb with details about the SQL
// event made.
//
// Additionally, if hnysql is used in conjunction with one of the Honeycomb HTTP
// wrappers, the trace ID picked up in the HTTP event will appear in the SQL
// event to allow easy identification of what HTTP call triggers which SQL
// calls.
//
// See the exmaples directory for a complete example of how to use the hnysql DB
// wrapper.
//
package hnysql
