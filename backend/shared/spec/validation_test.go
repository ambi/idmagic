package spec_test

// go test only instruments the package under test, so exercising these
// wrappers via a domain struct's own test (in a different package) does not
// count toward backend/shared/spec's coverage. These tests call the shared
// spec.Validate* wrappers directly against the same domain structs.

import (
	"testing"
	"time"

	approvaldomain "github.com/ambi/idmagic/backend/oauth2/approval/domain"
	authdomain "github.com/ambi/idmagic/backend/oauth2/authorization/domain"
	devicedomain "github.com/ambi/idmagic/backend/oauth2/device/domain"
	tokendomain "github.com/ambi/idmagic/backend/oauth2/token/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func mustUUID(t *testing.T) string {
	t.Helper()
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestValidateAuthorizationRequest(t *testing.T) {
	now := time.Now().UTC()
	valid := authdomain.AuthorizationRequest{
		ID: mustUUID(t), State: spec.AuthFlowReceived, ClientID: "client-1",
		RedirectURI: "https://client.example/callback", ResponseType: spec.ResponseTypeCode,
		CodeChallenge: "challenge", CodeChallengeMethod: spec.CodeChallengeMethodS256,
		CreatedAt: now, ExpiresAt: now.Add(time.Minute),
	}
	if err := spec.ValidateAuthorizationRequest(&valid); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
	invalid := valid
	invalid.State = "bogus"
	if err := spec.ValidateAuthorizationRequest(&invalid); err == nil {
		t.Fatal("expected rejection for invalid state")
	}
}

func TestValidateAuthorizationCodeRecord(t *testing.T) {
	now := time.Now().UTC()
	valid := authdomain.AuthorizationCodeRecord{
		Code: "code-1", AuthorizationRequestID: mustUUID(t), ClientID: "client-1", UserID: "user-1",
		RedirectURI: "https://client.example/callback", CodeChallenge: "challenge",
		CodeChallengeMethod: spec.CodeChallengeMethodS256, State: spec.AuthCodeRecordIssued,
		IssuedAt: now, ExpiresAt: now.Add(time.Minute),
	}
	if err := spec.ValidateAuthorizationCodeRecord(&valid); err != nil {
		t.Fatalf("valid record rejected: %v", err)
	}
	invalid := valid
	invalid.CodeChallengeMethod = "plain"
	if err := spec.ValidateAuthorizationCodeRecord(&invalid); err == nil {
		t.Fatal("expected rejection for invalid code_challenge_method")
	}
}

func TestValidateRefreshTokenRecord(t *testing.T) {
	now := time.Now().UTC()
	valid := tokendomain.RefreshTokenRecord{
		ID: mustUUID(t), Hash: "hash", FamilyID: mustUUID(t), ClientID: "client-1", UserID: "user-1",
		IssuedAt: now, ExpiresAt: now.Add(time.Minute), AbsoluteExpiresAt: now.Add(time.Hour),
	}
	if err := spec.ValidateRefreshTokenRecord(&valid); err != nil {
		t.Fatalf("valid record rejected: %v", err)
	}
	invalid := valid
	invalid.ID = "not-a-uuid"
	if err := spec.ValidateRefreshTokenRecord(&invalid); err == nil {
		t.Fatal("expected rejection for a non-UUID id")
	}
}

func TestValidatePARRecord(t *testing.T) {
	now := time.Now().UTC()
	valid := authdomain.PARRecord{
		RequestURI: "urn:ietf:params:oauth:request_uri:abc", ClientID: "client-1",
		IssuedAt: now, ExpiresAt: now.Add(time.Minute),
	}
	if err := spec.ValidatePARRecord(&valid); err != nil {
		t.Fatalf("valid record rejected: %v", err)
	}
	invalid := valid
	invalid.ClientID = ""
	if err := spec.ValidatePARRecord(&invalid); err == nil {
		t.Fatal("expected rejection for a missing client_id")
	}
}

func TestValidateDeviceAuthorization(t *testing.T) {
	now := time.Now().UTC()
	valid := devicedomain.DeviceAuthorization{
		DeviceCodeHash: "hash", UserCode: "USER-CODE", ClientID: "client-1",
		State: spec.DeviceFlowIssued, IntervalSeconds: 5, IssuedAt: now, ExpiresAt: now.Add(time.Minute),
	}
	if err := spec.ValidateDeviceAuthorization(&valid); err != nil {
		t.Fatalf("valid record rejected: %v", err)
	}
	invalid := valid
	invalid.IntervalSeconds = 0
	if err := spec.ValidateDeviceAuthorization(&invalid); err == nil {
		t.Fatal("expected rejection for a non-positive interval")
	}
}

func TestValidateApprovalRequest(t *testing.T) {
	now := time.Now().UTC()
	valid := approvaldomain.ApprovalRequest{
		ID: mustUUID(t), ClientID: "client-1", UserID: "user-1", AuthReqIDHash: "hash",
		State: spec.ApprovalPending, IntervalSeconds: 5, RequestedAt: now, ExpiresAt: now.Add(time.Minute),
	}
	if err := spec.ValidateApprovalRequest(&valid); err != nil {
		t.Fatalf("valid record rejected: %v", err)
	}
	invalid := valid
	invalid.IntervalSeconds = -1
	if err := spec.ValidateApprovalRequest(&invalid); err == nil {
		t.Fatal("expected rejection for a negative interval")
	}
}
