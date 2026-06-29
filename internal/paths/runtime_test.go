package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRuntimeRootFromEnv(t *testing.T) {
	root := t.TempDir()
	skill := filepath.Join(root, ResearchSkillSubdir)
	if err := os.MkdirAll(skill, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("D_RESEARCH_RUNTIME", root)
	got, err := RuntimeRoot()
	if err != nil {
		t.Fatal(err)
	}
	if got != root {
		t.Fatalf("RuntimeRoot() = %q, want %q", got, root)
	}
}

func TestScriptPathJoinsName(t *testing.T) {
	root := t.TempDir()
	scripts := filepath.Join(root, ScriptsSubdir)
	if err := os.MkdirAll(scripts, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("D_RESEARCH_SCRIPTS", scripts)
	got, err := ScriptPath("playwright_worker.mjs")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(scripts, "playwright_worker.mjs")
	if got != want {
		t.Fatalf("ScriptPath() = %q, want %q", got, want)
	}
}