package simulation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/d-init-d/d-research-cli/internal/kb"
	"github.com/d-init-d/d-research-cli/internal/store"
)

const EngineVersion = "aleph-engine-v0.1"

type RunMeta struct {
	GraphHash      string `json:"graph_hash"`
	EngineVersion  string `json:"engine_version"`
	Seed           int64  `json:"seed"`
	Context        string `json:"context"`
	BudgetTokens   int    `json:"budget_tokens"`
	Uncalibrated   bool   `json:"uncalibrated_warning"`
}

type Scenario struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Overlay     string `json:"overlay"`
}

type Outcome struct {
	ID          string  `json:"id"`
	Label       string  `json:"label"`
	Probability string  `json:"probability"`
	Value       float64 `json:"value,omitempty"`
}

type Service struct {
	ws *store.Workspace
}

func New(ws *store.Workspace) *Service {
	return &Service{ws: ws}
}

func (s *Service) EnsureKB() error {
	st := kb.StatusOf(s.ws.Root)
	switch {
	case !st.Exists || st.Empty:
		return kb.CreateSkeleton(s.ws.Root)
	case st.Valid:
		return nil
	default:
		return fmt.Errorf("kb invalid: %s; validate/repair required", st.Error)
	}
}

func (s *Service) ProposeEdges(evidenceID string, pairs [][2]string) ([]kb.Edge, error) {
	edges, err := kb.LoadEdges(s.ws.Root)
	if err != nil {
		return nil, err
	}
	for i, pair := range pairs {
		edges = append(edges, kb.Edge{
			ID:         fmt.Sprintf("E-%d-%d", time.Now().Unix(), i),
			Source:     pair[0],
			Target:     pair[1],
			Relation:   "causes",
			EvidenceID: evidenceID,
			Status:     "proposed",
			Calibrated: false,
		})
	}
	if err := kb.SaveEdges(s.ws.Root, edges); err != nil {
		return nil, err
	}
	return edges, nil
}

func (s *Service) BatchReview(edgeIDs []string, approve bool) error {
	edges, err := kb.LoadEdges(s.ws.Root)
	if err != nil {
		return err
	}
	set := map[string]bool{}
	for _, id := range edgeIDs {
		set[id] = true
	}
	status := "rejected"
	if approve {
		status = "approved"
	}
	for i := range edges {
		if set[edges[i].ID] {
			edges[i].Status = status
		}
	}
	return kb.SaveEdges(s.ws.Root, edges)
}

func (s *Service) Run(seed int64, context string, budget int) (RunMeta, []Outcome, error) {
	if err := s.EnsureKB(); err != nil {
		return RunMeta{}, nil, err
	}
	hash, err := kb.GraphHash(s.ws.Root)
	if err != nil {
		return RunMeta{}, nil, err
	}
	meta := RunMeta{
		GraphHash:     hash,
		EngineVersion: EngineVersion,
		Seed:          seed,
		Context:       context,
		BudgetTokens:  budget,
		Uncalibrated:  true,
	}
	outcomes := deterministicOutcomes(hash, seed)
	if err := s.persist(meta, outcomes); err != nil {
		return RunMeta{}, nil, err
	}
	return meta, outcomes, nil
}

func deterministicOutcomes(hash string, seed int64) []Outcome {
	base := float64((int(seed) + len(hash)) % 100)
	return []Outcome{
		{ID: "O1", Label: "Kịch bản cơ sở", Probability: "Uncalibrated", Value: base / 100},
		{ID: "O2", Label: "Kịch bản bi quan", Probability: "Uncalibrated", Value: (base + 7) / 100},
		{ID: "O3", Label: "Kịch bản lạc quan", Probability: "Uncalibrated", Value: (base + 13) / 100},
	}
}

func (s *Service) persist(meta RunMeta, outcomes []Outcome) error {
	dir := s.ws.SimulationDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	metaPath := filepath.Join(dir, "run-meta.json")
	outPath := filepath.Join(dir, "outcomes.json")
	if err := writeJSON(metaPath, meta); err != nil {
		return err
	}
	return writeJSON(outPath, outcomes)
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}