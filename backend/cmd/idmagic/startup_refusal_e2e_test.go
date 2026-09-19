package main

// docs/domain/system/scenarios.feature.md の REQ-SYSTEM-016 が宣言する、起動時設定の
// 検証に失敗した具体例を Run() の入口から観測する。
//
// 拒否が防いだ効果を読む必要がある (wi-392)。設定を読む関数だけを呼ぶ試験は「集約された
// エラーが返る」までしか言えず、そのエラーが返る前に listener を開き、PostgreSQL へ
// つなぎ、seed を当てた実装を落とせない。ここで観測するのは、到達しなかった先である。
//
// DATABASE_URL が閉じたポートを指しているのが観測の仕掛けである。検証より先に Assemble
// へ進む実装は接続を試み、返るのは接続の失敗になる。返ったエラーが設定の読み込みであり、
// 不正なキーを名指していることが、接続を始めていないことの証拠になる。

import (
	"net"
	"strings"
	"testing"
)

// freeLoopbackAddr は誰も待ち受けていない loopback のアドレスを返す。確保した listener は
// すぐ閉じるので、このアドレスは「開いていない」状態から始まる。
func freeLoopbackAddr(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve a loopback port: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("release the reserved port: %v", err)
	}
	return addr
}

//spec:covers EX-SYSTEM-016-02, EX-SYSTEM-016-03, EX-SYSTEM-016-04: 必須値の欠落・範囲外の値・矛盾する組み合わせを該当キーつきの集約エラーで返し、リスナーの待ち受けも永続化依存への接続も seed の適用も始めずに終了すること。
func TestRunRefusesInvalidStartupConfigurationBeforeAnySideEffect(t *testing.T) {
	addr := freeLoopbackAddr(t)
	t.Setenv("ADDR", addr)
	// 矛盾する組み合わせ (016-04): postgres を選んで DSN を空にする。
	t.Setenv("PERSISTENCE", "postgres")
	t.Setenv("DATABASE_URL", "")
	// 必須値の欠落 (016-02): SMTP を選んで host と from を与えない。
	t.Setenv("EMAIL_SENDER", "smtp")
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_FROM", "")
	// 範囲外の値 (016-03): 負のホップ数。
	t.Setenv("TRUSTED_FORWARDED_HOPS", "-1")

	err := Run()
	if err == nil {
		t.Fatal("Run() accepted an invalid startup configuration")
	}
	// Assemble へ進んでいれば、返るのは接続の失敗であって設定の読み込みではない。
	if !strings.Contains(err.Error(), "load startup configuration") {
		t.Fatalf("Run() error = %v, want the startup configuration to be refused before assembly", err)
	}
	for _, key := range []string{"DATABASE_URL", "SMTP_HOST", "SMTP_FROM", "TRUSTED_FORWARDED_HOPS"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("Run() error = %v, want it to name %s", err, key)
		}
	}

	// listener を開いていない。開いてから検証する実装は、返り値で誤りを報告しながら
	// ポートを掴んだままになる。
	conn, dialErr := net.Dial("tcp", addr)
	if dialErr == nil {
		_ = conn.Close()
		t.Fatalf("something is listening on %s although startup was refused", addr)
	}
}
