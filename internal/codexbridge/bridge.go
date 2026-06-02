package codexbridge

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/georgestander/ana-board/internal/board"
	"github.com/georgestander/ana-board/internal/client"
	"github.com/georgestander/ana-board/internal/messages"
)

const (
	defaultGlobalCooldown    = 30 * time.Second
	defaultKindCooldown      = 3 * time.Minute
	defaultDuplicateCooldown = 10 * time.Minute
	defaultMaxEventAge       = time.Hour
	defaultSendTimeout       = 1200 * time.Millisecond
	defaultSource            = "codex"
)

type Config struct {
	QueueDir          string
	StatePath         string
	BoardURL          string
	Source            string
	GlobalCooldown    time.Duration
	KindCooldown      time.Duration
	DuplicateCooldown time.Duration
	MaxEventAge       time.Duration
	SendTimeout       time.Duration
	Now               func() time.Time
}

type Signal struct {
	Swear       string `json:"swear,omitempty"`
	Celebration bool   `json:"celebration,omitempty"`
	Failure     string `json:"failure,omitempty"`
	Success     string `json:"success,omitempty"`
	Approval    bool   `json:"approval,omitempty"`
	Question    bool   `json:"question,omitempty"`
}

type QueuedEvent struct {
	ID         string    `json:"id"`
	Event      string    `json:"event"`
	ReceivedAt time.Time `json:"received_at"`
	Signal     Signal    `json:"signal"`
	Context    Context   `json:"context"`
	Digest     string    `json:"digest"`
}

type EnqueueResult struct {
	Queued bool
	Path   string
	Event  QueuedEvent
}

type ProcessStats struct {
	Seen    int `json:"seen"`
	Sent    int `json:"sent"`
	Dropped int `json:"dropped"`
	Errors  int `json:"errors"`
}

type State struct {
	LastSentAt   time.Time            `json:"last_sent_at,omitempty"`
	LastByKind   map[string]time.Time `json:"last_by_kind,omitempty"`
	LastByDigest map[string]time.Time `json:"last_by_digest,omitempty"`
}

type Action struct {
	Kind    string
	Request messages.SubmitRequest
}

type Sender interface {
	Send(ctx context.Context, req messages.SubmitRequest) error
}

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type HTTPSender struct {
	BaseURL    string
	Timeout    time.Duration
	HTTPClient httpDoer
}

func DefaultConfig() Config {
	cacheDir, err := os.UserCacheDir()
	if err != nil || strings.TrimSpace(cacheDir) == "" {
		cacheDir = os.TempDir()
	}

	base := filepath.Join(cacheDir, "ana-board", "codex-bridge")
	return Config{
		QueueDir:          envString("ANA_BOARD_CODEX_QUEUE_DIR", filepath.Join(base, "queue")),
		StatePath:         envString("ANA_BOARD_CODEX_STATE", filepath.Join(base, "state.json")),
		BoardURL:          envString("ANA_BOARD_URL", client.DefaultBaseURL),
		Source:            envString("ANA_BOARD_CODEX_SOURCE", defaultSource),
		GlobalCooldown:    envDurationSeconds("ANA_BOARD_CODEX_GLOBAL_COOLDOWN_SECONDS", defaultGlobalCooldown),
		KindCooldown:      envDurationSeconds("ANA_BOARD_CODEX_KIND_COOLDOWN_SECONDS", defaultKindCooldown),
		DuplicateCooldown: envDurationSeconds("ANA_BOARD_CODEX_DUPLICATE_COOLDOWN_SECONDS", defaultDuplicateCooldown),
		MaxEventAge:       envDurationSeconds("ANA_BOARD_CODEX_MAX_EVENT_AGE_SECONDS", defaultMaxEventAge),
		SendTimeout:       envDurationMillis("ANA_BOARD_CODEX_SEND_TIMEOUT_MS", defaultSendTimeout),
		Now:               time.Now,
	}
}

func Enqueue(config Config, eventName string, payload []byte) (EnqueueResult, error) {
	config = config.withDefaults()
	event, ok, err := BuildQueuedEvent(eventName, payload, config.now())
	if err != nil {
		return EnqueueResult{}, err
	}
	if !ok {
		return EnqueueResult{Queued: false}, nil
	}

	if err := os.MkdirAll(config.QueueDir, 0o700); err != nil {
		return EnqueueResult{}, err
	}

	finalPath := filepath.Join(config.QueueDir, event.ID+".json")
	tmpPath := filepath.Join(config.QueueDir, "."+event.ID+".tmp")

	data, err := json.Marshal(event)
	if err != nil {
		return EnqueueResult{}, err
	}
	data = append(data, '\n')

	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return EnqueueResult{}, err
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return EnqueueResult{}, err
	}

	return EnqueueResult{Queued: true, Path: finalPath, Event: event}, nil
}

func BuildQueuedEvent(eventName string, payload []byte, now time.Time) (QueuedEvent, bool, error) {
	eventName = normalizeEventName(eventName)
	fields, err := extractTextFields(payload)
	if err != nil {
		return QueuedEvent{}, false, err
	}

	signal := classify(eventName, fields)
	if signal.empty() {
		return QueuedEvent{}, false, nil
	}

	eventContext := extractContext(fields)
	if !signal.Question && !signal.Approval {
		eventContext.Topic = ""
	}
	digest := eventDigest(eventName, signal, eventContext)
	receivedAt := now.UTC()
	id := fmt.Sprintf("%s%09dZ-%d-%s", receivedAt.Format("20060102T150405"), receivedAt.Nanosecond(), os.Getpid(), digest[:10])
	return QueuedEvent{
		ID:         id,
		Event:      eventName,
		ReceivedAt: receivedAt,
		Signal:     signal,
		Context:    eventContext,
		Digest:     digest,
	}, true, nil
}

func ProcessOnce(ctx context.Context, config Config, sender Sender) (ProcessStats, error) {
	config = config.withDefaults()
	if sender == nil {
		sender = HTTPSender{BaseURL: config.BoardURL, Timeout: config.SendTimeout}
	}

	state, err := loadState(config.StatePath)
	if err != nil {
		return ProcessStats{}, err
	}

	entries, err := os.ReadDir(config.QueueDir)
	if err != nil {
		if os.IsNotExist(err) {
			return ProcessStats{}, nil
		}
		return ProcessStats{}, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	now := config.now()
	stats := ProcessStats{}
	stateChanged := false
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		stats.Seen++
		path := filepath.Join(config.QueueDir, entry.Name())
		event, err := readQueuedEvent(path)
		if err != nil {
			stats.Dropped++
			stateChanged = true
			_ = os.Remove(path)
			continue
		}

		if config.MaxEventAge > 0 && now.Sub(event.ReceivedAt) > config.MaxEventAge {
			stats.Dropped++
			_ = os.Remove(path)
			continue
		}

		action, ok := Decide(event, state, config)
		if !ok {
			stats.Dropped++
			_ = os.Remove(path)
			continue
		}

		sendCtx, cancel := context.WithTimeout(ctx, config.SendTimeout)
		err = sender.Send(sendCtx, action.Request)
		cancel()
		if err != nil {
			stats.Errors++
			stats.Dropped++
			_ = os.Remove(path)
			continue
		}

		stats.Sent++
		state.mark(action.Kind, event.Digest, now)
		stateChanged = true
		_ = os.Remove(path)
	}

	if stateChanged {
		if err := saveState(config.StatePath, state); err != nil {
			return stats, err
		}
	}

	return stats, nil
}

func Decide(event QueuedEvent, state State, config Config) (Action, bool) {
	config = config.withDefaults()
	state.ensure()
	now := config.now()

	kind := actionKind(event)
	if kind == "" {
		return Action{}, false
	}

	if last, ok := state.LastByKind[kind]; ok && config.KindCooldown > 0 && now.Sub(last) < config.KindCooldown {
		return Action{}, false
	}
	if last, ok := state.LastByDigest[event.Digest]; ok && config.DuplicateCooldown > 0 && now.Sub(last) < config.DuplicateCooldown {
		return Action{}, false
	}
	if !isInterruptiveKind(kind) && config.GlobalCooldown > 0 && !state.LastSentAt.IsZero() && now.Sub(state.LastSentAt) < config.GlobalCooldown {
		return Action{}, false
	}

	req := requestForKind(kind, event, config.Source)
	return Action{Kind: kind, Request: req}, true
}

func (s HTTPSender) Send(ctx context.Context, req messages.SubmitRequest) error {
	baseURL := strings.TrimSpace(s.BaseURL)
	if baseURL == "" {
		baseURL = client.DefaultBaseURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("parse board URL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("board URL must include scheme and host")
	}

	timeout := s.Timeout
	if timeout <= 0 {
		timeout = defaultSendTimeout
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpClient := s.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(parsed.String(), "/")+"/api/messages", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("board request failed: status %d %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	return nil
}

func (config Config) withDefaults() Config {
	defaults := DefaultConfig()
	if strings.TrimSpace(config.QueueDir) == "" {
		config.QueueDir = defaults.QueueDir
	}
	if strings.TrimSpace(config.StatePath) == "" {
		config.StatePath = defaults.StatePath
	}
	if strings.TrimSpace(config.BoardURL) == "" {
		config.BoardURL = defaults.BoardURL
	}
	if strings.TrimSpace(config.Source) == "" {
		config.Source = defaults.Source
	}
	if config.GlobalCooldown == 0 {
		config.GlobalCooldown = defaults.GlobalCooldown
	}
	if config.KindCooldown == 0 {
		config.KindCooldown = defaults.KindCooldown
	}
	if config.DuplicateCooldown == 0 {
		config.DuplicateCooldown = defaults.DuplicateCooldown
	}
	if config.MaxEventAge == 0 {
		config.MaxEventAge = defaults.MaxEventAge
	}
	if config.SendTimeout == 0 {
		config.SendTimeout = defaults.SendTimeout
	}
	if config.Now == nil {
		config.Now = defaults.Now
	}
	return config
}

func (config Config) now() time.Time {
	if config.Now == nil {
		return time.Now().UTC()
	}
	return config.Now().UTC()
}

func (signal Signal) empty() bool {
	return signal.Swear == "" && !signal.Celebration && signal.Failure == "" && signal.Success == "" && !signal.Approval && !signal.Question
}

func (state *State) ensure() {
	if state.LastByKind == nil {
		state.LastByKind = map[string]time.Time{}
	}
	if state.LastByDigest == nil {
		state.LastByDigest = map[string]time.Time{}
	}
}

func (state *State) mark(kind, digest string, now time.Time) {
	state.ensure()
	state.LastSentAt = now.UTC()
	state.LastByKind[kind] = now.UTC()
	state.LastByDigest[digest] = now.UTC()
}

func loadState(path string) (State, error) {
	var state State
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			state.ensure()
			return state, nil
		}
		return State{}, err
	}
	if strings.TrimSpace(string(data)) == "" {
		state.ensure()
		return state, nil
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, err
	}
	state.ensure()
	return state, nil
}

func saveState(path string, state State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

func readQueuedEvent(path string) (QueuedEvent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return QueuedEvent{}, err
	}
	var event QueuedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return QueuedEvent{}, err
	}
	if event.ID == "" || event.Event == "" || event.ReceivedAt.IsZero() || event.Digest == "" {
		return QueuedEvent{}, fmt.Errorf("queued event is missing required fields")
	}
	return event, nil
}

func normalizeEventName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return value
}

func eventDigest(eventName string, signal Signal, eventContext Context) string {
	data, _ := json.Marshal(struct {
		Event   string  `json:"event"`
		Signal  Signal  `json:"signal"`
		Context Context `json:"context"`
	}{Event: eventName, Signal: signal, Context: eventContext})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func actionKind(event QueuedEvent) string {
	eventName := strings.ToLower(event.Event)
	if event.Signal.Swear != "" && strings.Contains(eventName, "prompt") {
		return "swear"
	}
	if event.Signal.Failure != "" {
		if event.Signal.Failure == "test" {
			return "failure:test"
		}
		if event.Signal.Failure == "build" {
			return "failure:build"
		}
		return "failure"
	}
	if event.Signal.Question {
		return "question"
	}
	if event.Signal.Approval {
		return "approval"
	}
	if event.Signal.Swear != "" {
		return "swear"
	}
	if event.Signal.Celebration {
		return "celebration"
	}
	if event.Signal.Success != "" {
		switch event.Signal.Success {
		case "release", "deploy", "push", "tag", "merge", "milestone":
			return "success:" + event.Signal.Success
		}
	}
	return ""
}

func isInterruptiveKind(kind string) bool {
	return strings.HasPrefix(kind, "failure") || kind == "approval" || kind == "question"
}

func requestForKind(kind string, event QueuedEvent, source string) messages.SubmitRequest {
	req := messages.SubmitRequest{
		Source:    source,
		Priority:  messages.DefaultPriority,
		Animation: messages.DefaultAnimation,
		Kind:      messages.DefaultKind,
		Color:     messages.DefaultColor,
	}

	switch kind {
	case "failure", "failure:test":
		req.Kind = "error"
		req.Priority = "high"
		req.Color = "red"
	case "failure:build", "approval":
		req.Kind = "warning"
		req.Priority = "high"
		req.Color = "amber"
	case "question":
		req.Kind = "task"
		req.Priority = "high"
		req.Color = "amber"
	case "success", "success:test", "success:build", "success:release", "success:deploy", "success:push", "success:tag", "success:merge", "success:milestone", "celebration":
		req.Kind = "success"
		req.Color = "green"
	case "swear":
		req.Kind = "info"
		req.Color = "violet"
	}

	if placements, ok := composeFrame(kind, event); ok {
		req.Placements = placements
		return req
	}

	req.Text = fallbackText(kind, event)
	return req
}

func composeFrame(kind string, event QueuedEvent) ([]messages.PlacedTile, bool) {
	alert, ok := fallbackAlert(kind, event)
	if !ok {
		return nil, false
	}

	icon := iconPattern(kind)
	if len(icon) == 0 {
		return nil, false
	}

	placements := make([]messages.PlacedTile, 0, 80)
	if !addTextLine(&placements, 0, 0, board.DefaultCols, alert.project, "blue") {
		return nil, false
	}

	for rowOffset, line := range icon {
		for colOffset, marker := range line {
			color, ok := iconColors[marker]
			if !ok {
				continue
			}
			placements = append(placements, messages.PlacedTile{
				Row:    2 + rowOffset,
				Col:    1 + colOffset,
				Symbol: "█",
				Color:  color,
			})
		}
	}

	if !addTextLine(&placements, 2, 7, board.DefaultCols-7, alert.line1, alert.color) {
		return nil, false
	}
	if alert.line2 != "" && !addTextLine(&placements, 4, 7, board.DefaultCols-7, alert.line2, alert.color) {
		return nil, false
	}
	return placements, len(placements) != 0
}

func addTextLine(placements *[]messages.PlacedTile, row, col, max int, text, color string) bool {
	symbols, err := board.NormalizeSymbols(text)
	if err != nil {
		return false
	}
	symbols = fitSymbols(symbols, max)
	for colOffset, symbol := range symbols {
		if symbol == " " {
			continue
		}
		*placements = append(*placements, messages.PlacedTile{
			Row:    row,
			Col:    col + colOffset,
			Symbol: symbol,
			Color:  color,
		})
	}
	return true
}

type bridgeAlert struct {
	project string
	line1   string
	line2   string
	color   string
}

func fallbackAlert(kind string, event QueuedEvent) (bridgeAlert, bool) {
	alert := bridgeAlert{
		project: event.Context.DisplayLabel(),
		color:   textColorForKind(kind),
	}
	topic := strings.TrimSpace(event.Context.Topic)
	switch kind {
	case "success:test":
		alert.line1 = "TESTS GREEN ✅"
	case "success:build":
		alert.line1 = "BUILD OK ✅"
	case "success":
		alert.line1 = "DONE ✅"
	case "success:release":
		alert.line1 = "RELEASED ✅"
	case "success:deploy":
		alert.line1 = "DEPLOYED 🚀"
	case "success:push":
		alert.line1 = "PUSHED ✅"
	case "success:tag":
		alert.line1 = "TAGGED ✅"
	case "success:merge":
		alert.line1 = "MERGED ✅"
	case "success:milestone":
		alert.line1 = "MILESTONE ✅"
	case "failure:test":
		alert.line1 = "TESTS FAILED ❌"
	case "failure:build":
		alert.line1 = "BUILD FAILED ❌"
	case "failure":
		alert.line1 = "FAILED ❌"
	case "celebration":
		alert.line1 = "NICE ✅"
	case "swear":
		alert.line1 = swearFallback(event.Signal.Swear)
	case "approval":
		if topic != "" {
			alert.line1 = topic + "?"
			alert.line2 = "OK NEEDED ✋"
		} else {
			alert.line1 = "OK NEEDED ✋"
		}
	case "question":
		if topic != "" {
			alert.line1 = topic + "?"
			alert.line2 = "QUESTION ❓"
		} else {
			alert.line1 = "QUESTION ❓"
		}
	default:
		return bridgeAlert{}, false
	}
	if alert.line1 == "" {
		return bridgeAlert{}, false
	}
	return alert, true
}

func swearFallback(swear string) string {
	if swear == "" {
		return "I HEAR YOU 😂"
	}
	return strings.ToUpper(swear) + " NOTED 😂"
}

func fitSymbols(symbols []string, max int) []string {
	if len(symbols) <= max {
		return symbols
	}
	if max <= 0 {
		return nil
	}
	return symbols[:max]
}

func fallbackText(kind string, event QueuedEvent) string {
	alert, ok := fallbackAlert(kind, event)
	if !ok {
		return "[blue]" + event.Context.DisplayLabel() + " MOVED."
	}
	color := textColorForKind(kind)
	parts := []string{alert.project, alert.line1}
	if alert.line2 != "" {
		parts = append(parts, alert.line2)
	}
	return "[" + color + "]" + strings.Join(parts, " ")
}

func iconPattern(kind string) []string {
	switch {
	case strings.HasPrefix(kind, "success"), kind == "celebration":
		return []string{
			"....G",
			"...GG",
			"G.GG.",
			"GGG..",
			".G...",
		}
	case strings.HasPrefix(kind, "failure"):
		return []string{
			"R...R",
			".R.R.",
			"..R..",
			".R.R.",
			"R...R",
		}
	case kind == "approval":
		return []string{
			"..A..",
			".AAA.",
			"AA.AA",
			"AAAAA",
			"..A..",
		}
	case kind == "question":
		return []string{
			".AAA.",
			"A...A",
			"...A.",
			"..A..",
			"..A..",
		}
	case kind == "swear":
		return []string{
			"V...V",
			".V.V.",
			"..V..",
			"V...V",
			".VVV.",
		}
	default:
		return nil
	}
}

func textColorForKind(kind string) string {
	switch {
	case strings.HasPrefix(kind, "success"), kind == "celebration":
		return "green"
	case strings.HasPrefix(kind, "failure"):
		return "red"
	case kind == "approval" || kind == "question":
		return "amber"
	case kind == "swear":
		return "violet"
	default:
		return "blue"
	}
}

var iconColors = map[rune]string{
	'G': "green",
	'R': "red",
	'A': "amber",
	'V': "violet",
}

func envString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envDurationSeconds(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds < 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}

func envDurationMillis(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	millis, err := strconv.Atoi(value)
	if err != nil || millis <= 0 {
		return fallback
	}
	return time.Duration(millis) * time.Millisecond
}
