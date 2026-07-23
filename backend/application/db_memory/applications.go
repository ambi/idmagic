package db_memory

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/ambi/idmagic/backend/application/domain"
	"github.com/ambi/idmagic/backend/application/ports"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"
)

// =====================================================================
// ApplicationRepository (wi-69, ADR-064)
// =====================================================================

type ApplicationRepository struct {
	mu           sync.RWMutex
	applications map[string]*domain.Application // key: sharedmem.TenantKey(tenant_id, application_id)
}

func NewApplicationRepository() *ApplicationRepository {
	return &ApplicationRepository{applications: map[string]*domain.Application{}}
}

func cloneApplication(app *domain.Application) *domain.Application {
	cloned := *app
	if app.Protocol != nil {
		protocol := *app.Protocol
		cloned.Protocol = &protocol
	}
	cloned.CategoryIDs = slices.Clone(app.CategoryIDs)
	return &cloned
}

func cloneSignInPolicy(policy *domain.AppSignInPolicy) *domain.AppSignInPolicy {
	cloned := *policy
	cloned.Rules = slices.Clone(policy.Rules)
	return &cloned
}

// =====================================================================
// ApplicationIconStore (wi-74, ADR-073)
// =====================================================================

type ApplicationIconStore struct {
	mu    sync.RWMutex
	icons map[string]*domain.ApplicationIcon // key: tenant_id + application_id + object_key
}

func NewApplicationIconStore() *ApplicationIconStore {
	return &ApplicationIconStore{icons: map[string]*domain.ApplicationIcon{}}
}

func iconKey(tenantID, applicationID, objectKey string) string {
	return strings.Join([]string{tenantID, applicationID, objectKey}, "\x00")
}

func cloneIcon(icon *domain.ApplicationIcon) *domain.ApplicationIcon {
	cloned := *icon
	cloned.Data = slices.Clone(icon.Data)
	return &cloned
}

func (s *ApplicationIconStore) Save(_ context.Context, icon *domain.ApplicationIcon) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := iconKey(icon.TenantID, icon.ApplicationID, icon.ObjectKey)
	cloned := cloneIcon(icon)
	if existing := s.icons[key]; existing != nil && !existing.CreatedAt.IsZero() {
		cloned.CreatedAt = existing.CreatedAt
	}
	s.icons[key] = cloned
	return nil
}

func (s *ApplicationIconStore) Find(_ context.Context, tenantID, applicationID, objectKey string) (*domain.ApplicationIcon, error) {
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

func (r *ApplicationRepository) ListByTenant(_ context.Context, tenantID string) ([]*domain.Application, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.Application, 0)
	for _, app := range r.applications {
		if app.TenantID == tenantID {
			out = append(out, cloneApplication(app))
		}
	}
	slices.SortFunc(out, func(a, b *domain.Application) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

func (r *ApplicationRepository) FindByID(_ context.Context, tenantID, applicationID string) (*domain.Application, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	app := r.applications[sharedmem.TenantKey(tenantID, applicationID)]
	if app == nil {
		return nil, nil
	}
	return cloneApplication(app), nil
}

func protocolKey(protocol domain.ApplicationProtocol) string {
	switch protocol.Type {
	case domain.ApplicationProtocolOIDC:
		return protocol.ClientID
	case domain.ApplicationProtocolWsFed:
		return protocol.Wtrealm
	case domain.ApplicationProtocolSAML:
		return protocol.EntityID
	default:
		return ""
	}
}

func (r *ApplicationRepository) FindByProtocol(_ context.Context, tenantID string, protocolType domain.ApplicationProtocolType, key string) (*domain.Application, error) {
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

func (r *ApplicationRepository) Save(_ context.Context, app *domain.Application) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.applications[sharedmem.TenantKey(app.TenantID, app.ApplicationID)] = cloneApplication(app)
	return nil
}

func (r *ApplicationRepository) Create(ctx context.Context, app *domain.Application) error {
	return r.Save(ctx, app)
}

func (r *ApplicationRepository) Delete(_ context.Context, tenantID, applicationID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.applications, sharedmem.TenantKey(tenantID, applicationID))
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
// SignInPolicyRepository (wi-71, ADR-079)
// =====================================================================

type SignInPolicyRepository struct {
	mu       sync.RWMutex
	policies map[string]*domain.AppSignInPolicy // key: sharedmem.TenantKey(tenant_id, application_id)
}

func NewSignInPolicyRepository() *SignInPolicyRepository {
	return &SignInPolicyRepository{policies: map[string]*domain.AppSignInPolicy{}}
}

func (r *SignInPolicyRepository) Get(_ context.Context, tenantID, applicationID string) (*domain.AppSignInPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	policy := r.policies[sharedmem.TenantKey(tenantID, applicationID)]
	if policy == nil {
		return nil, nil
	}
	return cloneSignInPolicy(policy), nil
}

func (r *SignInPolicyRepository) ListByTenant(_ context.Context, tenantID string) ([]*domain.AppSignInPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.AppSignInPolicy, 0)
	for _, policy := range r.policies {
		if policy.TenantID == tenantID {
			out = append(out, cloneSignInPolicy(policy))
		}
	}
	return out, nil
}

func (r *SignInPolicyRepository) Save(_ context.Context, policy *domain.AppSignInPolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := sharedmem.TenantKey(policy.TenantID, policy.ApplicationID)
	cloned := cloneSignInPolicy(policy)
	if existing := r.policies[key]; existing != nil && !existing.CreatedAt.IsZero() {
		cloned.CreatedAt = existing.CreatedAt
	}
	r.policies[key] = cloned
	return nil
}

func (r *SignInPolicyRepository) Delete(_ context.Context, tenantID, applicationID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.policies, sharedmem.TenantKey(tenantID, applicationID))
	return nil
}

// =====================================================================
// DefaultSignInPolicyRepository (wi-115, ADR-081)
// =====================================================================

func cloneDefaultSignInPolicy(policy *domain.TenantDefaultSignInPolicy) *domain.TenantDefaultSignInPolicy {
	cloned := *policy
	cloned.Rules = slices.Clone(policy.Rules)
	return &cloned
}

type DefaultSignInPolicyRepository struct {
	mu       sync.RWMutex
	policies map[string]*domain.TenantDefaultSignInPolicy // key: tenant_id
}

func NewDefaultSignInPolicyRepository() *DefaultSignInPolicyRepository {
	return &DefaultSignInPolicyRepository{policies: map[string]*domain.TenantDefaultSignInPolicy{}}
}

func (r *DefaultSignInPolicyRepository) Get(_ context.Context, tenantID string) (*domain.TenantDefaultSignInPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	policy := r.policies[tenantID]
	if policy == nil {
		return nil, nil
	}
	return cloneDefaultSignInPolicy(policy), nil
}

func (r *DefaultSignInPolicyRepository) Save(_ context.Context, policy *domain.TenantDefaultSignInPolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cloned := cloneDefaultSignInPolicy(policy)
	if existing := r.policies[policy.TenantID]; existing != nil && !existing.CreatedAt.IsZero() {
		cloned.CreatedAt = existing.CreatedAt
	}
	r.policies[policy.TenantID] = cloned
	return nil
}

// =====================================================================
// AssignmentRepository (wi-69)
// =====================================================================

type ApplicationAssignmentRepository struct {
	mu          sync.RWMutex
	assignments map[string]*domain.ApplicationAssignment // key: assignmentKey(...)
}

func NewApplicationAssignmentRepository() *ApplicationAssignmentRepository {
	return &ApplicationAssignmentRepository{assignments: map[string]*domain.ApplicationAssignment{}}
}

func assignmentKey(tenantID, applicationID string, subjectType domain.AssignmentSubjectType, subjectID string) string {
	return strings.Join([]string{tenantID, applicationID, string(subjectType), subjectID}, "\x00")
}

func (r *ApplicationAssignmentRepository) ListByApplication(_ context.Context, tenantID, applicationID string) ([]*domain.ApplicationAssignment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.ApplicationAssignment, 0)
	for _, assignment := range r.assignments {
		if assignment.TenantID == tenantID && assignment.ApplicationID == applicationID {
			cloned := *assignment
			out = append(out, &cloned)
		}
	}
	slices.SortFunc(out, func(a, b *domain.ApplicationAssignment) int {
		if c := strings.Compare(string(a.SubjectType), string(b.SubjectType)); c != 0 {
			return c
		}
		return strings.Compare(a.SubjectID, b.SubjectID)
	})
	return out, nil
}

func (r *ApplicationAssignmentRepository) ListByTenant(_ context.Context, tenantID string) ([]*domain.ApplicationAssignment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.ApplicationAssignment, 0)
	for _, assignment := range r.assignments {
		if assignment.TenantID == tenantID {
			cloned := *assignment
			out = append(out, &cloned)
		}
	}
	return out, nil
}

func (r *ApplicationAssignmentRepository) ListBySubjects(_ context.Context, tenantID string, subjects []ports.SubjectRef) ([]*domain.ApplicationAssignment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.ApplicationAssignment, 0)
	for _, assignment := range r.assignments {
		if assignment.TenantID != tenantID {
			continue
		}
		if slices.ContainsFunc(subjects, func(s ports.SubjectRef) bool {
			return s.Type == assignment.SubjectType && s.ID == assignment.SubjectID
		}) {
			cloned := *assignment
			out = append(out, &cloned)
		}
	}
	return out, nil
}

func (r *ApplicationAssignmentRepository) Save(_ context.Context, assignment *domain.ApplicationAssignment) error {
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

func (r *ApplicationAssignmentRepository) Delete(_ context.Context, tenantID, applicationID string, subjectType domain.AssignmentSubjectType, subjectID string) error {
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
// ApplicationOrderingRepository (wi-70, ADR-069)
// =====================================================================

type ApplicationOrderingRepository struct {
	mu        sync.RWMutex
	orderings map[string]*domain.ApplicationOrdering // key: user_id (global unique)
}

func NewApplicationOrderingRepository() *ApplicationOrderingRepository {
	return &ApplicationOrderingRepository{orderings: map[string]*domain.ApplicationOrdering{}}
}

func cloneOrdering(o *domain.ApplicationOrdering) *domain.ApplicationOrdering {
	cloned := *o
	cloned.ApplicationIDs = slices.Clone(o.ApplicationIDs)
	return &cloned
}

func (r *ApplicationOrderingRepository) Get(_ context.Context, _ /*tenantID*/, userID string) (*domain.ApplicationOrdering, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o := r.orderings[userID]
	if o == nil {
		return nil, nil
	}
	return cloneOrdering(o), nil
}

func (r *ApplicationOrderingRepository) Save(_ context.Context, ordering *domain.ApplicationOrdering) error {
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
// ApplicationCategoryRepository (wi-70, ADR-069)
// =====================================================================

type ApplicationCategoryRepository struct {
	mu         sync.RWMutex
	categories map[string]*domain.ApplicationCategory // key: sharedmem.TenantKey(tenant_id, category_id)
}

func NewApplicationCategoryRepository() *ApplicationCategoryRepository {
	return &ApplicationCategoryRepository{categories: map[string]*domain.ApplicationCategory{}}
}

func (r *ApplicationCategoryRepository) ListByTenant(_ context.Context, tenantID string) ([]*domain.ApplicationCategory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.ApplicationCategory, 0)
	for _, category := range r.categories {
		if category.TenantID == tenantID {
			cloned := *category
			out = append(out, &cloned)
		}
	}
	slices.SortFunc(out, func(a, b *domain.ApplicationCategory) int {
		if a.Position != b.Position {
			return a.Position - b.Position
		}
		return strings.Compare(a.Name, b.Name)
	})
	return out, nil
}

func (r *ApplicationCategoryRepository) FindByID(_ context.Context, tenantID, categoryID string) (*domain.ApplicationCategory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	category := r.categories[sharedmem.TenantKey(tenantID, categoryID)]
	if category == nil {
		return nil, nil
	}
	cloned := *category
	return &cloned, nil
}

func (r *ApplicationCategoryRepository) Save(_ context.Context, category *domain.ApplicationCategory) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cloned := *category
	r.categories[sharedmem.TenantKey(category.TenantID, category.CategoryID)] = &cloned
	return nil
}

func (r *ApplicationCategoryRepository) Delete(_ context.Context, tenantID, categoryID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.categories, sharedmem.TenantKey(tenantID, categoryID))
	return nil
}
