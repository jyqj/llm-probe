package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"bedrock-gateway/internal/config"
)

// UpstreamProxy forwards requests to the upstream API provider.
type UpstreamProxy struct {
	cfg          config.UpstreamConfig
	syncClient   *http.Client
	streamClient *http.Client
}

// NewUpstreamProxy creates a new proxy to the upstream API.
func NewUpstreamProxy(cfg config.UpstreamConfig) *UpstreamProxy {
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	streamTimeout := timeout * 2
	if streamTimeout < 10*time.Minute {
		streamTimeout = 10 * time.Minute
	}
	return &UpstreamProxy{
		cfg: cfg,
		syncClient: &http.Client{
			Timeout: timeout,
		},
		streamClient: &http.Client{
			Timeout: streamTimeout,
			Transport: &http.Transport{
				ResponseHeaderTimeout: 30 * time.Second,
			},
		},
	}
}

// UpstreamTarget allows per-request override of upstream base/key.
type UpstreamTarget struct {
	BaseURL string
	APIKey  string
}

// buildMessagesRequest creates an HTTP request for /v1/messages with all
// required headers. accept should be "text/event-stream" or "application/json".
func buildMessagesRequest(ctx context.Context, url string, body []byte, apiKey, anthVersion, accept string, extraHeaders http.Header) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", anthVersion)
	if extraHeaders != nil {
		if beta := extraHeaders.Get("anthropic-beta"); beta != "" {
			req.Header.Set("anthropic-beta", beta)
		}
	}
	req.Header.Set("Accept", accept)
	return req, nil
}

// SendMessages forwards a request to upstream POST /v1/messages.
// If target is non-nil, it overrides the default upstream.
func (p *UpstreamProxy) SendMessages(ctx context.Context, body []byte, stream bool, target *UpstreamTarget, extraHeaders http.Header) (*http.Response, error) {
	baseURL := p.cfg.BaseURL
	apiKey := p.cfg.APIKey
	if target != nil {
		if target.BaseURL != "" {
			baseURL = strings.TrimRight(target.BaseURL, "/")
		}
		if target.APIKey != "" {
			apiKey = target.APIKey
		}
	}

	url := baseURL + "/v1/messages"

	// Forward client headers, fallback to defaults
	anthVersion := "2023-06-01"
	if extraHeaders != nil {
		if v := extraHeaders.Get("anthropic-version"); v != "" {
			anthVersion = v
		}
	}

	if stream {
		req, err := buildMessagesRequest(ctx, url, body, apiKey, anthVersion, "text/event-stream", extraHeaders)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		resp, err := p.streamClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("stream request: %w", err)
		}
		return resp, nil
	}

	// Simple retry for non-streaming: retry on 502/503/429, max 2 attempts
	var resp *http.Response
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
		}
		req, reqErr := buildMessagesRequest(ctx, url, body, apiKey, anthVersion, "application/json", extraHeaders)
		if reqErr != nil {
			return nil, fmt.Errorf("create request: %w", reqErr)
		}
		resp, err = p.syncClient.Do(req)
		if err != nil {
			if attempt < 2 {
				continue
			}
			return nil, fmt.Errorf("request: %w", err)
		}
		// Retry on transient errors
		if resp.StatusCode == 502 || resp.StatusCode == 503 || resp.StatusCode == 429 {
			if attempt < 2 {
				resp.Body.Close()
				continue
			}
		}
		break
	}
	return resp, nil
}

// MaxResponseSize is the maximum response body size (100MB).
const MaxResponseSize = 100 * 1024 * 1024

// ReadResponse reads and returns the full response body (capped at MaxResponseSize).
func ReadResponse(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, MaxResponseSize))
}

// PassthroughHandler forwards non-messages requests to upstream.
type PassthroughHandler struct {
	cfg    config.UpstreamConfig
	client *http.Client
	logger *slog.Logger
}

// NewPassthroughHandler creates a handler that proxies arbitrary paths to upstream.
func NewPassthroughHandler(cfg config.UpstreamConfig, logger *slog.Logger) *PassthroughHandler {
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	return &PassthroughHandler{
		cfg:    cfg,
		client: &http.Client{Timeout: timeout},
		logger: logger,
	}
}

func (p *PassthroughHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	baseURL := strings.TrimRight(p.cfg.BaseURL, "/")
	targetURL := baseURL + r.URL.Path
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	p.logger.Debug("passthrough", "method", r.Method, "path", r.URL.Path)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body error", http.StatusBadGateway)
		return
	}
	defer r.Body.Close()

	req, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, bytes.NewReader(body))
	if err != nil {
		http.Error(w, "create request error", http.StatusBadGateway)
		return
	}

	// Forward headers, skip hop-by-hop
	for k, vs := range r.Header {
		kl := strings.ToLower(k)
		if kl == "host" || kl == "content-length" || kl == "transfer-encoding" {
			continue
		}
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	// Override API key
	if p.cfg.APIKey != "" {
		req.Header.Set("x-api-key", p.cfg.APIKey)
		req.Header.Del("Authorization")
	}

	resp, err := p.client.Do(req)
	if err != nil {
		p.logger.Error("passthrough upstream error", "error", err)
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/json"
	}
	// Strip charset if present in JSON content type
	if strings.Contains(ct, "application/json") && strings.Contains(ct, "charset") {
		ct = "application/json"
	}
	w.Header().Set("Content-Type", ct)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, io.LimitReader(resp.Body, MaxResponseSize))
}
