---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-26
priority: p1
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: テスト用の embedded-postgres の起動と後始末だけを変える。製品の振る舞いも公開契約も変わらないので、リリースの読み手に見えるものが無い。
  references: []
spec_impact: { kind: none, reason: "テスト基盤の後始末の修正であり、シナリオも製品の振る舞いも変えない。" }
initial_context:
  specification: []
  typespec: []
  source:
    - backend/shared/storage/testing_postgres/pgtest.go
  tests: []
  stop_before_reading:
    - backend/shared/storage/db_postgres
    - frontend
---

# テスト用 embedded-postgres が SysV 共有メモリを漏らさないようにする

## 動機

ローカルで DB を使うテストを繰り返すと、SysV 共有メモリのセグメントが macOS の上限（`kern.sysv.shmmni` = 32）に達する。
上限に達すると initdb が `could not create shared memory segment: No space left on device` で失敗し、`testing_postgres` は DB を使うテストを**すべて黙ってスキップ**する。
`mise run verify` は成功のまま終わるので、PostgreSQL アダプターのテストが 1 件も実行されていないことに気づけない。
2026-09-26 には、接続数 0 の孤立したセグメントが 32 個残り、postgres のプロセスは 1 つも動いていなかった。

漏れる経路は 2 つあり、どちらも 2026-09-26 に再現した。

| 経路 | 再現 | 漏れる理由 |
|---|---|---|
| サンドボックス内での起動 | Claude Code のサンドボックス内でテストバイナリを 3 回起動すると、セグメントが 3 個増えた | サンドボックスは `shmget` の作成を通すが、`IPC_RMID` と `sysctl kern.sysv.*` の読み取りを拒否する。postgres は作成直後の操作に失敗して終了し、作られたセグメントを誰も消せない。DB テストのパッケージは 36 個あるので、サンドボックス内で `mise run verify` を 1 回走らせるだけで上限に届く |
| テストバイナリの強制終了 | テストバイナリを SIGKILL すると postmaster が孤児として動き続けた。その postmaster を SIGKILL すると、接続数 0 のセグメントが残った | `Main` の `defer cleanup()` が走らない。`go test` のタイムアウト、ツールによるプロセスの強制終了がこの経路に入る |

孤児の postmaster を SIGQUIT（即時停止）で止めた場合は、postgres 自身がセグメントを消した。

## 対象範囲

- SysV 共有メモリを扱えない環境では、postgres を起動せずに理由を示して DB テストをスキップする。
- 起動のたびに、持ち主のテストバイナリが既に終了している実行時ディレクトリを回収する。生きている postmaster は即時停止させ、死んでいる postmaster のセグメントは削除し、ディレクトリを消す。
- 持ち主を判定できるよう、実行時ディレクトリへ持ち主の pid を書く。

## 対象外

- 既に漏れて、対応する実行時ディレクトリが無いセグメントの自動削除。どの実行が作ったかを判定できないので、利用者のほかのソフトウェアのセグメントと区別できない。本変更の後は新たに生まれない。
- DB を起動できないときにテストを失敗させること。CI とローカルの扱いを決める別の判断である。
- `kern.sysv.shmmni` などの OS の上限値の変更。

## 設計

回収は、起動時の照合として行う。
強制終了を捕まえて後始末する方式は、SIGKILL を捕まえられないので採らない。
残っているものを次の起動で見つけて片づければ、どの経路で残ったかを問わず回収できる。

回収の判断は、副作用を持たない関数に置く。

```go
type abandonedInstance struct {
  dir        string
  owner      int // 実行時ディレクトリを作ったテストバイナリの pid
  postmaster int // data/postmaster.pid の 1 行目。無ければ 0
  shmID      int // data/postmaster.pid の 7 行目の ID。無ければ 0
}

type reclaim int // leaveAlone、stopPostmaster、removeLeftovers

func decideReclaim(inst abandonedInstance, alive func(pid int) bool) reclaim
```

作用（プロセスの生存確認、`pg_ctl stop -m immediate`、`shmctl`、ディレクトリの削除）は `Main` の側から与える。

| 状態 | 判断 |
|---|---|
| 持ち主の pid が読めない | 触らない。作成直後で書き込み前の実行と区別できない |
| 持ち主が生きている | 触らない。並行して走る別パッケージのテストである |
| 持ち主が死に、postmaster が生きている | `pg_ctl stop -m immediate` で止め、ディレクトリを消す |
| 持ち主も postmaster も死んでいる | 記録されたセグメントの作成者が記録の postmaster と一致し、接続数が 0 なら削除する。ディレクトリを消す |

セグメントを削除する前に作成者の pid を照合するのは、ID が再利用されて別のセグメントを指している場合に消さないためである。key は macOS の `SysvShmDesc` から読めないので照合に使わない。

SysV 共有メモリを扱えるかの事前検査は、macOS では `sysctl kern.sysv.shmmni` の読み取りで行う。
サンドボックスはこの読み取りを拒否し、しかもこの検査はセグメントを作らない。
`shmget` で試す方式は、サンドボックス内では試すこと自体がセグメントを 1 個漏らすので採らない。
Linux ではこの検査を持たない。

## 計画

1. `decideReclaim` を表の 4 状態について単体テストで固定する。
2. 回収の作用と事前検査を `Main` へ配線する。
3. 実際に postmaster を孤児にし、次の起動で回収されることを確かめる。サンドボックス内での起動でセグメントが増えないことを確かめる。

## タスク

- [x] T001 [Unit] `decideReclaim` の単体テストで RED を確認し、GREEN にする。
- [x] T002 [Wiring] 事前検査と回収を `Main` へ配線する。
- [x] T003 [Fault] 孤児の postmaster とサンドボックス内の起動で、セグメントが残らないことを確かめる。
- [x] T004 [Verify] サンドボックス外で `mise run verify` を通し、DB テストがスキップされていないことを確かめる。

## 検証

- `mise run test-go-package -- ./backend/shared/storage/testing_postgres`
- テストバイナリを SIGKILL して postmaster を孤児にし、次の起動の後にセグメント数が元へ戻る。
- サンドボックス内で DB テストのパッケージを起動しても、セグメント数が増えない。
- サンドボックス外で `mise run verify`

## リスク

- **生きている別のテストの postgres を止める。** 持ち主の pid が生きている間は触らない。`go test ./...` は 36 個のパッケージを並行して走らせるので、この条件が崩れると並行実行が互いの DB を止める。
- **pid の再利用。** 持ち主の pid が無関係のプロセスに再利用されていれば、回収しないだけで安全側に倒れる。postmaster の pid が再利用されている場合は、`pg_ctl` が対象を postgres と確かめずにシグナルを送る。`pg_ctl` 自身と同じ前提であり、本変更では追加の照合を持たない。

## 完了

- **Completed At**: 2026-09-26
- **Summary**:
  `mise run spec-diff` は規範仕様の差分を報告しない。変更はテスト基盤に閉じる。
  `testing_postgres.Main` は、SysV 共有メモリを扱えない環境（macOS のサンドボックス）では postgres を起動せずに DB テストをスキップする。
  起動のたびに、持ち主のテストバイナリが終了した `idmagic-pgtest-*` を回収する。孤児の postmaster は `pg_ctl stop -m immediate` で止め、postmaster まで死んでいる場合は記録された共有メモリを作成者の pid と接続数 0 を確かめて削除する。
  embedded-postgres は起動時に RuntimePath を消して作り直すので、持ち主の pid は `idmagic-pgtest-*/owner.pid` に置き、RuntimePath は `idmagic-pgtest-*/runtime` とした。
- **Acceptance RED Evidence**:
  - **Test**: サンドボックス内で DB テストのバイナリ（`backend/apitoken/db_postgres`）を 3 回起動し、サンドボックス外の `ipcs -m` でセグメント数を数えた。
  - **Requirement**: N/A: 製品の振る舞いを変えないテスト基盤の変更である。
  - **Observed Failure**: 修正前は 3 回の起動でセグメントが 3 個増えた。テストバイナリを SIGKILL した後に孤児の postmaster を SIGKILL すると、接続数 0 のセグメントが 1 個残った。
  - **Detection Reason**: セグメント数そのものを観測するので、テストのスキップや成功の表示に惑わされない。修正後は、サンドボックス内の 3 回の起動で増加 0、孤児の postmaster と SIGKILL された postmaster のどちらも、次の起動の後にセグメント数とディレクトリ数が 0 に戻った。
- **Unit RED Evidence**:
  - **Test**: `backend/shared/storage/testing_postgres/reclaim_test.go` の `TestDecideReclaim`、`TestParsePostmasterPID`
  - **Requirement**: N/A: 製品の振る舞いを変えないテスト基盤の変更である。
  - **Observed Failure**: 実装前は `undefined: abandonedInstance` でビルドに失敗した。
  - **Detection Reason**: 持ち主が生きている実行を止める誤り（並行実行の DB を止め合う）と、孤児を放置する誤りを、4 状態の表で区別する。
- **Change-Resistance Results**:
  - `mise run test-go-mutation -- backend/shared/storage/testing_postgres`: 5 件を試し 4 件を検出した。生き残った 1 件は `parsePostmasterPID` の `len(lines) < 7` → `<= 7` で、7 行目の末尾に改行が無いファイルでしか区別できない。postgres は各行を改行付きで書くので、実質的に等価と判断した。未被覆の 33 件はプロセスの停止、共有メモリの削除、ディレクトリの走査という作用の側で、上の実機での再現が受け持つ。
  - 実機での再現で、持ち主の pid を RuntimePath の中に置いた最初の実装は、embedded-postgres が起動時にそのディレクトリを消すため孤児を 1 件も回収できなかった。事前検査の最初の実装（`SysctlUint32`）は、値が 64 ビットのためサンドボックス外でも `ENOMEM` を返し、すべての DB テストをスキップさせた。どちらも実機の再現で見つけて直した。
- **Verification Results**:
  - `mise run test-go-package -- ./backend/shared/storage/testing_postgres` - 成功
  - `mise run lint-go` - 成功
  - `GOOS=linux go build ./backend/shared/storage/testing_postgres` - 成功
  - `mise run verify`（サンドボックス外で実行し、DB テストを実行させた） - 成功。実行後の SysV 共有メモリのセグメントは 0 個
