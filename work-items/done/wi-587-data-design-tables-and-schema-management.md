---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-16
priority: p2
depends_on: [wi-583-normalize-design-document-terminology]
change_kind: docs
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: "設計文書と文書検査だけを変え、製品の振る舞い、設定、運用者の手順を変えないため、リリースの読者に伝える差分がない。"
  references: []
spec_impact:
  kind: none
  reason: "現行スキーマの読み方とスキーマ管理の設計を書き下すだけで、列、制約、データ、移行手順は変えない。"
initial_context:
  specification: [docs/design/data/database.md, docs/design/data/lifecycle.md, docs/design/data/README.md, docs/operations/maintenance.md, docs/architecture/deployment.md, docs/design/infrastructure/platform.md, docs/design/reliability/recovery.md, docs/domain/tenancy/states.md, docs/domain/identity-management/states.md, docs/domain/audit/decisions.md, docs/domain/authentication/internals.md]
  typespec: []
  source: [infra/schema/postgres.sql, infra/schema/README.md, infra/schema/check-convergence.sh, infra/schema/data-migrations, backend/application/db_postgres/applications.sql, backend/audit/ports/tenant_salt_store.go, tools/check/src/registry.ts, tools/check/src/slo-references.ts, tools/check/src/check-slo-references.ts]
  tests: [tools/check/src/slo-references.test.ts]
  stop_before_reading: [frontend, spec, backend/datakeys/usecases, backend/idmanagement/user/usecases]
---

# データ設計にテーブルの役割と関係、スキーマ管理の設計を書く

## Motivation

[データベース設計](../../docs/design/data/database.md)の ER 図は、六つの機能領域に分けて全テーブルを示す。
しかし Mermaid の各テーブルは本体が空で、線が外部キーを表すだけである。
テーブル名から役割が読み取れるものばかりではなく、`tenant_usages`、`mfa_enrollment_bypasses`、`scim_user_refs` が何のために存在し、どの Bounded Context が所有し、隣のテーブルとどういう関係にあるのかは、正本の SQL を読むまで分からない。
図はテーブルの存在を見落とさないためには働くが、スキーマを理解するためには働いていない。

スキーマ管理とマイグレーションの設計も薄い。
`psqldef` で差分を適用すること、起動時にスキーマを移行しないこと、構造差分で表現できない変更は work item か専用 SQL に書くことが、[ポートとアダプター](../../docs/design/data/database.md#ポートとアダプター)の節に一段落あるだけである。
拡張と縮小の二段に分ける方針は [データライフサイクル設計](../../docs/design/data/lifecycle.md)に一行、[保守](../../docs/operations/maintenance.md)に一行あり、正本がどこかが決まっていない。
`infra/schema/data-migrations/` が何を持ち、いつ実行され、誰が後退を判断するのかはどこにも書かれていない。

[データライフサイクル設計](../../docs/design/data/lifecycle.md)は 15 行の表一枚で、各行が一文の原則を述べる。
書かれている内容は正しいが、他文書の要約に近く、この文書を読んだ人が何を知った状態になるのかが決まっていない。

## Scope

- [データベース設計](../../docs/design/data/database.md)の各 ER 図に、テーブルの役割、所有 Context、`tenant_id` の保持区分を示す表を添える。
- スキーマ管理とマイグレーションの設計を、独立した文書 `docs/design/data/schema-management.md` にまとめる。
- [データライフサイクル設計](../../docs/design/data/lifecycle.md)の担当範囲を確定し、その範囲を満たすまで書き足す。
- [データ設計の索引](../../docs/design/data/README.md)と `DOCUMENTATION_GUIDE.md` §4.9 の記述を追従させる。

## Out of Scope

- 列、型、索引、制約の一覧。`infra/schema/postgres.sql` が正本であり、文書へ写さない（`DOCUMENTATION_GUIDE.md` §4.9）。
- スキーマそのものの変更。テーブルの追加、列の変更、制約の追加は含めない。
- 保持期間の値。各 Context の `standards.md` と `decisions.md`、[品質要求](../../docs/requirements/quality.md)が正本である。
- データ層の拡張性。パーティショニングと読み取りレプリカは [[wi-164-data-tier-scalability-partitioning-read-replica-pooling]] が扱う。
- バックアップと復元の設計。[[wi-588-reliability-and-performance-design]] が扱う。
- CSV インポートの成果物が消えない欠陥。着手時の読み取りで見つかり、[[wi-611-csv-import-artifacts-are-never-deleted]] に起票した。
- 復元で消去済みのデータが戻る問題の仕組み。この work item は問題を `lifecycle.md` に書くだけで、仕組みは [[wi-612-reapply-erasure-after-restore]] に起票した。
- テナントの物理削除。[[wi-494-system-console-tenant-management-and-resource-inspection]] が扱い、`lifecycle.md` には現行のスキーマから決まる順序を（仮）として書く。

## Design

### テーブルの説明をどう持つか

手で書いてよいのは、**コードから復元できないもの**に限る。
列、型、索引、制約はスキーマファイルにあり、概念とその関係は TypeSpec のモデルにある。
復元できないのは次の三つで、これを機能領域ごとの表として ER 図の隣に置く。

| 列 | 内容 | なぜコードから読めないか |
| --- | --- | --- |
| テーブル | 名前 | — |
| 役割 | 何のために存在するか（一行） | 名前と列からは目的が決まらない |
| 所有 Context | 書き込む Bounded Context | スキーマは一枚のファイルで、Context 境界を持たない |
| `tenant_id` 列 | 単独主キー / 複合主キーの一部 / 非キー列 / なし | 現行の[保持区分](../../docs/design/data/database.md#tenant_id-の保持区分)が定める判断の適用結果。読み手が SQL を開かずに区分を知るために置き、SQL との一致は検査する |

列の一覧は書かない。一覧を生成する余地は残すが、この work item では作らない。
ER 図の本体が空であることは変えない。図は表の存在と外部キーだけを示し、意味は隣の表が持つ。

テーブル一覧には「テーブル種別」（`LOGGED` / `UNLOGGED`）の列も加える。
着手時の読み取りで、「再生成可能な認証状態と流量制御」の図が「次の `UNLOGGED` 表」と述べながら、`LOGGED` テーブルの `oauth2_access_token_denylist` と `login_throttle_counters` を含んでいることが分かった。
図の区分だけでは、この種の食い違いを読み手も検査も拾えない。

#### 表と SQL の照合

表は SQL と食い違ったまま古くなりやすいので、リポジトリ検査 `schema-tables` を `tools/check` に加え、`mise run check-schema-tables` と `mise run check` から実行する。
照合するのは SQL から機械的に決まる三つに限り、役割と所有 Context は照合しない。

| 照合内容 | SQL 側の根拠 | 文書側の根拠 |
| --- | --- | --- |
| テーブル名の集合 | `CREATE [UNLOGGED] TABLE <name>` | 先頭の見出しが「テーブル」である Markdown 表の行 |
| テーブル種別 | `UNLOGGED` の有無 | 「テーブル種別」列の `LOGGED` / `UNLOGGED` |
| `tenant_id` 列の区分 | `tenant_id` 列の有無と主キーの構成 | 「`tenant_id` 列」列の `単独主キー` / `複合主キーの一部` / `非キー列` / `なし` |

```ts
type DeclaredTable = { name: string; unlogged: boolean; tenantId: TenantIdPlacement }
type TenantIdPlacement = 'primary-key' | 'primary-key-part' | 'column' | 'absent'
type DescribedTable = { name: string; line: number; kind: string; tenantId: string }

function declaredTables(sql: string): DeclaredTable[]
function describedTables(markdown: string): DescribedTable[]
function compareTables(declared: readonly DeclaredTable[], described: readonly DescribedTable[]): Finding[]
```

入力はどちらも `WorkspaceSnapshot` から読む文字列であり、三つの関数は純粋である。

### スキーマ管理をどこに置くか

`database.md` は既に 411 行あり、型の選択、`tenant_id` の保持区分、エンベロープ暗号という別々の関心事を持つ。
スキーマ管理は読者が違う（スキーマを変える人が読む）ので、独立した文書にする。

`schema-management.md` が持つもの。

- **宣言的スキーマの原則**。`infra/schema/postgres.sql` が現在の構造を宣言し、差分は `psqldef` が計算する。マイグレーションの積み上げを持たない理由。
- **収束の検査**。空のデータベースに対する収束を `mise run check-schema` が確かめること。適用後と再適用後のプレビューが空になることを確かめる手順との関係。
- **拡張と縮小の二段**。新しい形を足す、書き込みを移す、読み出しを移す、古い形を落とすの四段階と、各段で後退できる範囲。新旧のアプリケーションが同時に動く期間に何が守られるか。
- **構造差分で表現できない変更**。`infra/schema/data-migrations/` の位置づけ、いつ実行するか、冪等性の要求、失敗時の扱い。
- **デプロイ工程との関係**。起動時にスキーマを移行しない理由と、デプロイプロファイルごとの適用地点（[[wi-584-ground-deployment-design-in-reference-profiles]] が定めるプロファイルを参照する）。
- **`psqldef` の制約**。同じ列に `CHECK` を二つ並べると差分が収束しないこと、`--enable-drop` を使わないことなど、道具の性質から来る規則。

手順そのもの（コマンドと実行順）は `infra/schema/README.md` が持ち、設計はこの文書が持つ。

### データライフサイクル設計の担当範囲

残す。ただし担当範囲を明示し、その範囲を満たすまで書く。
この文書が持つのは、**一つのデータがシステムに入ってから消えるまでを横断して見たときにだけ現れる設計**である。個々の保持期間や個々のテーブルの削除規則ではない。

具体的には次を持つ。

- データの区分（業務データ、短命な認証状態、監査イベント、鍵素材、資産）と、区分ごとの消え方の違い。
- 削除と匿名化の使い分け。どちらを選ぶかの判断基準。
- テナントの退去。何を消し、何を残し、どの順で消すか。
- 暗号学的消去。DEK を破棄したときに何が読めなくなり、何が残るか。
- **バックアップとの相互作用**。削除したデータが復元で戻る問題と、それをどう扱うか。この観点は現在どの文書にもない。
- 保持期間を変えたときに同時に動くもの（キャパシティ、プライバシー、復旧）と、変更の順序。

採らない案を二つ記録する。
一つは `lifecycle.md` を `database.md` へ統合する案である。`database.md` は物理スキーマの設計を持つ文書であり、テナント退去や暗号学的消去はその問いに属さない。
もう一つは `lifecycle.md` を削除する案である。削除すると、上に挙げた横断の観点（特にバックアップと削除の相互作用）を持つ場所が無くなる。現状が薄いのは担当範囲が決まっていないためであって、担当すべきものが無いからではない。

## Plan

1. `infra/schema/postgres.sql` と各 Context の実装から、テーブルの役割と所有 Context を読み取る。
2. 機能領域ごとに表を作り、ER 図の隣へ置く。
3. `schema-management.md` を書き、`database.md` と `lifecycle.md` と `maintenance.md` に散っている記述をそこへ寄せる。
4. `lifecycle.md` の担当範囲を冒頭で宣言し、その範囲を書き足す。
5. 索引と `DOCUMENTATION_GUIDE.md` §4.9 を追従させる。

## Tasks

- [x] T001 [Design] テーブルの役割と所有 Context を、スキーマと実装から読み取る。
- [x] T001a [Acceptance] `mise run check-schema-tables` が、表を持たない現行の `database.md` に対して 88 テーブルの欠落を報告して失敗することを確かめる（Acceptance RED）。
- [x] T001b [Unit] `mise run test-tools-file -- check/src/schema-tables.test.ts` で、SQL の解析、表の解析、照合の各規則の RED を確かめてから GREEN にする。
- [x] T002 [Docs] 機能領域ごとのテーブル説明の表を ER 図へ添える。
- [x] T003 [Docs] `docs/design/data/schema-management.md` を作り、散っている記述を寄せる。
- [x] T004 [Docs] データライフサイクル設計の担当範囲を宣言し、書き足す。
- [x] T005 [Docs] データ設計の索引と `DOCUMENTATION_GUIDE.md` §4.9 を追従させる。
- [x] T006 [Verify] リンク、スキーマ収束、仕様、全体検証を通す。

## Verification

- `mise run check-links`
- `mise run check-schema`
- `mise run check-spec`
- `mise run verify`

## Risk Notes

テーブルの役割を一行で書くと、正本の SQL と食い違ったまま気付かれない状態になりやすい。役割は列の言い換えではなく「なぜこの表があるか」を書き、列に触れない。所有 Context は `backend/<context>/db_postgres` の実装から読み取り、推測しない。

表の行数はテーブル数に等しく、スキーマが変われば古くなる。検査の対象にできるのはテーブル名の集合が一致することまでなので、この検査を追加するかを実装時に判断する。追加しないなら、古くなり方を risk として文書に残さず、`database.md` の冒頭で正本を明示する。

スキーマ管理を新しい文書へ切り出すと、`database.md`、`lifecycle.md`、`maintenance.md` に残った記述との二重化が起きる。移した側から重複を消し、参照に置き換える。

## Completion

- **Completed At**: 2026-09-18
- **Summary**:
  `mise run spec-diff` は main に対して規範の変更がないことを報告した。変更は設計文書と文書検査に限られる。`docs/design/data/database.md` の六つの ER 図に、全 88 テーブルの役割、所有 Context、テーブル種別、`tenant_id` 列の区分を示すテーブル一覧を添え、所有 Context の外から行われる四種類の書き込みを表にした。「再生成可能な認証状態」の図が `LOGGED` テーブルの `oauth2_access_token_denylist` と `login_throttle_counters` を `UNLOGGED` と説明していた誤りを直し、`LOGGED` にしている理由を書いた。スキーマの宣言、収束の検査、拡張と縮小の四段階、データ移行、適用する地点、`psqldef` の規則を `docs/design/data/schema-management.md` に集め、`database.md`、`maintenance.md`、`infra/schema/README.md` からは参照に置き換えた。README がデータ移行の置き場所を `infra/schema/` の外としていた誤りも直した。`lifecycle.md` は担当範囲を宣言し、データの区分ごとの消え方、削除と匿名化の判断基準、`RESTRICT` と `CASCADE` から決まるテナント退去の順序（仮）、暗号学的消去が消すものと消さないもの、復元で消去済みのデータが戻る問題を書いた。リポジトリ検査 `schema-tables`（`mise run check-schema-tables`）を加え、テーブル一覧のテーブル名の集合、テーブル種別、`tenant_id` 列の区分を `postgres.sql` と照合する。着手時の読み取りで見つかった二つの欠陥を wi-611 と wi-612 に起票した。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-schema-tables`
  - **Requirement**: N/A: 文書と文書検査の変更で、製品の規範要求を持たない。
  - **Observed Failure**: テーブル一覧のない変更前の `database.md` に対して、88 テーブルすべてについて `<table> is declared in the schema but not described` を報告し、終了コード 1 で失敗した。
  - **Detection Reason**: スキーマの `CREATE TABLE` の集合と文書のテーブル一覧の行の集合を比べるので、テーブルが一つでも欠ければ失敗する。説明のない文書を通してしまう実装は区別できる。
- **Unit RED Evidence**:
  - **Test**: `mise run test-tools-file -- check/src/schema-tables.test.ts`
  - **Requirement**: N/A: 文書検査の内部規則であり、製品の規範要求を持たない。
  - **Observed Failure**: `schema-tables.ts` が存在せず、テストファイルの読み込みで失敗した（0 pass、1 fail、1 error）。実装後は 11 件が通った。
  - **Detection Reason**: テストは、インラインとテーブルレベルの主キー、SQL コメント中の `tenant_id`、見出しが「テーブル」でない表の除外、欠落、余剰、重複、テーブル種別の食い違い、`tenant_id` 列の区分の食い違い、空のスキーマをそれぞれ別の期待値で固定する。
- **Change-Resistance Results**:
  `database.md` の `login_throttle_counters` のテーブル種別を `UNLOGGED` に、`consents` の `tenant_id` 列を `非キー列` に書き換えると、`mise run check-schema-tables` は両方の行番号を挙げて失敗した。元に戻すと通った。risk は low なので変異テストは実行していない。
- **Verification Results**:
  - `mise run check-schema-tables` - passed
  - `mise run check-links` - passed
  - `mise run check-terminology` - passed
  - `mise run check-work-items` - passed
  - `mise run check-schema` - passed（サンドボックスが Docker ソケットを拒否したため、サンドボックス外で実行した）
  - `mise run verify` - passed（`check-repository` の中で `schema-tables` も実行された）
  - `mise run test-ui-e2e` - 実行していない。UI とブラウザーに届く変更を含まない。
