package db_postgres

import (
	"context"
	"errors"
	"slices"
	"time"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appports "github.com/ambi/idmagic/backend/application/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ApplicationRepository は ApplicationCatalog の Application aggregate を PostgreSQL に
// 永続化する。protocol relation は各 protocol table の application_id から解決する。
// クエリは sqlc 生成 (wi-172); Pool は DBTX を構造的に満たす。
type ApplicationRepository struct {
	Pool            sharedpg.DB
	ProtocolLinkers ProtocolLinkers
}

func applicationFromRow(row *Application) *appdomain.Application {
	app := &appdomain.Application{
		TenantID:      row.TenantID,
		ID:            row.ID,
		Name:          row.Name,
		Kind:          appdomain.ApplicationKind(row.Kind),
		Status:        appdomain.ApplicationStatus(row.Status),
		IconURL:       row.IconUrl,
		IconObjectKey: row.IconObjectKey,
		LaunchURL:     row.LaunchUrl,
		CategoryIDs:   row.CategoryIds,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
	if row.ProtocolType.Valid {
		app.Protocol = &appdomain.ApplicationProtocol{Type: appdomain.ApplicationProtocolType(row.ProtocolType.String)}
	}
	if app.CategoryIDs == nil {
		app.CategoryIDs = []string{}
	}
	return app
}

func (r *ApplicationRepository) hydrateProtocol(ctx context.Context, app *appdomain.Application) error {
	if app.Protocol == nil {
		return nil
	}
	row, err := New(r.Pool).GetApplicationProtocolKey(ctx, GetApplicationProtocolKeyParams{
		TenantID: app.TenantID, ID: app.ID,
	})
	if err != nil {
		return err
	}
	switch app.Protocol.Type {
	case appdomain.ApplicationProtocolOIDC:
		app.Protocol.ClientID = row
	case appdomain.ApplicationProtocolSAML:
		app.Protocol.EntityID = row
	case appdomain.ApplicationProtocolWsFed:
		app.Protocol.Wtrealm = row
	}
	return nil
}

func (r *ApplicationRepository) ListAll(ctx context.Context, tenantID string) ([]*appdomain.Application, error) {
	rows, err := New(r.Pool).ListApplicationsByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*appdomain.Application, 0, len(rows))
	for _, row := range rows {
		app := applicationFromRow(row)
		if err := r.hydrateProtocol(ctx, app); err != nil {
			return nil, err
		}
		out = append(out, app)
	}
	return out, nil
}

// ListPage implements ports.ApplicationRepository.ListPage (wi-159):
// keyset pagination ordered by (name, id) ascending, strictly after the
// given keyset ("", "" for the first page).
func (r *ApplicationRepository) ListPage(ctx context.Context, tenantID, afterName, afterID string, limit int) ([]*appdomain.Application, error) {
	q := New(r.Pool)
	var rows []*Application
	var err error
	if afterName == "" && afterID == "" {
		rows, err = q.ListApplicationsByTenantPage(ctx, ListApplicationsByTenantPageParams{
			TenantID:  tenantID,
			PageLimit: int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
		})
	} else {
		rows, err = q.ListApplicationsByTenantPageAfter(ctx, ListApplicationsByTenantPageAfterParams{
			TenantID:  tenantID,
			AfterName: afterName,
			AfterID:   afterID,
			PageLimit: int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
		})
	}
	if err != nil {
		return nil, err
	}
	out := make([]*appdomain.Application, 0, len(rows))
	for _, row := range rows {
		app := applicationFromRow(row)
		if err := r.hydrateProtocol(ctx, app); err != nil {
			return nil, err
		}
		out = append(out, app)
	}
	return out, nil
}

func (r *ApplicationRepository) ListPageBefore(ctx context.Context, tenantID, beforeName, beforeID string, limit int) ([]*appdomain.Application, error) {
	q := New(r.Pool)
	var rows []*Application
	var err error
	if beforeName == "" && beforeID == "" {
		rows, err = q.ListApplicationsByTenantPageEnd(ctx, ListApplicationsByTenantPageEndParams{TenantID: tenantID, PageLimit: int32(limit)}) //nolint:gosec // caller clamps limit to a small positive bound
	} else {
		rows, err = q.ListApplicationsByTenantPageBefore(ctx, ListApplicationsByTenantPageBeforeParams{
			TenantID: tenantID, BeforeName: beforeName, BeforeID: beforeID,
			PageLimit: int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
		})
	}
	if err != nil {
		return nil, err
	}
	slices.Reverse(rows)
	out := make([]*appdomain.Application, 0, len(rows))
	for _, row := range rows {
		app := applicationFromRow(row)
		if err := r.hydrateProtocol(ctx, app); err != nil {
			return nil, err
		}
		out = append(out, app)
	}
	return out, nil
}

func (r *ApplicationRepository) Count(ctx context.Context, tenantID string) (int64, error) {
	return New(r.Pool).CountApplicationsByTenant(ctx, tenantID)
}

func (r *ApplicationRepository) FindByID(ctx context.Context, tenantID, applicationID string) (*appdomain.Application, error) {
	row, err := New(r.Pool).GetApplicationByID(ctx, GetApplicationByIDParams{
		TenantID: tenantID, ID: applicationID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	app := applicationFromRow(row)
	if err := r.hydrateProtocol(ctx, app); err != nil {
		return nil, err
	}
	return app, nil
}

func (r *ApplicationRepository) FindByProtocol(ctx context.Context, tenantID string, protocolType appdomain.ApplicationProtocolType, key string) (*appdomain.Application, error) {
	if key == "" {
		return nil, nil
	}
	row, err := New(r.Pool).FindApplicationByProtocol(ctx, FindApplicationByProtocolParams{
		TenantID:     tenantID,
		ProtocolType: pgtype.Text{String: string(protocolType), Valid: true},
		ClientID:     key,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	app := applicationFromRow(row)
	if err := r.hydrateProtocol(ctx, app); err != nil {
		return nil, err
	}
	return app, nil
}

func (r *ApplicationRepository) Save(ctx context.Context, app *appdomain.Application) error {
	return saveApplication(ctx, New(r.Pool), app)
}

func (r *ApplicationRepository) Update(ctx context.Context, tenantID, applicationID string, change func(*appdomain.Application) (*appdomain.Application, error)) (*appdomain.Application, error) {
	return sharedpg.InTransaction(ctx, r.Pool, func(tx pgx.Tx) (*appdomain.Application, error) {
		if _, err := New(tx).LockApplicationForDeletion(ctx, LockApplicationForDeletionParams{TenantID: tenantID, ID: applicationID}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, appports.ErrApplicationNotFound
			}
			return nil, err
		}
		bound := &ApplicationRepository{Pool: sharedpg.BindTransaction(tx), ProtocolLinkers: r.ProtocolLinkers}
		current, err := bound.FindByID(ctx, tenantID, applicationID)
		if err != nil {
			return nil, err
		}
		updated, err := change(current)
		if err != nil {
			return nil, err
		}
		if err := saveApplication(ctx, New(tx), updated); err != nil {
			return nil, err
		}
		return bound.FindByID(ctx, tenantID, applicationID)
	})
}

func saveApplication(ctx context.Context, queries *Queries, app *appdomain.Application) error {
	var protocolType pgtype.Text
	if app.Protocol != nil {
		protocolType = pgtype.Text{String: string(app.Protocol.Type), Valid: true}
	}
	categoryIDs := app.CategoryIDs
	if categoryIDs == nil {
		categoryIDs = []string{}
	}
	return queries.UpsertApplication(ctx, UpsertApplicationParams{
		TenantID:      app.TenantID,
		ID:            app.ID,
		Name:          app.Name,
		Kind:          string(app.Kind),
		Status:        string(app.Status),
		ProtocolType:  protocolType,
		IconUrl:       app.IconURL,
		IconObjectKey: app.IconObjectKey,
		LaunchUrl:     app.LaunchURL,
		CategoryIds:   categoryIDs,
		CreatedAt:     app.CreatedAt,
		UpdatedAt:     app.UpdatedAt,
	})
}

func (r *ApplicationRepository) linkApplicationProtocol(ctx context.Context, tx pgx.Tx, queries *Queries, app *appdomain.Application) error {
	if app.Protocol == nil {
		return nil
	}
	var key string
	var link func(context.Context, pgx.Tx, string, string, string) error
	switch app.Protocol.Type {
	case appdomain.ApplicationProtocolOIDC:
		key = app.Protocol.ClientID
		link = r.ProtocolLinkers.OIDC
	case appdomain.ApplicationProtocolSAML:
		key = app.Protocol.EntityID
		link = r.ProtocolLinkers.SAML
	case appdomain.ApplicationProtocolWsFed:
		key = app.Protocol.Wtrealm
		link = r.ProtocolLinkers.WsFed
	default:
		return appdomain.ErrInvalidProtocolType
	}
	if err := link(ctx, tx, app.TenantID, app.ID, key); err != nil {
		return err
	}
	linkedKey, err := queries.GetApplicationProtocolKey(ctx, GetApplicationProtocolKeyParams{
		TenantID: app.TenantID, ID: app.ID,
	})
	if err != nil {
		return err
	}
	if linkedKey != key {
		return errors.New("application protocol was not linked")
	}
	return nil
}

func (r *ApplicationRepository) Create(ctx context.Context, app *appdomain.Application) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := New(tx)
	if err := saveApplication(ctx, queries, app); err != nil {
		return err
	}
	if err := r.linkApplicationProtocol(ctx, tx, queries, app); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *ApplicationRepository) Delete(ctx context.Context, tenantID, applicationID string) error {
	return New(r.Pool).DeleteApplication(ctx, DeleteApplicationParams{
		TenantID: tenantID, ID: applicationID,
	})
}

func (r *ApplicationRepository) RemoveCategory(ctx context.Context, tenantID, categoryID string) error {
	return New(r.Pool).RemoveApplicationCategory(ctx, RemoveApplicationCategoryParams{
		TenantID: tenantID, ArrayRemove: categoryID,
	})
}

// ApplicationIconStore は Application icon blob を PostgreSQL に保存する (wi-74)。
type ApplicationIconStore struct{ Pool sharedpg.DB }

func (s *ApplicationIconStore) Save(ctx context.Context, icon *appdomain.ApplicationIcon) error {
	return New(s.Pool).UpsertApplicationIcon(ctx, UpsertApplicationIconParams{
		ApplicationID: icon.ApplicationID, ID: icon.ID,
		ContentType: icon.ContentType, SizeBytes: int32(icon.SizeBytes), Data: icon.Data, //nolint:gosec // G115: icon size is bounded by upload limits, well under int32 max
		CreatedAt: icon.CreatedAt, UpdatedAt: icon.UpdatedAt,
	})
}

func (s *ApplicationIconStore) Find(ctx context.Context, tenantID, applicationID, id string) (*appdomain.ApplicationIcon, error) {
	row, err := New(s.Pool).GetApplicationIcon(ctx, GetApplicationIconParams{
		TenantID: tenantID, ApplicationID: applicationID, ID: id,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &appdomain.ApplicationIcon{
		TenantID: row.TenantID, ApplicationID: row.ApplicationID, ID: row.ID,
		ContentType: row.ContentType, SizeBytes: int(row.SizeBytes), Data: row.Data,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

func (s *ApplicationIconStore) DeleteByApplication(ctx context.Context, tenantID, applicationID string) error {
	return New(s.Pool).DeleteApplicationIconsByApplication(ctx, DeleteApplicationIconsByApplicationParams{
		TenantID: tenantID, ApplicationID: applicationID,
	})
}

// ApplicationAssignmentRepository は Application 割当を PostgreSQL に永続化する (wi-69)。
type ApplicationAssignmentRepository struct{ Pool sharedpg.DB }

func assignmentFromFields(tenantID, applicationID, subjectType, subjectID, visibility string, createdAt, updatedAt time.Time) *appdomain.ApplicationAssignment {
	return &appdomain.ApplicationAssignment{
		TenantID: tenantID, ApplicationID: applicationID, SubjectType: appdomain.AssignmentSubjectType(subjectType),
		SubjectID: subjectID, Visibility: appdomain.AssignmentVisibility(visibility), CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
}

func (r *ApplicationAssignmentRepository) ListAll(ctx context.Context, tenantID string) ([]*appdomain.ApplicationAssignment, error) {
	rows, err := New(r.Pool).ListApplicationAssignmentsByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*appdomain.ApplicationAssignment, 0, len(rows))
	for _, row := range rows {
		out = append(out, assignmentFromFields(row.TenantID, row.ApplicationID, row.SubjectType, row.SubjectID, row.Visibility, row.CreatedAt, row.UpdatedAt))
	}
	return out, nil
}

func (r *ApplicationAssignmentRepository) ListByApplication(ctx context.Context, tenantID, applicationID string) ([]*appdomain.ApplicationAssignment, error) {
	rows, err := New(r.Pool).ListApplicationAssignmentsByApplication(ctx, ListApplicationAssignmentsByApplicationParams{
		TenantID: tenantID, ApplicationID: applicationID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*appdomain.ApplicationAssignment, 0, len(rows))
	for _, row := range rows {
		out = append(out, assignmentFromFields(row.TenantID, row.ApplicationID, row.SubjectType, row.SubjectID, row.Visibility, row.CreatedAt, row.UpdatedAt))
	}
	return out, nil
}

// ListPageByApplication implements ports.AssignmentRepository.ListPageByApplication
// (wi-159): keyset pagination ordered by (subject_type, subject_id)
// ascending, strictly after the given keyset ("", "" for the first page).
func (r *ApplicationAssignmentRepository) ListPageByApplication(ctx context.Context, tenantID, applicationID, afterSubjectType, afterSubjectID string, limit int) ([]*appdomain.ApplicationAssignment, error) {
	q := New(r.Pool)
	var out []*appdomain.ApplicationAssignment
	if afterSubjectType == "" && afterSubjectID == "" {
		rows, err := q.ListApplicationAssignmentsByApplicationPage(ctx, ListApplicationAssignmentsByApplicationPageParams{
			TenantID: tenantID, ApplicationID: applicationID,
			PageLimit: int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
		})
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			out = append(out, assignmentFromFields(r.TenantID, r.ApplicationID, r.SubjectType, r.SubjectID, r.Visibility, r.CreatedAt, r.UpdatedAt))
		}
		return out, nil
	}
	rows, err := q.ListApplicationAssignmentsByApplicationPageAfter(ctx, ListApplicationAssignmentsByApplicationPageAfterParams{
		TenantID: tenantID, ApplicationID: applicationID,
		AfterSubjectType: afterSubjectType, AfterSubjectID: afterSubjectID,
		PageLimit: int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
	})
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out = append(out, assignmentFromFields(r.TenantID, r.ApplicationID, r.SubjectType, r.SubjectID, r.Visibility, r.CreatedAt, r.UpdatedAt))
	}
	return out, nil
}

func (r *ApplicationAssignmentRepository) ListPageBeforeByApplication(ctx context.Context, tenantID, applicationID, beforeSubjectType, beforeSubjectID string, limit int) ([]*appdomain.ApplicationAssignment, error) {
	rows, err := New(r.Pool).ListApplicationAssignmentsByApplicationPageBefore(ctx, ListApplicationAssignmentsByApplicationPageBeforeParams{
		TenantID: tenantID, ApplicationID: applicationID, BeforeSubjectType: beforeSubjectType,
		BeforeSubjectID: beforeSubjectID,
		PageLimit:       int32(limit), //nolint:gosec // caller clamps limit to a small positive bound
	})
	if err != nil {
		return nil, err
	}
	slices.Reverse(rows)
	out := make([]*appdomain.ApplicationAssignment, 0, len(rows))
	for _, row := range rows {
		out = append(out, assignmentFromFields(row.TenantID, row.ApplicationID, row.SubjectType, row.SubjectID, row.Visibility, row.CreatedAt, row.UpdatedAt))
	}
	return out, nil
}

func (r *ApplicationAssignmentRepository) ListBySubjects(ctx context.Context, tenantID string, subjects []appports.SubjectRef) ([]*appdomain.ApplicationAssignment, error) {
	if len(subjects) == 0 {
		return []*appdomain.ApplicationAssignment{}, nil
	}
	types := make([]string, len(subjects))
	ids := make([]string, len(subjects))
	for i, s := range subjects {
		types[i] = string(s.Type)
		ids[i] = s.ID
	}
	rows, err := New(r.Pool).ListApplicationAssignmentsBySubjects(ctx, ListApplicationAssignmentsBySubjectsParams{
		TenantID: tenantID, SubjectTypes: types, SubjectIds: ids,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*appdomain.ApplicationAssignment, 0, len(rows))
	for _, row := range rows {
		out = append(out, assignmentFromFields(row.TenantID, row.ApplicationID, row.SubjectType, row.SubjectID, row.Visibility, row.CreatedAt, row.UpdatedAt))
	}
	return out, nil
}

func (r *ApplicationAssignmentRepository) Save(ctx context.Context, a *appdomain.ApplicationAssignment) error {
	return New(r.Pool).UpsertApplicationAssignment(ctx, UpsertApplicationAssignmentParams{
		ApplicationID: a.ApplicationID, SubjectType: string(a.SubjectType),
		SubjectID: a.SubjectID, Visibility: string(a.Visibility), CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	})
}

func (r *ApplicationAssignmentRepository) Delete(ctx context.Context, tenantID, applicationID string, subjectType appdomain.AssignmentSubjectType, subjectID string) error {
	return New(r.Pool).DeleteApplicationAssignment(ctx, DeleteApplicationAssignmentParams{
		TenantID: tenantID, ApplicationID: applicationID, SubjectType: string(subjectType), SubjectID: subjectID,
	})
}

func (r *ApplicationAssignmentRepository) DeleteByApplication(ctx context.Context, tenantID, applicationID string) error {
	return New(r.Pool).DeleteApplicationAssignmentsByApplication(ctx, DeleteApplicationAssignmentsByApplicationParams{
		TenantID: tenantID, ApplicationID: applicationID,
	})
}

// ApplicationOrderingRepository は利用者ごとのポータル手動並び順を PostgreSQL に永続化する
// (wi-70)。application_ids は順序を保つ text[] で格納する。user_id は global unique
// なため tenant_id 列は持たず、行は user_id で一意に識別する。tenantID 引数は port
// 契約の互換のために残すが SQL では用いない。
type ApplicationOrderingRepository struct{ Pool sharedpg.DB }

func (r *ApplicationOrderingRepository) Get(ctx context.Context, _ /*tenantID*/, userID string) (*appdomain.ApplicationOrdering, error) {
	row, err := New(r.Pool).GetApplicationOrdering(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &appdomain.ApplicationOrdering{
		UserID: row.UserID, ApplicationIDs: row.ApplicationIds,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *ApplicationOrderingRepository) Save(ctx context.Context, o *appdomain.ApplicationOrdering) error {
	ids := o.ApplicationIDs
	if ids == nil {
		ids = []string{}
	}
	return New(r.Pool).UpsertApplicationOrdering(ctx, UpsertApplicationOrderingParams{
		UserID: o.UserID, ApplicationIds: ids, CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt,
	})
}

// ApplicationCategoryRepository は ApplicationCategory を PostgreSQL に永続化する (wi-70)。
// すべてテナント境界に閉じる。
type ApplicationCategoryRepository struct{ Pool sharedpg.DB }

func categoryFromRow(row *ApplicationCategory) *appdomain.ApplicationCategory {
	return &appdomain.ApplicationCategory{
		TenantID: row.TenantID, ID: row.ID, Name: row.Name,
		Position: int(row.Position), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func (r *ApplicationCategoryRepository) ListAll(ctx context.Context, tenantID string) ([]*appdomain.ApplicationCategory, error) {
	rows, err := New(r.Pool).ListApplicationCategoriesByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*appdomain.ApplicationCategory, 0, len(rows))
	for _, row := range rows {
		out = append(out, categoryFromRow(row))
	}
	return out, nil
}

func (r *ApplicationCategoryRepository) FindByID(ctx context.Context, tenantID, categoryID string) (*appdomain.ApplicationCategory, error) {
	row, err := New(r.Pool).GetApplicationCategoryByID(ctx, GetApplicationCategoryByIDParams{
		TenantID: tenantID, ID: categoryID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return categoryFromRow(row), nil
}

func (r *ApplicationCategoryRepository) Save(ctx context.Context, c *appdomain.ApplicationCategory) error {
	return New(r.Pool).UpsertApplicationCategory(ctx, UpsertApplicationCategoryParams{
		TenantID: c.TenantID, ID: c.ID, Name: c.Name, Position: int32(c.Position), //nolint:gosec // G115: category position is a small manual-ordering index, well under int32 max
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	})
}

func (r *ApplicationCategoryRepository) Delete(ctx context.Context, tenantID, categoryID string) error {
	return New(r.Pool).DeleteApplicationCategory(ctx, DeleteApplicationCategoryParams{
		TenantID: tenantID, ID: categoryID,
	})
}
