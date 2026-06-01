package testutil

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func NewLocalServer(tb testing.TB, handler http.Handler) *httptest.Server {
	tb.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		tb.Skipf("skipping test that requires a local listener: %v", err)
	}
	ts := &httptest.Server{
		Listener: listener,
		Config:   &http.Server{Handler: handler},
	}
	ts.Start()
	return ts
}
