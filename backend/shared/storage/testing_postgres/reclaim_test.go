package testing_postgres

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestPostgresControlPathUsesRecordedDistribution(t *testing.T) {
	for _, executable := range []string{
		"/cache/v1/bin/postgres",
		"/cache/v2/18.6.1/darwin-arm64/bin/postgres",
		"/cache/with spaces/bin/postgres",
	} {
		t.Run(executable, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(dataDir(dir), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dataDir(dir), "postmaster.opts"), []byte(`"`+executable+`" "-D" "/data"`), 0o600); err != nil {
				t.Fatal(err)
			}
			got, err := postgresControlPath(dir)
			want := filepath.Join(filepath.Dir(executable), "pg_ctl")
			if err != nil || got != want {
				t.Fatalf("postgresControlPath = %q, %v; want %q, nil", got, err, want)
			}
		})
	}
}

func TestPostgresControlPathRejectsUnusableRecord(t *testing.T) {
	dir := t.TempDir()
	if _, err := postgresControlPath(dir); err == nil {
		t.Fatal("missing postmaster.opts must fail")
	}
	if err := os.MkdirAll(dataDir(dir), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, opts := range []string{"", "postgres -D /data", `"relative/bin/postgres"`, `"/cache/bin/other"`, `"/cache/bin/postgres`} {
		if err := os.WriteFile(filepath.Join(dataDir(dir), "postmaster.opts"), []byte(opts), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := postgresControlPath(dir); err == nil {
			t.Fatalf("invalid postmaster.opts %q must fail", opts)
		}
	}
}

func TestDecideReclaim(t *testing.T) {
	const owner, postmaster = 100, 200
	aliveOnly := func(pids ...int) func(int) bool {
		return func(pid int) bool { return slices.Contains(pids, pid) }
	}
	cases := []struct {
		name  string
		inst  abandonedInstance
		alive func(int) bool
		want  reclaim
	}{
		// 作成直後で持ち主の pid をまだ書いていない実行と区別できないので触らない。
		{"持ち主の pid が読めない", abandonedInstance{postmaster: postmaster}, aliveOnly(), leaveAlone},
		// go test ./... が並行して走らせる別パッケージの DB を止めない。
		{"持ち主が生きている", abandonedInstance{owner: owner, postmaster: postmaster}, aliveOnly(owner, postmaster), leaveAlone},
		{"持ち主が死に、postmaster が生きている", abandonedInstance{owner: owner, postmaster: postmaster}, aliveOnly(postmaster), stopPostmaster},
		{"持ち主も postmaster も死んでいる", abandonedInstance{owner: owner, postmaster: postmaster, shmID: 9}, aliveOnly(), removeLeftovers},
		// initdb の途中で死ぬと postmaster.pid が無い。消すセグメントも無く、ディレクトリだけが残る。
		{"持ち主が死に、postmaster.pid が無い", abandonedInstance{owner: owner}, aliveOnly(), removeLeftovers},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := decideReclaim(tc.inst, tc.alive); got != tc.want {
				t.Fatalf("decideReclaim(%+v) = %v, want %v", tc.inst, got, tc.want)
			}
		})
	}
}

func TestParsePostmasterPID(t *testing.T) {
	// PostgreSQL 18 の postmaster.pid。7 行目が「共有メモリの key と ID」である。
	content := "22103\n/tmp/idmagic-pgtest-1/data\n1790350684\n54321\n/tmp\nlocalhost\n102260796   5439489\nready   \n"
	if pid, id := parsePostmasterPID(content); pid != 22103 || id != 5439489 {
		t.Fatalf("parsePostmasterPID = (%d, %d), want (22103, 5439489)", pid, id)
	}
	// 起動の途中で書かれた短いファイルでは、読めた値だけを返す。
	if pid, id := parsePostmasterPID("22103\n"); pid != 22103 || id != 0 {
		t.Fatalf("truncated file = (%d, %d), want (22103, 0)", pid, id)
	}
}
