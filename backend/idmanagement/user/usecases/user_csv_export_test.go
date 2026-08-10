package usecases

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
)

type exportArtifactStore struct {
	content []byte
}

func (s *exportArtifactStore) PutUserCSVArtifact(_ context.Context, tenantID string, write func(io.Writer) error) (userports.UserCSVArtifact, error) {
	var content bytes.Buffer
	if err := write(&content); err != nil {
		return userports.UserCSVArtifact{}, err
	}
	s.content = append([]byte(nil), content.Bytes()...)
	digest := sha256.Sum256(s.content)
	return userports.UserCSVArtifact{Ref: "artifact-1", TenantID: tenantID, SHA256: hex.EncodeToString(digest[:]), ByteSize: int64(len(s.content))}, nil
}

func (s *exportArtifactStore) OpenUserCSVArtifact(_ context.Context, tenantID, ref string) (io.ReadCloser, userports.UserCSVArtifact, error) {
	digest := sha256.Sum256(s.content)
	return io.NopCloser(bytes.NewReader(s.content)), userports.UserCSVArtifact{Ref: ref, TenantID: tenantID, SHA256: hex.EncodeToString(digest[:]), ByteSize: int64(len(s.content))}, nil
}

func (s *exportArtifactStore) PutUserCSVArtifactPages(context.Context, string, func(func([]byte) error) error) (userports.UserCSVArtifact, error) {
	return userports.UserCSVArtifact{}, errors.New("not implemented")
}

func (s *exportArtifactStore) ReadUserCSVArtifactPage(context.Context, string, string, int) ([]byte, userports.UserCSVArtifact, error) {
	return nil, userports.UserCSVArtifact{}, errors.New("not implemented")
}

func TestExportUserCSVTenThousandRowsRoundTripAsUnchanged(t *testing.T) {
	ctx := importPlannerContext()
	repo := usermemory.NewUserRepository()
	createdAt := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	for i := range 10_000 {
		username := fmt.Sprintf("user-%05d", i)
		name := username
		if i == 0 {
			name = "'=SUM(1,1)\n既存'apostrophe"
		}
		department := "Engineering"
		repo.Seed(&userdomain.User{
			ID: fmt.Sprintf("00000000-0000-4000-8000-%012d", i), TenantID: "acme",
			PreferredUsername: username, PasswordHash: "hash", Name: &name,
			EmailVerified: i%2 == 0, Roles: []string{"member", "support"},
			Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive, RequiredActions: []idmdomain.RequiredAction{idmdomain.RequiredActionUpdatePassword}},
			Attributes: map[string]userdomain.AttributeValue{
				"department": {Type: idmdomain.AttributeTypeString, String: &department},
			},
			CreatedAt: createdAt, UpdatedAt: createdAt,
		})
	}
	defs := []userdomain.UserAttributeDef{{Key: "department", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityPrivate}}
	artifacts := &exportArtifactStore{}
	result, err := ExportUserCSV(ctx, UserCSVExportDeps{
		UserRepo: repo, SchemaReader: importSchemaReader{defs: defs}, Artifacts: artifacts,
	}, nil, "", userdomain.DefaultUserCSVTransferPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalRows != 10_000 || result.Artifact.Ref == "" || result.Artifact.ByteSize != int64(len(artifacts.content)) {
		t.Fatalf("result=%+v bytes=%d", result, len(artifacts.content))
	}

	var summary userdomain.UserImportPlanSummary
	_, err = PlanUserImport(ctx, UserImportPlanDeps{
		UserRepo: repo, SchemaReader: importSchemaReader{defs: defs}, OwnershipGuard: perUserImportOwnershipGuard{},
	}, bytes.NewReader(artifacts.content), userdomain.DefaultUserCSVTransferPolicy(), func(row userdomain.UserImportRowPlan) error {
		summary.Observe(row)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalRows != 10_000 || summary.UnchangedRows != 10_000 || summary.RejectedRows != 0 {
		t.Fatalf("summary=%+v", summary)
	}
	if !bytes.Contains(artifacts.content, []byte("''=SUM(1,1)")) {
		t.Fatalf("formula-safe apostrophe encoding missing from CSV prefix: %q", artifacts.content[:min(len(artifacts.content), 512)])
	}
}

func TestExportUserCSVUsesRequestedMachineColumnsAndPolicy(t *testing.T) {
	ctx := importPlannerContext()
	repo := usermemory.NewUserRepository()
	user := importPlannerUser("user-alice", "alice")
	repo.Seed(user)
	artifacts := &exportArtifactStore{}
	policy := userdomain.DefaultUserCSVTransferPolicy()
	result, err := ExportUserCSV(ctx, UserCSVExportDeps{
		UserRepo:     repo,
		SchemaReader: importSchemaReader{defs: []userdomain.UserAttributeDef{{Key: "department", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityPrivate}}},
		Artifacts:    artifacts,
	}, []string{"id", "preferred_username", "required_actions", "attr:department"}, "active", policy)
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalRows != 1 {
		t.Fatalf("result=%+v", result)
	}
	if got := string(artifacts.content); got != "id,preferred_username,required_actions,attr:department\nuser-alice,alice,,Old\n" {
		t.Fatalf("csv=%q", got)
	}

	policy.MaxRows = 1
	repo.Seed(importPlannerUser("user-bob", "bob"))
	_, err = ExportUserCSV(ctx, UserCSVExportDeps{
		UserRepo:     repo,
		SchemaReader: importSchemaReader{defs: []userdomain.UserAttributeDef{{Key: "department", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityPrivate}}},
		Artifacts:    &exportArtifactStore{},
	}, []string{"id", "preferred_username"}, "", policy)
	var csvErr *userdomain.UserCSVError
	if !errors.As(err, &csvErr) || csvErr.Code != userdomain.UserCSVErrorTooManyRows {
		t.Fatalf("err=%v", err)
	}
}
