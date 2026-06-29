package store

import "testing"

func TestSafeJoin(t *testing.T) {
	dir := t.TempDir()
	w, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.SafeJoin("..", "etc", "passwd"); err == nil {
		t.Fatal("expected path escape error")
	}
	if _, err := w.SafeJoin("research-output", "report.md"); err != nil {
		t.Fatal(err)
	}
}