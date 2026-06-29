package research

import "testing"

func TestTransitions(t *testing.T) {
	p := NewPlanFromPrompt("test")
	if err := p.UpdateTaskStatus("T1", TaskRunning, ""); err != nil {
		t.Fatal(err)
	}
	if err := p.UpdateTaskStatus("T1", TaskDone, ""); err != nil {
		t.Fatal(err)
	}
	if err := p.UpdateTaskStatus("T1", TaskTodo, ""); err == nil {
		t.Fatal("expected invalid transition from done")
	}
}

func TestApprovalNotRevokedOnStatusTransition(t *testing.T) {
	p := NewPlanFromPrompt("test")
	p.Approve("user", "ok")
	if err := p.UpdateTaskStatus("T1", TaskRunning, ""); err != nil {
		t.Fatal(err)
	}
	if err := p.UpdateTaskStatus("T1", TaskDone, ""); err != nil {
		t.Fatal(err)
	}
	if !p.IsApproved() {
		t.Fatal("approval should remain after task status transitions")
	}
}

func TestApprovalRevokedOnContentMutation(t *testing.T) {
	p := NewPlanFromPrompt("test")
	p.Approve("user", "ok")
	if err := p.MutateTaskContent("T1", "changed description", nil, nil); err != nil {
		t.Fatal(err)
	}
	if p.IsApproved() {
		t.Fatal("approval should be revoked after task content mutation")
	}
}

func TestApprovalRevokedOnPlanMetaMutation(t *testing.T) {
	p := NewPlanFromPrompt("test")
	p.Approve("user", "ok")
	p.MutatePlanMeta("new title", "", "")
	if p.IsApproved() {
		t.Fatal("approval should be revoked after plan meta mutation")
	}
}

func TestGates(t *testing.T) {
	p := NewPlanFromPrompt("test")
	ok, _ := p.GateStatus("plan_ready")
	if !ok {
		t.Fatal("plan_ready should pass")
	}
	ok, missing := p.GateStatus("execute_ready")
	if ok || len(missing) == 0 {
		t.Fatalf("execute_ready should fail without approval: %v", missing)
	}
	p.Approve("u", "")
	ok, missing = p.GateStatus("execute_ready")
	if !ok {
		t.Fatalf("execute_ready should pass: %v", missing)
	}
}