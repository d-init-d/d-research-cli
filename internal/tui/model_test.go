package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/d-init-d/d-research-cli/internal/app"
	"github.com/d-init-d/d-research-cli/internal/event"
)

func TestTUIApprovalAndTaskEventFlow(t *testing.T) {
	root := t.TempDir()
	svc := app.NewService(root)
	m := NewModel(svc)
	m.width = 120
	m.height = 40
	m.layoutViewports()

	cmd := m.startRun("integration prompt")
	msg := cmd()
	m = drainModel(m, msg)

	if !m.awaitApprove {
		t.Fatal("expected awaitApprove after prompt")
	}
	if !hasEventKind(m.events, "approval_required") {
		t.Fatalf("expected approval_required event, got %#v", eventKinds(m.events))
	}

	cmd = m.approvePlan()
	msg = cmd()
	m = drainModel(m, msg)

	if m.awaitApprove {
		t.Fatal("expected awaitApprove cleared after approval")
	}
	if !hasEventKind(m.events, "task_start") {
		t.Fatalf("expected task_start after approval, got %#v", eventKinds(m.events))
	}
	if hasEventKind(m.events, "task_done") {
		t.Fatal("fake task_done should not appear before runtime is wired")
	}
}

func TestTUIUsesMessagePatternNotDirectMutationInCmd(t *testing.T) {
	root := t.TempDir()
	svc := app.NewService(root)
	m := NewModel(svc)
	m.prompt = "held"
	m.awaitApprove = true

	before := len(m.events)
	cmd := m.approvePlan()
	if cmd == nil {
		t.Fatal("approvePlan must return tea.Cmd")
	}
	msg := cmd()
	if _, ok := msg.(approveFinishedMsg); !ok {
		t.Fatalf("approvePlan cmd must return approveFinishedMsg, got %T", msg)
	}
	if len(m.events) != before {
		t.Fatal("tea.Cmd must not mutate Model before Update handles message")
	}
}

func TestTUIWindowSizeAndBusEventMessages(t *testing.T) {
	root := t.TempDir()
	svc := app.NewService(root)
	m := NewModel(svc)

	next, cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = next.(Model)
	if m.width != 100 || m.height != 30 {
		t.Fatalf("unexpected size %dx%d", m.width, m.height)
	}
	if cmd != nil {
		t.Fatal("WindowSizeMsg should not schedule extra cmd")
	}

	m.bus.PublishSimple("research", "coordinator", "approval_required", "blocked", "test", "")
	m = drainBusEvents(m)
	if !m.awaitApprove {
		t.Fatal("bus event should set awaitApprove via Update")
	}
}

func drainModel(m Model, msg tea.Msg) Model {
	next, _ := m.Update(msg)
	m = next.(Model)
	return drainBusEvents(m)
}

func drainBusEvents(m Model) Model {
	for {
		select {
		case ev := <-m.evCh:
			next, _ := m.Update(busEventMsg{event: ev})
			m = next.(Model)
		default:
			return m
		}
	}
}

func hasEventKind(events []event.Event, kind string) bool {
	for _, ev := range events {
		if ev.Kind == kind {
			return true
		}
	}
	return false
}

func eventKinds(events []event.Event) []string {
	out := make([]string, 0, len(events))
	for _, ev := range events {
		out = append(out, ev.Kind)
	}
	return out
}