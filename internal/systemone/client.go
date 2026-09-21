package systemone

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/freepik-company/jev-mcp/internal/config"
)

const (
	maxRequestBytes  = 1 << 20
	maxResponseBytes = 2 << 20
)

type Client struct {
	endpoint string
	baseURL  string
	provider string
	apiKey   string
	model    string
	http     *http.Client
}

func NewClient(cfg config.Config, transport http.RoundTripper) *Client {
	return &Client{
		endpoint: strings.TrimRight(cfg.BaseURL, "/") + "/v1/systemone",
		baseURL:  strings.TrimRight(cfg.BaseURL, "/"),
		provider: cfg.Provider,
		apiKey:   cfg.APIKey,
		model:    cfg.Model,
		http: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
			// Credentials are never forwarded and paid inferences are never repeated.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

func (c *Client) Decide(ctx context.Context, in Request) (json.RawMessage, error) {
	if in.Model == "" {
		in.Model = c.model
	}
	body, err := json.Marshal(in)
	if err != nil || len(body) > maxRequestBytes {
		return nil, errors.New("Decision input must be valid JSON no larger than 1 MiB")
	}
	body, err = c.request(ctx, http.MethodPost, c.endpoint, body)
	if err != nil {
		return nil, err
	}
	var out response
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, errors.New("System One returned invalid JSON")
	}
	if out.Model == nil || out.Usage == nil || out.Usage.InputTokens == nil || out.Usage.OutputTokens == nil || *out.Usage.InputTokens < 0 || *out.Usage.OutputTokens < 0 || (out.Usage.Cost != nil && *out.Usage.Cost < 0) {
		return nil, errors.New("System One returned incomplete model or usage metadata")
	}
	if err := validateAnswers(in.Questions, out.Answers); err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}

func (c *Client) request(ctx context.Context, method, endpoint string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, errors.New("Cannot create the System One request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	res, err := c.http.Do(req)
	if err != nil {
		// Transport errors may include the URL or headers: they never reach the model.
		if ctx.Err() != nil {
			return nil, errors.New("System One request cancelled")
		}
		return nil, errors.New("System One request failed or timed out")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		// A provider may echo the key in its error body. Only the status code is surfaced.
		return nil, fmt.Errorf("System One returned HTTP %d; check the MCP credential, quota and endpoint", res.StatusCode)
	}
	body, err = io.ReadAll(io.LimitReader(res.Body, maxResponseBytes+1))
	if err != nil || len(body) > maxResponseBytes {
		return nil, errors.New("System One response is unreadable or exceeds 2 MiB")
	}

	return body, nil
}
