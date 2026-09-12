package usecases_test

// 主要ユースケース追跡: REQ-CLAIMMAPPING-001。

import (
	"errors"
	"slices"
	"testing"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	claimusecases "github.com/ambi/idmagic/backend/claimmapping/usecases"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

const (
	persistentFormat = "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent"
)

func employeeNumberDef() userdomain.UserAttributeDef {
	return userdomain.UserAttributeDef{Key: "employee_number", Visibility: idmdomain.AttrVisibilitySelfReadable}
}

func ssnDef() userdomain.UserAttributeDef {
	return userdomain.UserAttributeDef{Key: "ssn", Visibility: idmdomain.AttrVisibilityPrivate}
}

func departmentDef() userdomain.UserAttributeDef {
	return userdomain.UserAttributeDef{Key: "department", Visibility: idmdomain.AttrVisibilitySelfReadable}
}

// claimTypes は発行されたクレーム集合を、注記が名指しする「含まれる / 含まれない」で
// 読める形に落とす。型の比較ではなく集合の比較にしないと、部分的な発行と完全な発行を
// 区別する表明が書けない。
func claimTypes(result claimusecases.ClaimIssuanceResult) []string {
	types := make([]string, 0, len(result.Claims))
	for _, claim := range result.Claims {
		types = append(types, claim.ClaimType)
	}
	return types
}

// TestIsAttributeReleasable_SelfReadableAllowed は wi-73 の動機例 (employeeNumber は
// SelfReadable だが admin が明示的にアプリへ release できる) を検証する (scenario
// 「管理者はApplication単位でclaim releaseを絞り込める」)。
func TestIsAttributeReleasable_SelfReadableAllowed(t *testing.T) {
	defs := []userdomain.UserAttributeDef{employeeNumberDef()}
	if !claimusecases.IsAttributeReleasable("employee_number", defs) {
		t.Fatal("expected SelfReadable attribute to be releasable")
	}
}

func TestIsAttributeReleasable_PrivateRejected(t *testing.T) {
	defs := []userdomain.UserAttributeDef{ssnDef()}
	if claimusecases.IsAttributeReleasable("ssn", defs) {
		t.Fatal("expected Private attribute to be rejected")
	}
}

func TestIsAttributeReleasable_UnknownKeyRejected(t *testing.T) {
	defs := []userdomain.UserAttributeDef{employeeNumberDef()}
	if claimusecases.IsAttributeReleasable("not_defined_anywhere", defs) {
		t.Fatal("expected unknown attribute key to be rejected (fail-closed)")
	}
}

func TestIsAttributeReleasable_CoreAttributesAlwaysAllowed(t *testing.T) {
	for _, key := range []string{
		claimusecases.AttrUserID, claimusecases.AttrEmail, claimusecases.AttrName,
		claimusecases.AttrGivenName, claimusecases.AttrFamilyName,
		claimusecases.AttrPreferredUsername, claimusecases.AttrEmailVerified, claimusecases.AttrRoles,
	} {
		if !claimusecases.IsAttributeReleasable(key, nil) {
			t.Fatalf("expected core attribute %q to always be releasable", key)
		}
	}
}

func TestIsReservedClaimType(t *testing.T) {
	reserved := []string{"iss", "sub", "aud", "exp", "iat", "nbf", "jti", "azp", "nonce", "at_hash", "c_hash", "acr", "amr", "sid"}
	for _, ct := range reserved {
		if !claimusecases.IsReservedClaimType(ct) {
			t.Fatalf("expected %q to be a reserved claim type", ct)
		}
	}
	if claimusecases.IsReservedClaimType("employee_number") {
		t.Fatal("employee_number must not be treated as reserved")
	}
}

func TestIssueClaimsWithFloor_AllowsSelfReadableOverride(t *testing.T) {
	policy := claimdomain.ClaimMappingPolicy{
		NameID: claimdomain.NameIdConfiguration{Format: persistentFormat, SourceAttribute: claimusecases.AttrUserID},
		Rules: []claimdomain.ClaimMappingRule{
			{ClaimType: "employee_number", Source: claimdomain.ClaimSourceUserAttribute, SourceKey: "employee_number", Required: true},
		},
	}
	attrs := claimdomain.Attributes{claimusecases.AttrUserID: {"user-1"}, "employee_number": {"E-123"}}
	defs := []userdomain.UserAttributeDef{employeeNumberDef()}

	got, err := claimusecases.IssueClaimsWithFloor(policy, attrs, defs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Claims) != 1 || got.Claims[0].ClaimType != "employee_number" || got.Claims[0].Values[0] != "E-123" {
		t.Fatalf("unexpected claims: %+v", got.Claims)
	}
}

// EX-CLAIMMAPPING-001-01: 対応付け規則を持たないカスタム属性は、`visibility` が
// `Private` でなくても、解決済み属性に値があっても、発行されるクレーム集合に現れない。
// 固定しているのは「規則が発行の唯一の根拠である」ことであり、`Private` かどうかは
// ここでは関係しない。
//
// 素通りすれば、規則を 1 つも書いていない RP に対して、テナントが定義しただけの属性が
// 解決済みというだけで流れる。`IsAttributeReleasable` は「規則のソースとして使ってよいか」
// しか見ないので、発行側が attrs をそのまま撒く実装になっていても、この関数は止めない。
func TestIssueClaimsWithFloor_OmitsAttributeWithNoRule(t *testing.T) {
	policy := claimdomain.ClaimMappingPolicy{
		NameID: claimdomain.NameIdConfiguration{Format: persistentFormat, SourceAttribute: claimusecases.AttrUserID},
		Rules: []claimdomain.ClaimMappingRule{
			{ClaimType: "department", Source: claimdomain.ClaimSourceUserAttribute, SourceKey: "department"},
		},
	}
	// employee_number は定義済みで SelfReadable、値も解決済みにある。欠けているのは規則だけ。
	attrs := claimdomain.Attributes{
		claimusecases.AttrUserID: {"user-1"},
		"employee_number":        {"E-123"},
		"department":             {"sales"},
	}
	defs := []userdomain.UserAttributeDef{employeeNumberDef(), departmentDef()}

	got, err := claimusecases.IssueClaimsWithFloor(policy, attrs, defs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, claim := range got.Claims {
		if claim.ClaimType == "employee_number" || slices.Contains(claim.Values, "E-123") {
			t.Fatalf("規則の無い属性が発行された: %+v", got.Claims)
		}
	}
	// 対照: 規則のある department は発行される。これが無いと、何も発行しない実装が緑になる。
	if types := claimTypes(got); !slices.Equal(types, []string{"department"}) {
		t.Fatalf("claim types = %v, want [department]", types)
	}
}

// EX-CLAIMMAPPING-003-01: 必須規則のソース属性が解決済み属性に無いとき、解決できた
// 残りの規則だけを適用した部分的なクレーム集合は返らず、発行そのものが拒否される。
// 固定しているのは、拒否と「1 つも返らない」ことの両方である。
//
// 部分的な発行は、RP から見れば「その利用者はその属性を持たない」と区別が付かない。
// 必須と宣言した規則がそう読まれるなら、必須という宣言が意味を失う。解決できる規則を
// 必須規則より前に置いてあるので、蓄積した途中結果をそのまま返す実装は落ちる。
func TestIssueClaimsWithFloor_RejectsPartialIssuanceWhenRequiredSourceMissing(t *testing.T) {
	policy := claimdomain.ClaimMappingPolicy{
		NameID: claimdomain.NameIdConfiguration{Format: persistentFormat, SourceAttribute: claimusecases.AttrUserID},
		Rules: []claimdomain.ClaimMappingRule{
			{ClaimType: "department", Source: claimdomain.ClaimSourceUserAttribute, SourceKey: "department"},
			{ClaimType: "employee_number", Source: claimdomain.ClaimSourceUserAttribute, SourceKey: "employee_number", Required: true},
		},
	}
	// employee_number は定義済みなので規則としては正当。欠けているのは値だけ。
	attrs := claimdomain.Attributes{claimusecases.AttrUserID: {"user-1"}, "department": {"sales"}}
	defs := []userdomain.UserAttributeDef{employeeNumberDef(), departmentDef()}

	got, err := claimusecases.IssueClaimsWithFloor(policy, attrs, defs)
	if err == nil {
		t.Fatal("必須規則のソースが欠けているのに拒否されなかった")
	}
	if _, ok := errors.AsType[*claimusecases.ClaimReleaseDeniedError](err); !ok {
		t.Fatalf("err = %#v, want *ClaimReleaseDeniedError", err)
	}
	if types := claimTypes(got); len(types) != 0 {
		t.Fatalf("拒否されたのに部分的なクレーム集合が返った: %v", types)
	}
	if got.NameIDValue != "" {
		t.Fatalf("拒否されたのに NameID が返った: %q", got.NameIDValue)
	}

	// 対照: 値が揃えば同じ policy で 2 件とも発行される。拒否が必須規則の未解決だけに
	// 由来することを示す。
	complete := claimdomain.Attributes{
		claimusecases.AttrUserID: {"user-1"}, "department": {"sales"}, "employee_number": {"E-123"},
	}
	issued, err := claimusecases.IssueClaimsWithFloor(policy, complete, defs)
	if err != nil {
		t.Fatalf("対照が失敗した: %v", err)
	}
	if types := claimTypes(issued); !slices.Equal(types, []string{"department", "employee_number"}) {
		t.Fatalf("対照の claim types = %v", types)
	}
}

// EX-CLAIMMAPPING-002-01: `visibility=Private` のキーをソースとする規則がある policy は、
// クレームを 1 つも発行せずに拒否される。ssn そのものが漏れないことに加えて、同じ policy に
// 並んでいる正当な規則の結果も返らないことを固定する。
//
// 効果の不在は、返り値のクレーム集合と NameID が空であることで観測する。拒否の理由が
// Private 属性であることは、対照（同じキーを SelfReadable にすれば発行される）が示す。
func TestIssueClaimsWithFloor_RejectsPrivateSourceAttribute(t *testing.T) {
	// department の規則は正当で、単独なら発行される。拒否が policy 全体に効くことを
	// 見るために並べてある。
	policy := claimdomain.ClaimMappingPolicy{
		NameID: claimdomain.NameIdConfiguration{Format: persistentFormat, SourceAttribute: claimusecases.AttrUserID},
		Rules: []claimdomain.ClaimMappingRule{
			{ClaimType: "department", Source: claimdomain.ClaimSourceUserAttribute, SourceKey: "department"},
			{ClaimType: "ssn_claim", Source: claimdomain.ClaimSourceUserAttribute, SourceKey: "ssn"},
		},
	}
	attrs := claimdomain.Attributes{
		claimusecases.AttrUserID: {"user-1"}, "ssn": {"123-45-6789"}, "department": {"sales"},
	}
	defs := []userdomain.UserAttributeDef{ssnDef(), departmentDef()}

	got, err := claimusecases.IssueClaimsWithFloor(policy, attrs, defs)
	if err == nil {
		t.Fatal("expected Private attribute source to be rejected fail-closed")
	}
	if _, ok := errors.AsType[*claimusecases.ClaimReleaseDeniedError](err); !ok {
		t.Fatalf("err = %#v, want *ClaimReleaseDeniedError", err)
	}
	if types := claimTypes(got); len(types) != 0 {
		t.Fatalf("拒否されたのにクレームが発行された: %v", types)
	}
	if got.NameIDValue != "" {
		t.Fatalf("拒否されたのに NameID が返った: %q", got.NameIDValue)
	}

	// 対照: ssn の可視性だけを SelfReadable へ変えれば、同じ policy が 2 件とも発行する。
	// 拒否が可視性の判定に由来し、規則の組み立て不良ではないことを示す。
	releasable := []userdomain.UserAttributeDef{
		{Key: "ssn", Visibility: idmdomain.AttrVisibilitySelfReadable}, departmentDef(),
	}
	issued, err := claimusecases.IssueClaimsWithFloor(policy, attrs, releasable)
	if err != nil {
		t.Fatalf("対照が失敗した: %v", err)
	}
	if types := claimTypes(issued); !slices.Equal(types, []string{"department", "ssn_claim"}) {
		t.Fatalf("対照の claim types = %v", types)
	}
}

// EX-CLAIMMAPPING-002-01: `attribute_defs` に無いキーをソースとする規則がある policy は、
// クレームを 1 つも発行せずに拒否される。前のテストが Given の `visibility=Private` 側を、
// これが「定義そのものが無い」側を固定する。具体例が両方を or で並べているので、
// 観測も 2 つ要る。
//
// 効果の不在は返り値のクレーム集合と NameID が空であることで観測する。解決済み属性には
// 値が入っているので、フェイルクローズを持たない実装はここで値を発行してしまう。
func TestIssueClaimsWithFloor_RejectsUnknownSourceAttribute(t *testing.T) {
	policy := claimdomain.ClaimMappingPolicy{
		NameID: claimdomain.NameIdConfiguration{Format: persistentFormat, SourceAttribute: claimusecases.AttrUserID},
		Rules: []claimdomain.ClaimMappingRule{
			{ClaimType: "department", Source: claimdomain.ClaimSourceUserAttribute, SourceKey: "department"},
			{ClaimType: "mystery", Source: claimdomain.ClaimSourceUserAttribute, SourceKey: "not_defined_anywhere"},
		},
	}
	attrs := claimdomain.Attributes{
		claimusecases.AttrUserID: {"user-1"}, "not_defined_anywhere": {"leak"}, "department": {"sales"},
	}
	defs := []userdomain.UserAttributeDef{departmentDef()}

	got, err := claimusecases.IssueClaimsWithFloor(policy, attrs, defs)
	if err == nil {
		t.Fatal("expected unknown attribute source to be rejected fail-closed")
	}
	if _, ok := errors.AsType[*claimusecases.ClaimReleaseDeniedError](err); !ok {
		t.Fatalf("err = %#v, want *ClaimReleaseDeniedError", err)
	}
	if types := claimTypes(got); len(types) != 0 {
		t.Fatalf("拒否されたのにクレームが発行された: %v", types)
	}
	if got.NameIDValue != "" {
		t.Fatalf("拒否されたのに NameID が返った: %q", got.NameIDValue)
	}

	// 対照: 同じキーを定義へ足せば発行される。拒否が「定義が無い」ことに由来することを示す。
	defined := []userdomain.UserAttributeDef{
		departmentDef(),
		{Key: "not_defined_anywhere", Visibility: idmdomain.AttrVisibilitySelfReadable},
	}
	issued, err := claimusecases.IssueClaimsWithFloor(policy, attrs, defined)
	if err != nil {
		t.Fatalf("対照が失敗した: %v", err)
	}
	if types := claimTypes(issued); !slices.Equal(types, []string{"department", "mystery"}) {
		t.Fatalf("対照の claim types = %v", types)
	}
}

func TestIssueClaimsWithFloor_RejectsReservedClaimType(t *testing.T) {
	policy := claimdomain.ClaimMappingPolicy{
		NameID: claimdomain.NameIdConfiguration{Format: persistentFormat, SourceAttribute: claimusecases.AttrUserID},
		Rules: []claimdomain.ClaimMappingRule{
			{ClaimType: "sub", Source: claimdomain.ClaimSourceFixed, FixedValue: "attacker-controlled"},
		},
	}
	attrs := claimdomain.Attributes{claimusecases.AttrUserID: {"user-1"}}

	if _, err := claimusecases.IssueClaimsWithFloor(policy, attrs, nil); err == nil {
		t.Fatal("expected reserved claim_type rule to be rejected")
	}
}

func TestIssueClaimsWithFloor_RejectsPrivateNameIDSource(t *testing.T) {
	policy := claimdomain.ClaimMappingPolicy{
		NameID: claimdomain.NameIdConfiguration{Format: persistentFormat, SourceAttribute: "ssn"},
	}
	attrs := claimdomain.Attributes{"ssn": {"123-45-6789"}}
	defs := []userdomain.UserAttributeDef{ssnDef()}

	if _, err := claimusecases.IssueClaimsWithFloor(policy, attrs, defs); err == nil {
		t.Fatal("expected Private NameID source attribute to be rejected fail-closed")
	}
}
