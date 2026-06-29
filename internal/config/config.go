package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/d-init-d/d-research-cli/internal/paths"
)

type Config struct {
	Models  ModelsConfig  `json:"models"`
	Search  SearchConfig  `json:"search"`
	Browser BrowserConfig `json:"browser"`
	Locale  string        `json:"locale,omitempty"`
}

type ModelsConfig struct {
	Default ModelRef            `json:"default"`
	Roles   map[string]ModelRef `json:"roles,omitempty"`
}

type ModelRef struct {
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	CredentialRef string `json:"credential_ref,omitempty"`
	BaseURL       string `json:"base_url,omitempty"`
}

type SearchConfig struct {
	Strategy  string           `json:"strategy"`
	Providers []SearchProvider `json:"providers"`
}

type SearchProvider struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	Priority      int    `json:"priority"`
	CredentialRef string `json:"credential_ref,omitempty"`
	BaseURL       string `json:"base_url,omitempty"`
}

type BrowserConfig struct {
	Engine    string `json:"engine"`
	Headless  bool   `json:"headless"`
	TimeoutMS int    `json:"timeout_ms"`
}

type Sources struct {
	GlobalPath  string
	ProjectPath string
	FlagPath    string
}

func Default() Config {
	return Config{
		Models: ModelsConfig{
			Default: ModelRef{
				Provider: "openrouter",
				Model:    "google/gemini-2.5-flash",
			},
		},
		Search: SearchConfig{
			Strategy: "fallback",
			Providers: []SearchProvider{
				{ID: "ddg", Type: "duckduckgo", Priority: 10},
			},
		},
		Browser: BrowserConfig{
			Engine:    "chromium",
			Headless:  true,
			TimeoutMS: 30000,
		},
		Locale: "vi",
	}
}

func Load(sources Sources) (Config, error) {
	cfg := Default()
	ordered := []string{}
	if sources.GlobalPath != "" {
		ordered = append(ordered, sources.GlobalPath)
	}
	if sources.ProjectPath != "" {
		ordered = append(ordered, sources.ProjectPath)
	}
	if sources.FlagPath != "" {
		ordered = append(ordered, sources.FlagPath)
	}
	for _, path := range ordered {
		if path == "" {
			continue
		}
		overlay, err := readFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return Config{}, err
		}
		cfg = merge(cfg, overlay)
	}
	return cfg, nil
}

func ResolveSources(cwd, flagPath string) Sources {
	global, _ := paths.GlobalConfigPath()
	return Sources{
		GlobalPath:  global,
		ProjectPath: paths.ProjectConfigPath(cwd),
		FlagPath:    flagPath,
	}
}

func readFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}

func merge(base, overlay Config) Config {
	if overlay.Locale != "" {
		base.Locale = overlay.Locale
	}
	if overlay.Models.Default.Provider != "" || overlay.Models.Default.Model != "" || overlay.Models.Default.CredentialRef != "" || overlay.Models.Default.BaseURL != "" {
		base.Models.Default = mergeModelRef(base.Models.Default, overlay.Models.Default)
	}
	if overlay.Models.Roles != nil {
		if base.Models.Roles == nil {
			base.Models.Roles = map[string]ModelRef{}
		}
		for k, v := range overlay.Models.Roles {
			prev := base.Models.Roles[k]
			base.Models.Roles[k] = mergeModelRef(prev, v)
		}
	}
	if overlay.Search.Strategy != "" {
		base.Search.Strategy = overlay.Search.Strategy
	}
	if len(overlay.Search.Providers) > 0 {
		base.Search.Providers = mergeProviders(base.Search.Providers, overlay.Search.Providers)
	}
	if overlay.Browser.Engine != "" {
		base.Browser.Engine = overlay.Browser.Engine
	}
	if overlay.Browser.TimeoutMS > 0 {
		base.Browser.TimeoutMS = overlay.Browser.TimeoutMS
	}
	// Headless is a bool; overlay explicitly setting false must win when file exists.
	if overlay.Browser.Engine != "" || overlay.Browser.TimeoutMS > 0 {
		base.Browser.Headless = overlay.Browser.Headless
	}
	return base
}

func mergeModelRef(base, overlay ModelRef) ModelRef {
	if overlay.Provider != "" {
		base.Provider = overlay.Provider
	}
	if overlay.Model != "" {
		base.Model = overlay.Model
	}
	if overlay.CredentialRef != "" {
		base.CredentialRef = overlay.CredentialRef
	}
	if overlay.BaseURL != "" {
		base.BaseURL = overlay.BaseURL
	}
	return base
}

func mergeProviders(base, overlay []SearchProvider) []SearchProvider {
	byID := map[string]SearchProvider{}
	order := []string{}
	for _, p := range base {
		byID[p.ID] = p
		order = append(order, p.ID)
	}
	for _, p := range overlay {
		if _, ok := byID[p.ID]; !ok {
			order = append(order, p.ID)
		}
		prev := byID[p.ID]
		if p.Type != "" {
			prev.Type = p.Type
		}
		if p.Priority != 0 {
			prev.Priority = p.Priority
		}
		if p.CredentialRef != "" {
			prev.CredentialRef = p.CredentialRef
		}
		if p.BaseURL != "" {
			prev.BaseURL = p.BaseURL
		}
		if prev.ID == "" {
			prev.ID = p.ID
		}
		byID[p.ID] = prev
	}
	out := make([]SearchProvider, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	return out
}

func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, data)
}

func Validate(cfg Config) []string {
	var issues []string
	if strings.TrimSpace(cfg.Models.Default.Provider) == "" {
		issues = append(issues, "models.default.provider is required")
	}
	if strings.TrimSpace(cfg.Models.Default.Model) == "" {
		issues = append(issues, "models.default.model is required")
	}
	if cfg.Browser.TimeoutMS <= 0 {
		issues = append(issues, "browser.timeout_ms must be positive")
	}
	for _, p := range cfg.Search.Providers {
		if p.ID == "" || p.Type == "" {
			issues = append(issues, "search provider requires id and type")
		}
	}
	return issues
}

func ContainsSecret(cfg Config) bool {
	data, _ := json.Marshal(cfg)
	s := strings.ToLower(string(data))
	for _, needle := range []string{"api_key", "apikey", "secret", "password", "token", "bearer"} {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}