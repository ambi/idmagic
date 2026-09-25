package testing_postgres

import (
	"slices"
	"testing"
)

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
