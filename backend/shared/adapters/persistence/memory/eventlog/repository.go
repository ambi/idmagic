// Package eventlog is the in-memory adapter for event_logs / event_deliveries
// (ADR-094, wi-184 T002), used for tests and local demo runtime. It satisfies
// backend/shared/eventlog.Recorder.
package eventlog

import (
	"context"
	"fmt"
	"sync"

	sharedeventlog "github.com/ambi/idmagic/backend/shared/eventlog"
)

type Repository struct {
	mu         sync.Mutex
	logs       map[string]sharedeventlog.Record
	deliveries map[string]sharedeventlog.Delivery
}

var _ sharedeventlog.Recorder = (*Repository)(nil)

func New() *Repository {
	return &Repository{
		logs:       map[string]sharedeventlog.Record{},
		deliveries: map[string]sharedeventlog.Delivery{},
	}
}

func (r *Repository) Append(_ context.Context, rec sharedeventlog.Record) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.logs[rec.EventID]; exists {
		return fmt.Errorf("event_logs: duplicate event_id %q", rec.EventID)
	}
	r.logs[rec.EventID] = rec
	return nil
}

func (r *Repository) AppendDelivery(_ context.Context, eventID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.logs[eventID]; !exists {
		return fmt.Errorf("event_deliveries: unknown event_id %q", eventID)
	}
	if _, exists := r.deliveries[eventID]; exists {
		return fmt.Errorf("event_deliveries: duplicate event_id %q", eventID)
	}
	r.deliveries[eventID] = sharedeventlog.Delivery{
		EventID: eventID,
		Status:  sharedeventlog.DeliveryStatusPending,
	}
	return nil
}

func (r *Repository) FindByID(_ context.Context, eventID string) (*sharedeventlog.Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.logs[eventID]
	if !ok {
		return nil, nil
	}
	return &rec, nil
}

func (r *Repository) FindDeliveryByID(_ context.Context, eventID string) (*sharedeventlog.Delivery, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.deliveries[eventID]
	if !ok {
		return nil, nil
	}
	return &d, nil
}

// Count returns the number of event_logs rows appended so far. Test-only
// introspection; the real Recorder port has no such method since callers
// address rows by event_id, not by counting them.
func (r *Repository) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.logs)
}
