package log

import "testing"

func TestRedactBearer(t *testing.T) {
	r := NewRedactor("CANARY_SECRET_123")
	in := "Authorization: Bearer sk-live-abcdef123456"
	out := r.RedactString(in)
	if ContainsSecret(out, "CANARY_SECRET_123") {
		t.Fatalf("still contains secret: %s", out)
	}
	if out == in {
		t.Fatal("expected redaction")
	}
}

func TestCanary(t *testing.T) {
	canary := "CANARY_SECRET_123"
	r := NewRedactor(canary)
	s := "prefix " + canary + " suffix"
	if r.RedactString(s) == s {
		t.Fatal("canary not redacted")
	}
}