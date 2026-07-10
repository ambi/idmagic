package version

import (
	"testing"
)

func TestGet(t *testing.T) {
	// runtime/debug はテスト環境によって ReadBuildInfo の返り値が異なるが、
	// 少なくとも Get() がパニックせず、妥当なオブジェクトを返すことをテストする。
	info := Get()
	if info.Version == "" {
		t.Error("expected non-empty version")
	}
	if info.GitCommit == "" {
		t.Error("expected non-empty git commit")
	}
	if info.BuildDate == "" {
		t.Error("expected non-empty build date")
	}
}
