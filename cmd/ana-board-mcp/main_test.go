package main

import (
	"bytes"
	"encoding/json"
	"strings"
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

func TestInitializeReportsServerVersion(t *testing.T) {
	resp := handleLine(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`, nil)
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
		ServerInfo struct {
			Version string `json:"version"`
		} `json:"serverInfo"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if result.ServerInfo.Version != serverVersion {
		t.Fatalf("version = %q, want %q", result.ServerInfo.Version, serverVersion)
	}
}

func TestServeRejectsOversizedLineAndContinues(t *testing.T) {
	input := strings.NewReader(strings.Repeat("A", maxRequestLineBytes+1) + "\n" + `{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n")
	var output bytes.Buffer

	if err := serve(input, &output, nil); err != nil {
		t.Fatalf("serve returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("response lines = %d, want 2; output=%q", len(lines), output.String())
	}

	var tooLarge rpcResponse
	if err := json.Unmarshal([]byte(lines[0]), &tooLarge); err != nil {
		t.Fatalf("Unmarshal oversized response returned error: %v", err)
	}
	if tooLarge.Error == nil || tooLarge.Error.Code != -32600 {
		t.Fatalf("oversized error = %#v, want -32600", tooLarge.Error)
	}

	var ping rpcResponse
	if err := json.Unmarshal([]byte(lines[1]), &ping); err != nil {
		t.Fatalf("Unmarshal ping response returned error: %v", err)
	}
	if ping.Error != nil {
		t.Fatalf("ping error = %#v, want nil", ping.Error)
	}
}

func TestParseSendArgsSupportsExactPlacements(t *testing.T) {
	req, err := parseSendArgs([]byte(`{"placements":[{"row":0,"col":0,"symbol":"A","color":"green"}],"source":"codex"}`))
	if err != nil {
		t.Fatalf("parseSendArgs returned error: %v", err)
	}

	if len(req.Placements) != 1 {
		t.Fatalf("placements = %d, want 1", len(req.Placements))
	}
	if req.Placements[0].Symbol != "A" {
		t.Fatalf("symbol = %q, want A", req.Placements[0].Symbol)
	}
}

func TestParseSendArgsRejectsMixedTextAndFrame(t *testing.T) {
	_, err := parseSendArgs([]byte(`{"text":"hello","frame":{"cells":[]}}`))
	if err == nil {
		t.Fatal("parseSendArgs returned nil error, want mixed payload error")
	}
}

func TestRecentToolRejectsLimitAboveSchemaMaximum(t *testing.T) {
	got := recentTool([]byte(`{"limit":51}`), nil)
	if !got.IsError {
		t.Fatal("recentTool IsError = false, want true")
	}
	if got.Content[0].Text != "limit must be less than or equal to 50" {
		t.Fatalf("error = %q, want max limit error", got.Content[0].Text)
	}
}
