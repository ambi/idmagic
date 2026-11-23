package db_memory

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/ambi/idmagic/backend/shared/kernel"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appports "github.com/ambi/idmagic/backend/application/ports"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"
)

// =====================================================================
// ApplicationRepository (wi-69)
// =====================================================================

type ApplicationRepository struct {
	mu           sync.RWMutex
	applications map[string]*appdomain.Application // key: sharedmem.TenantKey(tenant_id, application_id)
}

func NewApplicationRepository() *ApplicationRepository {
	return &ApplicationRepository{applications: map[string]*appdomain.Application{}}
}

func cloneApplication(app *appdomain.Application) *appdomain.Application {
	cloned := *app
	if app.Protocol != nil {
		protocol := *app.Protocol
		cloned.Protocol = &protocol
	}
	cloned.CategoryIDs = slices.Clone(app.CategoryIDs)
	return &cloned
}

// =====================================================================
// ApplicationIconStore (wi-74)
// =====================================================================

type ApplicationIconStore struct {
	mu    sync.RWMutex
	icons map[string]*appdomain.ApplicationIcon // key: tenant_id + application_id + object_key
}

func NewApplicationIconStore() *ApplicationIconStore {
	return &ApplicationIconStore{icons: map[string]*appdomain.ApplicationIcon{}}
}

func iconKey(tenantID, applicationID, objectKey string) string {
	return strings.Join([]string{tenantID, applicationID, objectKey}, "\x00")
}

func cloneIcon(icon *appdomain.ApplicationIcon) *appdomain.ApplicationIcon {
	cloned := *icon
	cloned.Data = slices.Clone(icon.Data)
	return &cloned
}

func (s *ApplicationIconStore) Save(_ context.Context, icon *appdomain.ApplicationIcon) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := iconKey(icon.TenantID, icon.ApplicationID, icon.ID)
	cloned := cloneIcon(icon)
	if existing := s.icons[key]; existing != nil && !existing.CreatedAt.IsZero() {
		cloned.CreatedAt = existing.CreatedAt
	}
	s.icons[key] = cloned
	return nil
}

func (s *ApplicationIconStore) Find(_ context.Context, tenantID, applicationID, objectKey string) (*appdomain.ApplicationIcon, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	icon := s.icons[iconKey(tenantID, applicationID, objectKey)]
	if icon == nil {
		return nil, nil
	}
	return cloneIcon(icon), nil
}

func (s *ApplicationIconStore) DeleteByApplication(_ context.Context, tenantID, applicationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	prefix := tenantID + "\x00" + applicationID + "\x00"
	for key := range s.icons {
		if strings.HasPrefix(key, prefix) {
			delete(s.icons, key)
		}
	}
	return nil
}

func (r *ApplicationRepository) ListAll(_ context.Context, tenantID string) ([]*appdomain.Application, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*appdomain.Application, 0)
	for _, app := range r.applications {
		if app.TenantID == tenantID {
			out = append(out, cloneApplication(app))
		}
	}
	slices.SortFunc(out, func(a, b *appdomain.Application) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

// ListPage implements ports.ApplicationRepository.ListPage (wi-159):
// keyset pagination ordered by (Name, ID) ascending, strictly after the
// given keyset.
func (r *ApplicationRepository) ListPage(_ context.Context, tenantID, afterName, afterID string, limit int) ([]*appdomain.Application, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*appdomain.Application, 0)
	for _, app := range r.applications {
		if app.TenantID == tenantID {
			out = append(out, cloneApplication(app))
		}
	}
	key := func(a *appdomain.Application) (string, string) { return a.Name, a.ID }
	return sharedmem.KeysetPage(out, key, false, afterName, afterID, limit), nil
}

func (r *ApplicationRepository) ListPageBefore(_ context.Context, tenantID, beforeName, beforeID string, limit int) ([]*appdomain.Application, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*appdomain.Application, 0)
	for _, app := range r.applications {
		if app.TenantID == tenantID {
			out = append(out, cloneApplication(app))
		}
	}
	key := func(a *appdomain.Application) (string, string) { return a.Name, a.ID }
	return sharedmem.KeysetPageBefore(out, key, false, beforeName, beforeID, limit), nil
}

func (r *ApplicationRepository) Count(_ context.Context, tenantID string) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var count int64
	for _, app := range r.applications {
		if app.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

func (r *ApplicationRepository) FindByID(_ context.Context, tenantID, applicationID string) (*appdomain.Application, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	app := r.applications[sharedmem.ScopedKey(tenantID, applicationID, kernel.DefaultTenantID)]
	if app == nil {
		return nil, nil
	}
	return cloneApplication(app), nil
}

func protocolKey(protocol appdomain.ApplicationProtocol) string {
	switch protocol.Type {
	case appdomain.ApplicationProtocolOIDC:
		return protocol.ClientID
	case appdomain.ApplicationProtocolWsFed:
		return protocol.Wtrealm
	case appdomain.ApplicationProtocolSAML:
		return protocol.EntityID
	default:
		return ""
	}
}

func (r *ApplicationRepository) FindByProtocol(_ context.Context, tenantID string, protocolType appdomain.ApplicationProtocolType, key string) (*appdomain.Application, error) {
	if key == "" {
		return nil, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, app := range r.applications {
		if app.TenantID != tenantID {
			continue
		}
		if app.Protocol != nil && app.Protocol.Type == protocolType && protocolKey(*app.Protocol) == key {
			return cloneApplication(app), nil
		}
	}
	return nil, nil
}

func (r *ApplicationRepository) Save(_ context.Context, app *appdomain.Application) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.applications[sharedmem.ScopedKey(app.TenantID, app.ID, kernel.DefaultTenantID)] = cloneApplication(app)
	return nil
}

func (r *ApplicationRepository) Create(ctx context.Context, app *appdomain.Application) error {
	return r.Save(ctx, app)
}

func (r *ApplicationRepository) Update(_ context.Context, tenantID, applicationID string, change func(*appdomain.Application) (*appdomain.Application, error)) (*appdomain.Application, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := sharedmem.ScopedKey(tenantID, applicationID, kernel.DefaultTenantID)
	var current *appdomain.Application
	if stored := r.applications[key]; stored != nil {
		current = cloneApplication(stored)
	}
	updated, err := change(current)
	if err != nil {
		return nil, err
	}
	r.applications[key] = cloneApplication(updated)
	return cloneApplication(updated), nil
}

func (r *ApplicationRepository) Delete(_ context.Context, tenantID, applicationID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.applications, sharedmem.ScopedKey(tenantID, applicationID, kernel.DefaultTenantID))
	return nil
}

func (r *ApplicationRepository) RemoveCategory(_ context.Context, tenantID, categoryID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, app := range r.applications {
		if app.TenantID != tenantID {
			continue
		}
		app.CategoryIDs = slices.DeleteFunc(app.CategoryIDs, func(id string) bool { return id == categoryID })
	}
	return nil
}

// =====================================================================
// AssignmentRepository (wi-69)
// =====================================================================

type ApplicationAssignmentRepository struct {
	mu          sync.RWMutex
	assignments map[string]*appdomain.ApplicationAssignment // key: assignmentKey(...)
}

func NewApplicationAssignmentRepository() *ApplicationAssignmentRepository {
	return &ApplicationAssignmentRepository{assignments: map[string]*appdomain.ApplicationAssignment{}}
}

func assignmentKey(tenantID, applicationID string, subjectType appdomain.AssignmentSubjectType, subjectID string) string {
	return strings.Join([]string{tenantID, applicationID, string(subjectType), subjectID}, "\x00")
}

func (r *ApplicationAssignmentRepository) ListByApplication(_ context.Context, tenantID, applicationID string) ([]*appdomain.ApplicationAssignment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*appdomain.ApplicationAssignment, 0)
	for _, assignment := range r.assignments {
		if assignment.TenantID == tenantID && assignment.ApplicationID == applicationID {
			cloned := *assignment
			out = append(out, &cloned)
		}
	}
	slices.SortFunc(out, func(a, b *appdomain.ApplicationAssignment) int {
		if c := strings.Compare(string(a.SubjectType), string(b.SubjectType)); c != 0 {
			return c
		}
		return strings.Compare(a.SubjectID, b.SubjectID)
	})
	return out, nil
}

// ListPageByApplication implements ports.AssignmentRepository.ListPageByApplication
// (wi-159): keyset pagination ordered by (SubjectType, SubjectID)
// ascending, strictly after the given keyset.
func (r *ApplicationAssignmentRepository) ListPageByApplication(_ context.Context, tenantID, applicationID, afterSubjectType, afterSubjectID string, limit int) ([]*appdomain.ApplicationAssignment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*appdomain.ApplicationAssignment, 0)
	for _, assignment := range r.assignments {
		if assignment.TenantID == tenantID && assignment.ApplicationID == applicationID {
			cloned := *assignment
			out = append(out, &cloned)
		}
	}
	key := func(a *appdomain.ApplicationAssignment) (string, string) { return string(a.SubjectType), a.SubjectID }
	return sharedmem.KeysetPage(out, key, false, afterSubjectType, afterSubjectID, limit), nil
}

func (r *ApplicationAssignmentRepository) ListPageBeforeByApplication(_ context.Context, tenantID, applicationID, beforeSubjectType, beforeSubjectID string, limit int) ([]*appdomain.ApplicationAssignment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*appdomain.ApplicationAssignment, 0)
	for _, assignment := range r.assignments {
		if assignment.TenantID == tenantID && assignment.ApplicationID == applicationID {
			cloned := *assignment
			out = append(out, &cloned)
		}
	}
	key := func(a *appdomain.ApplicationAssignment) (string, string) { return string(a.SubjectType), a.SubjectID }
	return sharedmem.KeysetPageBefore(out, key, false, beforeSubjectType, beforeSubjectID, limit), nil
}

func (r *ApplicationAssignmentRepository) ListAll(_ context.Context, tenantID string) ([]*appdomain.ApplicationAssignment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*appdomain.ApplicationAssignment, 0)
	for _, assignment := range r.assignments {
		if assignment.TenantID == tenantID {
			cloned := *assignment
			out = append(out, &cloned)
		}
	}
	return out, nil
}

func (r *ApplicationAssignmentRepository) ListBySubjects(_ context.Context, tenantID string, subjects []appports.SubjectRef) ([]*appdomain.ApplicationAssignment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*appdomain.ApplicationAssignment, 0)
	for _, assignment := range r.assignments {
		if assignment.TenantID != tenantID {
			continue
		}
		if slices.ContainsFunc(subjects, func(s appports.SubjectRef) bool {
			return s.Type == assignment.SubjectType && s.ID == assignment.SubjectID
		}) {
			cloned := *assignment
			out = append(out, &cloned)
		}
	}
	return out, nil
}

func (r *ApplicationAssignmentRepository) Save(_ context.Context, assignment *appdomain.ApplicationAssignment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := assignmentKey(assignment.TenantID, assignment.ApplicationID, assignment.SubjectType, assignment.SubjectID)
	cloned := *assignment
	if existing := r.assignments[key]; existing != nil && !existing.CreatedAt.IsZero() {
		cloned.CreatedAt = existing.CreatedAt
	}
	r.assignments[key] = &cloned
	return nil
}

func (r *ApplicationAssignmentRepository) Delete(_ context.Context, tenantID, applicationID string, subjectType appdomain.AssignmentSubjectType, subjectID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.assignments, assignmentKey(tenantID, applicationID, subjectType, subjectID))
	return nil
}

func (r *ApplicationAssignmentRepository) DeleteByApplication(_ context.Context, tenantID, applicationID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, assignment := range r.assignments {
		if assignment.TenantID == tenantID && assignment.ApplicationID == applicationID {
			delete(r.assignments, key)
		}
	}
	return nil
}

// =====================================================================
// ApplicationOrderingRepository (wi-70)
// =====================================================================

type ApplicationOrderingRepository struct {
	mu        sync.RWMutex
	orderings map[string]*appdomain.ApplicationOrdering // key: user_id (global unique)
}

func NewApplicationOrderingRepository() *ApplicationOrderingRepository {
	return &ApplicationOrderingRepository{orderings: map[string]*appdomain.ApplicationOrdering{}}
}

func cloneOrdering(o *appdomain.ApplicationOrdering) *appdomain.ApplicationOrdering {
	cloned := *o
	cloned.ApplicationIDs = slices.Clone(o.ApplicationIDs)
	return &cloned
}

func (r *ApplicationOrderingRepository) Get(_ context.Context, _ /*tenantID*/, userID string) (*appdomain.ApplicationOrdering, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o := r.orderings[userID]
	if o == nil {
		return nil, nil
	}
	return cloneOrdering(o), nil
}

func (r *ApplicationOrderingRepository) Save(_ context.Context, ordering *appdomain.ApplicationOrdering) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := ordering.UserID
	cloned := cloneOrdering(ordering)
	if existing := r.orderings[key]; existing != nil && !existing.CreatedAt.IsZero() {
		cloned.CreatedAt = existing.CreatedAt
	}
	r.orderings[key] = cloned
	return nil
}

// =====================================================================
// ApplicationCategoryRepository (wi-70)
// =====================================================================

type ApplicationCategoryRepository struct {
	mu         sync.RWMutex
	categories map[string]*appdomain.ApplicationCategory // key: sharedmem.TenantKey(tenant_id, category_id)
}

func NewApplicationCategoryRepository() *ApplicationCategoryRepository {
	return &ApplicationCategoryRepository{categories: map[string]*appdomain.ApplicationCategory{}}
}

func (r *ApplicationCategoryRepository) ListAll(_ context.Context, tenantID string) ([]*appdomain.ApplicationCategory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*appdomain.ApplicationCategory, 0)
	for _, category := range r.categories {
		if category.TenantID == tenantID {
			cloned := *category
			out = append(out, &cloned)
		}
	}
	slices.SortFunc(out, func(a, b *appdomain.ApplicationCategory) int {
		if a.Position != b.Position {
			return a.Position - b.Position
		}
		return strings.Compare(a.Name, b.Name)
	})
	return out, nil
}

func (r *ApplicationCategoryRepository) FindByID(_ context.Context, tenantID, categoryID string) (*appdomain.ApplicationCategory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	category := r.categories[sharedmem.ScopedKey(tenantID, categoryID, kernel.DefaultTenantID)]
	if category == nil {
		return nil, nil
	}
	cloned := *category
	return &cloned, nil
}

func (r *ApplicationCategoryRepository) Save(_ context.Context, category *appdomain.ApplicationCategory) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cloned := *category
	r.categories[sharedmem.ScopedKey(category.TenantID, category.ID, kernel.DefaultTenantID)] = &cloned
	return nil
}

func (r *ApplicationCategoryRepository) Delete(_ context.Context, tenantID, categoryID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.categories, sharedmem.ScopedKey(tenantID, categoryID, kernel.DefaultTenantID))
	return nil
}
