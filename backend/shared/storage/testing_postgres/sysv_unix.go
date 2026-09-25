//go:build darwin || linux

package testing_postgres

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// removeSegment は、記録した postmaster が作り、接続しているプロセスが無いセグメントだけを
// 消す。ID は再利用されるので、作成者を照合しないと別のセグメントを消しうる。key は
// macOS の構造体から読めないため照合に使わない。
func removeSegment(creator, id int) error {
	var desc unix.SysvShmDesc
	if _, err := unix.SysvShmCtl(id, unix.IPC_STAT, &desc); err != nil {
		if err == unix.EINVAL {
			return nil // 既に消えている。
		}
		return err
	}
	if int(desc.Cpid) != creator {
		return nil // 別のセグメントが ID を再利用している。
	}
	if desc.Nattch != 0 {
		return fmt.Errorf("segment still has %d attached process(es)", desc.Nattch)
	}
	_, err := unix.SysvShmCtl(id, unix.IPC_RMID, nil)
	return err
}
