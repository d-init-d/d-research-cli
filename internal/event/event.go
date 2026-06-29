package event

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/d-init-d/d-research-cli/internal/log"
	"github.com/google/uuid"
)

const MaxRAMEvents = 2000

type Event struct {
	ID          string         `json:"id"`
	RunID       string         `json:"run_id"`
	Time        time.Time      `json:"time"`
	Mode        string         `json:"mode"`
	Agent       string         `json:"agent,omitempty"`
	Kind        string         `json:"kind"`
	Status      string         `json:"status,omitempty"`
	Message     string         `json:"message,omitempty"`
	ArtifactRef string         `json:"artifact_ref,omitempty"`
	DurationMS  int64          `json:"duration_ms,omitempty"`
	Usage       map[string]any `json:"usage,omitempty"`
}

type Bus struct {
	mu        sync.RWMutex
	events    []Event
	listeners []func(Event)
	redactor  *log.Redactor
	runID     string
}

func NewBus(redactor *log.Redactor) *Bus {
	return &Bus{redactor: redactor}
}

func (b *Bus) SetRunID(runID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.runID = runID
}

func (b *Bus) PublishSimple(mode, agent, kind, status, message, artifactRef string) Event {
	return b.Publish(mode, agent, kind, status, message, artifactRef, 0, nil)
}

func (b *Bus) Publish(mode, agent, kind, status, message, artifactRef string, durationMS int64, usage map[string]any) Event {
	ev := Event{
		ID:          uuid.NewString(),
		RunID:       b.currentRunID(),
		Time:        time.Now().UTC(),
		Mode:        mode,
		Agent:       agent,
		Kind:        kind,
		Status:      status,
		Message:     b.redactor.RedactString(message),
		ArtifactRef: artifactRef,
		DurationMS:  durationMS,
		Usage:       b.redactor.RedactMap(usage),
	}
	b.mu.Lock()
	b.events = append(b.events, ev)
	if len(b.events) > MaxRAMEvents {
		b.events = b.events[len(b.events)-MaxRAMEvents:]
	}
	listeners := append([]func(Event){}, b.listeners...)
	b.mu.Unlock()
	for _, fn := range listeners {
		fn(ev)
	}
	return ev
}

func (b *Bus) Subscribe(fn func(Event)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.listeners = append(b.listeners, fn)
}

func (b *Bus) Snapshot() []Event {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]Event, len(b.events))
	copy(out, b.events)
	return out
}

func (b *Bus) MarshalJSONL(ev Event) []byte {
	data, _ := json.Marshal(ev)
	return append(data, '\n')
}

func (b *Bus) currentRunID() string {
	if b.runID != "" {
		return b.runID
	}
	return "bootstrap"
}