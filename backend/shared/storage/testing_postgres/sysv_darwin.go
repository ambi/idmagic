package testing_postgres

import "golang.org/x/sys/unix"

// sysvUnavailable は、SysV 共有メモリを扱えない環境で理由を返す。
//
// macOS のサンドボックスは shmget による作成を通したうえで、その後の操作と削除を拒否する。
// postgres は起動に失敗し、作られたセグメントを誰も消せない。1 回の起動で 1 個漏れ、
// 上限の kern.sysv.shmmni は既定で 32 なので、DB テストの 36 パッケージを 1 巡するだけで
// 上限に届く。shmget で試すと試すこと自体が漏れるため、同じサンドボックスが拒否する
// sysctl の読み取りで判定する。
func sysvUnavailable() error {
	// 値は 64 ビットなので SysctlUint32 では ENOMEM になる。値そのものは使わない。
	_, err := unix.SysctlRaw("kern.sysv.shmmni")
	return err
}
