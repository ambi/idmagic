package db_postgres

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	agentpg "github.com/ambi/idmagic/backend/idmanagement/agent/db_postgres"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userpg "github.com/ambi/idmagic/backend/idmanagement/user/db_postgres"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"

	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	tenancypg "github.com/ambi/idmagic/backend/tenancy/db_postgres"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func testClock() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }

var idSeq atomic.Uint64

func uniqueID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, idSeq.Add(1))
}

func newUUID(t *testing.T) string {
	t.Helper()
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatalf("new uuid: %v", err)
	}
	return id
}

func seedTenant(t *testing.T, db sharedpg.DB) *tenancydomain.Tenant {
	t.Helper()
	now := testClock()
	tenant := &tenancydomain.Tenant{
		ID: newUUID(t), Realm: uniqueID("tenant"), DisplayName: "Test Tenant",
		Status: tenancydomain.TenantStatusActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := (&tenancypg.TenantRepository{Pool: db}).Save(context.Background(), tenant); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	return tenant
}

func seedAgent(t *testing.T, db sharedpg.DB, tenantID string) *agentdomain.Agent {
	t.Helper()
	now := testClock()
	user := &userdomain.User{
		ID: newUUID(t), TenantID: tenantID, PreferredUsername: uniqueID("username"),
		PasswordHash: "hash", Roles: []string{}, CreatedAt: now, UpdatedAt: now,
	}
	if err := (&userpg.UserRepository{Pool: db}).Save(context.Background(), user); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	agent := &agentdomain.Agent{
		ID: newUUID(t), TenantID: tenantID, Name: uniqueID("agent"), Kind: idmdomain.AgentKindAutonomous,
		OwnerUserID: user.ID, Status: idmdomain.AgentStatusActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := (&agentpg.AgentRepository{Pool: db}).Save(context.Background(), agent); err != nil {
		t.Fatalf("seed agent: %v", err)
	}
	return agent
}

func newStream(t *testing.T, tenantID string, direction ssdomain.SsfStreamDirection) *ssdomain.SsfStream {
	t.Helper()
	now := testClock()
	return &ssdomain.SsfStream{
		ID: newUUID(t), TenantID: tenantID, Direction: direction,
		EventTypes: []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked},
		Status:     ssdomain.SsfStreamStatusEnabled, CreatedAt: now,
	}
}

func newSecurityEventToken(agentID, tenantID string) ssdomain.SecurityEventToken {
	now := testClock()
	return ssdomain.SecurityEventToken{
		JTI: uniqueID("jti"), Issuer: "https://idmagic.example", Audience: "https://receiver.example",
		IssuedAt: now,
		Event: ssdomain.CaepEvent{
			EventType: ssdomain.CaepEventTypeSessionRevoked,
			Subject: ssdomain.SsfSubject{
				SubjectType: ssdomain.SsfSubjectTypeAgent, TenantID: tenantID, PrincipalID: agentID,
			},
			EventTimestamp:   now,
			InitiatingEntity: ssdomain.InitiatingEntityAdmin,
		},
		Compact: "eyJ.test.compact",
	}
}
