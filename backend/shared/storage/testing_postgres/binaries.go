package testing_postgres

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

// startWithSharedBinaries は、プロセス間で共有するバイナリ置き場 binariesPath を使って
// startPostgres を呼ぶ。`go test ./...` はパッケージのテストバイナリを並行に動かす。embedded-postgres の展開は
// プロセス内のロックしか取らず、ファイルを一つずつ最終位置へ移すので、別のプロセスからは
// bin/pg_ctl があるのに bin/postgres がまだない置き場が見える。そこで展開を終えた置き場に
// 完了の印を残し、印がない間は置き場のロックを持つ一つのプロセスだけが展開と起動を行う。
// 印のある置き場は読むだけなので、ロックを放してから並行に起動する。
func startWithSharedBinaries(binariesPath string, startPostgres func() error) error {
	if err := os.MkdirAll(filepath.Dir(binariesPath), 0o755); err != nil {
		return err
	}
	lock, err := os.OpenFile(binariesPath+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Close() }()
	fd := int(lock.Fd())
	if err := syscall.Flock(fd, syscall.LOCK_EX); err != nil {
		return err
	}
	unlock := func() { _ = syscall.Flock(fd, syscall.LOCK_UN) }

	complete := binariesPath + ".complete"
	if _, err := os.Stat(complete); err == nil {
		unlock()
		return startPostgres()
	} else if !errors.Is(err, os.ErrNotExist) {
		unlock()
		return err
	}
	defer unlock()
	// 印のない置き場は展開の途中で止まったものかもしれないので、消して展開し直させる。
	if err := os.RemoveAll(binariesPath); err != nil {
		return err
	}
	if err := startPostgres(); err != nil {
		return err
	}
	return os.WriteFile(complete, nil, 0o600)
}
