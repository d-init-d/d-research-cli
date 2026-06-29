package simulation

import (
	"testing"

	"github.com/d-init-d/d-research-cli/internal/store"
)

func TestDeterministicRun(t *testing.T) {
	root := t.TempDir()
	ws, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	svc := New(ws)
	meta1, out1, err := svc.Run(99, "ctx", 1000)
	if err != nil {
		t.Fatal(err)
	}
	meta2, out2, err := svc.Run(99, "ctx", 1000)
	if err != nil {
		t.Fatal(err)
	}
	if meta1.GraphHash != meta2.GraphHash {
		t.Fatalf("hash mismatch %s %s", meta1.GraphHash, meta2.GraphHash)
	}
	if out1[0].Value != out2[0].Value {
		t.Fatalf("outcome mismatch %v %v", out1[0], out2[0])
	}
	if !meta1.Uncalibrated || out1[0].Probability != "Uncalibrated" {
		t.Fatal("expected uncalibrated label")
	}
}