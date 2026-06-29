package log

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/d-init-d/d-research-cli/internal/config"
)

func TestCanaryNotInConfigArtifact(t *testing.T) {
	canary := "CANARY_SECRET_XYZ_998877"
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	cfg := config.Default()
	if err := config.Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data) + canary
	if !ContainsSecret(content, canary) {
		t.Fatal("sanity: canary detect failed")
	}
	redacted := NewRedactor(canary).RedactString(content)
	if ContainsSecret(redacted, canary) {
		t.Fatalf("canary leaked: %s", redacted)
	}
}