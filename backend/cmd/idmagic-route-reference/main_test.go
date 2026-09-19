package main

// docs/domain/system/scenarios.feature.md の REQ-SYSTEM-019 が宣言する、生成物と分類の
// 突き合わせを、この command の入口から観測する。
//
// 生成器そのものの中身は server_http の側のテストが持つ。ここで観測するのは突き合わせで
// ある。生成できることと、生成物が古くなったことを検出できることは別であり、後者が無いと
// 分類を変えた変更が ROUTE_PRIORITY.md を置き去りにしたまま通る。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

//spec:covers EX-SYSTEM-019-02: 生成物が分類の定義と一致しないとき、突き合わせが失敗して再生成すべきことを報告すること。
func TestCheckReportsAStaleRouteReference(t *testing.T) {
	t.Parallel()

	output := filepath.Join(t.TempDir(), "ROUTE_PRIORITY.md")
	if err := os.WriteFile(output, []byte("# Route Priority Reference\n\nhand-edited\n"), 0o600); err != nil {
		t.Fatalf("write a stale reference: %v", err)
	}

	err := run([]string{"--check", "--output", output})
	if err == nil {
		t.Fatal("--check accepted a reference that does not match the classification")
	}
	if !strings.Contains(err.Error(), "out of date") ||
		!strings.Contains(err.Error(), "generate-route-reference") {
		t.Errorf("error = %q, want it to say the file is out of date and name the regeneration command", err)
	}

	// 再生成すれば通る。ここまで見ないと、常に落ちる突き合わせと区別できない。
	if err := run([]string{"--output", output}); err != nil {
		t.Fatalf("regenerating the reference failed: %v", err)
	}
	if err := run([]string{"--check", "--output", output}); err != nil {
		t.Fatalf("--check rejected the freshly generated reference: %v", err)
	}
}
