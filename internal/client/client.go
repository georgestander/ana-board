package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/georgestander/ana-board/internal/messages"
)

const DefaultBaseURL = "http://localhost:8080"

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type FrameResponse struct {
	Rows   int        `json:"rows"`
	Cols   int        `json:"cols"`
	Cells  [][]string `json:"cells"`
	Colors [][]string `json:"colors"`
}

type CurrentFrameResponse struct {
	BoardID string        `json:"board_id"`
	Frame   FrameResponse `json:"frame"`
}

type SendMessageResponse struct {
	ID        string           `json:"id"`
	Status    string           `json:"status"`
	Message   messages.Message `json:"message"`
	Frame     FrameResponse    `json:"frame"`
	Animation string           `json:"animation"`
}

type ListMessagesResponse struct {
	Messages []messages.Message `json:"messages"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func New(baseURL string) (*Client, error) {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultBaseURL
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse board URL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("board URL must include scheme and host")
	}

	return &Client{
		baseURL: strings.TrimRight(parsed.String(), "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

func (c *Client) SendMessage(ctx context.Context, req messages.SubmitRequest) (SendMessageResponse, error) {
	var out SendMessageResponse
	if err := c.postJSON(ctx, "/api/messages", req, http.StatusCreated, &out); err != nil {
		return SendMessageResponse{}, err
	}

	return out, nil
}

func (c *Client) CurrentFrame(ctx context.Context) (CurrentFrameResponse, error) {
	var out CurrentFrameResponse
	if err := c.getJSON(ctx, "/api/current", &out); err != nil {
		return CurrentFrameResponse{}, err
	}

	return out, nil
}

func (c *Client) ListMessages(ctx context.Context, limit int) (ListMessagesResponse, error) {
	path := "/api/messages"
	if limit > 0 {
		path = fmt.Sprintf("%s?limit=%d", path, limit)
	}

	var out ListMessagesResponse
	if err := c.getJSON(ctx, path, &out); err != nil {
		return ListMessagesResponse{}, err
	}

	return out, nil
}

func (c *Client) Clear(ctx context.Context) (CurrentFrameResponse, error) {
	var out CurrentFrameResponse
	if err := c.postJSON(ctx, "/api/clear", map[string]string{}, http.StatusOK, &out); err != nil {
		return CurrentFrameResponse{}, err
	}

	return out, nil
}

func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return decodeJSONResponse(resp, http.StatusOK, out)
}

func (c *Client) postJSON(ctx context.Context, path string, value any, wantStatus int, out any) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return decodeJSONResponse(resp, wantStatus, out)
}

func decodeJSONResponse(resp *http.Response, wantStatus int, out any) error {
	if resp.StatusCode != wantStatus {
		var apiErr errorResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && apiErr.Error != "" {
			return fmt.Errorf("board request failed: %s", apiErr.Error)
		}

		return fmt.Errorf("board request failed: status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}
