package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	appports "github.com/ambi/idmagic/internal/application/ports"
	"github.com/ambi/idmagic/internal/shared/spec"
)

// ApplicationRepository は ApplicationCatalog の Application aggregate を PostgreSQL に
// 永続化する (wi-69)。protocol binding は JSONB に格納し、参照はテナント境界に閉じる。
type ApplicationRepository struct{ Pool DB }

const applicationSelect = `SELECT tenant_id,application_id,name,kind,status,icon_url,icon_object_key,launch_url,bindings,category_ids,created_at,updated_at FROM applications`

func scanApplication(row rowScanner) (*spec.Application, error) {
	var (
		app      spec.Application
		bindings []byte
	)
	err := row.Scan(&app.TenantID, &app.ApplicationID, &app.Name, &app.Kind, &app.Status,
		&app.IconURL, &app.IconObjectKey, &app.LaunchURL, &bindings, &app.CategoryIDs, &app.CreatedAt, &app.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	app.Bindings = []spec.ProtocolBinding{}
	if len(bindings) > 0 {
		if err := json.Unmarshal(bindings, &app.Bindings); err != nil {
			return nil, err
		}
	}
	if app.CategoryIDs == nil {
		app.CategoryIDs = []string{}
	}
	return &app, nil
}

func (r *ApplicationRepository) ListByTenant(ctx context.Context, tenantID string) ([]*spec.Application, error) {
	rows, err := r.Pool.Query(ctx, applicationSelect+" WHERE tenant_id=$1 ORDER BY name", tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*spec.Application{}
	for rows.Next() {
		app, err := scanApplication(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, app)
	}
	return out, rows.Err()
}

func (r *ApplicationRepository) FindByID(ctx context.Context, tenantID, applicationID string) (*spec.Application, error) {
	return scanApplication(r.Pool.QueryRow(ctx,
		applicationSelect+" WHERE tenant_id=$1 AND application_id=$2", tenantID, applicationID))
}

func (r *ApplicationRepository) FindByBinding(ctx context.Context, tenantID string, bindingType spec.ProtocolBindingType, key string) (*spec.Application, error) {
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
			case spec.ProtocolBindingWsFed:
				if binding.Wtrealm == key {
					return app, nil
				}
			case spec.ProtocolBindingSAML:
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

func (r *ApplicationRepository) Save(ctx context.Context, app *spec.Application) error {
	bindings := app.Bindings
	if bindings == nil {
		bindings = []spec.ProtocolBinding{}
	}
	encoded, err := json.Marshal(bindings)
	if err != nil {
		return err
	}
	categoryIDs := app.CategoryIDs
	if categoryIDs == nil {
		categoryIDs = []string{}
	}
	_, err = r.Pool.Exec(ctx, `
INSERT INTO applications (tenant_id,application_id,name,kind,status,icon_url,icon_object_key,launch_url,bindings,category_ids,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
ON CONFLICT (tenant_id,application_id) DO UPDATE SET name=EXCLUDED.name,kind=EXCLUDED.kind,
 status=EXCLUDED.status,icon_url=EXCLUDED.icon_url,icon_object_key=EXCLUDED.icon_object_key,launch_url=EXCLUDED.launch_url,
 bindings=EXCLUDED.bindings,category_ids=EXCLUDED.category_ids,updated_at=EXCLUDED.updated_at`,
		app.TenantID, app.ApplicationID, app.Name, app.Kind, app.Status, app.IconURL, app.IconObjectKey, app.LaunchURL,
		encoded, categoryIDs, app.CreatedAt, app.UpdatedAt)
	return err
}

func (r *ApplicationRepository) Delete(ctx context.Context, tenantID, applicationID string) error {
	_, err := r.Pool.Exec(ctx,
		"DELETE FROM applications WHERE tenant_id=$1 AND application_id=$2", tenantID, applicationID)
	return err
}

func (r *ApplicationRepository) RemoveCategory(ctx context.Context, tenantID, categoryID string) error {
	_, err := r.Pool.Exec(ctx,
		"UPDATE applications SET category_ids=array_remove(category_ids,$2),updated_at=now() WHERE tenant_id=$1 AND $2=ANY(category_ids)",
		tenantID, categoryID)
	return err
}

// SignInPolicyRepository は Application sign-in policy を PostgreSQL に永続化する (ADR-079)。
type SignInPolicyRepository struct{ Pool DB }

func (r *SignInPolicyRepository) Get(ctx context.Context, tenantID, applicationID string) (*spec.AppSignInPolicy, error) {
	var (
		policy spec.AppSignInPolicy
		rules  []byte
	)
	err := r.Pool.QueryRow(ctx, `
SELECT tenant_id,application_id,rules,created_at,updated_at
  FROM application_sign_in_policies
 WHERE tenant_id=$1 AND application_id=$2`, tenantID, applicationID).
		Scan(&policy.TenantID, &policy.ApplicationID, &rules, &policy.CreatedAt, &policy.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	policy.Rules = []spec.SignInRule{}
	if len(rules) > 0 {
		if err := json.Unmarshal(rules, &policy.Rules); err != nil {
			return nil, err
		}
	}
	return &policy, nil
}

func (r *SignInPolicyRepository) ListByTenant(ctx context.Context, tenantID string) ([]*spec.AppSignInPolicy, error) {
	rows, err := r.Pool.Query(ctx, `
SELECT tenant_id,application_id,rules,created_at,updated_at
  FROM application_sign_in_policies
 WHERE tenant_id=$1`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*spec.AppSignInPolicy
	for rows.Next() {
		var (
			policy spec.AppSignInPolicy
			rules  []byte
		)
		if err := rows.Scan(&policy.TenantID, &policy.ApplicationID, &rules, &policy.CreatedAt, &policy.UpdatedAt); err != nil {
			return nil, err
		}
		policy.Rules = []spec.SignInRule{}
		if len(rules) > 0 {
			if err := json.Unmarshal(rules, &policy.Rules); err != nil {
				return nil, err
			}
		}
		out = append(out, &policy)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *SignInPolicyRepository) Save(ctx context.Context, policy *spec.AppSignInPolicy) error {
	rules := policy.Rules
	if rules == nil {
		rules = []spec.SignInRule{}
	}
	encoded, err := json.Marshal(rules)
	if err != nil {
		return err
	}
	_, err = r.Pool.Exec(ctx, `
INSERT INTO application_sign_in_policies (tenant_id,application_id,rules,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5)
ON CONFLICT (tenant_id,application_id) DO UPDATE SET rules=EXCLUDED.rules,updated_at=EXCLUDED.updated_at`,
		policy.TenantID, policy.ApplicationID, encoded, policy.CreatedAt, policy.UpdatedAt)
	return err
}

func (r *SignInPolicyRepository) Delete(ctx context.Context, tenantID, applicationID string) error {
	_, err := r.Pool.Exec(ctx,
		"DELETE FROM application_sign_in_policies WHERE tenant_id=$1 AND application_id=$2", tenantID, applicationID)
	return err
}

// DefaultSignInPolicyRepository はテナント既定 sign-in policy を PostgreSQL に永続化する (ADR-081)。
type DefaultSignInPolicyRepository struct{ Pool DB }

func (r *DefaultSignInPolicyRepository) Get(ctx context.Context, tenantID string) (*spec.TenantDefaultSignInPolicy, error) {
	var (
		policy spec.TenantDefaultSignInPolicy
		rules  []byte
	)
	err := r.Pool.QueryRow(ctx, `
SELECT tenant_id,rules,created_at,updated_at
  FROM tenant_default_sign_in_policies
 WHERE tenant_id=$1`, tenantID).
		Scan(&policy.TenantID, &rules, &policy.CreatedAt, &policy.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	policy.Rules = []spec.SignInRule{}
	if len(rules) > 0 {
		if err := json.Unmarshal(rules, &policy.Rules); err != nil {
			return nil, err
		}
	}
	return &policy, nil
}

func (r *DefaultSignInPolicyRepository) Save(ctx context.Context, policy *spec.TenantDefaultSignInPolicy) error {
	rules := policy.Rules
	if rules == nil {
		rules = []spec.SignInRule{}
	}
	encoded, err := json.Marshal(rules)
	if err != nil {
		return err
	}
	_, err = r.Pool.Exec(ctx, `
INSERT INTO tenant_default_sign_in_policies (tenant_id,rules,created_at,updated_at)
VALUES ($1,$2,$3,$4)
ON CONFLICT (tenant_id) DO UPDATE SET rules=EXCLUDED.rules,updated_at=EXCLUDED.updated_at`,
		policy.TenantID, encoded, policy.CreatedAt, policy.UpdatedAt)
	return err
}

// ApplicationIconStore は Application icon blob を PostgreSQL に保存する (wi-74, ADR-073)。
type ApplicationIconStore struct{ Pool DB }

func (s *ApplicationIconStore) Save(ctx context.Context, icon *spec.ApplicationIcon) error {
	_, err := s.Pool.Exec(ctx, `
INSERT INTO application_icons (tenant_id,application_id,object_key,content_type,size_bytes,data,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (tenant_id,application_id,object_key) DO UPDATE SET
 content_type=EXCLUDED.content_type,size_bytes=EXCLUDED.size_bytes,data=EXCLUDED.data,updated_at=EXCLUDED.updated_at`,
		icon.TenantID, icon.ApplicationID, icon.ObjectKey, icon.ContentType, icon.SizeBytes, icon.Data, icon.CreatedAt, icon.UpdatedAt)
	return err
}

func (s *ApplicationIconStore) Find(ctx context.Context, tenantID, applicationID, objectKey string) (*spec.ApplicationIcon, error) {
	var icon spec.ApplicationIcon
	err := s.Pool.QueryRow(ctx, `
SELECT tenant_id,application_id,object_key,content_type,size_bytes,data,created_at,updated_at
  FROM application_icons
 WHERE tenant_id=$1 AND application_id=$2 AND object_key=$3`,
		tenantID, applicationID, objectKey).
		Scan(&icon.TenantID, &icon.ApplicationID, &icon.ObjectKey, &icon.ContentType, &icon.SizeBytes, &icon.Data, &icon.CreatedAt, &icon.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &icon, nil
}

func (s *ApplicationIconStore) DeleteByApplication(ctx context.Context, tenantID, applicationID string) error {
	_, err := s.Pool.Exec(ctx,
		"DELETE FROM application_icons WHERE tenant_id=$1 AND application_id=$2", tenantID, applicationID)
	return err
}

// ApplicationAssignmentRepository は Application 割当を PostgreSQL に永続化する (wi-69)。
type ApplicationAssignmentRepository struct{ Pool DB }

const assignmentSelect = `SELECT tenant_id,application_id,subject_type,subject_id,visibility,created_at,updated_at FROM application_assignments`

func scanAssignment(row rowScanner) (*spec.ApplicationAssignment, error) {
	var a spec.ApplicationAssignment
	err := row.Scan(&a.TenantID, &a.ApplicationID, &a.SubjectType, &a.SubjectID, &a.Visibility, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func collectAssignments(rows pgx.Rows) ([]*spec.ApplicationAssignment, error) {
	defer rows.Close()
	out := []*spec.ApplicationAssignment{}
	for rows.Next() {
		a, err := scanAssignment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *ApplicationAssignmentRepository) ListByTenant(ctx context.Context, tenantID string) ([]*spec.ApplicationAssignment, error) {
	rows, err := r.Pool.Query(ctx,
		assignmentSelect+" WHERE tenant_id=$1",
		tenantID)
	if err != nil {
		return nil, err
	}
	return collectAssignments(rows)
}

func (r *ApplicationAssignmentRepository) ListByApplication(ctx context.Context, tenantID, applicationID string) ([]*spec.ApplicationAssignment, error) {
	rows, err := r.Pool.Query(ctx,
		assignmentSelect+" WHERE tenant_id=$1 AND application_id=$2 ORDER BY subject_type,subject_id",
		tenantID, applicationID)
	if err != nil {
		return nil, err
	}
	return collectAssignments(rows)
}

func (r *ApplicationAssignmentRepository) ListBySubjects(ctx context.Context, tenantID string, subjects []appports.SubjectRef) ([]*spec.ApplicationAssignment, error) {
	if len(subjects) == 0 {
		return []*spec.ApplicationAssignment{}, nil
	}
	types := make([]string, len(subjects))
	ids := make([]string, len(subjects))
	for i, s := range subjects {
		types[i] = string(s.Type)
		ids[i] = s.ID
	}
	// (subject_type, subject_id) のペアを UNNEST で突き合わせる。
	rows, err := r.Pool.Query(ctx, assignmentSelect+`
 WHERE tenant_id=$1 AND (subject_type,subject_id) IN (
   SELECT subject_type, subject_id FROM UNNEST($2::text[], $3::text[]) AS s(subject_type, subject_id)
 )`, tenantID, types, ids)
	if err != nil {
		return nil, err
	}
	return collectAssignments(rows)
}

func (r *ApplicationAssignmentRepository) Save(ctx context.Context, a *spec.ApplicationAssignment) error {
	_, err := r.Pool.Exec(ctx, `
INSERT INTO application_assignments (tenant_id,application_id,subject_type,subject_id,visibility,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT (tenant_id,application_id,subject_type,subject_id) DO UPDATE SET
 visibility=EXCLUDED.visibility,updated_at=EXCLUDED.updated_at`,
		a.TenantID, a.ApplicationID, a.SubjectType, a.SubjectID, a.Visibility, a.CreatedAt, a.UpdatedAt)
	return err
}

func (r *ApplicationAssignmentRepository) Delete(ctx context.Context, tenantID, applicationID string, subjectType spec.AssignmentSubjectType, subjectID string) error {
	_, err := r.Pool.Exec(ctx, `
DELETE FROM application_assignments
 WHERE tenant_id=$1 AND application_id=$2 AND subject_type=$3 AND subject_id=$4`,
		tenantID, applicationID, subjectType, subjectID)
	return err
}

func (r *ApplicationAssignmentRepository) DeleteByApplication(ctx context.Context, tenantID, applicationID string) error {
	_, err := r.Pool.Exec(ctx,
		"DELETE FROM application_assignments WHERE tenant_id=$1 AND application_id=$2", tenantID, applicationID)
	return err
}

// ApplicationOrderingRepository は利用者ごとのポータル手動並び順を PostgreSQL に永続化する
// (wi-70, ADR-069)。application_ids は順序を保つ text[] で格納し、tenant 境界に閉じる。
type ApplicationOrderingRepository struct{ Pool DB }

func (r *ApplicationOrderingRepository) Get(ctx context.Context, tenantID, userID string) (*spec.ApplicationOrdering, error) {
	var o spec.ApplicationOrdering
	err := r.Pool.QueryRow(ctx,
		`SELECT tenant_id,user_id,application_ids,created_at,updated_at FROM application_orderings
 WHERE tenant_id=$1 AND user_id=$2`, tenantID, userID).
		Scan(&o.TenantID, &o.UserID, &o.ApplicationIDs, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *ApplicationOrderingRepository) Save(ctx context.Context, o *spec.ApplicationOrdering) error {
	ids := o.ApplicationIDs
	if ids == nil {
		ids = []string{}
	}
	_, err := r.Pool.Exec(ctx, `
INSERT INTO application_orderings (tenant_id,user_id,application_ids,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5)
ON CONFLICT (tenant_id,user_id) DO UPDATE SET
 application_ids=EXCLUDED.application_ids,updated_at=EXCLUDED.updated_at`,
		o.TenantID, o.UserID, ids, o.CreatedAt, o.UpdatedAt)
	return err
}

// ApplicationCategoryRepository は ApplicationCategory を PostgreSQL に永続化する (wi-70, ADR-069)。
// すべてテナント境界に閉じる。
type ApplicationCategoryRepository struct{ Pool DB }

const categorySelect = `SELECT tenant_id,category_id,name,position,created_at,updated_at FROM application_categories`

func scanCategory(row rowScanner) (*spec.ApplicationCategory, error) {
	var c spec.ApplicationCategory
	err := row.Scan(&c.TenantID, &c.CategoryID, &c.Name, &c.Position, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ApplicationCategoryRepository) ListByTenant(ctx context.Context, tenantID string) ([]*spec.ApplicationCategory, error) {
	rows, err := r.Pool.Query(ctx, categorySelect+" WHERE tenant_id=$1 ORDER BY position,name", tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*spec.ApplicationCategory{}
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *ApplicationCategoryRepository) FindByID(ctx context.Context, tenantID, categoryID string) (*spec.ApplicationCategory, error) {
	return scanCategory(r.Pool.QueryRow(ctx,
		categorySelect+" WHERE tenant_id=$1 AND category_id=$2", tenantID, categoryID))
}

func (r *ApplicationCategoryRepository) Save(ctx context.Context, c *spec.ApplicationCategory) error {
	_, err := r.Pool.Exec(ctx, `
INSERT INTO application_categories (tenant_id,category_id,name,position,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6)
ON CONFLICT (tenant_id,category_id) DO UPDATE SET
 name=EXCLUDED.name,position=EXCLUDED.position,updated_at=EXCLUDED.updated_at`,
		c.TenantID, c.CategoryID, c.Name, c.Position, c.CreatedAt, c.UpdatedAt)
	return err
}

func (r *ApplicationCategoryRepository) Delete(ctx context.Context, tenantID, categoryID string) error {
	_, err := r.Pool.Exec(ctx,
		"DELETE FROM application_categories WHERE tenant_id=$1 AND category_id=$2", tenantID, categoryID)
	return err
}
