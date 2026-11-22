package usecases

import (
	"sync"

	datakeysports "github.com/ambi/idmagic/backend/datakeys/internal/ports"
)

// MigratorRegistry maps a name to the ports.FieldMigrator an owning module
// registers at bootstrap (mirrors backend/jobs/usecases.HandlerRegistry).
// RotateTenantDataKey enqueues a data_key_reencryption job per registered
// name, and DestroyTenantDataKey's gate checks every registered migrator's
// PendingCount before erasing a wrapped_dek (wi-97 T006).
type MigratorRegistry struct {
	mu        sync.RWMutex
	migrators map[string]datakeysports.FieldMigrator
}

func NewMigratorRegistry() *MigratorRegistry {
	return &MigratorRegistry{migrators: map[string]datakeysports.FieldMigrator{}}
}

// Register adds m as the FieldMigrator for name, overwriting any previous
// registration under the same name.
func (r *MigratorRegistry) Register(name string, m datakeysports.FieldMigrator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.migrators[name] = m
}

// Lookup returns the FieldMigrator registered for name, or (nil, false).
func (r *MigratorRegistry) Lookup(name string) (datakeysports.FieldMigrator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.migrators[name]
	return m, ok
}

// Names returns every registered migrator name in no particular order:
// callers (Rotate's enqueue loop, Destroy's gate) process each name
// independently.
func (r *MigratorRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.migrators))
	for name := range r.migrators {
		names = append(names, name)
	}
	return names
}
