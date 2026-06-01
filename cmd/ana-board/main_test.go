package main

import (
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestExtraAddrsParsesCommaSeparatedAddresses(t *testing.T) {
	got := extraAddrs("127.0.0.1:18080, 100.72.2.6:18080,,127.0.0.1:18080")
	want := []string{"127.0.0.1:18080", "100.72.2.6:18080"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("extraAddrs = %#v, want %#v", got, want)
	}
}

func TestNewHTTPServerSetsResourceLimits(t *testing.T) {
	srv := newHTTPServer("127.0.0.1:0", http.NewServeMux())

	if srv.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("ReadHeaderTimeout = %s, want 5s", srv.ReadHeaderTimeout)
	}
	if srv.ReadTimeout != 15*time.Second {
		t.Fatalf("ReadTimeout = %s, want 15s", srv.ReadTimeout)
	}
	if srv.IdleTimeout != 60*time.Second {
		t.Fatalf("IdleTimeout = %s, want 60s", srv.IdleTimeout)
	}
	if srv.MaxHeaderBytes != 1<<20 {
		t.Fatalf("MaxHeaderBytes = %d, want %d", srv.MaxHeaderBytes, 1<<20)
	}
}
