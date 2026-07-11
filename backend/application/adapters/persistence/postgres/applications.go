package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ambi/idmagic/backend/application/adapters/persistence/postgres/sqlcgen"
	"github.com/ambi/idmagic/backend/application/domain"
	appports "github.com/ambi/idmagic/backend/application/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
)

// ApplicationRepository は ApplicationCatalog の Application aggregate を PostgreSQL に
// 永続化する (wi-69)。protocol binding は JSONB に格納し、参照はテナント境界に閉じる。
// クエリは sqlc 生成 (wi-172, ADR-090); Pool は sqlcgen.DBTX を構造的に満たす。
type ApplicationRepository struct{ Pool sharedpg.DB }

func applicationFromRow(row *sqlcgen.Application) (*domain.Application, error) {
	app := &domain.Application{
		TenantID:      row.TenantID,
		ApplicationID: row.ApplicationID,
		Name:          row.Name,
		Kind:          domain.ApplicationKind(row.Kind),
		Status:        domain.ApplicationStatus(row.Status),
		IconURL:       row.IconUrl,
		IconObjectKey: row.IconObjectKey,
		LaunchURL:     row.LaunchUrl,
		CategoryIDs:   row.CategoryIds,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
	app.Bindings = []domain.ProtocolBinding{}
	if len(row.Bindings) > 0 {
		if err := json.Unmarshal(row.Bindings, &app.Bindings); err != nil {
			return nil, err
		}
	}
	if app.CategoryIDs == nil {
		app.CategoryIDs = []string{}
	}
	return app, nil
}

func (r *ApplicationRepository) ListByTenant(ctx context.Context, tenantID string) ([]*domain.Application, error) {
	rows, err := sqlcgen.New(r.Pool).ListApplicationsByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Application, 0, len(rows))
	for _, row := range rows {
		app, err := applicationFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, app)
	}
	return out, nil
}

func (r *ApplicationRepository) FindByID(ctx context.Context, tenantID, applicationID string) (*domain.Application, error) {
	row, err := sqlcgen.New(r.Pool).GetApplicationByID(ctx, sqlcgen.GetApplicationByIDParams{
		TenantID: tenantID, ApplicationID: applicationID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return applicationFromRow(row)
}

func (r *ApplicationRepository) FindByBinding(ctx context.Context, tenantID string, bindingType domain.ProtocolBindingType, key string) (*domain.Application, error) {
	if key == "" {
		return nil, nil
	}
	apps, err := r.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	for _, app := range apps {
		for _, binding := range app.Bindings {
			if binding.Type != bindingType {
				continue
			}
			switch bindingType {
			case domain.ProtocolBindingWsFed:
				if binding.Wtrealm == key {
					return app, nil
				}
			case domain.ProtocolBindingSAML:
				if binding.EntityID == key {
					return app, nil
				}
			default:
				if binding.ClientID == key {
					return app, nil
				}
			}
		}
	}
	return nil, nil
}

func (r *ApplicationRepository) Save(ctx context.Context, app *domain.Application) error {
	bindings := app.Bindings
	if bindings == nil {
		bindings = []domain.ProtocolBinding{}
	}
	encoded, err := json.Marshal(bindings)
	if err != nil {
		return err
	}
	categoryIDs := app.CategoryIDs
	if categoryIDs == nil {
		categoryIDs = []string{}
	}
	return sqlcgen.New(r.Pool).UpsertApplication(ctx, sqlcgen.UpsertApplicationParams{
		TenantID:      app.TenantID,
		ApplicationID: app.ApplicationID,
		Name:          app.Name,
		Kind:          string(app.Kind),
		Status:        string(app.Status),
		IconUrl:       app.IconURL,
		IconObjectKey: app.IconObjectKey,
		LaunchUrl:     app.LaunchURL,
		Bindings:      encoded,
		CategoryIds:   categoryIDs,
		CreatedAt:     app.CreatedAt,
		UpdatedAt:     app.UpdatedAt,
	})
}

func (r *ApplicationRepository) Delete(ctx context.Context, tenantID, applicationID string) error {
	return sqlcgen.New(r.Pool).DeleteApplication(ctx, sqlcgen.DeleteApplicationParams{
		TenantID: tenantID, ApplicationID: applicationID,
	})
}

func (r *ApplicationRepository) RemoveCategory(ctx context.Context, tenantID, categoryID string) error {
	return sqlcgen.New(r.Pool).RemoveApplicationCategory(ctx, sqlcgen.RemoveApplicationCategoryParams{
		TenantID: tenantID, ArrayRemove: categoryID,
	})
}

// SignInPolicyRepository は Application sign-in policy を PostgreSQL に永続化する (ADR-079)。
type SignInPolicyRepository struct{ Pool sharedpg.DB }

func signInPolicyFromFields(tenantID, applicationID string, rules []byte, createdAt, updatedAt time.Time) (*domain.AppSignInPolicy, error) {
	policy := &domain.AppSignInPolicy{
		TenantID: tenantID, ApplicationID: applicationID,
		CreatedAt: createdAt, UpdatedAt: updatedAt,
		Rules: []domain.SignInRule{},
	}
	if len(rules) > 0 {
		if err := json.Unmarshal(rules, &policy.Rules); err != nil {
			return nil, err
		}
	}
	return policy, nil
}

func (r *SignInPolicyRepository) Get(ctx context.Context, tenantID, applicationID string) (*domain.AppSignInPolicy, error) {
	row, err := sqlcgen.New(r.Pool).GetAppSignInPolicy(ctx, sqlcgen.GetAppSignInPolicyParams{
		TenantID: tenantID, ApplicationID: applicationID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return signInPolicyFromFields(row.TenantID, row.ApplicationID, row.Rules, row.CreatedAt, row.UpdatedAt)
}

func (r *SignInPolicyRepository) ListByTenant(ctx context.Context, tenantID string) ([]*domain.AppSignInPolicy, error) {
	rows, err := sqlcgen.New(r.Pool).ListAppSignInPoliciesByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.AppSignInPolicy, 0, len(rows))
	for _, row := range rows {
		policy, err := signInPolicyFromFields(row.TenantID, row.ApplicationID, row.Rules, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		out = append(out, policy)
	}
	return out, nil
}

func (r *SignInPolicyRepository) Save(ctx context.Context, policy *domain.AppSignInPolicy) error {
	rules := policy.Rules
	if rules == nil {
		rules = []domain.SignInRule{}
	}
	encoded, err := json.Marshal(rules)
	if err != nil {
		return err
	}
	return sqlcgen.New(r.Pool).UpsertAppSignInPolicy(ctx, sqlcgen.UpsertAppSignInPolicyParams{
		ApplicationID: policy.ApplicationID, Rules: encoded,
		CreatedAt: policy.CreatedAt, UpdatedAt: policy.UpdatedAt,
	})
}

func (r *SignInPolicyRepository) Delete(ctx context.Context, tenantID, applicationID string) error {
	return sqlcgen.New(r.Pool).DeleteAppSignInPolicy(ctx, sqlcgen.DeleteAppSignInPolicyParams{
		TenantID: tenantID, ApplicationID: applicationID,
	})
}

// DefaultSignInPolicyRepository はテナント既定 sign-in policy を PostgreSQL に永続化する (ADR-081)。
type DefaultSignInPolicyRepository struct{ Pool sharedpg.DB }

func (r *DefaultSignInPolicyRepository) Get(ctx context.Context, tenantID string) (*domain.TenantDefaultSignInPolicy, error) {
	row, err := sqlcgen.New(r.Pool).GetTenantDefaultSignInPolicy(ctx, tenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	policy := &domain.TenantDefaultSignInPolicy{
		TenantID: row.TenantID, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		Rules: []domain.SignInRule{},
	}
	if len(row.Rules) > 0 {
		if err := json.Unmarshal(row.Rules, &policy.Rules); err != nil {
			return nil, err
		}
	}
	return policy, nil
}

func (r *DefaultSignInPolicyRepository) Save(ctx context.Context, policy *domain.TenantDefaultSignInPolicy) error {
	rules := policy.Rules
	if rules == nil {
		rules = []domain.SignInRule{}
	}
	encoded, err := json.Marshal(rules)
	if err != nil {
		return err
	}
	return sqlcgen.New(r.Pool).UpsertTenantDefaultSignInPolicy(ctx, sqlcgen.UpsertTenantDefaultSignInPolicyParams{
		TenantID: policy.TenantID, Rules: encoded, CreatedAt: policy.CreatedAt, UpdatedAt: policy.UpdatedAt,
	})
}

// ApplicationIconStore は Application icon blob を PostgreSQL に保存する (wi-74, ADR-073)。
type ApplicationIconStore struct{ Pool sharedpg.DB }

func (s *ApplicationIconStore) Save(ctx context.Context, icon *domain.ApplicationIcon) error {
	return sqlcgen.New(s.Pool).UpsertApplicationIcon(ctx, sqlcgen.UpsertApplicationIconParams{
		ApplicationID: icon.ApplicationID, ObjectKey: icon.ObjectKey,
		ContentType: icon.ContentType, SizeBytes: int32(icon.SizeBytes), Data: icon.Data, //nolint:gosec // G115: icon size is bounded by upload limits, well under int32 max
		CreatedAt: icon.CreatedAt, UpdatedAt: icon.UpdatedAt,
	})
}

func (s *ApplicationIconStore) Find(ctx context.Context, tenantID, applicationID, objectKey string) (*domain.ApplicationIcon, error) {
	row, err := sqlcgen.New(s.Pool).GetApplicationIcon(ctx, sqlcgen.GetApplicationIconParams{
		TenantID: tenantID, ApplicationID: applicationID, ObjectKey: objectKey,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.ApplicationIcon{
		TenantID: row.TenantID, ApplicationID: row.ApplicationID, ObjectKey: row.ObjectKey,
		ContentType: row.ContentType, SizeBytes: int(row.SizeBytes), Data: row.Data,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

func (s *ApplicationIconStore) DeleteByApplication(ctx context.Context, tenantID, applicationID string) error {
	return sqlcgen.New(s.Pool).DeleteApplicationIconsByApplication(ctx, sqlcgen.DeleteApplicationIconsByApplicationParams{
		TenantID: tenantID, ApplicationID: applicationID,
	})
}

// ApplicationAssignmentRepository は Application 割当を PostgreSQL に永続化する (wi-69)。
type ApplicationAssignmentRepository struct{ Pool sharedpg.DB }

func assignmentFromFields(tenantID, applicationID, subjectType, subjectID, visibility string, createdAt, updatedAt time.Time) *domain.ApplicationAssignment {
	return &domain.ApplicationAssignment{
		TenantID: tenantID, ApplicationID: applicationID, SubjectType: domain.AssignmentSubjectType(subjectType),
		SubjectID: subjectID, Visibility: domain.AssignmentVisibility(visibility), CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
}

func (r *ApplicationAssignmentRepository) ListByTenant(ctx context.Context, tenantID string) ([]*domain.ApplicationAssignment, error) {
	rows, err := sqlcgen.New(r.Pool).ListApplicationAssignmentsByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.ApplicationAssignment, 0, len(rows))
	for _, row := range rows {
		out = append(out, assignmentFromFields(row.TenantID, row.ApplicationID, row.SubjectType, row.SubjectID, row.Visibility, row.CreatedAt, row.UpdatedAt))
	}
	return out, nil
}

func (r *ApplicationAssignmentRepository) ListByApplication(ctx context.Context, tenantID, applicationID string) ([]*domain.ApplicationAssignment, error) {
	rows, err := sqlcgen.New(r.Pool).ListApplicationAssignmentsByApplication(ctx, sqlcgen.ListApplicationAssignmentsByApplicationParams{
		TenantID: tenantID, ApplicationID: applicationID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*domain.ApplicationAssignment, 0, len(rows))
	for _, row := range rows {
		out = append(out, assignmentFromFields(row.TenantID, row.ApplicationID, row.SubjectType, row.SubjectID, row.Visibility, row.CreatedAt, row.UpdatedAt))
	}
	return out, nil
}

// ListBySubjects は (subject_type, subject_id) ペア配列との UNNEST 突き合わせが必要で、
// sqlc の静的解析が UNNEST の引数型を解決できないため手書き pgx のままとする
// (動的クエリのエスケープハッチ、ADR-090)。
func (r *ApplicationAssignmentRepository) ListBySubjects(ctx context.Context, tenantID string, subjects []appports.SubjectRef) ([]*domain.ApplicationAssignment, error) {
	if len(subjects) == 0 {
		return []*domain.ApplicationAssignment{}, nil
	}
	types := make([]string, len(subjects))
	ids := make([]string, len(subjects))
	for i, s := range subjects {
		types[i] = string(s.Type)
		ids[i] = s.ID
	}
	// (subject_type, subject_id) のペアを UNNEST で突き合わせる。subject_id は UUID 列の
	// ため、パラメータは text[] のまま列側を text にキャストして比較する (ADR-084)。
	const assignmentSelect = `SELECT a.tenant_id,aa.application_id,aa.subject_type,aa.subject_id,aa.visibility,aa.created_at,aa.updated_at FROM application_assignments aa JOIN applications a ON a.application_id=aa.application_id`
	rows, err := r.Pool.Query(ctx, assignmentSelect+`
 WHERE a.tenant_id=$1 AND (aa.subject_type,aa.subject_id::text) IN (
   SELECT subject_type, subject_id FROM UNNEST($2::text[], $3::text[]) AS s(subject_type, subject_id)
 )`, tenantID, types, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.ApplicationAssignment{}
	for rows.Next() {
		var a domain.ApplicationAssignment
		if err := rows.Scan(&a.TenantID, &a.ApplicationID, &a.SubjectType, &a.SubjectID, &a.Visibility, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

func (r *ApplicationAssignmentRepository) Save(ctx context.Context, a *domain.ApplicationAssignment) error {
	return sqlcgen.New(r.Pool).UpsertApplicationAssignment(ctx, sqlcgen.UpsertApplicationAssignmentParams{
		ApplicationID: a.ApplicationID, SubjectType: string(a.SubjectType),
		SubjectID: a.SubjectID, Visibility: string(a.Visibility), CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	})
}

func (r *ApplicationAssignmentRepository) Delete(ctx context.Context, tenantID, applicationID string, subjectType domain.AssignmentSubjectType, subjectID string) error {
	return sqlcgen.New(r.Pool).DeleteApplicationAssignment(ctx, sqlcgen.DeleteApplicationAssignmentParams{
		TenantID: tenantID, ApplicationID: applicationID, SubjectType: string(subjectType), SubjectID: subjectID,
	})
}

func (r *ApplicationAssignmentRepository) DeleteByApplication(ctx context.Context, tenantID, applicationID string) error {
	return sqlcgen.New(r.Pool).DeleteApplicationAssignmentsByApplication(ctx, sqlcgen.DeleteApplicationAssignmentsByApplicationParams{
		TenantID: tenantID, ApplicationID: applicationID,
	})
}

// ApplicationOrderingRepository は利用者ごとのポータル手動並び順を PostgreSQL に永続化する
// (wi-70, ADR-069)。application_ids は順序を保つ text[] で格納する。user_id は global unique
// なため tenant_id 列は持たず、行は user_id で一意に識別する (ADR-082)。tenantID 引数は port
// 契約の互換のために残すが SQL では用いない。
type ApplicationOrderingRepository struct{ Pool sharedpg.DB }

func (r *ApplicationOrderingRepository) Get(ctx context.Context, _ /*tenantID*/, userID string) (*domain.ApplicationOrdering, error) {
	row, err := sqlcgen.New(r.Pool).GetApplicationOrdering(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.ApplicationOrdering{
		UserID: row.UserID, ApplicationIDs: row.ApplicationIds,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *ApplicationOrderingRepository) Save(ctx context.Context, o *domain.ApplicationOrdering) error {
	ids := o.ApplicationIDs
	if ids == nil {
		ids = []string{}
	}
	return sqlcgen.New(r.Pool).UpsertApplicationOrdering(ctx, sqlcgen.UpsertApplicationOrderingParams{
		UserID: o.UserID, ApplicationIds: ids, CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt,
	})
}

// ApplicationCategoryRepository は ApplicationCategory を PostgreSQL に永続化する (wi-70, ADR-069)。
// すべてテナント境界に閉じる。
type ApplicationCategoryRepository struct{ Pool sharedpg.DB }

func categoryFromRow(row *sqlcgen.ApplicationCategory) *domain.ApplicationCategory {
	return &domain.ApplicationCategory{
		TenantID: row.TenantID, CategoryID: row.CategoryID, Name: row.Name,
		Position: int(row.Position), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func (r *ApplicationCategoryRepository) ListByTenant(ctx context.Context, tenantID string) ([]*domain.ApplicationCategory, error) {
	rows, err := sqlcgen.New(r.Pool).ListApplicationCategoriesByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.ApplicationCategory, 0, len(rows))
	for _, row := range rows {
		out = append(out, categoryFromRow(row))
	}
	return out, nil
}

func (r *ApplicationCategoryRepository) FindByID(ctx context.Context, tenantID, categoryID string) (*domain.ApplicationCategory, error) {
	row, err := sqlcgen.New(r.Pool).GetApplicationCategoryByID(ctx, sqlcgen.GetApplicationCategoryByIDParams{
		TenantID: tenantID, CategoryID: categoryID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return categoryFromRow(row), nil
}

func (r *ApplicationCategoryRepository) Save(ctx context.Context, c *domain.ApplicationCategory) error {
	return sqlcgen.New(r.Pool).UpsertApplicationCategory(ctx, sqlcgen.UpsertApplicationCategoryParams{
		TenantID: c.TenantID, CategoryID: c.CategoryID, Name: c.Name, Position: int32(c.Position), //nolint:gosec // G115: category position is a small manual-ordering index, well under int32 max
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	})
}

func (r *ApplicationCategoryRepository) Delete(ctx context.Context, tenantID, categoryID string) error {
	return sqlcgen.New(r.Pool).DeleteApplicationCategory(ctx, sqlcgen.DeleteApplicationCategoryParams{
		TenantID: tenantID, CategoryID: categoryID,
	})
}
