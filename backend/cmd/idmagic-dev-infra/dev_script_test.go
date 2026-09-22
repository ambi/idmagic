package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

// fakeProcess は `dev.sh` が起動する各プロセスの代役である。起動したことと、受け取った
// 永続化の設定を記録する。基盤は FAKE_INFRA=fail のとき準備完了を公開せずに終わる。
const fakeProcess = `#!/bin/sh
case "$(basename "$0")" in
idmagic-dev-infra)
  echo "infra" >> "$DEV_LOG"
  if [ "$FAKE_INFRA" = fail ]; then exit 1; fi
  while [ $# -gt 0 ]; do
    case "$1" in --ready-file) printf '{}' > "$2"; shift 2 ;; *) shift ;; esac
  done
  exec sleep 30 ;;
idmagic)
  echo "api PERSISTENCE=$PERSISTENCE DATABASE_URL=$DATABASE_URL" >> "$DEV_LOG"
  exec sleep 30 ;;
idmagic-worker)
  echo "worker PERSISTENCE=$PERSISTENCE DATABASE_URL=$DATABASE_URL" >> "$DEV_LOG"
  # 自分から終わり、dev.sh に全体を止めさせる。他のプロセスが記録を書く時間を残す。
  sleep 2
  exit 0 ;;
esac
`

// fakeGo は `go build -o <dir>/ <packages>` の代わりに、代役を 3 つの名前で置く。
const fakeGo = `#!/bin/sh
while [ $# -gt 0 ]; do
  case "$1" in -o) out="$2"; shift 2 ;; *) shift ;; esac
done
for name in idmagic idmagic-worker idmagic-dev-infra; do
  cp "$FAKE_PROCESS" "$out/$name"
done
`

// fakeBun は依存の導入を何もせずに終え、Vite の起動を UI の起動として記録する。
const fakeBun = `#!/bin/sh
if [ "$1" = install ]; then exit 0; fi
echo "ui" >> "$DEV_LOG"
exec sleep 30
`

type devScriptRun struct {
	err     error
	started []string
}

// runDevScript は PATH の go と bun を代役へ差し替えて `dev.sh` を実行する。
// 実物のビルド、組込み PostgreSQL、Vite を起動せず、起動の順序と環境だけを観測する。
func runDevScript(t *testing.T, infraOutcome string) devScriptRun {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	script := filepath.Join(filepath.Dir(file), "..", "..", "..", "dev.sh")

	stubs := t.TempDir()
	for name, body := range map[string]string{"fake-process": fakeProcess, "go": fakeGo, "bun": fakeBun} {
		if err := os.WriteFile(filepath.Join(stubs, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	log := filepath.Join(t.TempDir(), "started.log")

	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", script)
	cmd.Env = []string{
		"PATH=" + stubs + ":/usr/bin:/bin",
		"HOME=" + t.TempDir(),
		"TMPDIR=" + t.TempDir(),
		"IDMAGIC_DEV_PG_DIR=" + filepath.Join(t.TempDir(), "postgres"),
		"FAKE_PROCESS=" + filepath.Join(stubs, "fake-process"),
		"FAKE_INFRA=" + infraOutcome,
		"DEV_LOG=" + log,
	}
	output, err := cmd.CombinedOutput()
	t.Logf("dev.sh output:\n%s", output)

	var started []string
	if recorded, readErr := os.ReadFile(log); readErr == nil {
		started = strings.Split(strings.TrimSpace(string(recorded)), "\n")
	}
	return devScriptRun{err: err, started: started}
}

func startedWith(run devScriptRun, prefix string) []string {
	var lines []string
	for _, line := range run.started {
		if strings.HasPrefix(line, prefix) {
			lines = append(lines, line)
		}
	}
	return lines
}

//spec:covers EX-JOBS-001-01: 標準開発コマンドは基盤、API、`worker`、UI を起動し、API と `worker` に同じ PostgreSQL の接続先を渡してキューを共有させる。
func TestDevScriptStartsEveryProcessAgainstTheSharedQueue(t *testing.T) {
	run := runDevScript(t, "ready")
	if run.err != nil {
		t.Fatalf("dev.sh exited with %v", run.err)
	}
	for _, name := range []string{"infra", "api", "worker", "ui"} {
		if len(startedWith(run, name)) != 1 {
			t.Fatalf("%s started %d times, want once; started=%q", name, len(startedWith(run, name)), run.started)
		}
	}
	api := strings.TrimPrefix(startedWith(run, "api")[0], "api ")
	worker := strings.TrimPrefix(startedWith(run, "worker")[0], "worker ")
	if !strings.Contains(api, "PERSISTENCE=postgres") || !strings.Contains(api, "DATABASE_URL=postgres://") {
		t.Fatalf("API did not start against PostgreSQL: %q", api)
	}
	if api != worker {
		t.Fatalf("API and worker start against different queues: api=%q worker=%q", api, worker)
	}
}

//spec:covers EX-JOBS-001-02: 基盤の準備に失敗した標準開発コマンドは失敗で終わり、API、`worker`、UI のどれも起動しない。
func TestDevScriptFailsFastWhenInfrastructureFails(t *testing.T) {
	run := runDevScript(t, "fail")
	if run.err == nil {
		t.Fatal("dev.sh succeeded although the infrastructure failed")
	}
	if !slices.Equal(run.started, []string{"infra"}) {
		t.Fatalf("started=%q, want only the infrastructure", run.started)
	}
}
