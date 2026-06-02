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
	if strings.Contains(rendered, event.Context.Thread) {
		t.Fatalf("message displayed internal thread hash %q: %q", event.Context.Thread, rendered)
	}
	assertNoBridgeFiller(t, rendered)
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

func TestBuildQueuedEventDropsGenericCompletionTurnEnded(t *testing.T) {
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	_, ok, err := BuildQueuedEvent("turn-ended", []byte(`{"last_message":"implemented and complete","cwd":"/Users/georgestander/dev/clients/valueinresearch/vir_2030","thread_id":"success-thread-secret"}`), now)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected generic completion to be skipped")
	}
}

func TestBuildQueuedEventIgnoresDelegationPromptFailureText(t *testing.T) {
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	_, ok, err := BuildQueuedEvent("turn-ended", []byte(`{"input":"Ana Board notifier smoke test failed loudly","last_message":"implemented and complete","cwd":"/Users/georgestander/Documents/ana-board","thread_id":"delegation-thread-secret"}`), now)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected prompt/input failure wording to be ignored")
	}
}

func TestBuildQueuedEventDetectsRequestUserInputQuestion(t *testing.T) {
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	event, ok, err := BuildQueuedEvent("PostToolUse", []byte(`{"tool_name":"request_user_input","questions":[{"header":"Scope","id":"release_scope"}],"cwd":"/Users/georgestander/Documents/ana-board","thread_id":"planning-thread-secret"}`), now)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected question event")
	}
	if !event.Signal.Question {
		t.Fatalf("signal = %+v, want question", event.Signal)
	}
	if event.Context.Project != "ANA" {
		t.Fatalf("project = %q, want ANA", event.Context.Project)
	}
	if event.Context.Topic != "SCOPE" {
		t.Fatalf("topic = %q, want SCOPE", event.Context.Topic)
	}

	action, ok := Decide(event, State{}, testConfig(t, now))
	if !ok {
		t.Fatal("expected action")
	}
	if action.Kind != "question" {
		t.Fatalf("kind = %q, want question", action.Kind)
	}
	req := action.Request
	if req.Kind != "task" || req.Priority != "high" || req.Color != "amber" {
		t.Fatalf("request = %+v", req)
	}
	if len(req.Placements) < 20 {
		t.Fatalf("expected a fallback block-art frame, got %d placements", len(req.Placements))
	}
	rendered := renderedRequestText(req)
	if !strings.Contains(rendered, "ANA") || !strings.Contains(rendered, "SCOPE?") || !strings.Contains(rendered, "QUESTION❓") {
		t.Fatalf("question frame missing useful context: %q", rendered)
	}
	assertNoBridgeFiller(t, rendered)
	if strings.Contains(rendered, "secret") || strings.Contains(rendered, "/Users/georgestander") || strings.Contains(rendered, "release") {
		t.Fatalf("question frame leaked raw context: %q", rendered)
	}
	if strings.Contains(rendered, event.Context.Thread) {
		t.Fatalf("question frame displayed internal thread hash %q: %q", event.Context.Thread, rendered)
	}
}

func TestBuildQueuedEventDetectsPlainAssistantQuestion(t *testing.T) {
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	event, ok, err := BuildQueuedEvent("turn-ended", []byte(`{"last_message":"Which route should I take?","cwd":"/Users/georgestander/Documents/ana-board"}`), now)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected question event")
	}
	if !event.Signal.Question {
		t.Fatalf("signal = %+v, want question", event.Signal)
	}
}

func TestBuildQueuedEventRendersMinimalApprovalFrame(t *testing.T) {
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	event, ok, err := BuildQueuedEvent("PermissionRequest", []byte(`{"tool_name":"shell_command","title":"Run Tests","cwd":"/Users/georgestander/Documents/ana-board","thread_id":"approval-thread-secret"}`), now)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected approval event")
	}
	if !event.Signal.Approval {
		t.Fatalf("signal = %+v, want approval", event.Signal)
	}
	if event.Context.Topic != "RUN TESTS" {
		t.Fatalf("topic = %q, want RUN TESTS", event.Context.Topic)
	}

	action, ok := Decide(event, State{}, testConfig(t, now))
	if !ok {
		t.Fatal("expected action")
	}
	if action.Kind != "approval" {
		t.Fatalf("kind = %q, want approval", action.Kind)
	}
	req := action.Request
	if req.Kind != "warning" || req.Priority != "high" || req.Color != "amber" {
		t.Fatalf("request = %+v", req)
	}
	if len(req.Placements) < 20 {
		t.Fatalf("expected a fallback block-art frame, got %d placements", len(req.Placements))
	}
	rendered := renderedRequestText(req)
	if !strings.Contains(rendered, "ANA") || !strings.Contains(rendered, "RUNTESTS?") || !strings.Contains(rendered, "OKNEEDED") {
		t.Fatalf("approval frame missing useful context: %q", rendered)
	}
	assertNoBridgeFiller(t, rendered)
	if strings.Contains(rendered, "secret") || strings.Contains(rendered, "/Users/georgestander") || strings.Contains(rendered, "shell") {
		t.Fatalf("approval frame leaked raw context: %q", rendered)
	}
	if strings.Contains(rendered, event.Context.Thread) {
		t.Fatalf("approval frame displayed internal thread hash %q: %q", event.Context.Thread, rendered)
	}
}

func TestBuildQueuedEventIgnoresPlainPlanUpdate(t *testing.T) {
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	_, ok, err := BuildQueuedEvent("PostToolUse", []byte(`{"tool_name":"update_plan","message":"plan updated","cwd":"/Users/georgestander/Documents/ana-board"}`), now)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected update_plan event without a question to be skipped")
	}
}

func TestBuildQueuedEventDropsRoutineTestPasses(t *testing.T) {
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	_, ok, err := BuildQueuedEvent("turn-ended", []byte(`{"last_message":"tests passed with no errors","cwd":"/Users/georgestander/Documents/ana-board"}`), now)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected routine test pass to be skipped")
	}
}

func TestBuildQueuedEventRendersMinimalMilestoneFrame(t *testing.T) {
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	event, ok, err := BuildQueuedEvent("turn-ended", []byte(`{"last_message":"v0.5.7 released and pushed to GitHub","cwd":"/Users/georgestander/Documents/ana-board","thread_id":"success-thread-secret"}`), now)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected milestone event")
	}
	if event.Signal.Success != "release" {
		t.Fatalf("success = %q, want release", event.Signal.Success)
	}

	action, ok := Decide(event, State{}, testConfig(t, now))
	if !ok {
		t.Fatal("expected action")
	}
	if action.Kind != "success:release" {
		t.Fatalf("kind = %q, want success:release", action.Kind)
	}
	if len(action.Request.Placements) < 20 {
		t.Fatalf("expected a fallback milestone frame, got %d placements", len(action.Request.Placements))
	}
	rendered := renderedRequestText(action.Request)
	if !strings.Contains(rendered, "ANA") || !strings.Contains(rendered, "RELEASED") || !strings.Contains(rendered, "✅") {
		t.Fatalf("milestone frame missing context: %q", rendered)
	}
	assertNoBridgeFiller(t, rendered)
	if strings.Contains(rendered, "success-thread") {
		t.Fatalf("milestone frame leaked thread: %q", rendered)
	}
	if strings.Contains(rendered, event.Context.Thread) {
		t.Fatalf("milestone frame displayed internal thread hash %q: %q", event.Context.Thread, rendered)
	}
}

func TestBuildQueuedEventRendersMinimalFailureFrame(t *testing.T) {
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	event, ok, err := BuildQueuedEvent("turn-ended", []byte(`{"last_message":"command failed with panic","cwd":"/Users/georgestander/Documents/ana-board","thread_id":"failure-thread-secret"}`), now)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected failure event")
	}

	action, ok := Decide(event, State{}, testConfig(t, now))
	if !ok {
		t.Fatal("expected action")
	}
	if action.Kind != "failure" {
		t.Fatalf("kind = %q, want failure", action.Kind)
	}
	if len(action.Request.Placements) < 15 {
		t.Fatalf("expected a fallback failure frame, got %d placements", len(action.Request.Placements))
	}
	rendered := renderedRequestText(action.Request)
	if !strings.Contains(rendered, "ANA") || !strings.Contains(rendered, "FAILED") || !strings.Contains(rendered, "❌") {
		t.Fatalf("failure frame missing context: %q", rendered)
	}
	assertNoBridgeFiller(t, rendered)
	if strings.Contains(rendered, "failure-thread") {
		t.Fatalf("failure frame leaked thread: %q", rendered)
	}
	if strings.Contains(rendered, event.Context.Thread) {
		t.Fatalf("failure frame displayed internal thread hash %q: %q", event.Context.Thread, rendered)
	}
}

func TestBuildQueuedEventClassifiesExplicitTestFailureOnly(t *testing.T) {
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	event, ok, err := BuildQueuedEvent("turn-ended", []byte(`{"last_message":"tests failed in fresh thread smoke","cwd":"/Users/georgestander/Documents/ana-board","thread_id":"test-failure-thread-secret"}`), now)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected explicit test failure event")
	}
	if event.Signal.Failure != "test" {
		t.Fatalf("failure = %q, want test", event.Signal.Failure)
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

func TestDecideLetsQuestionBypassGlobalCooldown(t *testing.T) {
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	config := testConfig(t, now)
	config.GlobalCooldown = time.Minute

	event, ok, err := BuildQueuedEvent("PostToolUse", []byte(`{"tool_name":"request_user_input"}`), now)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected question event")
	}

	state := State{LastSentAt: now.Add(-10 * time.Second)}
	if _, ok := Decide(event, state, config); !ok {
		t.Fatal("expected question to bypass global cooldown")
	}
}

func TestEnqueueAndProcessOncePostsThenDeletesSignal(t *testing.T) {
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	config := testConfig(t, now)

	result, err := Enqueue(config, "turn-ended", []byte(`{"last_message":"v0.5.7 released and pushed to GitHub","cwd":"/Users/georgestander/Documents/ana-board"}`))
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
	if !strings.Contains(rendered, "RELEASED") || !strings.Contains(rendered, "✅") {
		t.Fatalf("request should include contextual milestone success with emoji, got %q", rendered)
	}
	assertNoBridgeFiller(t, rendered)
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

func assertNoBridgeFiller(t *testing.T, rendered string) {
	t.Helper()
	forbidden := []string{
		"OPENQUESTION",
		"ANSWERTOUNSTICK",
		"UNSTICK",
		"FROMME",
		"YOURCALL",
		"APPROVEORNIX",
		"NEEDSYOU",
		"RE:",
		"THREAD",
		"LANDED",
		"SOMETHINGSNAPPED",
		"SNAPPED",
	}
	for _, phrase := range forbidden {
		if strings.Contains(rendered, phrase) {
			t.Fatalf("bridge rendered stale filler %q in %q", phrase, rendered)
		}
	}
}
