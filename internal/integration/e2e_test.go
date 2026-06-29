package integration

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/d-init-d/d-research-cli/internal/event"
	"github.com/d-init-d/d-research-cli/internal/host"
	"github.com/d-init-d/d-research-cli/internal/log"
	"github.com/d-init-d/d-research-cli/internal/research"
	"github.com/d-init-d/d-research-cli/internal/store"
)

func TestResearchHeadlessApprovalFlow(t *testing.T) {
	root := t.TempDir()
	ws, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	bus := event.NewBus(log.NewRedactor("CANARY_E2E_SECRET"))
	var buf bytes.Buffer
	h := host.New(ws, bus, host.Options{Headless: true, EventsOut: &buf, Mode: "research"})
	if err := h.Run(context.Background(), "test prompt"); err == nil {
		t.Fatal("expected approval required error")
	}
	planPath := filepath.Join(root, "research-output", "research-plan.json")
	plan, err := research.LoadPlan(planPath)
	if err != nil {
		t.Fatal(err)
	}
	plan.Approve("test", "ok")
	if err := research.SavePlan(planPath, plan); err != nil {
		t.Fatal(err)
	}
	err = h.Run(context.Background(), "test prompt")
	if !errors.Is(err, host.ErrRuntimeNotImplemented) {
		t.Fatalf("expected ErrRuntimeNotImplemented, got %v events=%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "task_start") {
		t.Fatalf("events=%s", buf.String())
	}
	if strings.Contains(buf.String(), "task_done") {
		t.Fatalf("fake completion leaked: %s", buf.String())
	}
}

func TestResumeKeepsApprovalAndPlanShape(t *testing.T) {
	root := t.TempDir()
	ws, _ := store.Open(root)
	bus := event.NewBus(log.NewRedactor())
	h := host.New(ws, bus, host.Options{Headless: true, Resume: true, Mode: "research"})
	_ = h.Run(context.Background(), "resume test")
	plan, _ := research.LoadPlan(ws.PlanPath())
	plan.Approve("t", "")
	_ = research.SavePlan(ws.PlanPath(), plan)
	_ = h.Run(context.Background(), "resume test")
	plan2, _ := research.LoadPlan(ws.PlanPath())
	if !plan2.IsApproved() {
		t.Fatal("approval should persist across resume")
	}
	if len(plan2.Tasks) != len(plan.Tasks) {
		t.Fatal("task count changed on resume")
	}
	_ = h.Run(context.Background(), "resume test")
	plan3, _ := research.LoadPlan(ws.PlanPath())
	if len(plan3.Tasks) != len(plan2.Tasks) {
		t.Fatal("task count changed on second resume")
	}
}