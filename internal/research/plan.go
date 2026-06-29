package research

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	TaskTodo    = "todo"
	TaskRunning = "running"
	TaskDone    = "done"
	TaskBlocked = "blocked"
)

type Plan struct {
	PlanID    string         `json:"plan_id"`
	Title     string         `json:"title"`
	Created   string         `json:"created,omitempty"`
	Approval  Approval       `json:"approval"`
	Tasks     []Task         `json:"tasks"`
	Gates     Gates          `json:"gates"`
	Notes     string         `json:"notes,omitempty"`
	Stopping  string         `json:"stopping_criteria,omitempty"`
	StoppingOK bool          `json:"stopping_criteria_satisfied"`
}

type Approval struct {
	ApprovedBy string `json:"approved_by"`
	ApprovedAt string `json:"approved_at"`
	Notes      string `json:"notes"`
}

type Task struct {
	ID            string   `json:"id"`
	Description   string   `json:"description"`
	DependsOn     []string `json:"depends_on"`
	Status        string   `json:"status"`
	BlockerReason string   `json:"blocker_reason"`
	Outputs       []string `json:"outputs"`
	Owner         string   `json:"owner,omitempty"`
}

type Gates struct {
	PlanReady       Gate `json:"plan_ready"`
	ExecuteReady    Gate `json:"execute_ready"`
	SynthesizeReady Gate `json:"synthesize_ready"`
	ReleaseReady    Gate `json:"release_ready"`
}

type Gate struct {
	Description string `json:"description,omitempty"`
}

func LoadPlan(path string) (Plan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Plan{}, err
	}
	var p Plan
	if err := json.Unmarshal(data, &p); err != nil {
		return Plan{}, err
	}
	return p, nil
}

func SavePlan(path string, p Plan) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".plan-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func (p *Plan) RevokeApproval() {
	p.Approval = Approval{}
}

func (p *Plan) Approve(by, notes string) {
	p.Approval = Approval{
		ApprovedBy: by,
		ApprovedAt: time.Now().UTC().Format(time.RFC3339),
		Notes:      notes,
	}
}

func (p *Plan) IsApproved() bool {
	return p.Approval.ApprovedBy != "" && p.Approval.ApprovedAt != ""
}

// UpdateTaskStatus changes task lifecycle state without revoking plan approval.
func (p *Plan) UpdateTaskStatus(taskID, status, blocker string) error {
	for i := range p.Tasks {
		if p.Tasks[i].ID != taskID {
			continue
		}
		if !validTransition(p.Tasks[i].Status, status) {
			return fmt.Errorf("invalid transition %s -> %s for task %s", p.Tasks[i].Status, status, taskID)
		}
		p.Tasks[i].Status = status
		p.Tasks[i].BlockerReason = blocker
		return nil
	}
	return fmt.Errorf("task %s not found", taskID)
}

// MutateTaskContent updates task definition fields and revokes approval when already approved.
func (p *Plan) MutateTaskContent(taskID, description string, dependsOn []string, outputs []string) error {
	for i := range p.Tasks {
		if p.Tasks[i].ID != taskID {
			continue
		}
		p.Tasks[i].Description = description
		p.Tasks[i].DependsOn = dependsOn
		p.Tasks[i].Outputs = outputs
		if p.IsApproved() {
			p.RevokeApproval()
		}
		return nil
	}
	return fmt.Errorf("task %s not found", taskID)
}

// MutatePlanMeta updates plan-level content and revokes approval when already approved.
func (p *Plan) MutatePlanMeta(title, notes, stopping string) {
	if title != "" {
		p.Title = title
	}
	if notes != "" {
		p.Notes = notes
	}
	if stopping != "" {
		p.Stopping = stopping
	}
	if p.IsApproved() {
		p.RevokeApproval()
	}
}

func validTransition(from, to string) bool {
	if from == to {
		return true
	}
	switch from {
	case TaskTodo:
		return to == TaskRunning || to == TaskBlocked
	case TaskRunning:
		return to == TaskDone || to == TaskBlocked || to == TaskTodo
	case TaskBlocked:
		return to == TaskTodo || to == TaskRunning
	case TaskDone:
		return false
	default:
		return false
	}
}

func (p *Plan) GateStatus(gate string) (bool, []string) {
	var missing []string
	switch gate {
	case "plan_ready":
		if len(p.Tasks) == 0 {
			missing = append(missing, "no_tasks")
		}
		if hasDependencyCycle(p.Tasks) {
			missing = append(missing, "dependency_cycle")
		}
	case "execute_ready":
		ok, m := p.GateStatus("plan_ready")
		if !ok {
			missing = append(missing, m...)
		}
		if !p.IsApproved() {
			missing = append(missing, "plan_not_approved")
		}
	case "synthesize_ready":
		for _, t := range p.Tasks {
			if t.Status != TaskDone && t.Status != TaskBlocked {
				missing = append(missing, "task_"+t.ID+"_not_terminal")
			}
		}
	case "release_ready":
		ok, m := p.GateStatus("synthesize_ready")
		if !ok {
			missing = append(missing, m...)
		}
		if !p.StoppingOK {
			missing = append(missing, "stopping_criteria_not_satisfied")
		}
	default:
		missing = append(missing, "unknown_gate")
	}
	return len(missing) == 0, missing
}

func hasDependencyCycle(tasks []Task) bool {
	index := map[string]int{}
	for i, t := range tasks {
		index[t.ID] = i
	}
	visiting := map[string]bool{}
	visited := map[string]bool{}
	var dfs func(id string) bool
	dfs = func(id string) bool {
		if visiting[id] {
			return true
		}
		if visited[id] {
			return false
		}
		visiting[id] = true
		i, ok := index[id]
		if ok {
			for _, dep := range tasks[i].DependsOn {
				if dfs(dep) {
					return true
				}
			}
		}
		delete(visiting, id)
		visited[id] = true
		return false
	}
	for _, t := range tasks {
		if dfs(t.ID) {
			return true
		}
	}
	return false
}

func NextRunnable(p Plan, completed map[string]bool) []Task {
	var out []Task
	for _, t := range p.Tasks {
		if t.Status != TaskTodo {
			continue
		}
		ready := true
		for _, dep := range t.DependsOn {
			if !completed[dep] {
				ready = false
				break
			}
		}
		if ready {
			out = append(out, t)
		}
	}
	return out
}

func NewPlanFromPrompt(prompt string) Plan {
	return Plan{
		PlanID:  fmt.Sprintf("plan-%d", time.Now().Unix()),
		Title:   prompt,
		Created: time.Now().UTC().Format("2006-01-02"),
		Tasks: []Task{
			{ID: "T1", Description: "Thu thập nguồn và khởi tạo evidence ledger", Status: TaskTodo},
			{ID: "T2", Description: "Xác minh claim và tìm mâu thuẫn", DependsOn: []string{"T1"}, Status: TaskTodo},
			{ID: "T3", Description: "Tổng hợp báo cáo cuối", DependsOn: []string{"T2"}, Status: TaskTodo},
		},
		Gates: Gates{
			PlanReady:       Gate{Description: "Plan shaped and rendered"},
			ExecuteReady:    Gate{Description: "Plan approved"},
			SynthesizeReady: Gate{Description: "All tasks terminal"},
			ReleaseReady:    Gate{Description: "Final report ready"},
		},
		Stopping: "Mọi sub-question có ít nhất một nguồn đã xác minh.",
	}
}

var ErrPlanNotApproved = errors.New("plan not approved")