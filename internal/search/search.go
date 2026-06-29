package search

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/d-init-d/d-research-cli/internal/config"
)

const TestQuery = "d-research connectivity probe 2026"

type Result struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet,omitempty"`
}

type TestReport struct {
	ProviderID string `json:"provider_id"`
	Available  bool   `json:"available"`
	LatencyMS  int64  `json:"latency_ms"`
	StatusCode int    `json:"status_code,omitempty"`
	Error      string `json:"error,omitempty"`
	QuotaHint  string `json:"quota_hint,omitempty"`
}

type Provider interface {
	ID() string
	Search(ctx context.Context, query string) ([]Result, error)
}

type HTTPClient struct {
	client *http.Client
}

func NewHTTPClient(timeout time.Duration) *HTTPClient {
	return &HTTPClient{client: &http.Client{Timeout: timeout}}
}

type DuckDuckGo struct {
	http *HTTPClient
	id   string
}

func NewDuckDuckGo(id string, http *HTTPClient) *DuckDuckGo {
	if id == "" {
		id = "ddg"
	}
	return &DuckDuckGo{id: id, http: http}
}

func (d *DuckDuckGo) ID() string { return d.id }

func (d *DuckDuckGo) Search(ctx context.Context, query string) ([]Result, error) {
	u := "https://api.duckduckgo.com/?q=" + url.QueryEscape(query) + "&format=json&no_redirect=1"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := d.http.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limited")
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Heading       string `json:"Heading"`
		AbstractText  string `json:"AbstractText"`
		AbstractURL   string `json:"AbstractURL"`
		RelatedTopics []struct {
			Text     string `json:"Text"`
			FirstURL string `json:"FirstURL"`
		} `json:"RelatedTopics"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	var out []Result
	if payload.AbstractURL != "" {
		out = append(out, Result{Title: payload.Heading, URL: payload.AbstractURL, Snippet: payload.AbstractText})
	}
	for _, t := range payload.RelatedTopics {
		if t.FirstURL != "" {
			out = append(out, Result{Title: t.Text, URL: t.FirstURL})
		}
	}
	return out, nil
}

type FakeProvider struct {
	IDValue    string
	StatusCode int
	Delay      time.Duration
	Err        error
	Results    []Result
}

func (f *FakeProvider) ID() string { return f.IDValue }

func (f *FakeProvider) Search(ctx context.Context, query string) ([]Result, error) {
	if f.Delay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(f.Delay):
		}
	}
	if f.Err != nil {
		return nil, f.Err
	}
	if f.StatusCode == 401 {
		return nil, fmt.Errorf("unauthorized")
	}
	if f.StatusCode == 429 {
		return nil, fmt.Errorf("rate limited")
	}
	return f.Results, nil
}

type Manager struct {
	providers []Provider
	strategy  string
}

func NewManager(cfg config.SearchConfig, resolver func(config.SearchProvider) (Provider, error)) (*Manager, error) {
	sorted := append([]config.SearchProvider{}, cfg.Providers...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Priority < sorted[j].Priority })
	var providers []Provider
	for _, p := range sorted {
		prov, err := resolver(p)
		if err != nil {
			return nil, err
		}
		providers = append(providers, prov)
	}
	return &Manager{providers: providers, strategy: cfg.Strategy}, nil
}

func (m *Manager) Search(ctx context.Context, query string) ([]Result, string, error) {
	if len(m.providers) == 0 {
		return nil, "", fmt.Errorf("no search providers configured")
	}
	var lastErr error
	for _, p := range m.providers {
		res, err := p.Search(ctx, query)
		if err == nil && len(res) > 0 {
			return res, p.ID(), nil
		}
		lastErr = err
		if m.strategy != "fallback" {
			break
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no results")
	}
	return nil, "", lastErr
}

func TestProvider(ctx context.Context, p Provider) TestReport {
	start := time.Now()
	_, err := p.Search(ctx, TestQuery)
	report := TestReport{
		ProviderID: p.ID(),
		LatencyMS:  time.Since(start).Milliseconds(),
		Available:  err == nil,
	}
	if err != nil {
		report.Error = err.Error()
		msg := strings.ToLower(err.Error())
		switch {
		case strings.Contains(msg, "rate"):
			report.QuotaHint = "rate_limit"
		case strings.Contains(msg, "401"), strings.Contains(msg, "unauthorized"):
			report.QuotaHint = "auth_failed"
		case strings.Contains(msg, "timeout"), errors.Is(ctx.Err(), context.DeadlineExceeded):
			report.QuotaHint = "timeout"
		}
	}
	return report
}

func BuildProvider(p config.SearchProvider, http *HTTPClient, apiKey string) (Provider, error) {
	switch p.Type {
	case "duckduckgo":
		return NewDuckDuckGo(p.ID, http), nil
	case "searxng":
		if p.BaseURL == "" {
			return nil, fmt.Errorf("searxng requires base_url")
		}
		return &SearXNG{id: p.ID, baseURL: strings.TrimRight(p.BaseURL, "/"), http: http}, nil
	case "brave":
		if apiKey == "" {
			return nil, fmt.Errorf("brave requires credential")
		}
		return &Brave{id: p.ID, apiKey: apiKey, http: http}, nil
	case "google_cse":
		if apiKey == "" {
			return nil, fmt.Errorf("google_cse requires credential")
		}
		return &GoogleCSE{id: p.ID, apiKey: apiKey, http: http}, nil
	default:
		return nil, fmt.Errorf("unsupported search type %s", p.Type)
	}
}

type SearXNG struct {
	id      string
	baseURL string
	http    *HTTPClient
}

func (s *SearXNG) ID() string { return s.id }

func (s *SearXNG) Search(ctx context.Context, query string) ([]Result, error) {
	u := s.baseURL + "/search?q=" + url.QueryEscape(query) + "&format=json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.http.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var payload struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	out := make([]Result, 0, len(payload.Results))
	for _, r := range payload.Results {
		out = append(out, Result{Title: r.Title, URL: r.URL, Snippet: r.Content})
	}
	return out, nil
}

type Brave struct {
	id     string
	apiKey string
	http   *HTTPClient
}

func (b *Brave) ID() string { return b.id }

func (b *Brave) Search(ctx context.Context, query string) ([]Result, error) {
	u := "https://api.search.brave.com/res/v1/web/search?q=" + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Subscription-Token", b.apiKey)
	resp, err := b.http.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized")
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limited")
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var payload struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	out := make([]Result, 0, len(payload.Web.Results))
	for _, r := range payload.Web.Results {
		out = append(out, Result{Title: r.Title, URL: r.URL, Snippet: r.Description})
	}
	return out, nil
}

type GoogleCSE struct {
	id     string
	apiKey string
	http   *HTTPClient
}

func (g *GoogleCSE) ID() string { return g.id }

func (g *GoogleCSE) Search(ctx context.Context, query string) ([]Result, error) {
	return nil, fmt.Errorf("google_cse requires cx configuration in base_url")
}