package usecases_test

// docs/standards.md の GDPR-ERASURE のうち、IdManagement が担う UserLifecycle の Purge
// 遷移を観測する。Authentication が担う資格情報の破棄は
// backend/authentication/usecases/credential_erasure_standards_test.go が別に観測する。
// 行が 2 つの Context を名指しているので、片方だけに注記を置くともう片方は素通りする。
//
// 観測は「消去できた」ではなく「消去のあとに読み出せない」側から書く。項目ごとの nil 検査は
// 項目が増えたときに黙って抜けるので、tombstone を丸ごと直列化して、投入した PII の文字列が
// 1 つも現れないことを読む。

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// 投入する PII。消去後に 1 つでも読み出せたら行が成り立っていない。
var erasurePII = []string{
	"erasure-subject",
	"Erasure Subject",
	"Erasure",
	"Subject",
	"erasure.subject@example.com",
	"+81-3-0000-0000",
}

type erasureFixture struct {
	users  *usermemory.UserRepository
	deps   userusecases.AdminUserDeps
	events *[]spec.DomainEvent
}

func newErasureFixture(t *testing.T, now time.Time) *erasureFixture {
	t.Helper()
	users := usermemory.NewUserRepository()
	events := &[]spec.DomainEvent{}
	pointer := func(value string) *string { return &value }
	users.Seed(&userdomain.User{
		ID: "user-erasure", TenantID: tenancydomain.DefaultTenantID,
		PreferredUsername: "erasure-subject", PasswordHash: "$argon2id$seeded",
		Name:       pointer("Erasure Subject"),
		GivenName:  pointer("Erasure"),
		FamilyName: pointer("Subject"),
		Email:      pointer("erasure.subject@example.com"), EmailVerified: true,
		Roles: []string{"support"},
		Attributes: map[string]userdomain.AttributeValue{
			"phone_number": {Type: idmdomain.AttributeTypeString, String: pointer("+81-3-0000-0000")},
		},
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour),
	})
	return &erasureFixture{
		users:  users,
		events: events,
		deps: userusecases.AdminUserDeps{
			UserRepo: users,
			Emit:     func(event spec.DomainEvent) error { *events = append(*events, event); return nil },
		},
	}
}

// readablePII は保存されている User を丸ごと読み直し、まだ読み出せる PII を返す。
func (f *erasureFixture) readablePII(t *testing.T) []string {
	t.Helper()
	stored, err := f.users.FindBySubIncludingDeleted(context.Background(), "user-erasure")
	if err != nil {
		t.Fatal(err)
	}
	if stored == nil {
		return nil
	}
	encoded, err := json.Marshal(stored)
	if err != nil {
		t.Fatal(err)
	}
	found := []string{}
	for _, value := range erasurePII {
		if strings.Contains(string(encoded), value) {
			found = append(found, value)
		}
	}
	return found
}

// GDPR-ERASURE: Purge 遷移を経た User からは、投入した PII をどの経路でも読み出せない。
// 消去の事実そのもの (tombstone と UserDeleted) は法的保存義務の側として残る。
func TestUserPurgeLeavesNoReadablePII(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	fixture := newErasureFixture(t, now)

	// 前提: 消去前は PII が読める。ここが読めないなら、あとの「読めない」は消去の結果ではない。
	if before := fixture.readablePII(t); len(before) != len(erasurePII) {
		t.Fatalf("消去前に読めた PII=%v, want %v", before, erasurePII)
	}

	if err := userusecases.DeleteUser(ctx, fixture.deps, userusecases.DeleteUserInput{
		ActorUserID: "admin", Sub: "user-erasure", Reason: "erasure request", Now: now,
	}); err != nil {
		t.Fatal(err)
	}

	if leaked := fixture.readablePII(t); len(leaked) != 0 {
		t.Fatalf("Purge のあとに読み出せた PII=%v", leaked)
	}
	if seen, _ := fixture.users.FindBySub(ctx, "user-erasure"); seen != nil {
		t.Fatalf("Purge のあとに通常の参照経路が User を返した: %+v", seen)
	}

	// 法的保存義務の側: 消去した事実は残る。tombstone ごと消えると、要求に応じたことを
	// 後から示せない。
	tombstone, err := fixture.users.FindBySubIncludingDeleted(ctx, "user-erasure")
	if err != nil {
		t.Fatal(err)
	}
	if tombstone == nil || tombstone.Lifecycle.Status != idmdomain.UserStatusDeleted {
		t.Fatalf("tombstone=%+v, want status=deleted", tombstone)
	}
	deleted := false
	for _, event := range *fixture.events {
		if event, ok := event.(*idmdomain.UserDeleted); ok && event.TargetUserID == "user-erasure" {
			deleted = true
		}
	}
	if !deleted {
		t.Fatalf("UserDeleted が記録されていない: events=%+v", *fixture.events)
	}
}

// GDPR-ERASURE: 消去は「定義済み期間内に」起きる。削除予約の猶予期間の内側では PII が
// 残り、期間を過ぎると Purge が走って読み出せなくなる。境界の片側だけを見るテストは、
// 予約した瞬間に消す実装とも、永久に消さない実装とも区別できない。
func TestUserErasureHappensWithinTheDefinedGracePeriod(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	fixture := newErasureFixture(t, now)
	grace := time.Duration(userusecases.UserSoftDeleteGracePeriodSeconds) * time.Second

	if err := userusecases.SoftDeleteUser(ctx, fixture.deps, userusecases.SoftDeleteUserInput{
		ActorUserID: "admin", Sub: "user-erasure", Reason: "erasure request", Now: now,
	}); err != nil {
		t.Fatal(err)
	}

	// 期間の内側: sweep を通しても消えない。復元できる間は PII が要る。
	if err := userusecases.PurgeExpiredSoftDeleted(ctx, fixture.deps, now.Add(grace-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if inside := fixture.readablePII(t); len(inside) != len(erasurePII) {
		t.Fatalf("猶予期間の内側で消えた PII がある: 残り=%v, want %v", inside, erasurePII)
	}

	// 期間の外側: 同じ sweep が Purge する。
	if err := userusecases.PurgeExpiredSoftDeleted(ctx, fixture.deps, now.Add(grace+time.Hour)); err != nil {
		t.Fatal(err)
	}
	if leaked := fixture.readablePII(t); len(leaked) != 0 {
		t.Fatalf("猶予期間を過ぎても読み出せた PII=%v", leaked)
	}
}
