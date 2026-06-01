package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestHealthz(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestCurrentFrameStartsBlank(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/current", nil)
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got currentFrameResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}

	if got.Frame.Rows != 6 {
		t.Fatalf("Rows = %d, want 6", got.Frame.Rows)
	}

	if got.Frame.Cols != 22 {
		t.Fatalf("Cols = %d, want 22", got.Frame.Cols)
	}
}

func TestCreateMessageUpdatesCurrentFrame(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/messages",
		bytes.NewBufferString(`{"text":"hello","source":"test","animation":"row"}`),
	)
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	currentReq := httptest.NewRequest(http.MethodGet, "/api/current", nil)
	currentRec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(currentRec, currentReq)

	var got currentFrameResponse
	if err := json.NewDecoder(currentRec.Body).Decode(&got); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}

	if got.Frame.Cells[2][8] != "H" {
		t.Fatalf("centered cell = %q, want %q", got.Frame.Cells[2][8], "H")
	}
}

func TestCreateMessageRejectsHostReflectedOrigin(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/messages",
		bytes.NewBufferString(`{"text":"hello"}`),
	)
	req.Host = "attacker.test"
	req.Header.Set("Origin", "http://attacker.test")
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestCreateMessageAllowsConfiguredTrustedOrigin(t *testing.T) {
	srv, err := NewServer(WithTrustedOrigins([]string{"http://trusted.test"}))
	if err != nil {
		t.Fatalf("NewServer returned error: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/messages",
		bytes.NewBufferString(`{"text":"hello"}`),
	)
	req.Host = "attacker.test"
	req.Header.Set("Origin", "http://trusted.test")
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

func TestCreateMessageSupportsEmojiAndColor(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/messages",
		bytes.NewBufferString(`{"text":"[green]hello [blue]🌍","source":"test","animation":"row","kind":"info","color":"white"}`),
	)
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var got createMessageResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}

	if got.Frame.Cells[2][7] != "H" {
		t.Fatalf("first cell = %q, want H", got.Frame.Cells[2][7])
	}

	if got.Frame.Colors[2][7] != "green" {
		t.Fatalf("first color = %q, want green", got.Frame.Colors[2][7])
	}

	if got.Frame.Cells[2][13] != "🌍" {
		t.Fatalf("emoji cell = %q, want globe emoji", got.Frame.Cells[2][13])
	}

	if got.Frame.Colors[2][13] != "blue" {
		t.Fatalf("emoji color = %q, want blue", got.Frame.Colors[2][13])
	}
}

func TestCreateMessageSupportsColoredSegmentsAndNativeEmoji(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/messages",
		bytes.NewBufferString(`{"segments":[{"text":"ANA ","color":"green"},{"text":"READY 🎯","color":"red"}],"source":"test","animation":"row"}`),
	)
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var got createMessageResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}

	if got.Frame.Cells[2][15] != "🎯" {
		t.Fatalf("emoji cell = %q, want target emoji", got.Frame.Cells[2][15])
	}

	if got.Frame.Colors[2][5] != "green" {
		t.Fatalf("first color = %q, want green", got.Frame.Colors[2][5])
	}

	if got.Frame.Colors[2][15] != "red" {
		t.Fatalf("emoji color = %q, want red", got.Frame.Colors[2][15])
	}
}

func TestCreateMessageSupportsTileLevelColors(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/messages",
		bytes.NewBufferString(`{"tiles":[{"symbol":"A","color":"green"},{"symbol":"N","color":"amber"},{"symbol":"A","color":"red"},{"symbol":" ","color":"white"},{"symbol":"🫶","color":"violet"}],"source":"test"}`),
	)
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var got createMessageResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}

	if got.Frame.Cells[2][8] != "A" || got.Frame.Colors[2][8] != "green" {
		t.Fatalf("first tile = %q/%q, want A/green", got.Frame.Cells[2][8], got.Frame.Colors[2][8])
	}

	if got.Frame.Cells[2][9] != "N" || got.Frame.Colors[2][9] != "amber" {
		t.Fatalf("second tile = %q/%q, want N/amber", got.Frame.Cells[2][9], got.Frame.Colors[2][9])
	}

	if got.Frame.Cells[2][10] != "A" || got.Frame.Colors[2][10] != "red" {
		t.Fatalf("third tile = %q/%q, want A/red", got.Frame.Cells[2][10], got.Frame.Colors[2][10])
	}

	if got.Frame.Cells[2][12] != "🫶" || got.Frame.Colors[2][12] != "violet" {
		t.Fatalf("emoji tile = %q/%q, want emoji/violet", got.Frame.Cells[2][12], got.Frame.Colors[2][12])
	}

	if len(got.Message.Tiles) != 5 {
		t.Fatalf("stored tiles = %d, want 5", len(got.Message.Tiles))
	}

	currentReq := httptest.NewRequest(http.MethodGet, "/api/current", nil)
	currentRec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(currentRec, currentReq)

	if currentRec.Code != http.StatusOK {
		t.Fatalf("current status = %d, want %d; body=%s", currentRec.Code, http.StatusOK, currentRec.Body.String())
	}

	var current currentFrameResponse
	if err := json.NewDecoder(currentRec.Body).Decode(&current); err != nil {
		t.Fatalf("Decode current returned error: %v", err)
	}

	if current.Frame.Cells[2][9] != "N" || current.Frame.Colors[2][9] != "amber" {
		t.Fatalf("current second tile = %q/%q, want N/amber", current.Frame.Cells[2][9], current.Frame.Colors[2][9])
	}

	if current.Frame.Cells[2][12] != "🫶" || current.Frame.Colors[2][12] != "violet" {
		t.Fatalf("current emoji tile = %q/%q, want emoji/violet", current.Frame.Cells[2][12], current.Frame.Colors[2][12])
	}
}

func TestCreateMessageSupportsExactPlacements(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/messages",
		bytes.NewBufferString(`{"placements":[{"row":0,"col":0,"symbol":"A","color":"green"},{"row":5,"col":21,"symbol":"✅","color":"blue"}],"source":"test","kind":"info"}`),
	)
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var got createMessageResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}

	if got.Frame.Cells[0][0] != "A" || got.Frame.Colors[0][0] != "green" {
		t.Fatalf("top-left tile = %q/%q, want A/green", got.Frame.Cells[0][0], got.Frame.Colors[0][0])
	}
	if got.Frame.Cells[5][21] != "✅" || got.Frame.Colors[5][21] != "blue" {
		t.Fatalf("bottom-right tile = %q/%q, want check/blue", got.Frame.Cells[5][21], got.Frame.Colors[5][21])
	}
	if len(got.Message.Placements) != 2 {
		t.Fatalf("stored placements = %d, want 2", len(got.Message.Placements))
	}
}

func TestCreateMessageSupportsFullFrame(t *testing.T) {
	srv := newTestServer(t)

	cells := make([][]string, 6)
	colors := make([][]string, 6)
	for row := range cells {
		cells[row] = make([]string, 22)
		colors[row] = make([]string, 22)
		for col := range cells[row] {
			cells[row][col] = " "
			colors[row][col] = "white"
		}
	}
	cells[1][3] = "F"
	colors[1][3] = "amber"
	cells[4][20] = "🚀"
	colors[4][20] = "violet"

	body, err := json.Marshal(map[string]any{
		"frame": map[string]any{
			"cells":  cells,
			"colors": colors,
		},
		"source": "test",
	})
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var got createMessageResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}

	if got.Frame.Cells[1][3] != "F" || got.Frame.Colors[1][3] != "amber" {
		t.Fatalf("placed letter = %q/%q, want F/amber", got.Frame.Cells[1][3], got.Frame.Colors[1][3])
	}
	if got.Frame.Cells[4][20] != "🚀" || got.Frame.Colors[4][20] != "violet" {
		t.Fatalf("placed emoji = %q/%q, want rocket/violet", got.Frame.Cells[4][20], got.Frame.Colors[4][20])
	}
	if got.Message.Frame == nil {
		t.Fatal("stored frame = nil, want frame")
	}
}

func TestCreateMessageRejectsDuplicateExactPlacement(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/messages",
		bytes.NewBufferString(`{"placements":[{"row":0,"col":0,"symbol":"A"},{"row":0,"col":0,"symbol":"B"}]}`),
	)
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateMessageRejectsOutOfBoundsPlacement(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/messages",
		bytes.NewBufferString(`{"placements":[{"row":6,"col":0,"symbol":"A"}]}`),
	)
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateMessageRejectsMixedTextAndPlacements(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/messages",
		bytes.NewBufferString(`{"text":"hello","placements":[{"row":0,"col":0,"symbol":"A"}]}`),
	)
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateMessageRejectsTileWithMoreThanOneSymbol(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/messages",
		bytes.NewBufferString(`{"tiles":[{"symbol":"AB","color":"green"}]}`),
	)
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateMessageRejectsUnsupportedAnimation(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/messages",
		bytes.NewBufferString(`{"text":"hello","animation":"instant"}`),
	)
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateMessageRejectsMalformedJSON(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader(`{"text":`))
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateMessageRejectsTrailingJSON(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader(`{"text":"hello"} {}`))
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateMessageRejectsInvalidColor(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/messages",
		bytes.NewBufferString(`{"text":"hello","color":"neon"}`),
	)
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateMessageRejectsTooLargeMessage(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/messages",
		bytes.NewBufferString(`{"text":"ABCDEFGHIJKLMNOPQRSTUVW"}`),
	)
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestListMessages(t *testing.T) {
	srv := newTestServer(t)

	for _, text := range []string{"first", "second"} {
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/messages",
			bytes.NewBufferString(`{"text":"`+text+`"}`),
		)
		rec := httptest.NewRecorder()
		srv.Routes().ServeHTTP(rec, req)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/messages?limit=1", nil)
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got listMessagesResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}

	if len(got.Messages) != 1 {
		t.Fatalf("len(Messages) = %d, want 1", len(got.Messages))
	}
}

func TestListMessagesRejectsNonPositiveLimit(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/messages?limit=0", nil)
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestAdminPageLoads(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if !strings.Contains(rec.Body.String(), "Send Message") {
		t.Fatalf("admin page did not contain send form")
	}

	if got := rec.Header().Get("Content-Security-Policy"); got != "frame-ancestors 'self'" {
		t.Fatalf("Content-Security-Policy = %q, want frame-ancestors 'self'", got)
	}
	if got := rec.Header().Get("X-Frame-Options"); got != "SAMEORIGIN" {
		t.Fatalf("X-Frame-Options = %q, want SAMEORIGIN", got)
	}
}

func TestAdminCreateMessageUpdatesCurrentFrame(t *testing.T) {
	srv := newTestServer(t)
	form := url.Values{
		"text":   {"browser admin"},
		"source": {"admin-test"},
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/messages", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusSeeOther, rec.Body.String())
	}

	if location := rec.Header().Get("Location"); location != "/admin?status=sent" {
		t.Fatalf("Location = %q, want %q", location, "/admin?status=sent")
	}

	currentReq := httptest.NewRequest(http.MethodGet, "/api/current", nil)
	currentRec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(currentRec, currentReq)

	var got currentFrameResponse
	if err := json.NewDecoder(currentRec.Body).Decode(&got); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}

	if got.Frame.Cells[2][4] != "B" {
		t.Fatalf("centered cell = %q, want %q", got.Frame.Cells[2][4], "B")
	}
}

func TestAdminClearBlanksCurrentFrame(t *testing.T) {
	srv := newTestServer(t)

	createReq := httptest.NewRequest(
		http.MethodPost,
		"/api/messages",
		bytes.NewBufferString(`{"text":"clear me","animation":"row"}`),
	)
	createRec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(createRec, createReq)

	req := httptest.NewRequest(http.MethodPost, "/admin/clear", nil)
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}

	currentReq := httptest.NewRequest(http.MethodGet, "/api/current", nil)
	currentRec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(currentRec, currentReq)

	var got currentFrameResponse
	if err := json.NewDecoder(currentRec.Body).Decode(&got); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}

	for row := range got.Frame.Cells {
		for col := range got.Frame.Cells[row] {
			if got.Frame.Cells[row][col] != " " {
				t.Fatalf("cell [%d][%d] = %q, want blank", row, col, got.Frame.Cells[row][col])
			}
		}
	}
}

func TestAPIClearBlanksCurrentFrame(t *testing.T) {
	srv := newTestServer(t)

	createReq := httptest.NewRequest(
		http.MethodPost,
		"/api/messages",
		bytes.NewBufferString(`{"text":"clear me","animation":"row"}`),
	)
	createRec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(createRec, createReq)

	req := httptest.NewRequest(http.MethodPost, "/api/clear", nil)
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got currentFrameResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}

	for row := range got.Frame.Cells {
		for col := range got.Frame.Cells[row] {
			if got.Frame.Cells[row][col] != " " {
				t.Fatalf("cell [%d][%d] = %q, want blank", row, col, got.Frame.Cells[row][col])
			}
		}
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()

	srv, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer returned error: %v", err)
	}

	return srv
}
