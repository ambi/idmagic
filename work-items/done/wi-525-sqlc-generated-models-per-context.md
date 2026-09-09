---
depends_on: []
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-10
priority: p1
change_kind: refactor
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 生成コードから未参照の構造体が消えるだけで、利用者が観測する振る舞い、HTTP 契約、設定、運用手順のいずれも変わらない。リリースの読み手に伝える差分がない。
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - sqlc.yaml
    - backend/jobs/db_postgres
  tests: []
  stop_before_reading:
    - infra/schema/postgres.sql
    - backend/oauth2
    - frontend
spec_impact:
  kind: none
  reason: "sqlc が出す構造体のうち、その context の query が参照しないものを出力しなくなるだけである。SQL、query 関数の署名、HTTP 契約、データベーススキーマのいずれも変わらず、規範的シナリオと TypeSpec シンボルの追加、変更、退役を伴わない。"
---

# sqlc の生成モデルを、その context の query が使う型だけに絞る

## Motivation

`sqlc.yaml` の 30 個の生成エントリーは、すべて同じ `schema: infra/schema/postgres.sql` を指している。
sqlc は既定でスキーマ内の全テーブルの構造体を出すので、**生成対象の 30 個の `db_postgres` パッケージが、1054 行 88 型の同一の `models.go` を持っている**。
`backend` 配下の `db_postgres` パッケージは 36 個あり、残り 6 個は生成エントリーを持たないので `models.go` を持たない。
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

- [x] T001 [Verify] 変更前の `models.go` の行数と型数を、代表 3 パッケージについて記録する。
- [x] T002 [App] `sqlc.yaml` の全生成エントリーに `omit_unused_structs: true` を加え、意図をコメントに残す。実行: `mise run sqlc-generate`。
- [x] T003 [Verify] `mise run build-go` と `mise run test-go-changed` を通し、削除された型への参照がないことを確かめる。
- [x] T004 [Verify] `mise run verify` を通し、前後の行数比較を Completion に残す。

## Verification

規範的な製品要求を持たない変更なので、Acceptance RED と Unit RED の代わりに、この項目が消そうとしている状態そのものを測る 2 つの検査を置く。
どちらも変更前に実際に失敗することを確かめてから実装に入る。

- Acceptance 相当 RED (生成物の観測境界):
  `test "$(rg -c '^type ' backend/jobs/db_postgres/models.go)" -eq 1`。
  jobs の query が触れるテーブルは `jobs` の 1 つだけなので、生成される型も 1 つであるべきである。
- Unit 相当 RED (生成設定の観測境界):
  `sqlc.yaml` の全エントリーが `omit_unused_structs: true` を宣言していること。
  `yq '[.sql[].gen.go.omit_unused_structs | select(. == true)] | length' sqlc.yaml` が `yq '.sql | length' sqlc.yaml` と一致する。
- `mise run sqlc-generate`
- `mise run build-go`
- `mise run test-go-changed`
- `mise run verify`

## Risk Notes

- 試行で走らせたのは `build-go` と `test-go-changed` だけである。`test-go-changed` は変更パッケージとその逆依存を対象にするので、削除された型を参照する production と test は原理的にこの範囲へ入るが、`verify` 全体は未確認である。実装時に通す。
- 型が減ることで、いま参照されていないが将来参照する予定の構造体が消える。query を足せば再び生成されるので、失われる情報はない。
- 生成エントリーを新しく足すときに `omit_unused_structs` を書き忘れると、その context だけ 88 型に戻る。`sqlc.yaml` のコメントで意図を示すが、機械的な強制は入れない。エントリー追加は稀であり、追加時の diff に 1000 行の `models.go` が現れれば気付く。

## Completion

- **Completed At**: 2026-09-10
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。
  規範的シナリオ、標準要件、TypeSpec 宣言のいずれも追加、変更、退役していない。
  意味上の差分は生成コードの側だけにある。
  `sqlc.yaml` の 30 個の生成エントリーすべてが `omit_unused_structs: true` を宣言するようになり、
  各 context の `models.go` は、その context の query が参照するテーブルの構造体だけを持つ。
  30 ファイルの合計は 31,620 行 2,640 型から 810 行 45 型へ減り、30,810 行 2,595 型が消えた。
  代表 3 パッケージでは `backend/jobs/db_postgres` が 1054 行 88 型から 30 行 1 型へ、
  `backend/oauth2/db_postgres` が 106 行 7 型へ、`backend/tenancy/db_postgres` が 87 行 5 型へ変わった。
  `querier.go` と `*.sql.go` は 1 バイトも変わっておらず、
  変更されたのは 30 個の `models.go` と `sqlc.yaml` だけである。
  query 関数の署名が変わらないという Design の前提は、この diff の形そのもので確かめられた。
  意図は `sqlc.yaml` 冒頭のコメントに置いた。
- **Acceptance RED Evidence**:
  - **Test**: `test "$(rg -c '^type ' backend/jobs/db_postgres/models.go)" -eq 1`
  - **Requirement**: N/A: 生成設定の変更であり、製品の規範要件を持たない。
    この項目が消そうとしている状態そのものを測る検査を代わりに置いた。
  - **Observed Failure**: 実装前に `FAIL: actual=88` を観測した。
    `backend/jobs/db_postgres` の query が触れるテーブルは `jobs` の 1 つだけなのに、
    生成された構造体は 88 個あった。
  - **Detection Reason**: 観測可能な境界は「リポジトリに入っている生成物が、その context の query が必要とする型だけを持つこと」である。
    この検査は生成物そのものを数えるので、設定は書いたが生成を走らせていない、
    一部のエントリーにしか設定が届いていない、
    設定名を綴り間違えて sqlc が黙って無視した、の 3 つをいずれも落とす。
    `sqlc.yaml` の中身だけを読む検査では、この 3 つを区別できない。
- **Unit RED Evidence**:
  - **Test**: `yq '[.sql[].gen.go.omit_unused_structs | select(. == true)] | length' sqlc.yaml` が
    `yq '.sql | length' sqlc.yaml` と一致すること。
  - **Requirement**: N/A: 同上。生成設定の変更であり、製品の規範要件を持たない。
  - **Observed Failure**: 実装前に `FAIL: declared=0 expected=30` を観測した。
  - **Detection Reason**: 生成物の検査は 1 つの context しか見ないので、
    30 個のエントリーのうち 29 個にしか設定が届いていない状態を落とせない。
    この検査は設定側を全数で見るので、その取りこぼしを検出する。
    2 つの検査は入力が異なり、片方だけでは通ってしまう誤りが互いにある。
- **Change-Resistance Results**:
  `risk: low` は代表的な誤実装の検出を求めないが、検査の対をこの費用で確かめられるので記録する。
  `sqlc.yaml` の `backend/jobs/db_postgres` エントリーからだけ `omit_unused_structs: true` を外して
  `mise run sqlc-generate` を走らせると、Unit 側は `FAIL: declared=29 expected=30`、
  Acceptance 側は `FAIL: actual=88` を返し、両方が検出した。
  設定を戻して再生成すると、両方が通る状態へ戻った。
  **方法の限界を記録する。** この 2 つの検査が見るのは「未参照の構造体が出力から外れていること」だけである。
  生成された構造体の中身が正しいこと、query 関数の署名が変わっていないことは検査していない。
  後者は `mise run build-go` と `mise run test-go-race` が担保しており、
  `querier.go` と `*.sql.go` に差分が出ていないことでも裏付けた。
  また `mise run verify` に sqlc の生成物ずれを検出するゲートはないので、
  `sqlc.yaml` を変えて生成を忘れた状態は、この 2 つの検査を人が走らせない限り通ってしまう。
- **Verification Results**:
  - `mise run sqlc-generate` - passed (1.6 秒)
  - `mise run build-go` - passed
  - `mise run test-go-changed` - passed (43 パッケージ、失敗なし)。
    2026-09-10 の試行で同じ生成物に対して走っているため、結果はすべて Go のテストキャッシュからの再利用である。
    キャッシュを介さない実行は下の `mise run verify` の `test-go-race` が担う。
  - `mise run check-work-items` - passed
  - `mise run check-ids` - passed
  - `mise run spec-diff` - `no normative specification change against main`
  - `mise run verify` - passed (31 タスク、44.02 秒)。
    `test-ui-e2e` は走らせていない。変更は Go の生成構造体だけで、
    HTTP 契約、TypeSpec、`frontend/` のいずれにも差分がなく、ブラウザへ届く経路がない。
