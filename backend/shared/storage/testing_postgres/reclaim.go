package testing_postgres

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// runtimeDirPattern は start が作る実行時ディレクトリの名前である。回収はこの名前の
// ディレクトリだけを対象にする。
const runtimeDirPattern = "idmagic-pgtest-*"

// ownerFile は実行時ディレクトリを作ったテストバイナリの pid を置くファイルである。
const ownerFile = "owner.pid"

// runtimeSubdir は embedded-postgres に渡す RuntimePath である。ライブラリは起動時に
// RuntimePath を消して作り直すので、ownerFile と同じ階層には置けない。
const runtimeSubdir = "runtime"

func dataDir(instanceDir string) string {
	return filepath.Join(instanceDir, runtimeSubdir, "data")
}

// abandonedInstance は、ある実行時ディレクトリについて回収の判断に要る事実である。
// pid と共有メモリの値は、読めなければ 0 とする。
type abandonedInstance struct {
	dir        string
	owner      int
	postmaster int
	shmID      int
}

type reclaim int

const (
	leaveAlone reclaim = iota
	// stopPostmaster は、持ち主が死んで孤児になった postmaster を即時停止させる。
	// 即時停止なら postgres 自身がセグメントを消す。
	stopPostmaster
	// removeLeftovers は、postmaster まで死んでいる実行の、記録されたセグメントと
	// ディレクトリを消す。postmaster が SIGKILL されるとセグメントは残る。
	removeLeftovers
)

func (r reclaim) String() string {
	switch r {
	case leaveAlone:
		return "leaveAlone"
	case stopPostmaster:
		return "stopPostmaster"
	case removeLeftovers:
		return "removeLeftovers"
	default:
		return fmt.Sprintf("reclaim(%d)", int(r))
	}
}

// decideReclaim は、実行時ディレクトリを回収してよいか、どう回収するかを決める。
// 持ち主が読めないか生きている間は触らない。go test ./... は 36 個のパッケージを
// 並行して走らせるので、この条件が崩れると互いの DB を止め合う。
func decideReclaim(inst abandonedInstance, alive func(pid int) bool) reclaim {
	if inst.owner == 0 || alive(inst.owner) {
		return leaveAlone
	}
	if inst.postmaster != 0 && alive(inst.postmaster) {
		return stopPostmaster
	}
	return removeLeftovers
}

// parsePostmasterPID は data/postmaster.pid から postmaster の pid と、7 行目の
// 共有メモリの ID を読む。読めない値は 0 を返す。
func parsePostmasterPID(content string) (pid, shmID int) {
	lines := strings.Split(content, "\n")
	pid, _ = strconv.Atoi(strings.TrimSpace(lines[0]))
	if len(lines) < 7 {
		return pid, 0
	}
	fields := strings.Fields(lines[6])
	if len(fields) != 2 {
		return pid, 0
	}
	shmID, _ = strconv.Atoi(fields[1])
	return pid, shmID
}

// reclaimAbandonedInstances は、持ち主のテストバイナリが既に終了した実行時ディレクトリを
// 片づける。強制終了は defer の後始末を飛ばすので、残ったものを次の起動で照合して回収する。
// SIGKILL を捕まえる方式では、この経路を塞げない。
func reclaimAbandonedInstances(tempDir string) {
	dirs, err := filepath.Glob(filepath.Join(tempDir, runtimeDirPattern))
	if err != nil {
		return
	}
	for _, dir := range dirs {
		inst := readInstance(dir)
		switch decideReclaim(inst, processAlive) {
		case leaveAlone:
			continue
		case stopPostmaster:
			if err := stopOrphanedPostmaster(dir); err != nil {
				fmt.Fprintf(os.Stderr, "pgtest: cannot stop orphaned postmaster %d in %s: %v\n", inst.postmaster, dir, err)
				continue
			}
		case removeLeftovers:
			if inst.shmID != 0 {
				if err := removeSegment(inst.postmaster, inst.shmID); err != nil {
					fmt.Fprintf(os.Stderr, "pgtest: cannot remove shared memory segment %d of %s: %v\n", inst.shmID, dir, err)
					continue
				}
			}
		}
		if err := os.RemoveAll(dir); err != nil {
			fmt.Fprintf(os.Stderr, "pgtest: cannot remove abandoned runtime directory %s: %v\n", dir, err)
		}
	}
}

func readInstance(dir string) abandonedInstance {
	inst := abandonedInstance{dir: dir}
	if owner, err := os.ReadFile(filepath.Join(dir, ownerFile)); err == nil {
		inst.owner, _ = strconv.Atoi(strings.TrimSpace(string(owner)))
	}
	if pidFile, err := os.ReadFile(filepath.Join(dataDir(dir), "postmaster.pid")); err == nil {
		inst.postmaster, inst.shmID = parsePostmasterPID(string(pidFile))
	}
	return inst
}

// processAlive は pid のプロセスが存在するかを返す。他人のプロセスで EPERM が返る場合も
// 存在するとみなし、回収しない側に倒す。
func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

// stopOrphanedPostmaster は pg_ctl の即時停止で孤児の postmaster を止める。
func stopOrphanedPostmaster(dir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pgCtl := filepath.Join(postgresBinariesPath(), "bin", "pg_ctl")
	out, err := exec.CommandContext(ctx, pgCtl, "stop", "-m", "immediate", "-w", "-D", dataDir(dir)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
