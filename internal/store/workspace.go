package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/d-init-d/d-research-cli/internal/paths"
)

type Workspace struct {
	Root string
}

func Open(root string) (*Workspace, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return &Workspace{Root: abs}, nil
}

func (w *Workspace) MetaDir() string {
	return filepath.Join(w.Root, paths.ProjectMetaDir)
}

func (w *Workspace) StatePath() string {
	return filepath.Join(w.MetaDir(), "state.json")
}

func (w *Workspace) CheckpointPath() string {
	return filepath.Join(w.MetaDir(), "checkpoints.jsonl")
}

func (w *Workspace) EventsPath() string {
	return filepath.Join(w.MetaDir(), "events.jsonl")
}

func (w *Workspace) ResearchOutputDir() string {
	return filepath.Join(w.Root, paths.ResearchOutputDir)
}

func (w *Workspace) PlanPath() string {
	return filepath.Join(w.ResearchOutputDir(), "research-plan.json")
}

func (w *Workspace) SimulationDir() string {
	return filepath.Join(w.Root, paths.SimulationDir)
}

func (w *Workspace) KBDir() string {
	return filepath.Join(w.Root, paths.KBDir)
}

func (w *Workspace) EnsureLayout() error {
	dirs := []string{
		w.MetaDir(),
		w.ResearchOutputDir(),
		w.SimulationDir(),
		w.KBDir(),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}

type RunState struct {
	RunID        string    `json:"run_id"`
	Prompt       string    `json:"prompt,omitempty"`
	Mode         string    `json:"mode"`
	Phase        string    `json:"phase"`
	Approved     bool      `json:"plan_approved"`
	GraphHash    string    `json:"graph_hash,omitempty"`
	EngineSeed   int64     `json:"engine_seed,omitempty"`
	UpdatedAt    time.Time `json:"updated_at"`
	CheckpointID string    `json:"checkpoint_id,omitempty"`
}

func (w *Workspace) LoadState() (RunState, error) {
	data, err := os.ReadFile(w.StatePath())
	if errors.Is(err, os.ErrNotExist) {
		return RunState{Mode: "research", Phase: "init"}, nil
	}
	if err != nil {
		return RunState{}, err
	}
	var st RunState
	if err := json.Unmarshal(data, &st); err != nil {
		return RunState{}, err
	}
	return st, nil
}

func (w *Workspace) SaveState(st RunState) error {
	st.UpdatedAt = time.Now().UTC()
	return writeJSONAtomic(w.StatePath(), st)
}

type Checkpoint struct {
	ID        string    `json:"id"`
	Time      time.Time `json:"time"`
	Kind      string    `json:"kind"`
	TaskID    string    `json:"task_id,omitempty"`
	Agent     string    `json:"agent,omitempty"`
	Artifact  string    `json:"artifact,omitempty"`
}

func (w *Workspace) AppendCheckpoint(cp Checkpoint) error {
	if cp.Time.IsZero() {
		cp.Time = time.Now().UTC()
	}
	line, err := json.Marshal(cp)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(w.CheckpointPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

func (w *Workspace) AppendEventLine(line []byte) error {
	f, err := os.OpenFile(w.EventsPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(line); err != nil {
		return err
	}
	return f.Sync()
}

func (w *Workspace) SafeJoin(elem ...string) (string, error) {
	p := filepath.Join(append([]string{w.Root}, elem...)...)
	rel, err := filepath.Rel(w.Root, p)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path escapes workspace: %s", p)
	}
	return p, nil
}

func writeJSONAtomic(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".state-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
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
	return os.Rename(tmpName, path)
}