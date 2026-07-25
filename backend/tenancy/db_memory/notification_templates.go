package db_memory

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"sync"

	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
)

// NotificationTemplateRepository (wi-288, ADR-142)。key は
// tenant_id + template_key + locale で、postgres 側の PK と同じ粒度。
type NotificationTemplateRepository struct {
	mu        sync.RWMutex
	overrides map[string]*notificationports.TemplateOverride
}

func NewNotificationTemplateRepository() *NotificationTemplateRepository {
	return &NotificationTemplateRepository{overrides: map[string]*notificationports.TemplateOverride{}}
}

func notificationTemplateKey(tenantID string, key notificationports.TemplateKey, locale string) string {
	return strings.Join([]string{tenantID, string(key), locale}, "\x00")
}

func (r *NotificationTemplateRepository) FindByKey(
	_ context.Context, tenantID string, key notificationports.TemplateKey, locale string,
) (*notificationports.TemplateOverride, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	stored := r.overrides[notificationTemplateKey(tenantID, key, locale)]
	if stored == nil {
		return nil, nil
	}
	cloned := *stored
	return &cloned, nil
}

func (r *NotificationTemplateRepository) ListByTenant(
	_ context.Context, tenantID string,
) ([]*notificationports.TemplateOverride, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*notificationports.TemplateOverride{}
	for _, stored := range r.overrides {
		if stored.TenantID != tenantID {
			continue
		}
		cloned := *stored
		out = append(out, &cloned)
	}
	slices.SortFunc(out, func(a, b *notificationports.TemplateOverride) int {
		if byKey := cmp.Compare(string(a.Key), string(b.Key)); byKey != 0 {
			return byKey
		}
		return cmp.Compare(a.Locale, b.Locale)
	})
	return out, nil
}

func (r *NotificationTemplateRepository) Save(_ context.Context, override *notificationports.TemplateOverride) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	storageKey := notificationTemplateKey(override.TenantID, override.Key, override.Locale)
	cloned := *override
	if existing := r.overrides[storageKey]; existing != nil && !existing.CreatedAt.IsZero() {
		cloned.CreatedAt = existing.CreatedAt
	}
	r.overrides[storageKey] = &cloned
	return nil
}

func (r *NotificationTemplateRepository) Delete(
	_ context.Context, tenantID string, key notificationports.TemplateKey, locale string,
) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	storageKey := notificationTemplateKey(tenantID, key, locale)
	if _, ok := r.overrides[storageKey]; !ok {
		return false, nil
	}
	delete(r.overrides, storageKey)
	return true, nil
}
