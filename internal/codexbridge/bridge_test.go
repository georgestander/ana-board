package codexbridge

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/georgestander/ana-board/internal/messages"
)

type recordingSender struct {
	requests []messages.SubmitRequest
	err      error
}

func (sender *recordingSender) Send(_ context.Context, req messages.SubmitRequest) error {
	sender.requests = append(sender.requests, req)
	return sender.err
}

func testConfig(t *testing.T, now time.Time) Config {
	t.Helper()
	base := t.TempDir()
	return Config{
		QueueDir:          filepath.Join(base, "queue"),
		StatePath:         filepath.Join(base, "state.json"),
		BoardURL:          "http://example.test",
		Source:            "codex",
		GlobalCooldown:    -1,
		KindCooldown:      -1,
		DuplicateCooldown: -1,
		MaxEventAge:       time.Hour,
		SendTimeout:       20 * time.Millisecond,
		Now: func() time.Time {
			return now
		},
	}
}

func TestBuildQueuedEventDistillsSwearAndSafeContextWithoutRawPrompt(t *testing.T) {
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	event, ok, err := BuildQueuedEvent("UserPromptSubmit", []byte(`{"prompt":"why the fuck is this not natural","cwd":"/Users/georgestander/Documents/ana-board","thread_id":"thread-secret-123"}`), now)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected event to be queued")
	}
	if event.Signal.Swear != "fuck" {
		t.Fatalf("swear = %q, want fuck", event.Signal.Swear)
	}
	if event.Context.Project != "ANA" {
		t.Fatalf("project = %q, want ANA", event.Context.Project)
	}
	if !strings.HasPrefix(event.Context.Thread, "T") || strings.Contains(event.Context.Thread, "secret") {
		t.Fatalf("thread context = %q, want hashed thread label", event.Context.Thread)
	}

	encoded := mustJSON(t, event)
	if strings.Contains(encoded, "why the fuck") || strings.Contains(encoded, "natural") || strings.Contains(encoded, "/Users/georgestander") || strings.Contains(encoded, "thread-secret") {
		t.Fatalf("queued event leaked raw prompt: %s", encoded)
	}

	action, ok := Decide(event, State{}, testConfig(t, now))
	if !ok {
		t.Fatal("expected action")
	}
	rendered := renderedRequestText(action.Request)
	if !strings.Contains(rendered, "FUCK") {
		t.Fatalf("message should keep profanity natural, got %q", rendered)
	}
	if !strings.Contains(rendered, "ANA") {
		t.Fatalf("message should include safe project context, got %q", rendered)
	}
	if len(action.Request.Placements) == 0 {
		t.Fatal("expected a block-art frame")
	}
	if strings.Contains(strings.ToLower(rendered), "why") {
		t.Fatalf("message leaked prompt content: %q", rendered)
	}
}

func TestBuildQueuedEventDropsBlandTurnEnded(t *testing.T) {
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	_, ok, err := BuildQueuedEvent("turn-ended", []byte(`{"last_message":"I inspected the files."}`), now)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected bland turn-ended event to be skipped")
	}
}

func TestBuildQueuedEventDoesNotTreatNoErrorsAsFailure(t *testing.T) {
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	event, ok, err := BuildQueuedEvent("turn-ended", []byte(`{"last_message":"tests passed with no errors","cwd":"/Users/georgestander/Documents/ana-board"}`), now)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected success event")
	}
	if event.Signal.Failure != "" {
		t.Fatalf("failure = %q, want empty", event.Signal.Failure)
	}
	if event.Signal.Success != "test" {
		t.Fatalf("success = %q, want test", event.Signal.Success)
	}
	if event.Context.Project != "ANA" {
		t.Fatalf("project = %q, want ANA", event.Context.Project)
	}
}

func TestDecideSuppressesDuplicateKindWithinCooldown(t *testing.T) {
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	config := testConfig(t, now)
	config.KindCooldown = time.Minute

	event, ok, err := BuildQueuedEvent("turn-ended", []byte(`{"last_message":"tests failed"}`), now)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected failure event")
	}

	action, ok := Decide(event, State{}, config)
	if !ok {
		t.Fatal("expected first event to send")
	}
	state := State{}
	state.mark(action.Kind, event.Digest, now)

	config.Now = func() time.Time { return now.Add(30 * time.Second) }
	if _, ok := Decide(event, state, config); ok {
		t.Fatal("expected duplicate failure to be suppressed inside cooldown")
	}
}

func TestDecideLetsFailureBypassGlobalCooldown(t *testing.T) {
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	config := testConfig(t, now)
	config.GlobalCooldown = time.Minute

	event, ok, err := BuildQueuedEvent("turn-ended", []byte(`{"last_message":"build failed"}`), now)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected failure event")
	}

	state := State{LastSentAt: now.Add(-10 * time.Second)}
	if _, ok := Decide(event, state, config); !ok {
		t.Fatal("expected failure to bypass global cooldown")
	}
}

func TestEnqueueAndProcessOncePostsThenDeletesSignal(t *testing.T) {
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	config := testConfig(t, now)

	result, err := Enqueue(config, "turn-ended", []byte(`{"last_message":"tests passed"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Queued {
		t.Fatal("expected event to queue")
	}
	if _, err := os.Stat(result.Path); err != nil {
		t.Fatal(err)
	}

	sender := &recordingSender{}
	stats, err := ProcessOnce(context.Background(), config, sender)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Sent != 1 || stats.Seen != 1 || stats.Dropped != 0 || stats.Errors != 0 {
		t.Fatalf("stats = %+v", stats)
	}
	if len(sender.requests) != 1 {
		t.Fatalf("sent requests = %d, want 1", len(sender.requests))
	}
	req := sender.requests[0]
	if req.Source != "codex" || req.Kind != "success" || req.Color != "green" {
		t.Fatalf("request = %+v", req)
	}
	rendered := renderedRequestText(req)
	if !strings.Contains(rendered, "TESTS") || !strings.Contains(rendered, "✅") {
		t.Fatalf("request should include contextual test success with emoji, got %q", rendered)
	}
	if len(req.Placements) == 0 {
		t.Fatal("expected block-art placements")
	}
	if _, err := os.Stat(result.Path); !os.IsNotExist(err) {
		t.Fatalf("queue file still exists or unexpected error: %v", err)
	}
}

func TestProcessOnceDropsOnSendErrorWithoutReturningError(t *testing.T) {
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	config := testConfig(t, now)
	result, err := Enqueue(config, "turn-ended", []byte(`{"last_message":"build failed"}`))
	if err != nil {
		t.Fatal(err)
	}

	sender := &recordingSender{err: errors.New("board down")}
	stats, err := ProcessOnce(context.Background(), config, sender)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Errors != 1 || stats.Dropped != 1 || stats.Sent != 0 {
		t.Fatalf("stats = %+v", stats)
	}
	if _, err := os.Stat(result.Path); !os.IsNotExist(err) {
		t.Fatalf("queue file still exists or unexpected error: %v", err)
	}
}

func TestHTTPSenderPostsMessage(t *testing.T) {
	var got messages.SubmitRequest
	sender := HTTPSender{
		BaseURL: "http://board.test",
		Timeout: time.Second,
		HTTPClient: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost || r.URL.Path != "/api/messages" {
				t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
			}
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			return &http.Response{
				StatusCode: http.StatusCreated,
				Body:       io.NopCloser(strings.NewReader(`{"status":"displayed"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	err := sender.Send(context.Background(), messages.SubmitRequest{
		Text:      "[green]TESTS ARE HAPPY.",
		Source:    "codex",
		Kind:      "success",
		Priority:  "normal",
		Animation: "row",
		Color:     "green",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != "codex" || got.Kind != "success" || got.Text == "" {
		t.Fatalf("request = %+v", got)
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := jsonMarshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

var jsonMarshal = json.Marshal

func renderedRequestText(req messages.SubmitRequest) string {
	if strings.TrimSpace(req.Text) != "" {
		return req.Text
	}
	var builder strings.Builder
	for _, placement := range req.Placements {
		builder.WriteString(placement.Symbol)
	}
	return builder.String()
}
