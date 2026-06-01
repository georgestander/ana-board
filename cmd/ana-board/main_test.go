package main

import (
	"reflect"
	"testing"
)

func TestExtraAddrsParsesCommaSeparatedAddresses(t *testing.T) {
	got := extraAddrs("127.0.0.1:18080, 100.72.2.6:18080,,127.0.0.1:18080")
	want := []string{"127.0.0.1:18080", "100.72.2.6:18080"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("extraAddrs = %#v, want %#v", got, want)
	}
}
