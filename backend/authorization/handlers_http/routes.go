// Package handlers_http は Authorization bounded context の管理 API を所有する。
// 認可モデルと関係タプルの管理、および診断用の判定エンドポイントを、テナント解決済み
// グループに登録する。
package handlers_http

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/ambi/idmagic/backend/authorization/domain"
	"github.com/ambi/idmagic/backend/authorization/ports"
	"github.com/ambi/idmagic/backend/authorization/usecases"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

// Deps は Authorization の HTTP ハンドラーが必要とする依存。
type Deps struct {
	support.Deps
	*support.Authenticator
	TupleRepo  ports.RelationTupleRepository
	ModelRepo  ports.AuthorizationModelRepository
	Principals ports.PrincipalStatusResolver
	// Authorizer は OAuth2 が所有する AuthZEN ポート。合成規則はこの先にある。
	Authorizer oauthports.Authorizer
	// MaxDepth と MaxEnumeratedResources はゼロなら usecases / domain の既定を使う。
	MaxDepth               int
	MaxEnumeratedResources int
}

func (d Deps) usecaseDeps() usecases.Deps {
	return usecases.Deps{
		Tuples: d.TupleRepo, Models: d.ModelRepo, Principals: d.Principals,
		Authorizer: d.Authorizer, Emit: d.Emit,
		MaxDepth: d.MaxDepth, MaxEnumeratedResources: d.MaxEnumeratedResources,
	}
}

// RegisterRoutes はテナント解決済みグループに Authorization 管理 API を登録する。
func RegisterRoutes(g *echo.Group, d Deps) {
	g.GET("/api/admin/v1/authorization/model", d.handleGetAuthorizationModel)
	g.POST("/api/admin/v1/authorization/model", d.handlePutAuthorizationModel)
	g.GET("/api/admin/v1/authorization/relation-tuples", d.handleListRelationTuples)
	g.POST("/api/admin/v1/authorization/relation-tuples", d.handleWriteRelationTuples)
	g.POST("/api/admin/v1/authorization/check", d.handleCheckAccess)
	g.POST("/api/admin/v1/authorization/list-accessible-resources", d.handleListAccessibleResources)
}

// ---- ワイヤ表現 ----

type relationRewriteWire struct {
	Kind               string   `json:"kind"`
	ComputedRelation   string   `json:"computed_relation,omitempty"`
	TuplesetRelation   string   `json:"tupleset_relation,omitempty"`
	DirectSubjectTypes []string `json:"direct_subject_types,omitempty"`
}

type relationDefinitionWire struct {
	Name     string                `json:"name"`
	Rewrites []relationRewriteWire `json:"rewrites"`
}

type resourceTypeWire struct {
	Name      string                   `json:"name"`
	Relations []relationDefinitionWire `json:"relations"`
}

type authorizationModelWire struct {
	ID            string             `json:"id"`
	TenantID      string             `json:"tenant_id"`
	Version       int                `json:"version"`
	ResourceTypes []resourceTypeWire `json:"resource_types"`
	CreatedAt     time.Time          `json:"created_at"`
}

type relationTupleWire struct {
	ResourceType    string `json:"resource_type"`
	ResourceID      string `json:"resource_id"`
	Relation        string `json:"relation"`
	SubjectType     string `json:"subject_type"`
	SubjectID       string `json:"subject_id"`
	SubjectRelation string `json:"subject_relation,omitempty"`
}

type objectReferenceWire struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type actorWire struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

func toResourceTypes(wire []resourceTypeWire) []domain.ResourceTypeDefinition {
	out := make([]domain.ResourceTypeDefinition, 0, len(wire))
	for _, t := range wire {
		relations := make([]domain.RelationDefinition, 0, len(t.Relations))
		for _, r := range t.Relations {
			rewrites := make([]domain.RelationRewrite, 0, len(r.Rewrites))
			for _, rw := range r.Rewrites {
				rewrites = append(rewrites, domain.RelationRewrite{
					Kind:               domain.RelationRewriteKind(rw.Kind),
					ComputedRelation:   rw.ComputedRelation,
					TuplesetRelation:   rw.TuplesetRelation,
					DirectSubjectTypes: rw.DirectSubjectTypes,
				})
			}
			relations = append(relations, domain.RelationDefinition{Name: r.Name, Rewrites: rewrites})
		}
		out = append(out, domain.ResourceTypeDefinition{Name: t.Name, Relations: relations})
	}
	return out
}

func toModelWire(model *domain.AuthorizationModel) authorizationModelWire {
	types := make([]resourceTypeWire, 0, len(model.ResourceTypes))
	for _, t := range model.ResourceTypes {
		relations := make([]relationDefinitionWire, 0, len(t.Relations))
		for _, r := range t.Relations {
			rewrites := make([]relationRewriteWire, 0, len(r.Rewrites))
			for _, rw := range r.Rewrites {
				rewrites = append(rewrites, relationRewriteWire{
					Kind:             string(rw.Kind),
					ComputedRelation: rw.ComputedRelation,
					TuplesetRelation: rw.TuplesetRelation,
					// nil と空配列を分けず、常に宣言された形の一覧として返す。
					DirectSubjectTypes: rw.DirectSubjectTypes,
				})
			}
			relations = append(relations, relationDefinitionWire{Name: r.Name, Rewrites: rewrites})
		}
		types = append(types, resourceTypeWire{Name: t.Name, Relations: relations})
	}
	return authorizationModelWire{
		ID: model.ID, TenantID: model.TenantID, Version: model.Version,
		ResourceTypes: types, CreatedAt: model.CreatedAt,
	}
}

func toTuple(wire relationTupleWire) domain.RelationTuple {
	return domain.RelationTuple{
		Resource: domain.ObjectRef{Type: wire.ResourceType, ID: wire.ResourceID},
		Relation: wire.Relation,
		Subject: domain.SubjectRef{
			Type: wire.SubjectType, ID: wire.SubjectID, Relation: wire.SubjectRelation,
		},
	}
}

func toTupleWire(t domain.RelationTuple) relationTupleWire {
	return relationTupleWire{
		ResourceType: t.Resource.Type, ResourceID: t.Resource.ID, Relation: t.Relation,
		SubjectType: t.Subject.Type, SubjectID: t.Subject.ID, SubjectRelation: t.Subject.Relation,
	}
}

func toActors(wire []actorWire) []usecases.Actor {
	out := make([]usecases.Actor, 0, len(wire))
	for _, a := range wire {
		out = append(out, usecases.Actor{Type: a.Type, ID: a.ID})
	}
	return out
}

// ---- 認可モデル ----

func (d Deps) handleGetAuthorizationModel(c *echo.Context) error {
	if err := d.requireAuthorizationAdmin(c); err != nil {
		return err
	}
	version, err := strconv.Atoi(c.Request().URL.Query().Get("version"))
	if err != nil {
		version = 0
	}
	published, err := usecases.GetAuthorizationModel(c.Request().Context(), d.usecaseDeps(), support.RequestTenantID(c), version)
	if err != nil {
		return writeAuthorizationError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{
		"authorization_model": toModelWire(published.Model),
		"consistency":         published.Consistency,
	})
}

type putAuthorizationModelRequest struct {
	ResourceTypes []resourceTypeWire `json:"resource_types"`
}

func (d Deps) handlePutAuthorizationModel(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if err := d.requireAuthorizationAdmin(c); err != nil {
		return err
	}
	var req putAuthorizationModelRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	published, err := usecases.PutAuthorizationModel(c.Request().Context(), d.usecaseDeps(),
		support.RequestTenantID(c), toResourceTypes(req.ResourceTypes), time.Now().UTC())
	if err != nil {
		return writeAuthorizationError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusCreated, map[string]any{
		"authorization_model": toModelWire(published.Model),
		"consistency":         published.Consistency,
	})
}

// ---- 関係タプル ----

func (d Deps) handleListRelationTuples(c *echo.Context) error {
	if err := d.requireAuthorizationAdmin(c); err != nil {
		return err
	}
	query := c.Request().URL.Query()
	limit, err := strconv.Atoi(query.Get("limit"))
	if err != nil {
		limit = 0
	}
	listed, err := usecases.ListRelationTuples(c.Request().Context(), d.usecaseDeps(), support.RequestTenantID(c),
		ports.RelationTupleFilter{
			ResourceType: query.Get("resource_type"),
			ResourceID:   query.Get("resource_id"),
			Relation:     query.Get("relation"),
			SubjectType:  query.Get("subject_type"),
			SubjectID:    query.Get("subject_id"),
		}, limit)
	if err != nil {
		return writeAuthorizationError(c, err)
	}
	tuples := make([]relationTupleWire, 0, len(listed.Tuples))
	for _, t := range listed.Tuples {
		tuples = append(tuples, toTupleWire(t))
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{
		"tuples": tuples, "consistency": listed.Consistency,
	})
}

type writeRelationTuplesRequest struct {
	Writes        []relationTupleWire   `json:"writes"`
	Deletes       []relationTupleWire   `json:"deletes"`
	DeleteObjects []objectReferenceWire `json:"delete_objects"`
}

func (d Deps) handleWriteRelationTuples(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if err := d.requireAuthorizationAdmin(c); err != nil {
		return err
	}
	var req writeRelationTuplesRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	write := ports.TupleWrite{}
	for _, t := range req.Writes {
		write.Writes = append(write.Writes, toTuple(t))
	}
	for _, t := range req.Deletes {
		write.Deletes = append(write.Deletes, toTuple(t))
	}
	for _, o := range req.DeleteObjects {
		write.DeleteObjects = append(write.DeleteObjects, domain.ObjectRef{Type: o.Type, ID: o.ID})
	}
	outcome, err := usecases.WriteRelationTuples(c.Request().Context(), d.usecaseDeps(),
		support.RequestTenantID(c), write, time.Now().UTC())
	if err != nil {
		return writeAuthorizationError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{
		"written_count": outcome.WrittenCount,
		"deleted_count": outcome.DeletedCount,
		"consistency":   outcome.Consistency,
	})
}

// ---- 判定 ----

type checkAccessRequest struct {
	ResourceType       string      `json:"resource_type"`
	ResourceID         string      `json:"resource_id"`
	Relation           string      `json:"relation"`
	SubjectType        string      `json:"subject_type"`
	SubjectID          string      `json:"subject_id"`
	ActorChain         []actorWire `json:"actor_chain"`
	MinimumConsistency string      `json:"minimum_consistency"`
	GrantedScopes      []string    `json:"granted_scopes"`
	RequiredScopes     []string    `json:"required_scopes"`
}

func (d Deps) handleCheckAccess(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if err := d.requireAuthorizationAdmin(c); err != nil {
		return err
	}
	var req checkAccessRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	result, err := usecases.CheckAccess(c.Request().Context(), d.usecaseDeps(), usecases.CheckAccessInput{
		TenantID:           support.RequestTenantID(c),
		Resource:           domain.ObjectRef{Type: req.ResourceType, ID: req.ResourceID},
		Relation:           req.Relation,
		Subject:            domain.SubjectRef{Type: req.SubjectType, ID: req.SubjectID},
		ActorChain:         toActors(req.ActorChain),
		MinimumConsistency: req.MinimumConsistency,
		GrantedScopes:      req.GrantedScopes,
		RequiredScopes:     req.RequiredScopes,
	}, time.Now().UTC())
	if err != nil {
		return writeAuthorizationError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"result": map[string]any{
		"permitted":     result.Permitted,
		"model_version": result.ModelVersion,
		"consistency":   result.Consistency,
		"relation_path": nonNil(result.RelationPath),
		"reasons":       nonNil(result.Reasons),
	}})
}

type listAccessibleResourcesRequest struct {
	ResourceType       string      `json:"resource_type"`
	Relation           string      `json:"relation"`
	SubjectType        string      `json:"subject_type"`
	SubjectID          string      `json:"subject_id"`
	ActorChain         []actorWire `json:"actor_chain"`
	MinimumConsistency string      `json:"minimum_consistency"`
	GrantedScopes      []string    `json:"granted_scopes"`
	RequiredScopes     []string    `json:"required_scopes"`
}

func (d Deps) handleListAccessibleResources(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if err := d.requireAuthorizationAdmin(c); err != nil {
		return err
	}
	var req listAccessibleResourcesRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	listed, err := usecases.ListAccessibleResources(c.Request().Context(), d.usecaseDeps(), usecases.ListAccessibleResourcesInput{
		TenantID: support.RequestTenantID(c), ResourceType: req.ResourceType, Relation: req.Relation,
		Subject:            domain.SubjectRef{Type: req.SubjectType, ID: req.SubjectID},
		ActorChain:         toActors(req.ActorChain),
		MinimumConsistency: req.MinimumConsistency,
		GrantedScopes:      req.GrantedScopes,
		RequiredScopes:     req.RequiredScopes,
	}, time.Now().UTC())
	if err != nil {
		return writeAuthorizationError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"result": map[string]any{
		"resource_ids":  nonNil(listed.ResourceIDs),
		"truncated":     listed.Truncated,
		"model_version": listed.ModelVersion,
		"consistency":   listed.Consistency,
	}})
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

// requireAuthorizationAdmin は AdminAuthorizationModelManage を要する操作の入口。
// 関係の有無そのものが情報になるため、判定エンドポイントも同じ権限で守る。
func (d Deps) requireAuthorizationAdmin(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	return nil
}

func writeAuthorizationError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrModelNotFound):
		return support.WriteProblem(c, http.StatusNotFound, "authorization_model_not_found",
			"No authorization model has been published for this tenant.")
	case errors.Is(err, domain.ErrModelInvalid):
		return support.WriteProblem(c, http.StatusUnprocessableEntity, "authorization_model_invalid", err.Error())
	case errors.Is(err, domain.ErrTupleInvalid):
		return support.WriteProblem(c, http.StatusUnprocessableEntity, "relation_tuple_invalid", err.Error())
	case errors.Is(err, domain.ErrConsistencyNotSatisfied):
		return support.WriteProblem(c, http.StatusUnprocessableEntity, "consistency_not_satisfied",
			"The store has not caught up with the supplied consistency token.")
	default:
		return err
	}
}
