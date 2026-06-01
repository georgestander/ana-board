package server

import "testing"

func TestTrustedOriginsForAddrsUsesConfiguredBindAddresses(t *testing.T) {
	got := TrustedOriginsForAddrs("127.0.0.1:8080", "100.72.2.6:18080", "0.0.0.0:8081")
	want := []string{
		"http://localhost:8080",
		"http://127.0.0.1:8080",
		"http://[::1]:8080",
		"http://100.72.2.6:18080",
		"http://localhost:8081",
		"http://127.0.0.1:8081",
		"http://[::1]:8081",
	}

	if len(got) != len(want) {
		t.Fatalf("len(origins) = %d, want %d: %#v", len(got), len(want), got)
	}

	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("origin[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}

func TestParseTrustedOriginsRejectsPaths(t *testing.T) {
	_, err := ParseTrustedOrigins("http://ana-board.local:8080/admin")
	if err == nil {
		t.Fatal("ParseTrustedOrigins returned nil error, want invalid origin error")
	}
}
