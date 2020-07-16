package hnynethttp

import (
	"context"
	"crypto/tls"
	"net/http/httptrace"
	"strings"
	"sync"

	"github.com/honeycombio/beeline-go/trace"
)

func NewHttpClientTrace(ctx context.Context) *httptrace.ClientTrace {
	if span := trace.GetSpanFromContext(ctx); span == nil {
		// there's no span so just short circuit all this
		return nil
	}
	tracer := newTracer(ctx)

	return &httptrace.ClientTrace{
		GetConn: func(hostPort string) {
			tracer.GetConn(hostPort)
		},
		GotConn: func(info httptrace.GotConnInfo) {
			tracer.GotConn(info)
		},
		DNSStart: func(info httptrace.DNSStartInfo) {
			tracer.DNSStart(info)
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			tracer.DNSDone(info)
		},
		ConnectStart: func(network, addr string) {
			tracer.ConnectStart(network, addr)
		},
		ConnectDone: func(network, addr string, err error) {
			tracer.ConnectDone(network, addr, err)
		},
		TLSHandshakeStart: func() {
			tracer.TLSHandshakeStart()
		},
		TLSHandshakeDone: func(connState tls.ConnectionState, err error) {
			tracer.TLSHandshakeDone(connState, err)
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			tracer.WroteRequest(info)
		},
		GotFirstResponseByte: func() {
			tracer.GotFirstResponseByte()
		},
	}
}

type httpTracer struct {
	l             sync.Mutex
	rootSpan      *trace.Span
	rootCtx       context.Context
	connectionCtx context.Context
	dnsCtx        context.Context
	tlsCtx        context.Context
	poolCtx       context.Context
	requestCtx    context.Context
}

func newTracer(ctx context.Context) *httpTracer {
	span := trace.GetSpanFromContext(ctx)

	return &httpTracer{rootCtx: ctx, rootSpan: span}
}

func (h *httpTracer) GetConn(hostPort string) {
	h.l.Lock()
	defer h.l.Unlock()
	if h.rootSpan == nil {
		return
	}
	c, span := h.rootSpan.CreateChild(h.rootCtx)
	span.AddField("name", "connection")
	span.AddField("host_port", hostPort)
	h.connectionCtx = c
}

func (h *httpTracer) GotConn(info httptrace.GotConnInfo) {
	h.l.Lock()
	defer h.l.Unlock()
	if h.rootSpan == nil {
		return
	}
	span := trace.GetSpanFromContext(h.connectionCtx)
	span.AddField("conn_was_idle", info.WasIdle)
	span.AddField("conn_reused", info.Reused)
	span.AddField("conn_idle_dur_ms", info.IdleTime.Milliseconds)
	span.Send()
}

func (h *httpTracer) DNSStart(info httptrace.DNSStartInfo) {
	h.l.Lock()
	defer h.l.Unlock()
	if h.rootSpan == nil {
		return
	}
	c, span := h.rootSpan.CreateChild(h.rootCtx)
	span.AddField("name", "dns")
	span.AddField("dns_host", info.Host)
	h.dnsCtx = c
}

func (h *httpTracer) DNSDone(info httptrace.DNSDoneInfo) {
	h.l.Lock()
	defer h.l.Unlock()
	if h.rootSpan == nil {
		return
	}
	span := trace.GetSpanFromContext(h.dnsCtx)
	span.AddField("dns_coalesced", info.Coalesced)

	addrs := []string{}
	for _, addr := range info.Addrs {
		addrs = append(addrs, addr.IP.String())
	}
	strings.Join(addrs, ",")
	span.AddField("dns_addrs", info.Addrs)
	span.Send()
}

func (h *httpTracer) TLSHandshakeStart() {
	h.l.Lock()
	defer h.l.Unlock()
	if h.rootSpan == nil {
		return
	}
	c, span := h.rootSpan.CreateChild(h.rootCtx)
	span.AddField("name", "tls")
	h.tlsCtx = c
}
func (h *httpTracer) TLSHandshakeDone(connState tls.ConnectionState, err error) {
	h.l.Lock()
	defer h.l.Unlock()
	if h.rootSpan == nil {
		return
	}
	span := trace.GetSpanFromContext(h.tlsCtx)

	span.AddField("tls_version", connState.Version)
	span.AddField("tls_server_name", connState.ServerName)
	span.AddField("tls_negotiated_proto_name", connState.NegotiatedProtocol)
	span.AddField("tls_cipher_suite", connState.CipherSuite)
	span.Send()
}

func (h *httpTracer) ConnectStart(network, addr string) {
	h.l.Lock()
	defer h.l.Unlock()
	if h.rootSpan == nil {
		return
	}
	c, span := h.rootSpan.CreateChild(h.rootCtx)
	span.AddField("name", "connect")
	span.AddField("network", network)
	span.AddField("addr", addr)
	h.connectionCtx = c
}

func (h *httpTracer) ConnectDone(network, addr string, err error) {
	h.l.Lock()
	defer h.l.Unlock()
	if h.rootSpan == nil {
		return
	}
	span := trace.GetSpanFromContext(h.connectionCtx)
	if err != nil {
		span.AddField("error", err.Error())
	}
	span.Send()
}

func (h *httpTracer) WroteRequest(info httptrace.WroteRequestInfo) {
	h.l.Lock()
	defer h.l.Unlock()
	if h.rootSpan == nil {
		return
	}
	c, span := h.rootSpan.CreateChild(h.rootCtx)
	span.AddField("name", "request")
	if info.Err != nil {
		span.AddField("error", info.Err.Error())
	}
	h.requestCtx = c
}

func (h *httpTracer) GotFirstResponseByte() {
	h.l.Lock()
	defer h.l.Unlock()
	if h.rootSpan == nil {
		return
	}

	span := trace.GetSpanFromContext(h.requestCtx)
	span.Send()
}
