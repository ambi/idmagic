package domain_test

import (
	"errors"
	"testing"

	accessdomain "github.com/ambi/idmagic/backend/accesscontrol/internal/domain"
)

func direct(types ...string) accessdomain.RelationRewrite {
	return accessdomain.RelationRewrite{Kind: accessdomain.RewriteDirect, DirectSubjectTypes: types}
}

//spec:covers REQ-AUTHORIZATION-001: 整合しないモデルは登録時点で拒否する。
func TestValidateRejectsUnknownRelation(t *testing.T) {
	cases := []struct {
		name  string
		types []accessdomain.ResourceTypeDefinition
	}{
		{"unknown computed relation", []accessdomain.ResourceTypeDefinition{
			{Name: "user"},
			{Name: "document", Relations: []accessdomain.RelationDefinition{
				{Name: "viewer", Rewrites: []accessdomain.RelationRewrite{
					{Kind: accessdomain.RewriteComputedUserset, ComputedRelation: "editor"},
				}},
			}},
		}},
		{"unknown tupleset relation", []accessdomain.ResourceTypeDefinition{
			{Name: "user"},
			{Name: "document", Relations: []accessdomain.RelationDefinition{
				{Name: "viewer", Rewrites: []accessdomain.RelationRewrite{
					{Kind: accessdomain.RewriteTupleToUserset, TuplesetRelation: "parent", ComputedRelation: "viewer"},
				}},
			}},
		}},
		{"unknown direct subject type", []accessdomain.ResourceTypeDefinition{
			{Name: "document", Relations: []accessdomain.RelationDefinition{
				{Name: "viewer", Rewrites: []accessdomain.RelationRewrite{direct("user")}},
			}},
		}},
		{"unknown subject set relation", []accessdomain.ResourceTypeDefinition{
			{Name: "user"},
			{Name: "group"},
			{Name: "document", Relations: []accessdomain.RelationDefinition{
				{Name: "viewer", Rewrites: []accessdomain.RelationRewrite{direct("group#member")}},
			}},
		}},
		{"malformed type name", []accessdomain.ResourceTypeDefinition{
			{Name: "Document", Relations: []accessdomain.RelationDefinition{
				{Name: "viewer", Rewrites: []accessdomain.RelationRewrite{direct("Document")}},
			}},
		}},
		{"relation without a rewrite", []accessdomain.ResourceTypeDefinition{
			{Name: "user"},
			{Name: "document", Relations: []accessdomain.RelationDefinition{{Name: "viewer"}}},
		}},
		{"computed userset cycle", []accessdomain.ResourceTypeDefinition{
			{Name: "user"},
			{Name: "document", Relations: []accessdomain.RelationDefinition{
				{Name: "viewer", Rewrites: []accessdomain.RelationRewrite{
					{Kind: accessdomain.RewriteComputedUserset, ComputedRelation: "editor"},
				}},
				{Name: "editor", Rewrites: []accessdomain.RelationRewrite{
					{Kind: accessdomain.RewriteComputedUserset, ComputedRelation: "viewer"},
				}},
			}},
		}},
		{"duplicate relation", []accessdomain.ResourceTypeDefinition{
			{Name: "user"},
			{Name: "document", Relations: []accessdomain.RelationDefinition{
				{Name: "viewer", Rewrites: []accessdomain.RelationRewrite{direct("user")}},
				{Name: "viewer", Rewrites: []accessdomain.RelationRewrite{direct("user")}},
			}},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			model := &accessdomain.AuthorizationModel{TenantID: "tenant-a", Version: 1, ResourceTypes: tc.types}
			if err := model.Validate(); !errors.Is(err, accessdomain.ErrModelInvalid) {
				t.Fatalf("Validate() = %v, want %v", err, accessdomain.ErrModelInvalid)
			}
		})
	}
}

func TestValidateAcceptsTheReferenceModel(t *testing.T) {
	if err := documentModel().Validate(); err != nil {
		t.Fatalf("the reference model must validate: %v", err)
	}
}

//spec:covers REQ-AUTHORIZATION-002: モデルに適合しないタプルは書き込みを拒否する。
func TestValidateTupleRejectsFormsTheModelDoesNotDeclare(t *testing.T) {
	model := documentModel()
	cases := []struct {
		name  string
		tuple accessdomain.RelationTuple
	}{
		{"unknown relation", accessdomain.RelationTuple{
			Resource: accessdomain.ObjectRef{Type: "document", ID: "d1"}, Relation: "shredder", Subject: user("alice"),
		}},
		{"unknown resource type", accessdomain.RelationTuple{
			Resource: accessdomain.ObjectRef{Type: "spreadsheet", ID: "s1"}, Relation: "viewer", Subject: user("alice"),
		}},
		{"undeclared subject type", accessdomain.RelationTuple{
			Resource: accessdomain.ObjectRef{Type: "document", ID: "d1"}, Relation: "editor",
			Subject: accessdomain.SubjectRef{Type: "group", ID: "eng", Relation: "member"},
		}},
		{"wildcard where the relation does not declare one", accessdomain.RelationTuple{
			Resource: accessdomain.ObjectRef{Type: "document", ID: "d1"}, Relation: "editor",
			Subject: accessdomain.SubjectRef{Type: "user", ID: accessdomain.Wildcard},
		}},
		{"wildcard resource", accessdomain.RelationTuple{
			Resource: accessdomain.ObjectRef{Type: "document", ID: accessdomain.Wildcard}, Relation: "viewer", Subject: user("alice"),
		}},
		{"identifier carrying a separator", accessdomain.RelationTuple{
			Resource: accessdomain.ObjectRef{Type: "document", ID: "d1#viewer"}, Relation: "viewer", Subject: user("alice"),
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := model.ValidateTuple(tc.tuple); !errors.Is(err, accessdomain.ErrTupleInvalid) {
				t.Fatalf("ValidateTuple() = %v, want %v", err, accessdomain.ErrTupleInvalid)
			}
		})
	}
}

func TestValidateTupleAcceptsDeclaredForms(t *testing.T) {
	model := documentModel()
	accepted := []accessdomain.RelationTuple{
		{Resource: accessdomain.ObjectRef{Type: "document", ID: "d1"}, Relation: "viewer", Subject: user("alice")},
		{Resource: accessdomain.ObjectRef{Type: "document", ID: "d1"}, Relation: "viewer", Subject: accessdomain.SubjectRef{Type: "user", ID: accessdomain.Wildcard}},
		{Resource: accessdomain.ObjectRef{Type: "document", ID: "d1"}, Relation: "viewer", Subject: accessdomain.SubjectRef{Type: "group", ID: "eng", Relation: "member"}},
		{Resource: accessdomain.ObjectRef{Type: "document", ID: "d1"}, Relation: "parent", Subject: accessdomain.SubjectRef{Type: "folder", ID: "f1"}},
	}
	for _, tuple := range accepted {
		if err := model.ValidateTuple(tuple); err != nil {
			t.Fatalf("ValidateTuple(%s) = %v, want nil", tuple, err)
		}
	}
}

//spec:covers REQ-AUTHORIZATION-006: 整合トークンは発行テナントに束縛される。
func TestConsistencyTokenIsBoundToItsTenant(t *testing.T) {
	token := accessdomain.EncodeConsistencyToken("tenant-a", 7)
	version, err := accessdomain.DecodeConsistencyToken(token, "tenant-a")
	if err != nil || version != 7 {
		t.Fatalf("DecodeConsistencyToken() = (%d, %v), want (7, nil)", version, err)
	}
	if _, err := accessdomain.DecodeConsistencyToken(token, "tenant-b"); !errors.Is(err, accessdomain.ErrConsistencyNotSatisfied) {
		t.Fatalf("a token from another tenant must be rejected, got %v", err)
	}
	if _, err := accessdomain.DecodeConsistencyToken("not a token", "tenant-a"); !errors.Is(err, accessdomain.ErrConsistencyNotSatisfied) {
		t.Fatalf("a malformed token must be rejected, got %v", err)
	}
}
