package main

import (
	"context"
	"slices"
	"testing"
	"time"

	auditports "github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/cmd/internal/bootstrap"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

const (
	xmlProfileScope = "profile-x"
	freshScope      = "profile-fresh"
)

// xmlLifecycleWorld は、本番と同じ組み立てで作ったメモリの構成と、組み立ての直後に置いた鍵である。
// tenant-a の default と profile-x のスコープには、作成から 100 日たった有効な XML 鍵と、
// 猶予期間を過ぎた旧鍵がある。profile-fresh のスコープの有効な鍵は今作ったものである。
// tenant-b は XML 鍵を持たない。
type xmlLifecycleWorld struct {
	deps             *bootstrap.Dependencies
	tenantA, tenantB context.Context
	active, expired  map[string]string // スコープごとの kid
}

func newXMLLifecycleWorld(t *testing.T) (*xmlLifecycleWorld, launch) {
	t.Helper()
	w := &xmlLifecycleWorld{active: map[string]string{}, expired: map[string]string{}}
	in := launch{
		loader: bootstrap.NewConfigLoader(func(string) string { return "" }),
		assemble: func(ctx context.Context, cfg bootstrap.SharedConfig) (*bootstrap.Dependencies, error) {
			deps, err := bootstrap.Assemble(ctx, cfg)
			if err != nil {
				return nil, err
			}
			w.deps = deps
			tenantA := w.saveTenant(ctx, t, "tenant-a")
			w.tenantA, w.tenantB = tenantA, w.saveTenant(ctx, t, "tenant-b")
			for _, scope := range []string{signingports.DefaultKeyScope, xmlProfileScope} {
				w.seedAgedKeys(tenantA, t, scope)
			}
			if _, err := deps.SigningKeys.KeyStore.GetActiveKey(w.xml(tenantA, freshScope)); err != nil {
				t.Fatal(err)
			}
			return deps, nil
		},
	}
	return w, in
}

func (w *xmlLifecycleWorld) saveTenant(ctx context.Context, t *testing.T, realm string) context.Context {
	t.Helper()
	now := time.Now().UTC()
	tenant := &tenancydomain.Tenant{ID: realm + "-id", Realm: realm, DisplayName: realm, Status: tenancydomain.TenantStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := w.deps.Tenancy.TenantRepo.Save(ctx, tenant); err != nil {
		t.Fatal(err)
	}
	return tenantports.WithTenant(ctx, tenant, "", "")
}

// seedAgedKeys は、tenantCtx のテナントで、管理者の準備と確認付きの切り替えを 100 日前に行った状態を作る。
func (w *xmlLifecycleWorld) seedAgedKeys(tenantCtx context.Context, t *testing.T, scope string) {
	t.Helper()
	ctx := w.xml(tenantCtx, scope)
	store := w.deps.SigningKeys.KeyStore
	first, err := store.GetActiveKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	staged := store.(signingports.FederationKeyStore)
	aged := time.Now().UTC().Add(-100 * 24 * time.Hour)
	prepared, err := staged.Prepare(ctx, aged)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := staged.Activate(ctx, prepared.Next.Kid, aged, 7*24*time.Hour); err != nil {
		t.Fatal(err)
	}
	w.active[scope], w.expired[scope] = prepared.Next.Kid, first.Kid
}

func (w *xmlLifecycleWorld) xml(ctx context.Context, scope string) context.Context {
	return signingports.WithKeyScope(signingports.WithKeyUsage(ctx, signingdomain.KeyUsageXMLFederationSigning), scope)
}

func (w *xmlLifecycleWorld) keys(t *testing.T, scope string) []*signingdomain.SigningKey {
	t.Helper()
	keys, err := w.deps.SigningKeys.KeyStore.GetAllKeys(w.xml(w.tenantA, scope))
	if err != nil {
		t.Fatal(err)
	}
	return keys
}

func (w *xmlLifecycleWorld) published(t *testing.T, scope string) []string {
	t.Helper()
	keys, err := w.deps.SigningKeys.KeyStore.ListPublicKeys(w.xml(w.tenantA, scope), time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	kids := []string{}
	for _, key := range keys {
		kids = append(kids, key.Kid)
	}
	return kids
}

func (w *xmlLifecycleWorld) auditedKids(t *testing.T, eventType string) []string {
	t.Helper()
	records, err := w.deps.Audit.AuditEventRepo.List(w.tenantA, auditports.AuditEventQuery{TenantID: "tenant-a-id", Type: eventType})
	if err != nil {
		t.Fatal(err)
	}
	kids := []string{}
	for _, record := range records {
		kid, _ := record.Payload["kid"].(string)
		kids = append(kids, kid)
	}
	return kids
}

func runLifecycle(t *testing.T, in launch) {
	t.Helper()
	if err := run(t.Context(), []string{"signing-key-lifecycle", "--cadence-days=90", "--grace-days=7"}, in); err != nil {
		t.Fatal(err)
	}
}

//spec:covers EX-SIGNINGKEYS-003-02: 周期を過ぎた XML 鍵は、default と専用プロファイルのスコープでバッチの実行により準備されてメタデータに公開され、有効な署名鍵は変わらず、SigningKeyPrepared が監査に残り、XML 鍵のないテナントには鍵ができず、再実行では鍵もイベントも増えない。
//spec:covers REQ-SIGNINGKEYS-003: 作成から周期前の XML 鍵を持つスコープには、バッチの実行で鍵を準備しないこと。
func TestSigningKeyLifecyclePreparesDueXMLKeysWithoutActivating(t *testing.T) {
	w, in := newXMLLifecycleWorld(t)
	runLifecycle(t, in)

	var preparedKids []string
	for _, scope := range []string{signingports.DefaultKeyScope, xmlProfileScope} {
		keys := w.keys(t, scope)
		active, prepared := signingdomain.ActiveKey(keys), signingdomain.PreparedKey(keys)
		if active == nil || active.Kid != w.active[scope] {
			t.Fatalf("%s: active key changed, want %s kept", scope, w.active[scope])
		}
		if prepared == nil || prepared.Kid == w.active[scope] {
			t.Fatalf("%s: no new prepared key", scope)
		}
		if !slices.Contains(w.published(t, scope), prepared.Kid) {
			t.Fatalf("%s: prepared key %s is not published", scope, prepared.Kid)
		}
		preparedKids = append(preparedKids, prepared.Kid)
	}
	audited := w.auditedKids(t, "SigningKeyPrepared")
	slices.Sort(audited)
	slices.Sort(preparedKids)
	if !slices.Equal(audited, preparedKids) {
		t.Fatalf("SigningKeyPrepared kids = %v, want %v", audited, preparedKids)
	}
	if prepared := signingdomain.PreparedKey(w.keys(t, freshScope)); prepared != nil {
		t.Fatalf("%s: a key before the cadence was prepared: %s", freshScope, prepared.Kid)
	}
	if scopes, err := w.deps.SigningKeys.KeyStore.KeyScopes(w.xml(w.tenantB, signingports.DefaultKeyScope)); err != nil || len(scopes) != 0 {
		t.Fatalf("tenant-b XML scopes = %v %v, want none", scopes, err)
	}

	before := len(w.keys(t, signingports.DefaultKeyScope)) + len(w.keys(t, xmlProfileScope))
	runLifecycle(t, in)
	if after := len(w.keys(t, signingports.DefaultKeyScope)) + len(w.keys(t, xmlProfileScope)); after != before {
		t.Fatalf("rerun changed the key count from %d to %d", before, after)
	}
	if again := w.auditedKids(t, "SigningKeyPrepared"); len(again) != len(preparedKids) {
		t.Fatalf("rerun emitted SigningKeyPrepared again: %v", again)
	}
}

//spec:covers REQ-SIGNINGKEYS-002: 猶予期間を過ぎた XML 鍵は、default と専用プロファイルのスコープでバッチの実行によりメタデータの公開から外れ、鍵ごとに SigningKeyArchived が監査に残ること。
func TestSigningKeyLifecycleArchivesExpiredXMLKeysInEveryScope(t *testing.T) {
	w, in := newXMLLifecycleWorld(t)
	runLifecycle(t, in)

	var expired []string
	for _, scope := range []string{signingports.DefaultKeyScope, xmlProfileScope} {
		if slices.Contains(w.published(t, scope), w.expired[scope]) {
			t.Fatalf("%s: expired key %s is still published", scope, w.expired[scope])
		}
		expired = append(expired, w.expired[scope])
	}
	audited := w.auditedKids(t, "SigningKeyArchived")
	slices.Sort(audited)
	slices.Sort(expired)
	if !slices.Equal(audited, expired) {
		t.Fatalf("SigningKeyArchived kids = %v, want %v", audited, expired)
	}
}
