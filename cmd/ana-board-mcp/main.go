package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/georgestander/ana-board/internal/art"
	"github.com/georgestander/ana-board/internal/board"
	"github.com/georgestander/ana-board/internal/capabilities"
	"github.com/georgestander/ana-board/internal/client"
	"github.com/georgestander/ana-board/internal/layout"
	"github.com/georgestander/ana-board/internal/messages"
)

const protocolVersion = "2025-11-25"
const serverVersion = "0.5.5"
const maxRequestLineBytes = 1024 * 1024
const maxRecentMessages = 50

var supportedProtocolVersions = []string{"2025-11-25", "2025-06-18"}
var errRequestLineTooLarge = errors.New("request line exceeds maximum size")

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type toolResult struct {
	Content           []toolContent `json:"content"`
	StructuredContent any           `json:"structuredContent,omitempty"`
	IsError           bool          `json:"isError,omitempty"`
}

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type sendArgs struct {
	Text       string                `json:"text"`
	Segments   []messages.Segment    `json:"segments"`
	Tiles      []messages.Tile       `json:"tiles"`
	Placements []messages.PlacedTile `json:"placements"`
	Frame      *messages.FrameInput  `json:"frame"`
	Source     string                `json:"source"`
	Priority   string                `json:"priority"`
	Animation  string                `json:"animation"`
	Kind       string                `json:"kind"`
	Color      string                `json:"color"`
}

type spriteArgs struct {
	Sprite    string `json:"sprite"`
	Source    string `json:"source"`
	Priority  string `json:"priority"`
	Animation string `json:"animation"`
	Kind      string `json:"kind"`
	Color     string `json:"color"`
}

type initializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
}

func main() {
	baseURL := flag.String("url", defaultBaseURL(), "Ana Board URL")
	flag.Parse()

	boardClient, err := client.New(*baseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ana-board-mcp: %v\n", err)
		os.Exit(1)
	}

	if err := serve(os.Stdin, os.Stdout, boardClient); err != nil {
		fmt.Fprintf(os.Stderr, "ana-board-mcp: %v\n", err)
		os.Exit(1)
	}
}

func serve(input io.Reader, output io.Writer, boardClient *client.Client) error {
	reader := bufio.NewReaderSize(input, 64*1024)
	encoder := json.NewEncoder(output)

	for {
		line, err := readRequestLine(reader, maxRequestLineBytes)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if errors.Is(err, errRequestLineTooLarge) {
			if encodeErr := encoder.Encode(errorResponse(nil, -32600, "request line exceeds maximum size")); encodeErr != nil {
				return encodeErr
			}
			continue
		}
		if err != nil {
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		response := handleLine(line, boardClient)
		if response == nil {
			continue
		}

		if err := encoder.Encode(response); err != nil {
			return err
		}
	}
}

func readRequestLine(reader *bufio.Reader, maxBytes int) (string, error) {
	var builder strings.Builder

	for {
		chunk, err := reader.ReadString('\n')
		if len(chunk) > 0 {
			if builder.Len()+len(chunk) > maxBytes {
				if !strings.Contains(chunk, "\n") {
					drainRequestLine(reader)
				}
				return "", errRequestLineTooLarge
			}
			builder.WriteString(chunk)
		}

		switch {
		case err == nil:
			return builder.String(), nil
		case errors.Is(err, io.EOF):
			if builder.Len() == 0 {
				return "", io.EOF
			}
			return builder.String(), nil
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		default:
			return "", err
		}
	}
}

func drainRequestLine(reader *bufio.Reader) {
	for {
		chunk, err := reader.ReadString('\n')
		if strings.Contains(chunk, "\n") || err == nil || errors.Is(err, io.EOF) || !errors.Is(err, bufio.ErrBufferFull) {
			return
		}
	}
}

func handleLine(line string, boardClient *client.Client) *rpcResponse {
	var req rpcRequest
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		return errorResponse(nil, -32700, "parse error")
	}
	if len(req.ID) == 0 && strings.HasPrefix(req.Method, "notifications/") {
		return nil
	}
	if req.JSONRPC != "2.0" {
		return errorResponse(req.ID, -32600, "invalid JSON-RPC version")
	}

	switch req.Method {
	case "initialize":
		negotiatedProtocolVersion := negotiateProtocolVersion(req.Params)
		return resultResponse(req.ID, map[string]any{
			"protocolVersion": negotiatedProtocolVersion,
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]string{
				"name":    "ana-board",
				"version": serverVersion,
			},
			"instructions": "Use ana_board_capabilities first. Send concise board-safe status updates. Native iOS/macOS emoji can be sent directly; there is no emoji whitelist. Use sprite tools for named block art, tiles JSON for exact per-letter color, or placements/frame JSON for exact row and column control. Only row animation is supported.",
		})
	case "ping":
		return resultResponse(req.ID, map[string]any{})
	case "notifications/initialized":
		return nil
	case "tools/list":
		return resultResponse(req.ID, map[string]any{"tools": tools()})
	case "tools/call":
		result := callTool(req.Params, boardClient)
		return resultResponse(req.ID, result)
	default:
		return errorResponse(req.ID, -32601, "method not found")
	}
}

func negotiateProtocolVersion(raw json.RawMessage) string {
	var params initializeParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return protocolVersion
	}

	for _, supported := range supportedProtocolVersions {
		if params.ProtocolVersion == supported {
			return supported
		}
	}

	return protocolVersion
}

func callTool(raw json.RawMessage, boardClient *client.Client) toolResult {
	var params toolCallParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return textToolError("invalid tool call params: " + err.Error())
	}

	switch params.Name {
	case "ana_board_capabilities":
		return jsonToolResult(capabilities.Current())
	case "ana_board_preview_message":
		return previewTool(params.Arguments)
	case "ana_board_send_message":
		return sendTool(params.Arguments, boardClient)
	case "ana_board_list_sprites":
		return jsonToolResult(map[string][]string{"sprites": art.ListSprites()})
	case "ana_board_preview_sprite":
		return previewSpriteTool(params.Arguments)
	case "ana_board_send_sprite":
		return sendSpriteTool(params.Arguments, boardClient)
	case "ana_board_current":
		resp, err := boardClient.CurrentFrame(context.Background())
		if err != nil {
			return textToolError(err.Error())
		}
		return jsonToolResult(resp)
	case "ana_board_recent_messages":
		return recentTool(params.Arguments, boardClient)
	case "ana_board_clear":
		return clearTool(params.Arguments, boardClient)
	default:
		return textToolError("unknown Ana Board tool: " + params.Name)
	}
}

func previewTool(raw json.RawMessage) toolResult {
	req, err := parseSendArgs(raw)
	if err != nil {
		return textToolError(err.Error())
	}

	frame, err := requestFrame(req)
	if err != nil {
		return textToolError(err.Error())
	}

	return jsonToolResult(frameToOutput(frame))
}

func sendTool(raw json.RawMessage, boardClient *client.Client) toolResult {
	req, err := parseSendArgs(raw)
	if err != nil {
		return textToolError(err.Error())
	}

	resp, err := boardClient.SendMessage(context.Background(), req)
	if err != nil {
		return textToolError(err.Error())
	}

	return jsonToolResult(resp)
}

func previewSpriteTool(raw json.RawMessage) toolResult {
	req, err := parseSpriteArgs(raw)
	if err != nil {
		return textToolError(err.Error())
	}

	frame, err := requestFrame(req)
	if err != nil {
		return textToolError(err.Error())
	}

	return jsonToolResult(frameToOutput(frame))
}

func sendSpriteTool(raw json.RawMessage, boardClient *client.Client) toolResult {
	req, err := parseSpriteArgs(raw)
	if err != nil {
		return textToolError(err.Error())
	}

	resp, err := boardClient.SendMessage(context.Background(), req)
	if err != nil {
		return textToolError(err.Error())
	}

	return jsonToolResult(resp)
}

func recentTool(raw json.RawMessage, boardClient *client.Client) toolResult {
	var args struct {
		Limit *int `json:"limit"`
	}
	if len(raw) != 0 {
		if err := json.Unmarshal(raw, &args); err != nil {
			return textToolError("invalid recent messages arguments: " + err.Error())
		}
	}
	limit := 10
	if args.Limit != nil {
		if *args.Limit <= 0 {
			return textToolError("limit must be a positive integer")
		}
		if *args.Limit > maxRecentMessages {
			return textToolError(fmt.Sprintf("limit must be less than or equal to %d", maxRecentMessages))
		}
		limit = *args.Limit
	}

	resp, err := boardClient.ListMessages(context.Background(), limit)
	if err != nil {
		return textToolError(err.Error())
	}

	return jsonToolResult(resp)
}

func clearTool(raw json.RawMessage, boardClient *client.Client) toolResult {
	var args struct {
		Confirm bool `json:"confirm"`
	}
	if len(raw) != 0 {
		if err := json.Unmarshal(raw, &args); err != nil {
			return textToolError("invalid clear arguments: " + err.Error())
		}
	}
	if !args.Confirm {
		return textToolError("clear requires confirm=true")
	}

	resp, err := boardClient.Clear(context.Background())
	if err != nil {
		return textToolError(err.Error())
	}

	return jsonToolResult(resp)
}

func parseSpriteArgs(raw json.RawMessage) (messages.SubmitRequest, error) {
	var args spriteArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return messages.SubmitRequest{}, fmt.Errorf("invalid sprite arguments: %w", err)
	}

	frame, err := art.SpriteFrame(args.Sprite)
	if err != nil {
		return messages.SubmitRequest{}, err
	}

	req := messages.SubmitRequest{
		Frame:     &frame,
		Source:    args.Source,
		Priority:  args.Priority,
		Animation: args.Animation,
		Kind:      args.Kind,
		Color:     args.Color,
	}

	return normalizeRequestMetadata(req)
}

func parseSendArgs(raw json.RawMessage) (messages.SubmitRequest, error) {
	var args sendArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return messages.SubmitRequest{}, fmt.Errorf("invalid message arguments: %w", err)
	}

	req := messages.SubmitRequest{
		Text:       strings.TrimSpace(args.Text),
		Segments:   args.Segments,
		Tiles:      args.Tiles,
		Placements: args.Placements,
		Frame:      args.Frame,
		Source:     args.Source,
		Priority:   args.Priority,
		Animation:  args.Animation,
		Kind:       args.Kind,
		Color:      args.Color,
	}

	payloadCount := 0
	for _, hasPayload := range []bool{req.Text != "", len(req.Segments) != 0, len(req.Tiles) != 0, len(req.Placements) != 0, req.Frame != nil} {
		if hasPayload {
			payloadCount++
		}
	}
	if payloadCount == 0 {
		return messages.SubmitRequest{}, fmt.Errorf("text is required")
	}
	if payloadCount > 1 {
		return messages.SubmitRequest{}, fmt.Errorf("use either text, segments, tiles, placements, or frame")
	}

	return normalizeRequestMetadata(req)
}

func normalizeRequestMetadata(req messages.SubmitRequest) (messages.SubmitRequest, error) {
	var err error
	req.Source, err = messages.NormalizeSource(req.Source)
	if err != nil {
		return messages.SubmitRequest{}, err
	}
	req.Priority, err = messages.NormalizePriority(req.Priority)
	if err != nil {
		return messages.SubmitRequest{}, err
	}
	req.Animation, err = messages.NormalizeAnimation(req.Animation)
	if err != nil {
		return messages.SubmitRequest{}, err
	}
	req.Kind, err = messages.NormalizeKind(req.Kind)
	if err != nil {
		return messages.SubmitRequest{}, err
	}
	req.Color, err = messages.NormalizeColor(req.Color)
	if err != nil {
		return messages.SubmitRequest{}, err
	}

	return req, nil
}

func jsonToolResult(value any) toolResult {
	text, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return textToolError(err.Error())
	}

	return toolResult{
		Content: []toolContent{
			{Type: "text", Text: string(text)},
		},
		StructuredContent: value,
	}
}

func textToolError(text string) toolResult {
	return toolResult{
		Content: []toolContent{
			{Type: "text", Text: text},
		},
		IsError: true,
	}
}

func tools() []map[string]any {
	messageSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"text":       map[string]string{"type": "string", "description": "Message text. Native iOS/macOS emoji can be used directly and each visible emoji grapheme counts as one tile. Inline color tokens like [green]A[red]B can color individual letters."},
			"segments":   segmentSchema(),
			"tiles":      tileSchema(),
			"placements": placementSchema(),
			"frame":      frameSchema(),
			"source":     map[string]string{"type": "string", "description": "Short sender name such as codex, hermes, claude, opencode."},
			"kind":       map[string]any{"type": "string", "enum": messages.AllowedKinds()},
			"color":      map[string]any{"type": "string", "enum": messages.AllowedColors(), "description": "Default color for tiles without an inline token, segment color, or tile color."},
			"animation":  map[string]any{"type": "string", "enum": messages.AllowedAnimations(), "description": "Only row is supported."},
			"priority":   map[string]any{"type": "string", "enum": messages.AllowedPriorities()},
		},
	}
	spriteMessageSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"sprite":    map[string]any{"type": "string", "enum": art.ListSprites(), "description": "Named block-art sprite to render as colored █ pixels."},
			"source":    map[string]string{"type": "string", "description": "Short sender name such as codex, hermes, claude, opencode."},
			"kind":      map[string]any{"type": "string", "enum": messages.AllowedKinds()},
			"color":     map[string]any{"type": "string", "enum": messages.AllowedColors(), "description": "Message default color metadata. Sprite pixels keep their own colors."},
			"animation": map[string]any{"type": "string", "enum": messages.AllowedAnimations(), "description": "Only row is supported."},
			"priority":  map[string]any{"type": "string", "enum": messages.AllowedPriorities()},
		},
		"required": []string{"sprite"},
	}

	return []map[string]any{
		{
			"name":        "ana_board_capabilities",
			"title":       "Ana Board Capabilities",
			"description": "Read Ana Board limits, allowed colors, kinds, animations, native emoji support, and color syntax.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
			"annotations": map[string]any{"readOnlyHint": true, "openWorldHint": false},
		},
		{
			"name":        "ana_board_preview_message",
			"title":       "Preview Ana Board Message",
			"description": "Validate and preview how a message or exact frame will fit before sending it.",
			"inputSchema": messageSchema,
			"annotations": map[string]any{"readOnlyHint": true, "openWorldHint": false},
		},
		{
			"name":        "ana_board_send_message",
			"title":       "Send Ana Board Message",
			"description": "Display a concise status message or exact placed frame on Ana Board.",
			"inputSchema": messageSchema,
			"annotations": map[string]any{"readOnlyHint": false, "idempotentHint": false, "openWorldHint": false},
		},
		{
			"name":        "ana_board_list_sprites",
			"title":       "List Ana Board Sprites",
			"description": "List named block-art sprites available for Ana Board.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
			"annotations": map[string]any{"readOnlyHint": true, "openWorldHint": false},
		},
		{
			"name":        "ana_board_preview_sprite",
			"title":       "Preview Ana Board Sprite",
			"description": "Preview a named block-art sprite as a 10x22 colored frame.",
			"inputSchema": spriteMessageSchema,
			"annotations": map[string]any{"readOnlyHint": true, "openWorldHint": false},
		},
		{
			"name":        "ana_board_send_sprite",
			"title":       "Send Ana Board Sprite",
			"description": "Display a named block-art sprite on Ana Board.",
			"inputSchema": spriteMessageSchema,
			"annotations": map[string]any{"readOnlyHint": false, "idempotentHint": false, "openWorldHint": false},
		},
		{
			"name":        "ana_board_current",
			"title":       "Current Ana Board Frame",
			"description": "Read the current displayed board frame.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
			"annotations": map[string]any{"readOnlyHint": true, "openWorldHint": false},
		},
		{
			"name":        "ana_board_recent_messages",
			"title":       "Recent Ana Board Messages",
			"description": "Read recent Ana Board messages.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"limit": map[string]any{"type": "integer", "minimum": 1, "maximum": maxRecentMessages},
				},
			},
			"annotations": map[string]any{"readOnlyHint": true, "openWorldHint": false},
		},
		{
			"name":        "ana_board_clear",
			"title":       "Clear Ana Board",
			"description": "Clear the displayed board. Requires confirm=true.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"confirm": map[string]any{"type": "boolean"},
				},
				"required": []string{"confirm"},
			},
			"annotations": map[string]any{"readOnlyHint": false, "destructiveHint": true, "idempotentHint": true, "openWorldHint": false},
		},
	}
}

func segmentSchema() map[string]any {
	return map[string]any{
		"type":        "array",
		"description": "Optional colored text segments. Use tiles instead when individual letters need different colors.",
		"items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"text":  map[string]string{"type": "string"},
				"color": map[string]any{"type": "string", "enum": messages.AllowedColors(), "description": "Color for tiles produced by this segment unless the segment text contains inline color tokens."},
			},
			"required": []string{"text"},
		},
	}
}

func tileSchema() map[string]any {
	return map[string]any{
		"type":        "array",
		"description": "Exact tile list. Use this when each individual letter or emoji tile may have its own color.",
		"items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"symbol": map[string]string{"type": "string", "description": "One visible tile: one letter, one digit, one punctuation mark, one space, or one native emoji grapheme."},
				"color":  map[string]any{"type": "string", "enum": messages.AllowedColors()},
			},
			"required": []string{"symbol"},
		},
	}
}

func placementSchema() map[string]any {
	return map[string]any{
		"type":        "array",
		"description": "Exact sparse tile placements. Use this when the agent needs row and column control. Rows are 0-9 and columns are 0-21.",
		"items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"row":    map[string]any{"type": "integer", "minimum": 0, "maximum": board.DefaultRows - 1},
				"col":    map[string]any{"type": "integer", "minimum": 0, "maximum": board.DefaultCols - 1},
				"symbol": map[string]string{"type": "string", "description": "One visible tile: one letter, one digit, one punctuation mark, one space, or one native emoji grapheme."},
				"color":  map[string]any{"type": "string", "enum": messages.AllowedColors()},
			},
			"required": []string{"row", "col", "symbol"},
		},
	}
}

func frameSchema() map[string]any {
	return map[string]any{
		"type":        "object",
		"description": "Full exact frame. cells must be a 10 row x 22 column array. colors is optional but must be the same shape when provided.",
		"properties": map[string]any{
			"cells":  map[string]any{"type": "array", "description": "10 rows x 22 columns of symbols. Empty strings are treated as blank tiles."},
			"colors": map[string]any{"type": "array", "description": "Optional 10 rows x 22 columns of tile colors."},
		},
		"required": []string{"cells"},
	}
}

func requestFrame(req messages.SubmitRequest) (board.Frame, error) {
	if len(req.Placements) != 0 {
		frame, _, _, err := layout.ExactFrameFromPlacements(req.Placements, req.Color)
		return frame, err
	}

	if req.Frame != nil {
		frame, _, _, err := layout.ExactFrameFromInput(*req.Frame, req.Color)
		return frame, err
	}

	cells, err := requestCells(req)
	if err != nil {
		return board.Frame{}, err
	}

	return layout.CenterCells(cells)
}

func requestCells(req messages.SubmitRequest) ([]board.Cell, error) {
	if len(req.Tiles) != 0 {
		cells := make([]board.Cell, 0, len(req.Tiles))
		for _, tile := range req.Tiles {
			color, err := messages.NormalizeColor(tile.Color)
			if err != nil {
				return nil, err
			}
			if tile.Color == "" {
				color = req.Color
			}

			cell, err := layout.NormalizeTileCell(tile.Symbol, color)
			if err != nil {
				return nil, err
			}
			cells = append(cells, cell)
		}

		return cells, nil
	}

	if len(req.Segments) == 0 {
		return board.NormalizeCells(req.Text, req.Color)
	}

	segments := make([]board.TextSegment, len(req.Segments))
	for index, segment := range req.Segments {
		segments[index] = board.TextSegment{Text: segment.Text, Color: segment.Color}
	}

	return board.NormalizeSegmentCells(segments, req.Color)
}

func frameToOutput(frame board.Frame) client.FrameResponse {
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

	return client.FrameResponse{
		Rows:   frame.Rows,
		Cols:   frame.Cols,
		Cells:  cells,
		Colors: colors,
	}
}

func resultResponse(id json.RawMessage, result any) *rpcResponse {
	return &rpcResponse{JSONRPC: "2.0", ID: id, Result: result}
}

func errorResponse(id json.RawMessage, code int, message string) *rpcResponse {
	if len(id) == 0 {
		id = []byte("null")
	}

	return &rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: message},
	}
}

func defaultBaseURL() string {
	value := os.Getenv("ANA_BOARD_URL")
	if strings.TrimSpace(value) == "" {
		return client.DefaultBaseURL
	}

	return value
}
