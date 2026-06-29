package graph

import (
	"fmt"
	"testing"

	"github.com/d-init-d/d-research-cli/internal/kb"
)

func TestProjectionLimit(t *testing.T) {
	root := t.TempDir()
	if err := kb.CreateSkeleton(root); err != nil {
		t.Fatal(err)
	}
	edges := make([]kb.Edge, 0, 200)
	for i := 0; i < 200; i++ {
		edges = append(edges, kb.Edge{
			ID: fmtID(i), Source: fmtID(i), Target: fmtID(i + 1), Status: "approved",
		})
	}
	if err := kb.SaveEdges(root, edges); err != nil {
		t.Fatal(err)
	}
	proj, err := Project(root, 7, DefaultMaxNodes, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(proj.Nodes) > DefaultMaxNodes {
		t.Fatalf("nodes=%d", len(proj.Nodes))
	}
}

func fmtID(i int) string {
	return fmt.Sprintf("N%d", i)
}