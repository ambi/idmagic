package domain_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/ambi/idmagic/backend/sourcing/scim/domain"
)

// scimBody は fuzz 入力の JSON を SCIM のボディ表現へ変換する。
// 変換できない入力はこの target の対象外なので落とす。
func scimBody(raw string) (map[string]any, bool) {
	if len(raw) > 64*1024 {
		return nil, false
	}
	var body map[string]any
	if json.Unmarshal([]byte(raw), &body) != nil {
		return nil, false
	}
	return body, true
}

// FuzzParseUserWrite は、拒否した書き込みボディから値を持ち出さないことを表明する。
// SCIM の書き込みは外部 IdP から届くので、error と一緒に埋まった UserWrite を返すと、
// 呼び出し側が err を握り潰した瞬間に未検証の userName で上書きが走る。
func FuzzParseUserWrite(f *testing.F) {
	f.Add(`{"userName":"alice","active":true}`)
	f.Add(`{"userName":"  "}`)
	f.Add(`{"userName":"alice","phoneNumbers":[]}`)
	f.Add(`{"userName":"alice","name":{"givenName":"A","familyName":"B"}}`)
	f.Add(`{}`)

	f.Fuzz(func(t *testing.T, raw string) {
		body, ok := scimBody(raw)
		if !ok {
			return
		}
		write, err := domain.ParseUserWrite(body)
		if err == nil {
			return
		}
		if write != (domain.UserWrite{}) {
			t.Fatalf("ParseUserWrite returned %+v together with an error", write)
		}
	})
}

// FuzzParseUserPatchOps は、受理した PATCH 操作が宣言済みの op だけからなることを表明する。
// 未知の op がそのまま下位へ流れると、許可属性の判定を経ないまま適用される経路ができる。
func FuzzParseUserPatchOps(f *testing.F) {
	f.Add(`{"Operations":[{"op":"replace","path":"userName","value":"alice"}]}`)
	f.Add(`{"Operations":[{"op":"REPLACE","path":"userName","value":"alice"}]}`)
	f.Add(`{"Operations":[{"op":"delete","path":"userName"}]}`)
	f.Add(`{"Operations":[{"op":"replace","path":"id","value":"x"}]}`)
	f.Add(`{"Operations":[]}`)
	f.Add(`{"Operations":[{"op":"remove","path":"emails"}]}`)

	f.Fuzz(func(t *testing.T, raw string) {
		body, ok := scimBody(raw)
		if !ok {
			return
		}
		ops, err := domain.ParseUserPatchOps(body)
		if err != nil {
			if ops != nil {
				t.Fatalf("ParseUserPatchOps returned %d operations together with an error", len(ops))
			}
			return
		}
		if len(ops) == 0 {
			t.Fatalf("ParseUserPatchOps accepted an empty operation list from %q", raw)
		}
		for _, op := range ops {
			if !slices.Contains([]string{"add", "replace", "remove"}, op.Op) {
				t.Fatalf("ParseUserPatchOps accepted the undeclared op %q from %q", op.Op, raw)
			}
			if op.Attr == "" {
				t.Fatalf("ParseUserPatchOps accepted an operation without an attribute from %q", raw)
			}
		}
	})
}

// FuzzParseGroupPatchOps は ParseUserPatchOps と同じ性質を Group について表明する。
func FuzzParseGroupPatchOps(f *testing.F) {
	f.Add(`{"Operations":[{"op":"replace","path":"displayName","value":"g"}]}`)
	f.Add(`{"Operations":[{"op":"add","path":"members","value":[{"value":"u1"}]}]}`)
	f.Add(`{"Operations":[{"op":"delete","path":"members"}]}`)
	f.Add(`{"Operations":[]}`)

	f.Fuzz(func(t *testing.T, raw string) {
		body, ok := scimBody(raw)
		if !ok {
			return
		}
		ops, err := domain.ParseGroupPatchOps(body)
		if err != nil {
			if ops != nil {
				t.Fatalf("ParseGroupPatchOps returned %d operations together with an error", len(ops))
			}
			return
		}
		if len(ops) == 0 {
			t.Fatalf("ParseGroupPatchOps accepted an empty operation list from %q", raw)
		}
		for _, op := range ops {
			if !slices.Contains([]string{"add", "replace", "remove"}, op.Op) {
				t.Fatalf("ParseGroupPatchOps accepted the undeclared op %q from %q", op.Op, raw)
			}
			if op.Attr == "" {
				t.Fatalf("ParseGroupPatchOps accepted an operation without an attribute from %q", raw)
			}
		}
	})
}

// FuzzParseGroupWrite は、拒否したグループ書き込みから値を持ち出さないことを表明する。
func FuzzParseGroupWrite(f *testing.F) {
	f.Add(`{"displayName":"group"}`)
	f.Add(`{"displayName":"  "}`)
	f.Add(`{"displayName":"g","members":[{"value":"u1"}]}`)
	f.Add(`{"displayName":"g","members":"not-an-array"}`)

	f.Fuzz(func(t *testing.T, raw string) {
		body, ok := scimBody(raw)
		if !ok {
			return
		}
		write, err := domain.ParseGroupWrite(body)
		if err == nil {
			return
		}
		if write.DisplayName != "" || len(write.MemberScimIDs) != 0 {
			t.Fatalf("ParseGroupWrite returned %+v together with an error", write)
		}
	})
}
