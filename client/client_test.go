package client

import (
	"fmt"
	"testing"
)

func TestClientWrappersWorkWithoutInit(t *testing.T) {
	// None of these should cause panics
	Close()
	Flush()
	AddField("foo", "bar")
	b := NewBuilder()
	e := b.NewEvent()
	e.AddField("beep", "boop")
	e.Send()
	// we should get a closed channel back that doesn't panic or block forever
	resp := TxResponses()
	for r := range resp {
		fmt.Println(r.Body)
	}
}
