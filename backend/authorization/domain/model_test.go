package domain_test

import (
	"errors"
	"testing"

	"github.com/ambi/idmagic/backend/authorization/domain"
)

func direct(types ...string) domain.RelationRewrite {
	return domain.RelationRewrite{Kind: domain.RewriteDirect, DirectSubjectTypes: types}
}

// REQ-AUTHORIZATION-001: 整合しないモデルは登録時点で拒否する。
func TestValidateRejectsUnknownRelation(t *testing.T) {
	cases := []struct {
		name  string
		types []domain.ResourceTypeDefinition
	}{
		{"unknown computed relation", []domain.ResourceTypeDefinition{
			{Name: "user"},
			{Name: "document", Relations: []domain.RelationDefinition{
				{Name: "viewer", Rewrites: []domain.RelationRewrite{
					{Kind: domain.RewriteComputedUserset, ComputedRelation: "editor"},
				}},
			}},
		}},
		{"unknown tupleset relation", []domain.ResourceTypeDefinition{
			{Name: "user"},
			{Name: "document", Relations: []domain.RelationDefinition{
				{Name: "viewer", Rewrites: []domain.RelationRewrite{
					{Kind: domain.RewriteTupleToUserset, TuplesetRelation: "parent", ComputedRelation: "viewer"},
				}},
			}},
		}},
		{"unknown direct subject type", []domain.ResourceTypeDefinition{
			{Name: "document", Relations: []domain.RelationDefinition{
				{Name: "viewer", Rewrites: []domain.RelationRewrite{direct("user")}},
			}},
		}},
		{"unknown subject set relation", []domain.ResourceTypeDefinition{
			{Name: "user"},
			{Name: "group"},
			{Name: "document", Relations: []domain.RelationDefinition{
				{Name: "viewer", Rewrites: []domain.RelationRewrite{direct("group#member")}},
			}},
		}},
		{"malformed type name", []domain.ResourceTypeDefinition{
			{Name: "Document", Relations: []domain.RelationDefinition{
				{Name: "viewer", Rewrites: []domain.RelationRewrite{direct("Document")}},
			}},
		}},
		{"relation without a rewrite", []domain.ResourceTypeDefinition{
			{Name: "user"},
			{Name: "document", Relations: []domain.RelationDefinition{{Name: "viewer"}}},
		}},
		{"computed userset cycle", []domain.ResourceTypeDefinition{
			{Name: "user"},
			{Name: "document", Relations: []domain.RelationDefinition{
				{Name: "viewer", Rewrites: []domain.RelationRewrite{
					{Kind: domain.RewriteComputedUserset, ComputedRelation: "editor"},
				}},
				{Name: "editor", Rewrites: []domain.RelationRewrite{
					{Kind: domain.RewriteComputedUserset, ComputedRelation: "viewer"},
				}},
			}},
		}},
		{"duplicate relation", []domain.ResourceTypeDefinition{
			{Name: "user"},
			{Name: "document", Relations: []domain.RelationDefinition{
				{Name: "viewer", Rewrites: []domain.RelationRewrite{direct("user")}},
				{Name: "viewer", Rewrites: []domain.RelationRewrite{direct("user")}},
			}},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			model := &domain.AuthorizationModel{TenantID: "tenant-a", Version: 1, ResourceTypes: tc.types}
			if err := model.Validate(); !errors.Is(err, domain.ErrModelInvalid) {
				t.Fatalf("Validate() = %v, want %v", err, domain.ErrModelInvalid)
			}
		})
	}
}

func TestValidateAcceptsTheReferenceModel(t *testing.T) {
	if err := documentModel().Validate(); err != nil {
		t.Fatalf("the reference model must validate: %v", err)
	}
}

// REQ-AUTHORIZATION-002: モデルに適合しないタプルは書き込みを拒否する。
func TestValidateTupleRejectsFormsTheModelDoesNotDeclare(t *testing.T) {
	model := documentModel()
	cases := []struct {
		name  string
		tuple domain.RelationTuple
	}{
		{"unknown relation", domain.RelationTuple{
			Resource: domain.ObjectRef{Type: "document", ID: "d1"}, Relation: "shredder", Subject: user("alice"),
		}},
		{"unknown resource type", domain.RelationTuple{
			Resource: domain.ObjectRef{Type: "spreadsheet", ID: "s1"}, Relation: "viewer", Subject: user("alice"),
		}},
		{"undeclared subject type", domain.RelationTuple{
			Resource: domain.ObjectRef{Type: "document", ID: "d1"}, Relation: "editor",
			Subject: domain.SubjectRef{Type: "group", ID: "eng", Relation: "member"},
		}},
		{"wildcard where the relation does not declare one", domain.RelationTuple{
			Resource: domain.ObjectRef{Type: "document", ID: "d1"}, Relation: "editor",
			Subject: domain.SubjectRef{Type: "user", ID: domain.Wildcard},
		}},
		{"wildcard resource", domain.RelationTuple{
			Resource: domain.ObjectRef{Type: "document", ID: domain.Wildcard}, Relation: "viewer", Subject: user("alice"),
		}},
		{"identifier carrying a separator", domain.RelationTuple{
			Resource: domain.ObjectRef{Type: "document", ID: "d1#viewer"}, Relation: "viewer", Subject: user("alice"),
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := model.ValidateTuple(tc.tuple); !errors.Is(err, domain.ErrTupleInvalid) {
				t.Fatalf("ValidateTuple() = %v, want %v", err, domain.ErrTupleInvalid)
			}
		})
	}
}

func TestValidateTupleAcceptsDeclaredForms(t *testing.T) {
	model := documentModel()
	accepted := []domain.RelationTuple{
		{Resource: domain.ObjectRef{Type: "document", ID: "d1"}, Relation: "viewer", Subject: user("alice")},
		{Resource: domain.ObjectRef{Type: "document", ID: "d1"}, Relation: "viewer", Subject: domain.SubjectRef{Type: "user", ID: domain.Wildcard}},
		{Resource: domain.ObjectRef{Type: "document", ID: "d1"}, Relation: "viewer", Subject: domain.SubjectRef{Type: "group", ID: "eng", Relation: "member"}},
		{Resource: domain.ObjectRef{Type: "document", ID: "d1"}, Relation: "parent", Subject: domain.SubjectRef{Type: "folder", ID: "f1"}},
	}
	for _, tuple := range accepted {
		if err := model.ValidateTuple(tuple); err != nil {
			t.Fatalf("ValidateTuple(%s) = %v, want nil", tuple, err)
		}
	}
}

// REQ-AUTHORIZATION-006: 整合トークンは発行テナントに束縛される。
func TestConsistencyTokenIsBoundToItsTenant(t *testing.T) {
	token := domain.EncodeConsistencyToken("tenant-a", 7)
	version, err := domain.DecodeConsistencyToken(token, "tenant-a")
	if err != nil || version != 7 {
		t.Fatalf("DecodeConsistencyToken() = (%d, %v), want (7, nil)", version, err)
	}
	if _, err := domain.DecodeConsistencyToken(token, "tenant-b"); !errors.Is(err, domain.ErrConsistencyNotSatisfied) {
		t.Fatalf("a token from another tenant must be rejected, got %v", err)
	}
	if _, err := domain.DecodeConsistencyToken("not a token", "tenant-a"); !errors.Is(err, domain.ErrConsistencyNotSatisfied) {
		t.Fatalf("a malformed token must be rejected, got %v", err)
	}
}
