---
depends_on: []
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-10
priority: p1
change_kind: refactor
spec_impact:
  kind: none
  reason: "sqlc が出す構造体のうち、その context の query が参照しないものを出力しなくなるだけである。SQL、query 関数の署名、HTTP 契約、データベーススキーマのいずれも変わらず、規範的シナリオと TypeSpec シンボルの追加、変更、退役を伴わない。"
---

# sqlc の生成モデルを、その context の query が使う型だけに絞る

## Motivation

`sqlc.yaml` の 30 個の生成エントリーは、すべて同じ `schema: infra/schema/postgres.sql` を指している。
sqlc は既定でスキーマ内の全テーブルの構造体を出すので、**36 個の `db_postgres` パッケージが、1054 行 88 型の同一の `models.go` を持っている**。
`backend/jobs/db_postgres` が実際に query で使うテーブルは 1 つである。

この重複は 3 か所で費用に変わる。

1 つ目は diff の膨張である。
wi-257 の commit `3d8da80c` は 77 ファイル、2876 行の追加だが、そのうち **15 ファイルは `db_postgres/models.go` で、各 27 行、すべて機械生成**である。
logout と無関係な jobs、saml、tenancy、wsfederation が含まれる。

2 つ目は検証範囲の膨張である。
wi-257 の Timing Analysis は、単一最長コマンドである変更 package の race test (初回 76.70 秒、最終 84.22 秒) について「共有 PostgreSQL schema により多数 context の `models.go` が機械更新され、reverse dependency の対象が広がったことが原因である」と記録している。

3 つ目は読む時間である。
同じ Timing Analysis は、PostgreSQL・discovery の約 5 分について「実行より fixture と生成コード確認が中心だった」と記録している。
確認すべき生成差分の大半は、その work item と関係のない context のものである。

### 試行の実測

2026-09-10、30 個の生成エントリーすべてに `omit_unused_structs: true` を足して生成し、検証してから元に戻した。

| 項目 | 結果 |
| --- | --- |
| 生成コードの削減 | 30 ファイル、30,810 行削除 |
| `backend/jobs/db_postgres/models.go` | 1054 行 88 型 → 30 行 1 型 |
| `backend/oauth2/db_postgres/models.go` | 1054 行 88 型 → 106 行 7 型 |
| `mise run sqlc-generate` | 1.67 秒で成功 |
| `mise run build-go` | 成功 |
| `mise run test-go-changed` | 終了コード 0、36 パッケージ通過、失敗なし |

削除された型を参照しているコードは 1 か所もなかった。

## Scope

- `sqlc.yaml` の全生成エントリーに `omit_unused_structs: true` を加える。
- 生成物を更新し、リポジトリに入っている `models.go` を絞り込んだ状態にする。
- 生成設定の意図を `sqlc.yaml` のコメントに残し、将来のエントリー追加で同じ設定が漏れないようにする。

## Out of Scope

- `infra/schema/postgres.sql` を context 別のスキーマファイルへ分割すること。Design のとおり、より小さい手段で同じ効果が得られる。
- 共有スキーマそのものの分解、および context 別データベースへの移行。データ配置の決定であり、生成コードの重複とは別の問題である。
- `db_memory` adapter の扱い。wi-257 の logout では 72 行であり、律速を示す材料がない。
- work item の所要時間に関するほかの施策。wi-526 から wi-529 が持つ。

## Design

`omit_unused_structs` は sqlc の Go 出力オプションで、その生成エントリーの query が参照しない構造体を出力から外す。
使っている sqlc は 1.31.1 である。
query が参照する型は残るので、`querier.go` と `*.sql.go` の署名は変わらない。

**退けた案: スキーマファイルを context 別に分割する。**
`sqlc.yaml` の各エントリーに、その context が使うテーブルだけを含むスキーマを与えれば同じ結果になる。
しかしテーブル間の外部キーは context をまたぐので、分割の単位を決める作業と、分割後のスキーマが本番の 1 枚のスキーマと一致し続けることを保証する仕掛けが要る。
`mise run check-schema` が担保しているのは 1 枚のスキーマの収束であり、分割した複数枚には及ばない。
30 行の設定追加で同じ効果が得られる以上、この費用を払う理由がない。

**退けた案: 生成物をリポジトリから外す。**
生成コードが commit に現れないので diff は静かになるが、検証範囲は縮まず、生成の再現に依存が増える。
wi-267 が生成ディレクトリを平坦化した経緯もあり、現在の配置を変える理由がない。

## Plan

1. `sqlc.yaml` を変更する前に、現在の `models.go` の行数と型数を記録する。前後比較の基準にする。
2. 全エントリーに `omit_unused_structs: true` を加え、`mise run sqlc-generate` を走らせる。
3. `mise run build-go` と `mise run test-go-changed` で、削除された型への参照がないことを確かめる。
4. `mise run verify` を通す。試行では走らせていないので、ここが本番の確認になる。

実装内容を変える未決事項はない。

## Tasks

- [ ] T001 [Verify] 変更前の `models.go` の行数と型数を、代表 3 パッケージについて記録する。
- [ ] T002 [App] `sqlc.yaml` の全生成エントリーに `omit_unused_structs: true` を加え、意図をコメントに残す。実行: `mise run sqlc-generate`。
- [ ] T003 [Verify] `mise run build-go` と `mise run test-go-changed` を通し、削除された型への参照がないことを確かめる。
- [ ] T004 [Verify] `mise run verify` を通し、前後の行数比較を Completion に残す。

## Verification

- Acceptance RED: 変更前の `backend/jobs/db_postgres/models.go` が、jobs の query が参照しない 87 個の型を含むことを観測する。
- `mise run sqlc-generate`
- `mise run build-go`
- `mise run test-go-changed`
- `mise run verify`

## Risk Notes

- 試行で走らせたのは `build-go` と `test-go-changed` だけである。`test-go-changed` は変更パッケージとその逆依存を対象にするので、削除された型を参照する production と test は原理的にこの範囲へ入るが、`verify` 全体は未確認である。実装時に通す。
- 型が減ることで、いま参照されていないが将来参照する予定の構造体が消える。query を足せば再び生成されるので、失われる情報はない。
- 生成エントリーを新しく足すときに `omit_unused_structs` を書き忘れると、その context だけ 88 型に戻る。`sqlc.yaml` のコメントで意図を示すが、機械的な強制は入れない。エントリー追加は稀であり、追加時の diff に 1000 行の `models.go` が現れれば気付く。
