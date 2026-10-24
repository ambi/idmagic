---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-24
priority: p3
depends_on: []
change_kind: maintenance
spec_impact:
  kind: none
  reason: "PostgreSQL を使うテストと開発用インフラの起動方法だけを替える。製品の HTTP 契約、永続状態のスキーマ、ドメインイベント、外向きの呼び出し、本番の DB 接続は変更しない。"
---

# embedded-postgres を v2 へ移行し、テストハーネスと開発用インフラを新しい起動モデルに合わせる

## 動機

2026-10-24 の依存関係の一括更新で、`github.com/fergusstrange/embedded-postgres` は v1.34.0 のまま見送った。
v2（`github.com/fergusstrange/embedded-postgres/v2`、v2.0.1）は module path が変わる major 更新であり、`go get -u` では上がらない。
v1 の系列にはこれ以上の更新が出ない前提で、v2 の起動モデルへ移る。

v2 の変更のうち、このリポジトリに影響するものは次のとおりである（[移行ガイド](https://github.com/fergusstrange/embedded-postgres/blob/master/docs/migration-v2.md)）。

| v2 の変更 | 影響する箇所 |
| --- | --- |
| 独立した supervisor 実行ファイルを、実行時に GitHub Release から取得する | ネットワークを遮断した CI とサンドボックス内のテスト実行 |
| `RuntimePath` は一意な workspace の親になり、渡したパスを消さない | `testing_postgres` の実行時ディレクトリと持ち主の記録 |
| `BinariesPath` は `LocalProvider` に置き換わり、postgres、initdb、pg_ctl、psql、createdb を要求する | `binaries.go` の共有置き場 |
| `CachePath` が展開済みバイナリを共有し、v1 の archive cache は再利用されない | `binaries.go` の flock と完了印 |
| supervisor が持ち主の終了と panic の後に後始末をする | `reclaim.go` の孤児 postmaster の回収 |
| lib/pq の暗黙登録を削除した | lib/pq を間接依存として使う箇所の有無 |

## 対象範囲

- `backend/shared/storage/testing_postgres/` の `pgtest.go`、`binaries.go`、`reclaim.go` を v2 の API と起動モデルへ合わせる。
- `backend/cmd/idmagic-dev-infra/internal/devinfra/devinfra.go` を v2 の API へ合わせる。
- `go.mod` の require を `/v2` に置き換え、v1 を取り除く。

## 対象外

- PostgreSQL の major 版の変更。v2 の既定は PostgreSQL 18.6 であり、現在の V18 と同じ系列に留める。
- テスト以外の DB 接続と本番のスキーマ適用手順。

## 設計

着手時に次の問いを解決する。

- supervisor の取得経路。実行時に GitHub Release から取る方式を CI とサンドボックスで受け入れるか、`EP_SUPERVISOR` または `Config.Supervisor` で導入済みの実行ファイルを渡すか。後者なら、その実行ファイルの版の固定方法（`mise.toml` の `go:` バックエンドなど）も決める。
- `CachePath` による共有で、`binaries.go` の flock と完了印が要らなくなるか。`go test ./...` がパッケージのテストバイナリを並行に起動しても、展開途中の置き場が見えないことを確かめてから外す。
- supervisor の後始末で、`reclaim.go` が要らなくなるか。SIGKILL でテストバイナリが終わったとき、postmaster と SysV 共有メモリが残らないことを確かめてから外す。
- `Port(0)` と `ConnectionURL()` を使い、`freePort` と DSN の組み立てを外すか。

## 計画

1. 上の問いを、v2 の挙動を観測して解決し、設計の節に結論を書く。
2. import を `/v2` に替え、ハーネスと dev-infra を書き換える。
3. 不要になった仕組みを取り除く。

## タスク

- [ ] T001 [Design] supervisor の取得経路、`binaries.go` と `reclaim.go` の要否を観測して決める。
- [ ] T002 [App] `testing_postgres` と `devinfra` を v2 へ移行する。
- [ ] T003 [Verify] DB テスト全件と開発用インフラの起動を確かめる。

## 検証

- `mise run test-go-race` をサンドボックスの外で実行し、DB テストがスキップされずに全件通ることを確かめる。サンドボックス内では DB テストが黙ってスキップされるためである。
- テストバイナリを強制終了した後の再実行で、postmaster と共有メモリが残らないことを確かめる。
- `mise run dev` で開発用インフラが起動することを確かめる。
- `mise run verify` を通す。

## リスク

- supervisor を実行時に取得する方式を採ると、GitHub Release への到達性がテストの前提に加わる。到達できない環境では DB テストがスキップされ、緑のまま検査が抜け落ちる。
- 共有キャッシュと回収の仕組みを早く外しすぎると、並行実行での相互停止や共有メモリの枯渇が再発する。
