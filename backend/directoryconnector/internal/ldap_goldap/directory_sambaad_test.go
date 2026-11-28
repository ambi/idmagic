//go:build sambaad

package ldap_goldap_test

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/go-ldap/ldap/v3"

	directoryconnectordomain "github.com/ambi/idmagic/backend/directoryconnector/internal/domain"
	"github.com/ambi/idmagic/backend/directoryconnector/internal/ldap_goldap"
	"github.com/ambi/idmagic/backend/directoryconnector/internal/testing_sambaad"
)

func sambaDirectory(t *testing.T, server *testing_sambaad.Server) *ldap_goldap.Directory {
	t.Helper()
	directory, err := ldap_goldap.New(ldap_goldap.Config{
		URL: server.LDAPSURL, TLS: server.TLS(t), Dialect: directoryconnectordomain.ActiveDirectory,
		BindDN: testing_sambaad.AdminDN, BindPassword: testing_sambaad.AdminPassword, PageSize: 2, Timeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	return directory
}

// readUsers は、Active Directory の読み取り方で一回の取り込みの計画を立て、その検索の結果を写した項目を返す。
func readUsers(t *testing.T, directory *ldap_goldap.Directory, prev *directoryconnectordomain.Cursor) (directoryconnectordomain.Plan, map[string]directoryconnectordomain.Item) {
	t.Helper()
	session, err := directory.Connect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = session.Close() }()
	root, err := session.RootInfo(t.Context())
	if err != nil {
		t.Fatalf("RootInfo: %v", err)
	}
	scope := directoryconnectordomain.Scope{BaseDNs: []string{testing_sambaad.UsersDN}, Filter: directoryconnectordomain.DefaultUserFilter(directoryconnectordomain.ActiveDirectory)}
	plan := directoryconnectordomain.PlanRun(directoryconnectordomain.ActiveDirectory, scope, prev, root, time.Now(), 24*time.Hour)
	mapping, err := directoryconnectordomain.EffectiveMapping(directoryconnectordomain.ActiveDirectory, nil)
	if err != nil {
		t.Fatal(err)
	}
	items := map[string]directoryconnectordomain.Item{}
	err = session.Search(t.Context(), testing_sambaad.UsersDN, plan.Filter,
		directoryconnectordomain.RequestedAttributes(directoryconnectordomain.ActiveDirectory, mapping),
		func(page []directoryconnectordomain.Entry) error {
			for _, entry := range page {
				item, err := directoryconnectordomain.MapEntry(directoryconnectordomain.ActiveDirectory, mapping, entry)
				if err != nil {
					t.Logf("skipped %s: %v", entry.DN, err)
					continue
				}
				items[item.PreferredUsername] = item
			}
			return nil
		})
	if err != nil {
		t.Fatalf("Search(%s): %v", plan.Filter, err)
	}
	return plan, items
}

// 隔離した Samba AD の DC から、Active Directory の読み取り方で利用者を取り込む。
// 期待値は MS-ADTS の属性の意味から書く。objectGUID は 16 バイトの不変 ID、userAccountControl の 0x2 は
// ACCOUNTDISABLE、変更は uSNChanged を rootDSE の highestCommittedUSN の範囲で進め、
// userPrincipalName は start.sh の samba-tool が作った alice@example.com である。
//
//spec:covers REQ-DIRECTORYCONNECTOR-004, REQ-DIRECTORYCONNECTOR-005, REQ-DIRECTORYCONNECTOR-006: 実物の AD の DC から objectGUID を不変 ID に、userPrincipalName を利用者名に、userAccountControl を有効の可否に写し、uSNChanged の差分の同期で無効化を同じ利用者へ反映すること
func TestActiveDirectoryDialectReadsSambaAD(t *testing.T) {
	server := testing_sambaad.Start(t)
	directory := sambaDirectory(t, server)

	full, users := readUsers(t, directory, nil)
	if full.Mode != directoryconnectordomain.ModeFull || full.Next.USN <= 0 || full.Next.Server == "" {
		t.Fatalf("first plan = %+v, want a full run that remembers the DC and its highestCommittedUSN", full)
	}
	alice, ok := users["alice@example.com"]
	if !ok {
		t.Fatalf("users = %v, want alice@example.com", keys(users))
	}
	if guid, err := base64.StdEncoding.DecodeString(alice.ExternalID); err != nil || len(guid) != 16 {
		t.Fatalf("alice's immutable id %q is not a 16-byte objectGUID", alice.ExternalID)
	}
	if !alice.Enabled || alice.OnPremisesImmutableID != alice.ExternalID {
		t.Fatalf("alice = %+v, want enabled with the objectGUID as the on-premises immutable id", alice)
	}

	server.SambaTool(t, "user", "disable", testing_sambaad.Alice)
	incremental, changed := readUsers(t, directory, &full.Next)
	if incremental.Mode != directoryconnectordomain.ModeIncremental || incremental.Next.USN <= full.Next.USN {
		t.Fatalf("second plan = %+v, want an incremental run past USN %d", incremental, full.Next.USN)
	}
	disabled, ok := changed["alice@example.com"]
	if !ok || disabled.Enabled || disabled.ExternalID != alice.ExternalID {
		t.Fatalf("incremental users = %+v, want alice disabled with the same immutable id", changed)
	}
	for name := range changed {
		if name != "alice@example.com" {
			t.Errorf("the incremental run returned %s, which did not change", name)
		}
	}
}

// 隔離した Samba AD の DC で、委任認証の bind の結果を確かめる。
// 期待値は Microsoft の LDAP の bind の誤りの診断（data 52e は資格情報の誤り、533 は無効、773 は次回のログオンで
// パスワードの変更が要る）から書く。
//
//spec:covers REQ-DIRECTORYCONNECTOR-008: 実物の AD の DC で、objectGUID で利用者を解決して bind し、誤ったパスワード、無効な利用者、変更が要るパスワードを区別すること
func TestDelegatedPasswordAgainstSambaAD(t *testing.T) {
	server := testing_sambaad.Start(t)
	directory := sambaDirectory(t, server)
	_, users := readUsers(t, directory, nil)
	alice, ok := users["alice@example.com"]
	if !ok {
		t.Fatalf("users = %v, want alice@example.com", keys(users))
	}
	scope := directoryconnectordomain.Scope{BaseDNs: []string{testing_sambaad.UsersDN}, Filter: directoryconnectordomain.DefaultUserFilter(directoryconnectordomain.ActiveDirectory)}

	if got := directory.VerifyPassword(t.Context(), scope, alice.ExternalID, testing_sambaad.AlicePassword); got != "accepted" {
		t.Fatalf("alice's password: %s, want accepted", got)
	}
	if got := directory.VerifyPassword(t.Context(), scope, alice.ExternalID, "wrong-password"); got != "invalid_credentials" {
		t.Fatalf("a wrong password: %s, want invalid_credentials", got)
	}

	server.Do(t, func(conn *ldap.Conn) error {
		change := ldap.NewModifyRequest("CN=alice,"+testing_sambaad.UsersDN, nil)
		change.Replace("pwdLastSet", []string{"0"})
		return conn.Modify(change)
	})
	if got := directory.VerifyPassword(t.Context(), scope, alice.ExternalID, testing_sambaad.AlicePassword); got != "password_change_required" {
		t.Errorf("a password that must change: %s, want password_change_required", got)
	}

	server.SambaTool(t, "user", "disable", testing_sambaad.Alice)
	if got := directory.VerifyPassword(t.Context(), scope, alice.ExternalID, testing_sambaad.AlicePassword); got != "account_disabled" {
		t.Errorf("a disabled user: %s, want account_disabled", got)
	}
}

func keys(items map[string]directoryconnectordomain.Item) []string {
	names := make([]string, 0, len(items))
	for name := range items {
		names = append(names, name)
	}
	return names
}
