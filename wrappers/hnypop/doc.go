// Package hnypop wraps the gobuffalo/pop ORM.
//
// Summary
//
// hnypop provides a minimal implementation of the pop Store interface. There
// are a few flaws - when starting a pop Transaction, you'll get a Honeycomb
// event for the start of the transaction but none of the incremental
// statements.
//
// Most other operations should come through ok.
//
package hnypop
