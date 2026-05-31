package main

import (
	"encoding/json"
	"testing"
)

func TestHandleLineRespondsToPing(t *testing.T) {
	resp := handleLine(`{"jsonrpc":"2.0","id":1,"method":"ping"}`, nil)
	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.Error != nil {
		t.Fatalf("error = %#v, want nil", resp.Error)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("result = %T, want map", resp.Result)
	}
	if len(result) != 0 {
		t.Fatalf("result len = %d, want 0", len(result))
	}
}

func TestHandleLineNegotiatesSupportedProtocolVersion(t *testing.T) {
	resp := handleLine(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`, nil)
	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.Error != nil {
		t.Fatalf("error = %#v, want nil", resp.Error)
	}

	raw, err := json.Marshal(resp.Result)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	var result struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if result.ProtocolVersion != "2025-06-18" {
		t.Fatalf("protocolVersion = %q, want 2025-06-18", result.ProtocolVersion)
	}
}
