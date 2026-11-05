package testing_openldap

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestStartFailsWhenDockerIsUnavailable(t *testing.T) {
	if os.Getenv("OPENLDAP_FAILURE_PROBE") == "1" {
		Start(t)
		return
	}
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "docker"), []byte("#!/bin/sh\necho docker-probe-unavailable >&2\nexit 42\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	t.Setenv("OPENLDAP_FAILURE_PROBE", "1")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	child := exec.CommandContext(t.Context(), executable, "-test.run=^TestStartFailsWhenDockerIsUnavailable$", "-test.v")
	output, err := child.CombinedOutput()
	if err == nil || strings.Contains(string(output), "--- SKIP:") {
		t.Fatalf("Docker が使えない試験の結果 = %v\n%s\n期待値はスキップを伴わない失敗", err, output)
	}
	if !strings.Contains(string(output), "docker-probe-unavailable") {
		t.Fatalf("Docker の診断情報がない: %s", output)
	}
}
