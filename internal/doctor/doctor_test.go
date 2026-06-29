package doctor

import (
	"testing"

	"github.com/d-init-d/d-research-cli/internal/config"
)

func TestClassifyReady(t *testing.T) {
	status, ok := classify([]Check{
		{Name: "config", OK: true},
		{Name: "workspace", OK: true},
		{Name: "chromium", OK: true},
		{Name: "d_research_skill", OK: true},
	})
	if status != StatusReady || !ok {
		t.Fatalf("status=%s ok=%v", status, ok)
	}
}

func TestClassifyDegradedMissingChromium(t *testing.T) {
	status, ok := classify([]Check{
		{Name: "config", OK: true},
		{Name: "workspace", OK: true},
		{Name: "chromium", OK: false},
	})
	if status != StatusDegraded || ok {
		t.Fatalf("status=%s ok=%v", status, ok)
	}
}

func TestClassifyFailedConfig(t *testing.T) {
	status, ok := classify([]Check{
		{Name: "config", OK: false},
		{Name: "chromium", OK: true},
	})
	if status != StatusFailed || ok {
		t.Fatalf("status=%s ok=%v", status, ok)
	}
}

func TestRunDefaultsToDegradedWithoutRuntime(t *testing.T) {
	rep := Run(config.Default(), t.TempDir())
	if rep.OK {
		t.Fatal("expected OK=false without pinned runtime")
	}
	if rep.Status != StatusDegraded && rep.Status != StatusFailed {
		t.Fatalf("unexpected status %q", rep.Status)
	}
}