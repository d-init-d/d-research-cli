package event

import (
	"testing"

	"github.com/d-init-d/d-research-cli/internal/log"
)

func TestRAMCap(t *testing.T) {
	bus := NewBus(log.NewRedactor("SECRET_CANARY"))
	for i := 0; i < MaxRAMEvents+10; i++ {
		bus.Publish("research", "a", "k", "ok", "m", "", 0, nil)
	}
	if len(bus.Snapshot()) != MaxRAMEvents {
		t.Fatalf("len=%d", len(bus.Snapshot()))
	}
}