package testing_scimtwin

import (
	"strings"
	"testing"
)

// recordedCapabilities は RFC 7643 §8.5 の例が広告する対応可否である。
var recordedCapabilities = Capabilities{Patch: true, Bulk: true, Filter: true, Etag: true, Sort: true, ChangePassword: true}

// 模倣は、記録した往復の要求に記録と同じ応答を返す。
func TestTwin_MatchesRFC7644Recording(t *testing.T) {
	tw := NewTwin()
	tw.Capabilities = recordedCapabilities
	for _, diff := range Diverge(tw, RFC7644Recording(t)) {
		t.Error(diff)
	}
}

// 照合が空虚でないことを確かめる。模倣の応答を記録から外すと、照合が食い違いを報告する。
func TestDiverge_ReportsTwinThatLeavesTheRecording(t *testing.T) {
	tests := []struct {
		name  string
		setup func(tw *Twin)
		want  string
	}{
		{"PATCH が 204 でなく 200 を返す", func(tw *Twin) { tw.PatchReturnsResource = true }, "add-group-member: status = 200, recorded 204"},
		{"広告する対応可否が記録と違う", func(tw *Twin) { tw.Capabilities.Sort = false }, "discover: body.sort.supported = false, recorded true"},
		{"作成が 500 を返す", func(tw *Twin) {
			tw.InjectFaults(Fault{}, Fault{Status: 500})
		}, "create-user: status = 500, recorded 201"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tw := NewTwin()
			tw.Capabilities = recordedCapabilities
			tc.setup(tw)
			diffs := Diverge(tw, RFC7644Recording(t))
			if !containsLine(diffs, tc.want) {
				t.Errorf("Diverge() = %q, want a line %q", diffs, tc.want)
			}
		})
	}
}

func containsLine(lines []string, want string) bool {
	for _, line := range lines {
		if strings.Contains(line, want) {
			return true
		}
	}
	return false
}

// 記録は出どころを持たなければ読み込めない。古くなった記録を区別する手がかりを失わないためである。
func TestDecodeRecording_RequiresProvenance(t *testing.T) {
	if _, err := decodeRecording([]byte(`{"counterpart":"x","counterpart_version":"1","method":"m","exchanges":[]}`)); err == nil {
		t.Error("decodeRecording() accepted a recording without retrieved_at")
	}
}
