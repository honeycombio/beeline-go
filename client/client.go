package client

import (
	libhoney "github.com/honeycombio/libhoney-go"
	"github.com/honeycombio/libhoney-go/transmission"
)

var client = &libhoney.Client{}

func Init(config *libhoney.ClientConfig) {
	client, _ = libhoney.NewClient(*config)
}

func Close() {
	if client != nil {
		client.Close()
	}
}

func Flush() {
	if client != nil {
		client.Flush()
	}
}

func AddField(name string, val interface{}) {
	if client != nil {
		client.AddField(name, val)
	}
}

func NewBuilder() *libhoney.Builder {
	if client != nil {
		return client.NewBuilder()
	}
	return &libhoney.Builder{}
}

func TxResponses() chan transmission.Response {
	if client != nil {
		client.TxResponses()
	}

	c := make(chan transmission.Response)
	close(c)
	return c
}
