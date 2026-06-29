package config

import (
	"path/filepath"
	"testing"
)

func TestPrecedence(t *testing.T) {
	dir := t.TempDir()
	global := filepath.Join(dir, "global.json")
	project := filepath.Join(dir, "project.json")
	flag := filepath.Join(dir, "flag.json")
	_ = Save(global, Config{Models: ModelsConfig{Default: ModelRef{Provider: "openai", Model: "gpt"}}})
	_ = Save(project, Config{Models: ModelsConfig{Default: ModelRef{Model: "claude"}}})
	_ = Save(flag, Config{Browser: BrowserConfig{Headless: false, TimeoutMS: 45000}})

	cfg, err := Load(Sources{GlobalPath: global, ProjectPath: project, FlagPath: flag})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Models.Default.Provider != "openai" {
		t.Fatalf("provider=%s", cfg.Models.Default.Provider)
	}
	if cfg.Models.Default.Model != "claude" {
		t.Fatalf("model=%s", cfg.Models.Default.Model)
	}
	if cfg.Browser.Headless {
		t.Fatal("expected headless false from flag overlay")
	}
	if cfg.Browser.TimeoutMS != 45000 {
		t.Fatalf("timeout=%d", cfg.Browser.TimeoutMS)
	}
}

func TestContainsSecret(t *testing.T) {
	cfg := Default()
	cfg.Models.Default.CredentialRef = "llm/openrouter"
	if ContainsSecret(cfg) {
		t.Fatal("credential_ref should not count as secret field name alone in json - actually it contains credential word")
	}
}