package server

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/georgestander/ana-board/internal/board"
	"github.com/georgestander/ana-board/internal/layout"
	"github.com/georgestander/ana-board/internal/messages"
	"github.com/georgestander/ana-board/web"
)

type createMessageRequest = messages.SubmitRequest

type createMessageResponse struct {
	ID        string           `json:"id"`
	Status    string           `json:"status"`
	Message   messages.Message `json:"message"`
	Frame     frameResponse    `json:"frame"`
	Animation string           `json:"animation"`
}

type currentFrameResponse struct {
	BoardID string        `json:"board_id"`
	Frame   frameResponse `json:"frame"`
}

type listMessagesResponse struct {
	Messages []messages.Message `json:"messages"`
}

type adminPageData struct {
	Error    string
	Notice   string
	Text     string
	Source   string
	Kind     string
	Color    string
	Messages []adminMessageView
}

type adminMessageView struct {
	Text      string
	Source    string
	Kind      string
	Color     string
	Status    string
	CreatedAt string
}

type frameResponse struct {
	Rows   int        `json:"rows"`
	Cols   int        `json:"cols"`
	Cells  [][]string `json:"cells"`
	Colors [][]string `json:"colors"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	html, err := web.IndexHTML()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Errorf("could not load board page"))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(html)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleAdmin(w http.ResponseWriter, r *http.Request) {
	data, err := s.adminPageData(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err)
		return
	}

	switch r.URL.Query().Get("status") {
	case "sent":
		data.Notice = "Message sent"
	case "cleared":
		data.Notice = "Board cleared"
	}

	s.renderAdmin(w, http.StatusOK, data)
}

func (s *Server) handleAdminCreateMessage(w http.ResponseWriter, r *http.Request) {
	if !allowSameOriginWrite(w, r) {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
	if err := r.ParseForm(); err != nil {
		s.renderAdminError(w, r, http.StatusBadRequest, "Could not read form", adminPageData{})
		return
	}

	data := adminPageData{
		Text:   r.FormValue("text"),
		Source: r.FormValue("source"),
		Kind:   r.FormValue("kind"),
		Color:  r.FormValue("color"),
	}
	if strings.TrimSpace(data.Source) == "" {
		data.Source = "admin"
	}

	_, err := s.createMessage(r.Context(), createMessageRequest{
		Text:   data.Text,
		Source: data.Source,
		Kind:   data.Kind,
		Color:  data.Color,
	})
	if err != nil {
		s.renderAdminError(w, r, http.StatusBadRequest, err.Error(), data)
		return
	}

	http.Redirect(w, r, "/admin?status=sent", http.StatusSeeOther)
}

func (s *Server) handleAdminClear(w http.ResponseWriter, r *http.Request) {
	if !allowSameOriginWrite(w, r) {
		return
	}

	if _, err := s.clearCurrentFrame(r.Context(), messages.DefaultAnimation); err != nil {
		s.renderAdminError(w, r, http.StatusInternalServerError, err.Error(), adminPageData{})
		return
	}

	http.Redirect(w, r, "/admin?status=cleared", http.StatusSeeOther)
}

func (s *Server) handleAPIClear(w http.ResponseWriter, r *http.Request) {
	if !allowSameOriginWrite(w, r) {
		return
	}

	frame, err := s.clearCurrentFrame(r.Context(), messages.DefaultAnimation)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, currentFrameResponse{
		BoardID: defaultBoardID,
		Frame:   frameToResponse(frame),
	})
}

func (s *Server) handleCurrentFrame(w http.ResponseWriter, r *http.Request) {
	frame, err := s.store.CurrentFrame(r.Context(), defaultBoardID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, currentFrameResponse{
		BoardID: defaultBoardID,
		Frame:   frameToResponse(frame),
	})
}

func (s *Server) handleCreateMessage(w http.ResponseWriter, r *http.Request) {
	if !allowSameOriginWrite(w, r) {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
	defer r.Body.Close()

	var req createMessageRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Errorf("invalid JSON request: %w", err))
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeJSONError(w, http.StatusBadRequest, fmt.Errorf("request body must contain one JSON object"))
		return
	}

	response, err := s.createMessage(r.Context(), req)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusCreated, response)
}

func (s *Server) createMessage(ctx context.Context, req createMessageRequest) (createMessageResponse, error) {
	req.Text = strings.TrimSpace(req.Text)
	var err error
	req.Source, err = messages.NormalizeSource(req.Source)
	if err != nil {
		return createMessageResponse{}, err
	}
	req.Priority, err = messages.NormalizePriority(req.Priority)
	if err != nil {
		return createMessageResponse{}, err
	}
	req.Animation, err = messages.NormalizeAnimation(req.Animation)
	if err != nil {
		return createMessageResponse{}, err
	}
	req.Kind, err = messages.NormalizeKind(req.Kind)
	if err != nil {
		return createMessageResponse{}, err
	}
	req.Color, err = messages.NormalizeColor(req.Color)
	if err != nil {
		return createMessageResponse{}, err
	}

	cells, storedSegments, storedTiles, displayText, err := normalizeMessagePayload(req)
	if err != nil {
		return createMessageResponse{}, err
	}
	req.Text = displayText

	frame, err := layout.CenterCells(cells)
	if err != nil {
		return createMessageResponse{}, err
	}

	now := time.Now().UTC()
	msg := messages.Message{
		ID:        fmt.Sprintf("msg_%d", now.UnixNano()),
		Text:      req.Text,
		Segments:  storedSegments,
		Tiles:     storedTiles,
		Source:    req.Source,
		Priority:  req.Priority,
		Animation: req.Animation,
		Kind:      req.Kind,
		Color:     req.Color,
		CreatedAt: now,
		Status:    "displayed",
	}

	if err := s.store.SaveMessage(ctx, msg); err != nil {
		return createMessageResponse{}, err
	}

	if err := s.store.SaveCurrentFrame(ctx, defaultBoardID, frame); err != nil {
		return createMessageResponse{}, err
	}

	frameResponse := frameToResponse(frame)
	s.broker.Broadcast(FrameEvent{
		Frame:     frameResponse,
		Animation: req.Animation,
	})

	return createMessageResponse{
		ID:        msg.ID,
		Status:    msg.Status,
		Message:   msg,
		Frame:     frameResponse,
		Animation: req.Animation,
	}, nil
}

func normalizeMessagePayload(req createMessageRequest) ([]board.Cell, []messages.Segment, []messages.Tile, string, error) {
	if len(req.Tiles) != 0 {
		if req.Text != "" || len(req.Segments) != 0 {
			return nil, nil, nil, "", fmt.Errorf("send either text, segments, or tiles, not more than one")
		}

		return normalizeMessageTiles(req.Tiles, req.Color)
	}

	if len(req.Segments) == 0 {
		if req.Text == "" {
			return nil, nil, nil, "", fmt.Errorf("text is required")
		}

		cells, err := board.NormalizeCells(req.Text, req.Color)
		if err != nil {
			return nil, nil, nil, "", err
		}

		return cells, nil, nil, req.Text, nil
	}

	segments := make([]board.TextSegment, 0, len(req.Segments))
	stored := make([]messages.Segment, 0, len(req.Segments))
	var text strings.Builder

	for _, segment := range req.Segments {
		if strings.TrimSpace(segment.Text) == "" {
			continue
		}

		color, err := messages.NormalizeColor(segment.Color)
		if err != nil {
			return nil, nil, nil, "", err
		}
		if segment.Color == "" {
			color = req.Color
		}

		segments = append(segments, board.TextSegment{Text: segment.Text, Color: color})
		stored = append(stored, messages.Segment{Text: segment.Text, Color: color})
		text.WriteString(segment.Text)
	}

	if len(segments) == 0 {
		return nil, nil, nil, "", fmt.Errorf("text is required")
	}

	cells, err := board.NormalizeSegmentCells(segments, req.Color)
	if err != nil {
		return nil, nil, nil, "", err
	}

	return cells, stored, nil, text.String(), nil
}

func normalizeMessageTiles(tiles []messages.Tile, defaultColor string) ([]board.Cell, []messages.Segment, []messages.Tile, string, error) {
	cells := make([]board.Cell, 0, len(tiles))
	stored := make([]messages.Tile, 0, len(tiles))
	var text strings.Builder

	for _, tile := range tiles {
		color, err := messages.NormalizeColor(tile.Color)
		if err != nil {
			return nil, nil, nil, "", err
		}
		if tile.Color == "" {
			color = defaultColor
		}

		cell, err := normalizeTileCell(tile.Symbol, color)
		if err != nil {
			return nil, nil, nil, "", err
		}

		cells = append(cells, cell)
		stored = append(stored, messages.Tile{Symbol: cell.Symbol, Color: cell.Color})
		text.WriteString(cell.Symbol)
	}

	if len(cells) == 0 {
		return nil, nil, nil, "", fmt.Errorf("text is required")
	}

	return cells, nil, stored, text.String(), nil
}

func normalizeTileCell(symbol, color string) (board.Cell, error) {
	if symbol == " " {
		return board.NewCell(" ", color), nil
	}

	cells, err := board.NormalizeCells(symbol, color)
	if err != nil {
		return board.Cell{}, err
	}
	if len(cells) != 1 {
		return board.Cell{}, fmt.Errorf("tile symbol %q must normalize to exactly one tile", symbol)
	}

	return cells[0], nil
}

func (s *Server) clearCurrentFrame(ctx context.Context, animation string) (board.Frame, error) {
	frame, err := board.NewFrame(board.DefaultRows, board.DefaultCols)
	if err != nil {
		return board.Frame{}, err
	}

	if err := s.store.SaveCurrentFrame(ctx, defaultBoardID, frame); err != nil {
		return board.Frame{}, err
	}

	s.broker.Broadcast(FrameEvent{
		Frame:     frameToResponse(frame),
		Animation: animation,
	})

	return frame, nil
}

func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeJSONError(w, http.StatusBadRequest, fmt.Errorf("limit must be a positive integer"))
			return
		}

		limit = parsed
	}

	messages, err := s.store.ListMessages(r.Context(), limit)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, listMessagesResponse{Messages: messages})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, fmt.Errorf("streaming is not supported"))
		return
	}

	ch := s.broker.Subscribe()
	defer s.broker.Unsubscribe(ch)

	frame, err := s.store.CurrentFrame(r.Context(), defaultBoardID)
	if err == nil {
		_ = writeSSE(w, "frame", FrameEvent{Frame: frameToResponse(frame), Animation: messages.DefaultAnimation})
		flusher.Flush()
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}

			if err := writeSSE(w, "frame", event); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (s *Server) renderAdminError(w http.ResponseWriter, r *http.Request, status int, message string, data adminPageData) {
	freshData, err := s.adminPageData(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err)
		return
	}

	freshData.Error = message
	freshData.Text = data.Text
	freshData.Source = data.Source
	freshData.Kind = data.Kind
	freshData.Color = data.Color
	s.renderAdmin(w, status, freshData)
}

func (s *Server) renderAdmin(w http.ResponseWriter, status int, data adminPageData) {
	if data.Source == "" {
		data.Source = "admin"
	}
	if data.Kind == "" {
		data.Kind = messages.DefaultKind
	}
	if data.Color == "" {
		data.Color = messages.DefaultColor
	}

	html, err := web.AdminHTML()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Errorf("could not load admin page"))
		return
	}

	tmpl, err := template.New("admin").Parse(string(html))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Errorf("could not parse admin page"))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = tmpl.Execute(w, data)
}

func (s *Server) adminPageData(ctx context.Context) (adminPageData, error) {
	recent, err := s.store.ListMessages(ctx, 20)
	if err != nil {
		return adminPageData{}, err
	}

	views := make([]adminMessageView, len(recent))
	for index, msg := range recent {
		views[index] = adminMessageView{
			Text:      msg.Text,
			Source:    msg.Source,
			Kind:      msg.Kind,
			Color:     msg.Color,
			Status:    msg.Status,
			CreatedAt: msg.CreatedAt.Local().Format("2006-01-02 15:04:05"),
		}
	}

	return adminPageData{
		Source:   "admin",
		Kind:     messages.DefaultKind,
		Color:    messages.DefaultColor,
		Messages: views,
	}, nil
}

func frameToResponse(frame board.Frame) frameResponse {
	cells := make([][]string, frame.Rows)
	colors := make([][]string, frame.Rows)
	for row := range cells {
		cells[row] = make([]string, frame.Cols)
		colors[row] = make([]string, frame.Cols)
		for col := range cells[row] {
			cell, err := frame.CellAt(row, col)
			if err != nil {
				cells[row][col] = " "
				colors[row][col] = board.DefaultColor
				continue
			}

			cells[row][col] = cell.Symbol
			colors[row][col] = cell.Color
		}
	}

	return frameResponse{
		Rows:   frame.Rows,
		Cols:   frame.Cols,
		Cells:  cells,
		Colors: colors,
	}
}

func allowSameOriginWrite(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	if origin == "http://"+r.Host || origin == "https://"+r.Host {
		return true
	}

	writeJSONError(w, http.StatusForbidden, fmt.Errorf("origin %q is not allowed", origin))
	return false
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeJSONError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, errorResponse{Error: err.Error()})
}
