package domain_test

import (
	"regexp"
	"testing"

	"github.com/ambi/idmagic/backend/authorization/domain"
)

// tenantIDShape は tenancy が受け入れるテナント ID の形 (tenancy.tenantIDPattern と同じ)。
// 整合トークンは tenantID と版を ":" で連結して符号化するため、往復が成り立つのは
// テナント ID が ":" を含まないというこの制約に依っている。
var tenantIDShape = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

// FuzzConsistencyTokenRoundTrip は、整合トークンが発行元テナントに束縛されていることを表明する。
//
// 別テナントでの復号が必ず拒否されることは、テナント ID の形によらず成り立たなければならない。
// 束縛が緩むと、あるテナントで得たトークンを別テナントの読み取りへ持ち込める。
// 同じテナントでの往復は、tenancy が受け入れる形のテナント ID についてのみ表明する。
func FuzzConsistencyTokenRoundTrip(f *testing.F) {
	f.Add("tenant-a", int64(1), "tenant-b")
	f.Add("tenant-a", int64(0), "tenant-a")
	f.Add("", int64(-1), "")
	f.Add("tenant:a", int64(5), "tenant")
	f.Add("tenant", int64(5), "tenant:a")

	f.Fuzz(func(t *testing.T, issuingTenant string, version int64, readingTenant string) {
		if len(issuingTenant) > 4096 || len(readingTenant) > 4096 {
			return
		}
		token := domain.EncodeConsistencyToken(issuingTenant, version)

		if readingTenant != issuingTenant {
			if _, err := domain.DecodeConsistencyToken(token, readingTenant); err == nil {
				t.Fatalf("a token issued for %q was accepted by tenant %q", issuingTenant, readingTenant)
			}
		}

		if !tenantIDShape.MatchString(issuingTenant) {
			return
		}
		decoded, err := domain.DecodeConsistencyToken(token, issuingTenant)
		if err != nil {
			t.Fatalf("a token issued for %q was rejected by its own tenant: %v", issuingTenant, err)
		}
		if decoded != version {
			t.Fatalf("round trip changed the version: %d -> %d", version, decoded)
		}
	})
}

// FuzzDecodeConsistencyToken は、任意の文字列から版を取り出さないことを表明する。
func FuzzDecodeConsistencyToken(f *testing.F) {
	f.Add("", "tenant-a")
	f.Add("dGVuYW50LWE6MQ", "tenant-a")
	f.Add("dGVuYW50LWE6MQ", "tenant-b")
	f.Add("!!!", "tenant-a")
	f.Add("dGVuYW50LWE6", "tenant-a")

	f.Add("dGVuYW50LWE6M0", "tenant-a") // 末尾の余剰ビットが立った非正規な符号化。

	f.Fuzz(func(t *testing.T, token, tenantID string) {
		if len(token) > 8192 || len(tenantID) > 4096 {
			return
		}
		version, err := domain.DecodeConsistencyToken(token, tenantID)
		if err != nil {
			if version != 0 {
				t.Fatalf("DecodeConsistencyToken returned version %d together with an error", version)
			}
			return
		}
		// 受理したなら、そのテナントで同じ版を符号化し直したものと一致しなければならない。
		if reissued := domain.EncodeConsistencyToken(tenantID, version); reissued != token {
			t.Fatalf("accepted %q for tenant %q, but that tenant and version encode to %q",
				token, tenantID, reissued)
		}
	})
}
