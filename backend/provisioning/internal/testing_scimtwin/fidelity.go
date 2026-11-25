package testing_scimtwin

import (
	"fmt"
	"strings"

	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
)

// volatileKeys は相手が割り当てる、または相手ごとに違ってよい値を持つ属性である。
// 模倣との照合では、これらの値は型だけを比べる。
var volatileKeys = map[string]bool{
	"id": true, "value": true, "$ref": true, "location": true, "version": true,
	"created": true, "lastModified": true, "documentationUri": true, "specUri": true,
	"description": true, "maxOperations": true, "maxPayloadSize": true, "maxResults": true,
	"totalResults": true, "detail": true,
}

// comparedHeaders は記録が持つときに模倣も返すべき応答のヘッダーである。
var comparedHeaders = []string{"Content-Type", "ETag", "Location"}

// Diverge は記録の往復を順に模倣へ送り、模倣の応答が記録と食い違う点を返す。
//
// 記録の識別子は模倣が割り当てたものへ読み替える。作成の応答の id で対応を学び、
// 後の要求のパスと本文の中の同じ文字列を置き換える。
// 比べるのは状態コード、記録が持つヘッダーの有無と Content-Type、本文の構造と値である。
// 本文は記録の属性がすべて模倣の応答にあることを求め、模倣が多く返すことは許す。
func Diverge(tw *Twin, rec Recording) []string {
	ids := map[string]string{}
	var setupErrs []string
	fidelity := testing_httprecord.Fidelity{
		VolatileKeys: volatileKeys,
		Headers:      comparedHeaders,
		Setup: func(ex Exchange) {
			resources, err := setupResources(ex)
			if err != nil {
				setupErrs = append(setupErrs, fmt.Sprintf("%s: setup: %v", ex.Name, err))
				return
			}
			for _, res := range resources {
				kind := "Users"
				if res.Kind == "group" {
					kind = "Groups"
				}
				tw.Put(kind, res.ID, map[string]any{"displayName": res.DisplayName})
			}
		},
		Rewrite: func(s string) string {
			for recorded, actual := range ids {
				s = strings.ReplaceAll(s, recorded, actual)
			}
			return s
		},
		Learn: func(want, got any) { learnID(ids, want, got) },
	}
	diffs := fidelity.Diverge(tw, rec)
	return append(setupErrs, diffs...)
}

// learnID は作成の応答から、記録の id と模倣の id の対応を覚える。
func learnID(ids map[string]string, want, got any) {
	w, ok1 := want.(map[string]any)
	g, ok2 := got.(map[string]any)
	if !ok1 || !ok2 {
		return
	}
	wid, ok1 := w["id"].(string)
	gid, ok2 := g["id"].(string)
	if ok1 && ok2 && wid != gid {
		ids[wid] = gid
	}
}
