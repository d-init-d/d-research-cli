package kb

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/d-init-d/d-research-cli/internal/paths"
)

const (
	SchemaVersion = "aleph-kb-v0.1"
	GraphVersion  = "causal-graph-v0.1"
)

type Status struct {
	Exists       bool   `json:"exists"`
	Valid        bool   `json:"valid"`
	Empty        bool   `json:"empty"`
	Schema       string `json:"schema_version,omitempty"`
	GraphVersion string `json:"graph_version,omitempty"`
	Error        string `json:"error,omitempty"`
}

type Manifest struct {
	SchemaVersion string    `json:"schema_version"`
	GraphVersion  string    `json:"graph_version"`
	CreatedAt     time.Time `json:"created_at"`
	Provenance    string    `json:"provenance"`
}

type Edge struct {
	ID         string  `json:"id"`
	Source     string  `json:"source"`
	Target     string  `json:"target"`
	Relation   string  `json:"relation"`
	EvidenceID string  `json:"evidence_id"`
	Status     string  `json:"status"`
	Weight     float64 `json:"weight,omitempty"`
	Calibrated bool    `json:"calibrated"`
}

func Dir(root string) string {
	return filepath.Join(root, paths.KBDir)
}

func StatusOf(root string) Status {
	dir := Dir(root)
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return Status{Exists: false}
		}
		return Status{Exists: true, Error: err.Error()}
	}
	if !info.IsDir() {
		return Status{Exists: true, Error: "kb path is not a directory"}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return Status{Exists: true, Error: err.Error()}
	}
	if len(entries) == 0 {
		return Status{Exists: true, Empty: true}
	}
	manifestPath := filepath.Join(dir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return Status{Exists: true, Error: "missing manifest.json"}
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Status{Exists: true, Error: "invalid manifest.json"}
	}
	if m.SchemaVersion != SchemaVersion {
		return Status{Exists: true, Error: fmt.Sprintf("unsupported schema %s", m.SchemaVersion)}
	}
	return Status{
		Exists:       true,
		Valid:        true,
		Schema:       m.SchemaVersion,
		GraphVersion: m.GraphVersion,
	}
}

func CreateSkeleton(root string) error {
	dir := Dir(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	manifest := Manifest{
		SchemaVersion: SchemaVersion,
		GraphVersion:  GraphVersion,
		CreatedAt:     time.Now().UTC(),
		Provenance:    "d-research-cli simulation",
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0o644); err != nil {
		return err
	}
	graph := struct {
		Nodes []map[string]string `json:"nodes"`
		Edges []Edge              `json:"edges"`
	}{Nodes: []map[string]string{}, Edges: []Edge{}}
	gdata, _ := json.MarshalIndent(graph, "", "  ")
	return os.WriteFile(filepath.Join(dir, "graph.json"), gdata, 0o644)
}

func LoadEdges(root string) ([]Edge, error) {
	path := filepath.Join(Dir(root), "graph.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Edges []Edge `json:"edges"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return payload.Edges, nil
}

func SaveEdges(root string, edges []Edge) error {
	path := filepath.Join(Dir(root), "graph.json")
	var payload struct {
		Nodes []map[string]string `json:"nodes"`
		Edges []Edge              `json:"edges"`
	}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &payload)
	}
	payload.Edges = edges
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func GraphHash(root string) (string, error) {
	edges, err := LoadEdges(root)
	if err != nil {
		return "", err
	}
	approved := make([]Edge, 0)
	for _, e := range edges {
		if e.Status == "approved" {
			approved = append(approved, e)
		}
	}
	data, err := json.Marshal(approved)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func Export(root, outZip string) error {
	dir := Dir(root)
	st := StatusOf(root)
	if !st.Valid {
		return fmt.Errorf("kb invalid: %s", st.Error)
	}
	hash, err := GraphHash(root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outZip), 0o755); err != nil {
		return err
	}
	f, err := os.Create(outZip)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()
	meta := map[string]string{
		"schema_version": SchemaVersion,
		"graph_version":  GraphVersion,
		"graph_hash":     hash,
		"exported_at":    time.Now().UTC().Format(time.RFC3339),
	}
	metaData, _ := json.MarshalIndent(meta, "", "  ")
	w, err := zw.Create("export-manifest.json")
	if err != nil {
		return err
	}
	if _, err := w.Write(metaData); err != nil {
		return err
	}
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		base := filepath.Base(path)
		if strings.Contains(base, "cache") || strings.Contains(base, "credential") || strings.HasSuffix(base, ".log") {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		w, err := zw.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(w, src)
		return err
	})
}