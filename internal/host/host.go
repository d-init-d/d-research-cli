package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/d-init-d/d-research-cli/internal/event"
	"github.com/d-init-d/d-research-cli/internal/research"
	"github.com/d-init-d/d-research-cli/internal/store"
	"github.com/google/uuid"
)

type Options struct {
	Headless   bool
	EventsOut  io.Writer
	Resume     bool
	Mode       string
	Simulation bool
}

var ErrRuntimeNotImplemented = errors.New("task execution runtime not implemented")

type Host struct {
	ws    *store.Workspace
	bus   *event.Bus
	opts  Options
}

func New(ws *store.Workspace, bus *event.Bus, opts Options) *Host {
	if opts.Mode == "" {
		opts.Mode = "research"
	}
	return &Host{ws: ws, bus: bus, opts: opts}
}

func (h *Host) Run(ctx context.Context, prompt string) error {
	if err := h.ws.EnsureLayout(); err != nil {
		return err
	}
	st, err := h.ws.LoadState()
	if err != nil {
		return err
	}
	if st.RunID == "" || !h.opts.Resume {
		st.RunID = uuid.NewString()
		st.Prompt = prompt
		st.Mode = h.opts.Mode
		st.Phase = "planning"
	}
	h.bus.SetRunID(st.RunID)
	h.publish("system", "run_start", "running", "Bắt đầu phiên nghiên cứu", "")

	planPath := h.ws.PlanPath()
	plan, err := research.LoadPlan(planPath)
	if err != nil {
		plan = research.NewPlanFromPrompt(prompt)
		if err := research.SavePlan(planPath, plan); err != nil {
			return err
		}
		h.publish("planner", "plan_created", "done", "Đã tạo research-plan.json", planPath)
	}

	if !plan.IsApproved() {
		st.Phase = "awaiting_approval"
		_ = h.ws.SaveState(st)
		h.publish("coordinator", "approval_required", "blocked", "Cần phê duyệt plan trước khi thực thi", planPath)
		if h.opts.Headless {
			return fmt.Errorf("plan requires approval; approve in TUI or set approved state")
		}
		return nil
	}

	ok, missing := plan.GateStatus("execute_ready")
	if !ok {
		return fmt.Errorf("execute gate blocked: %v", missing)
	}

	st.Phase = "executing"
	_ = h.ws.SaveState(st)
	return h.execute(ctx, &plan, &st)
}

func (h *Host) ApprovePlan(by, notes string) error {
	plan, err := research.LoadPlan(h.ws.PlanPath())
	if err != nil {
		return err
	}
	plan.Approve(by, notes)
	if err := research.SavePlan(h.ws.PlanPath(), plan); err != nil {
		return err
	}
	st, _ := h.ws.LoadState()
	st.Approved = true
	st.Phase = "approved"
	return h.ws.SaveState(st)
}

func (h *Host) execute(ctx context.Context, plan *research.Plan, st *store.RunState) error {
	completed := map[string]bool{}
	for _, t := range plan.Tasks {
		if t.Status == research.TaskDone {
			completed[t.ID] = true
		}
	}
	runnable := research.NextRunnable(*plan, completed)
	if len(runnable) == 0 {
		st.Phase = "synthesis"
		_ = h.ws.SaveState(*st)
		h.publish("synthesizer", "execute_complete", "done", "Không còn task runnable", "")
		return nil
	}
	for _, task := range runnable {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := plan.UpdateTaskStatus(task.ID, research.TaskRunning, ""); err != nil {
			return err
		}
		_ = research.SavePlan(h.ws.PlanPath(), *plan)
		agent := roleForTask(task.ID)
		h.publish(agent, "task_start", "running", task.Description, task.ID)
		cp := store.Checkpoint{ID: uuid.NewString(), Kind: "task_start", TaskID: task.ID, Agent: agent}
		_ = h.ws.AppendCheckpoint(cp)
		if err := plan.UpdateTaskStatus(task.ID, research.TaskBlocked, ErrRuntimeNotImplemented.Error()); err != nil {
			return err
		}
		_ = research.SavePlan(h.ws.PlanPath(), *plan)
		h.publish(agent, "task_blocked", "blocked", ErrRuntimeNotImplemented.Error(), task.ID)
		return ErrRuntimeNotImplemented
	}
	st.Phase = "executing"
	return h.ws.SaveState(*st)
}

func roleForTask(id string) string {
	switch id {
	case "T1":
		return "searcher"
	case "T2":
		return "verifier"
	case "T3":
		return "synthesizer"
	default:
		return "coordinator"
	}
}

func (h *Host) publish(agent, kind, status, message, artifact string, durationMS ...int64) {
	var d int64
	if len(durationMS) > 0 {
		d = durationMS[0]
	}
	ev := h.bus.Publish(h.opts.Mode, agent, kind, status, message, artifact, d, nil)
	if h.opts.Headless && h.opts.EventsOut != nil {
		_, _ = h.opts.EventsOut.Write(h.bus.MarshalJSONL(ev))
	}
	_ = h.ws.AppendEventLine(h.bus.MarshalJSONL(ev))
}

func (h *Host) ExportState() ([]byte, error) {
	st, err := h.ws.LoadState()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(st, "", "  ")
}