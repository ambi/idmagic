---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-26
priority: p2
depends_on: []
change_kind: refactor
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 永続化アダプターの内部で SQL の渡し方を変え、リポジトリ検査を足すだけである。製品の振る舞いも公開契約も変わらないので、リリースの読み手に見えるものがない。
  references: []
spec_impact:
  kind: none
  reason: "SQL 文の置き場所を sqlc の入力へ移し、検出の仕組みを足すだけで、読み書きする行と結果は変えない。"
initial_context:
  specification: []
  typespec: []
  source:
    - sqlc.yaml
    - docs/design/data/database.md
    - tools/check/src/check-boundaries.ts
    - tools/check/src/registry.ts
    - backend/idgovernance/db_postgres/lifecycle_workflows.go
    - backend/sourcing/scim/db_postgres/scim.go
    - backend/application/db_postgres/applications.go
    - backend/idmanagement/db_postgres/csv_artifacts.go
    - backend/idmanagement/user/db_postgres/user_import_committer.go
    - backend/idmanagement/group/db_postgres/group_import_committer.go
    - backend/idmanagement/group/db_postgres/group_membership_import_committer.go
    - backend/shared/security/salts_postgres/tenant_salt_store.go
    - backend/shared/storage/db_postgres/base.go
    - backend/cmd/idmagic-batch/restore_consistency.go
    - backend/tenancy/db_postgres/quota_repository.go
    - backend/jobs/db_postgres/postgres.go
    - backend/audit/db_postgres/audit_events.go
  tests:
    - backend/cmd/idmagic-batch/restore_consistency_test.go
  stop_before_reading:
    - frontend
---

# 静的な SQL 文を sqlc へ移し、直接の SQL 文字列を検査で拒否する

## 動機

[データベース設計](../docs/design/data/database.md#ポートとアダプター)は、`db_postgres` の静的な SQL 文をすべて `sqlc` の入力とし、SQL 文字列を直接渡す `Pool.Query` と `Pool.Exec` は問い合わせの構造が実行時まで決まらない場合に限ると定める。
しかし、この規則を機械的に確かめる仕組みはない。
2026-09-26 に `backend/` の `_test.go` 以外を調べたところ、sqlc の生成コードを通さずに SQL 文字列を渡す呼び出しが 15 ファイルにあり、そのうち 11 ファイルの文は構造が固定された静的な文だった。
`idgovernance` の `lifecycle_workflows.go` のように、同じ文が `lifecycle_workflows.sql` に sqlc の問い合わせとして宣言済みで、生成された関数を使わずに手書きの文字列を重複させている箇所もある。

手書きの文字列は、列の追加や改名を `sqlc generate` のコンパイルエラーとして検出できない。
列の並びと `Scan` の引数の対応も手で保つことになる。
規則が文書にしかないため、同じ逸脱は今後も入り得る。

## 調査結果

各呼び出しを、sqlc で表現できるかどうかで分類する。

### sqlc へ移すもの

| ファイル | 対象の文 | sqlc へ移せる理由 |
| --- | --- | --- |
| `backend/idgovernance/db_postgres/lifecycle_workflows.go` | `lifecycle_workflows` と `lifecycle_workflow_revisions` の一覧、取得、保存（5 文） | 同じ文が `lifecycle_workflows.sql` に宣言済みで、生成された関数がある。 |
| `backend/sourcing/scim/db_postgres/scim.go` | `FindUserRefsByUserIDs`、`FindGroupRefsByGroupIDs` | `= ANY(@ids)` の固定の文である。 |
| `backend/application/db_postgres/applications.go` | 利用者とグループの組に対する割り当ての一括取得 | 固定の文である。現行のコメントは「sqlc が UNNEST の引数型を解決できない」と述べ、実際に引数 2 つの `UNNEST` は sqlc のカタログにない。1 次元の `UNNEST` を `WITH ORDINALITY` の序数で突き合わせれば同じ組を作れる。 |
| `backend/idmanagement/db_postgres/csv_artifacts.go` | `csv_artifacts` と `csv_artifact_chunks` の挿入、更新、読み取り（7 文） | すべて固定の文である。このディレクトリには sqlc の生成エントリーがないので追加する。 |
| `backend/idmanagement/user/db_postgres/user_import_committer.go` | `password_history` と `audit_events` の挿入 | 固定の文である。所有 Context の外への書き込みだが、同じトランザクションで確定する必要から既存の一覧に載っており、sqlc の問い合わせを置く場所だけを変える。 |
| `backend/idmanagement/group/db_postgres/group_import_committer.go` | `audit_events` と `jobs` の挿入 | 同上。 |
| `backend/idmanagement/group/db_postgres/group_membership_import_committer.go` | `audit_events` の挿入 | 同上。 |
| `backend/shared/storage/db_postgres/tenant_salt_store.go` | `tenant_correlation_salts` の挿入と取得 | 固定の文である。 |
| `backend/cmd/idmagic-batch/restore_consistency.go` | 件数、有効な署名鍵のないテナント、重複した `dedup_key`、一時テーブルの件数 | 一時テーブルは固定の 9 表であり、表ごとの `EXISTS` を `VALUES` に並べた 1 文で表せる。 |
| `backend/tenancy/db_postgres/quota_repository.go` | `tenant_usages` の加算と減算 | 列名は `switch` が選ぶ 11 種の閉じた集合から決まる。資源ごとの固定の文として表せる。 |
| `backend/jobs/db_postgres/postgres.go` | `ListForAdmin` | 絞り込みはすべて省略可能な固定の条件であり、`sqlc.narg` と空配列の判定で 1 文に表せる。現行のコメントは「sqlc の静的な文にできない」と述べるが、条件の数も種類も固定なので当たらない。 |

### 直接の SQL 文字列を残すもの

| ファイル | 対象 | 残す理由 |
| --- | --- | --- |
| `backend/audit/db_postgres/audit_events.go` | `List`、`Count` | `Filters` の件数が要求ごとに変わり、条件の数だけ `EXISTS` 副問い合わせを連言でつなぐ。演算子（`eq`、`in`、`contains`）も式ごとに変わる。配列や JSONB に詰めて 1 文にすることもできるが、各式の演算子を SQL 内で分岐させる必要があり、索引の効き方も読み取りにくくなる。 |
| `backend/shared/storage/db_postgres/base.go` | `ResilientDB` の `Query`、`QueryRow`、`Exec` | sqlc の `DBTX` を実装する中継であり、SQL 文字列は呼び出し元（生成コード）から渡される。 |
| `backend/cmd/idmagic-dev-infra/internal/devinfra/devinfra.go` | スキーマの削除と `infra/schema/postgres.sql` の適用 | 問い合わせではなく DDL の適用である。 |
| `backend/shared/storage/testing_postgres/pgtest.go` | テスト用データベースへのスキーマの適用 | 同上。 |
| `backend/shared/storage/db_postgres/base.go` | `AfterConnect` の `SET statement_timeout …` | 接続ごとのセッション設定である。`pgx.ConnConfig.RuntimeParams` の起動パラメーターにすれば SQL をなくせるが、PgBouncer などの前段は許可したもの以外の起動パラメーターを拒否する。前段の構成は運用者が決め、リポジトリの外にあるので、送り方は変えない。 |

`_test.go` のテストデータの投入と `TRUNCATE` は、本番の問い合わせではないので対象外とする。

## 対象範囲

- 上の「sqlc へ移すもの」の各文を、対応するディレクトリの `.sql` に sqlc の問い合わせとして宣言し、手書きの呼び出しを生成された関数へ置き換える。
- `sqlc.yaml` へ `backend/idmanagement/db_postgres`、`backend/shared/security/salts_postgres`、`backend/cmd/idmagic-batch/internal/restorecheck/db_postgres` の生成エントリーを追加する。
- 相関 salt の PostgreSQL 実装を、メモリ実装の `salts_memory` と並ぶ `backend/shared/security/salts_postgres` へ移す。
- 移行によって PostgreSQL のテストが一つも通らなくなるメソッドへ、移行前のコードでも通る特性テストを足す。
- 特性テストで見つかった SCIM の一括取得の不具合（下の「設計」）を直す。
- 直接の SQL 文字列を検出するリポジトリ検査を追加し、`mise run check-repository` から実行する。
- [データベース設計](../docs/design/data/database.md#ポートとアダプター)の規則に、例外の書き方と検査の存在を追記する。

## 対象外

- `tenant_usages` を資源ごとの行へ正規化するスキーマ変更。加算の文を資源ごとに持たずに済むが、クォータの読み書き全体の設計変更になる。
- 監査イベントの検索条件を 1 文の静的な SQL へ詰め直すこと。
- `_test.go` と `fixtures_postgres` のテストデータ投入の SQL。
- TypeScript 側のツール。PostgreSQL へ問い合わせるコードはない。

## 設計

### sqlc への移行

各ディレクトリの既存の `.sql` に問い合わせを追加し、`mise run sqlc-generate` で生成する。
リポジトリのメソッドのシグネチャとポートは変えない。
変えるのはメソッド内部の SQL の渡し方と、行から domain 型への変換だけである。

資源ごとの問い合わせが必要なものは、次のように扱う。

- **クォータの加算と減算**：`IncrementTenantUsageUsers` と `DecrementTenantUsageUsers` のように、資源ごとに加算と減算の文を 11 組宣言する（`tenant_usages.sql`）。sqlc は文ごとに引数の型を生成するので、`quota_repository.go` の `usageCounters` が資源名から加算と減算の関数の組を引く対応表になり、`switch` を置き換える。既定の上限値は `default_limit` の引数で渡す。
  - 採用しない案 A：`CASE @resource WHEN 'users' THEN … END` を `SET`、条件、`RETURNING` の 3 か所に並べた 1 文。11 種を 3 か所で同期させる必要があり、誤った列を更新しても型検査で検出できない。
  - 採用しない案 B：閉じた集合から列名を選ぶので直接の SQL 文字列として残す。sqlc で表現できる以上、規則の例外に当たらない。
- **管理者向けジョブ一覧**：`sqlc.narg(tenant_id)::text IS NULL OR tenant_id = sqlc.narg(tenant_id)::text::uuid` のように、省略可能な条件を 1 文に並べる。`uuid` の NULL 可能な引数は `pgtype.UUID` として生成されるので、`text` で受けて SQL 側で `uuid` にする。空の配列は「条件を外す」ではなく「どれにも一致しない」になるため、状態と種類は指定があるときだけ配列を渡し、ないときは `nil`（NULL）にする。`AllTenants` と `TenantID` の組み合わせの検証（`ErrAdminJobFilterUnscoped`）は Go 側に残す。
  - `jobs` の索引は `(lane, run_at)`、`(lane, lease_expires_at)` と、`dedup_key` を条件に持つ `(tenant_id, dedup_key)` の部分索引だけであり、テナント指定の一覧に使える索引は移行前からない。省略可能な条件を `OR` でつないでも、失う索引はない。
- **復元後の整合性検査**：件数を 1 行で返す問い合わせ、有効な署名鍵のないテナント、重複した `dedup_key`、行の残る一時テーブルを返す問い合わせを宣言する。一時テーブルの一覧は `VALUES` の中だけに置き、Go 側の `ephemeralTables` を消す。一覧を 2 か所で同期させずに済む。sqlc の出力先はパッケージ単位なので、問い合わせを `backend/cmd/idmagic-batch/internal/restorecheck/db_postgres` に置く。
- **CSV の成果物の読み取り**：`OpenCSVArtifact` が返す読み取り器は、移行前は `pgx.Rows` を開いたまま 1 チャンクずつ返していた。sqlc の `:many` は全行をメモリへ載せるため、チャンク番号を起点に `csvChunkReadBatch`（16 チャンク、1 MiB）ずつ問い合わせる形に変える。成果物全体をメモリへ載せず、読み取りの間に接続を占有することもなくなる。
- **相関 salt の保存先**：`backend/shared/storage/db_postgres` へ sqlc の生成先を足すと、共有パッケージが `New`、`Queries`、`DBTX` を公開し、既存の `DB` と役割が重なる。メモリ実装が `backend/shared/security/salts_memory` にあるので、PostgreSQL 実装をその隣の `salts_postgres` へ移し、そこを生成先にする。
- **SCIM の一括取得の不具合**：特性テストを移行前のコードで流すと、`FindUserRefsByUserIDs` は `improper binary format in array element 1 (SQLSTATE 22P03)` で失敗した。`RegisterUUIDAsText` は `uuid` だけを text として扱うよう登録し、`uuid[]` の要素はバイナリで符号化されたままになるため、`[]string` を `uuid[]` の引数へ渡すと PostgreSQL では必ず失敗する。CSV の計画器が所有権を解決する経路であり、E2E はメモリ実装で動くため気づかれていなかった。引数を `text[]` で受けて SQL 側で `uuid[]` にする（`ANY(sqlc.arg(user_ids)::text[]::uuid[])`）。型変換は引数の側にかかるので、列の索引は引き続き使える。この修正は振る舞いの変更なので、移行とは別のコミットにする。
  - 採用しない案：`RegisterUUIDAsText` で `uuid[]` にも text の要素を登録する。全接続の型の扱いを変え、影響がこの 2 文にとどまらない。`uuid[]` の引数を持つ文はこの 2 つだけである。

### 検出の仕組み

`tools/check/src/` に `raw-sql` 検査を追加し、`registry.ts` の `all` グループへ登録する。
既存の `boundaries` 検査と同じく、`WorkspaceSnapshot` から Go ソースを読み、正規表現で判定する。

| 項目 | 内容 |
| --- | --- |
| 対象ファイル | `backend/` 配下の `.go` のうち、`_test.go` と、先頭に `// Code generated by sqlc` を持つ生成ファイルを除いたもの |
| 違反とする呼び出し | `.Exec(`、`.Query(`、`.QueryRow(`、`.SendBatch(`、`.CopyFrom(` のうち、第 1 引数が `ctx` で終わる識別子または `context.Background()` などのもの。`url.Values` を返す `u.Query()` のような引数のない呼び出しは一致しない。 |
| 例外の書き方 | 呼び出しを含む行の直前の行に `//sql:raw <理由>` を置く。理由が空の場合は違反とする。 |
| 失敗時の出力 | `fail  <パス>:<行>: declare the SQL as a sqlc query, or state why sqlc cannot express it with //sql:raw <reason>` |

例外の印の名前は、当初 `//sql:dynamic` とした。
しかし残す理由は、構造が実行時に決まる文のほかに、DDL の適用、SQL の中継、接続の設定と幅があるので、`//sql:raw` とした。
例外は上の「直接の SQL 文字列を残すもの」の 4 ファイル、9 か所だけに置く。
理由を呼び出しの隣に置くため、例外の一覧を別ファイルで管理しない。

- 採用しない案 A：golangci-lint の `forbidigo` で `pgxpool.Pool.Query` などを禁止する。型を解析すれば精度は上がるが、`pgx.Tx`、`DBTX`、`queryer` など受け手の型が複数あり、パターンの保守がかえって難しい。例外の書式を強制することもできない。
- 採用しない案 B：`go/analysis` で専用の解析器を書く。型に基づく判定ができるが、この規模の規則にはビルドと配布の負担が大きい。正規表現の検査で誤検知が問題になった時点で再検討する。

## 計画

1. 検査を先に書き、現状の 15 ファイルで失敗すること（受け入れ RED）を確かめる。検査の単体テストは、違反、生成ファイル、`_test.go`、引数のない `Query()`、理由のある例外、理由のない例外を固定する。
2. 移行するメソッドのうち、PostgreSQL のテストがないもの（クォータの加算と減算、インポートの確定処理 3 種、ライフサイクルワークフローの読み取り、SCIM の一括取得）へ特性テストを足し、移行前のコードで通ることを確かめる。
3. 移行が単純なもの（`lifecycle_workflows`、`scim`、`applications`、`csv_artifacts`、インポートの確定処理、`tenant_salt_store`）を移し、移行前後で同じテストが通ることを確かめる。
4. クォータ、ジョブ一覧、復元検査を移す。クォータは 11 資源すべての加算と上限超過を、ジョブ一覧は各絞り込みの有無の組み合わせとキーセットの継続をテストで固定する。
5. 例外の 4 ファイルへ `//sql:raw` を置き、検査を GREEN にする。`jobs` の `ListForAdmin` の古いコメントは削除する。
6. [データベース設計](../docs/design/data/database.md#ポートとアダプター)へ例外の書き方と検査を追記する。

未解決の問いはない。

予定する RED 検査は次のとおりである。

| 種類 | 検査 | 期待する失敗 |
| --- | --- | --- |
| Acceptance RED | `mise run check-repository`（`raw-sql` 検査を登録した状態） | 現状の 15 ファイルの直接の SQL 呼び出しを違反として報告する。 |
| Unit RED | `tools/check/src/raw-sql.test.ts` | 検査の実装前に、関数が存在しないため失敗する。 |

sqlc への移行そのものは振る舞いを変えないリファクタリングなので、独自の RED は求めない。
各パッケージの既存の `db_postgres` テストを移行の前後で通し、`mise run test-go-package -- <package>` をサンドボックスの外で実行する。

## タスク

- [x] T001 [Acceptance] `raw-sql` 検査を追加し、現状のコードで失敗することを確認する。
- [x] T002 [App] 検査の単体テストを書き、RED から GREEN にする。
- [x] T003 [App] `lifecycle_workflows`、`scim`、`applications` を生成された関数へ置き換える。
- [x] T004 [App] `csv_artifacts` と `tenant_salt_store` の生成エントリーを追加し、置き換える。
- [x] T005 [App] インポートの確定処理の挿入を、`idmanagement/user` と `idmanagement/group` の sqlc 問い合わせへ置き換える。
- [x] T006 [Test] 移行するメソッドのうち PostgreSQL のテストがないものへ特性テストを足し、移行前のコードで通ることを確かめる。
- [x] T007 [App] クォータの加算と減算を資源ごとの問い合わせへ置き換える。
- [x] T008 [App] `ListForAdmin` を省略可能な条件の 1 文へ置き換える。
- [x] T009 [App] 復元後の整合性検査を `restorecheck/db_postgres` の問い合わせへ置き換える。
- [x] T010 [App] 例外の 4 ファイルへ `//sql:raw` を置く。
- [x] T011 [Docs] データベース設計へ例外の書き方と検査を追記する。
- [x] T013 [Fix] SCIM の一括取得で `uuid[]` の引数を `text[]` で受ける。
- [x] T012 [Verify] 変更を検証する。

## 検証

- `mise run sqlc-generate` の後に差分がないこと
- `mise run check-repository`
- `mise run test-go-race`（PostgreSQL を使うテストはサンドボックスの外で実行する）
- `mise run verify`

## リスク

- **クォータの資源と文の対応の誤り**：資源ごとに 11 組の文を宣言するため、資源名と列の対応を取り違えると別の資源を加算する。11 資源すべてについて、加算後にその列だけが増えることをテストで確かめる。
- **ジョブ一覧の実行計画**：省略可能な条件を `OR` でつなぐと、準備済み文の汎用計画で索引が使われない可能性がある。`jobs` にはテナント指定の一覧に使える索引が移行前からなく（「設計」を参照）、失うものはない。索引を足すときは、条件の組み合わせごとに文を分けるかを見直す。
- **検査の誤検知と見逃し**：正規表現による判定なので、受け手の変数名や引数の書き方によっては一致しない。第 1 引数を `ctx` 以外の名前で渡す呼び出しは見逃す。リポジトリの Go コードは `ctx` を使う慣習がそろっているので、見逃しは実害が出た時点で型に基づく解析器へ移す判断材料とする。

## 完了

- **Completed At**: 2026-09-26
- **Summary**:
  `mise run spec-diff` は規範仕様の差分を報告しない（`no normative specification change against main`）。
  `backend/` のテスト以外の Go コードで SQL 文字列を直接渡していた 15 ファイル 42 か所のうち、11 ファイルの文を sqlc の問い合わせへ移した。
  残る 4 ファイル 9 か所（監査イベントの検索、`ResilientDB` の中継、DDL の適用、接続ごとのセッション設定）には `//sql:raw <理由>` を置いた。
  リポジトリ検査 `raw-sql` を `check-repository` に加え、理由のある印のない直接の SQL 呼び出しを拒否する。
  データベース設計に、sqlc で表せない場合の一覧と例外の書き方を追記した。
  移行の途中で、相関 salt の PostgreSQL 実装を `backend/shared/security/salts_postgres` へ移し、CSV の成果物の読み取りを 16 チャンクずつの問い合わせに変えた。
  追加した特性テストで、SCIM の一括取得（`FindUserRefsByUserIDs`、`FindGroupRefsByGroupIDs`）が PostgreSQL では必ず失敗する既存の不具合が見つかったので、引数を `text[]` で受けるよう直した。
- **Acceptance RED Evidence**:
  - **Test**: `raw-sql` 検査を登録した `bun run check/src/runner.ts raw-sql`
  - **Requirement**: N/A: 製品の振る舞いを変えない、リポジトリ規則の機械検査と永続化アダプターの内部変更である。
  - **Observed Failure**: 移行前のコードに対して終了コード 1 で、15 ファイル 42 か所を `declare the SQL as a sqlc query, …` として報告した。
  - **Detection Reason**: 調査で手作業で数えた 15 ファイルと一致し、sqlc の生成ファイル、`_test.go`、`url.Values` の `Query()` を報告しなかった。移行と印の付与の後は `ok  976 Go file(s), 9 //sql:raw exception(s)` になった。
- **Unit RED Evidence**:
  - **Test**: `tools/check/src/raw-sql.test.ts`
  - **Requirement**: N/A: 製品の振る舞いを変えないリポジトリ検査である。
  - **Observed Failure**: 実装前は `Cannot find module './raw-sql.ts'` で失敗した。
  - **Detection Reason**: 違反、生成ファイル、引数のない `Query()`、次の行に置いた context、理由のある印、理由のない印、2 行離れた印を区別する 10 件の例で、印の置き場所と理由の有無を取り違える実装を検出する。
  - **Test**: `backend/sourcing/scim/db_postgres/repositories_test.go` の `TestScimRepositoryResolvesRefsInBulkWithinTheTenant`
  - **Requirement**: N/A: 仕様に書かれた振る舞いは変えず、PostgreSQL の実装がメモリ実装と同じ結果を返すよう直す不具合修正である。
  - **Observed Failure**: 移行前のコードでも移行後のコードでも `ERROR: improper binary format in array element 1 (SQLSTATE 22P03)` で失敗した。
  - **Detection Reason**: 実際の PostgreSQL へ `[]string` の ID を渡し、自テナントの参照だけが返ることを表明するので、符号化の誤りとテナントの取り違えの両方を検出する。
- **Change-Resistance Results**:
  - 移行するメソッドのうち PostgreSQL のテストがなかったものへ特性テストを足し、実装ファイルだけを `HEAD` の版へ戻した状態でも通ることを確かめた（`TestQuotaRepository*`、`TestGroupImportRowCommitter*`、`TestGroupMembershipImportRowCommitter*`、`TestUserImportRowCommitter*`、`TestLifecycleWorkflowRepositoryReadsBackWhatItSaved`）。
  - 手作業の障害注入：クォータの対応表で `groups` の加算を `users` の問い合わせに差し替えると、`TestQuotaRepositoryCountsOnlyTheNamedResourceUpToItsLimit/groups` が失敗した。CSV の読み取りで `r.done` を常に真にすると、`TestCSVArtifactStoreReadsAcrossChunkBatches` の両方の事例が 1048576 バイトで読み終えて失敗した。
  - `mise run test-go-mutation -- backend/tenancy/db_postgres`：31 件中 28 件を検出し、2 件は時間切れになった。生き残った 1 件は `GetUsage` の `err != nil` の否定で、PostgreSQL の読み取り失敗を起こすテストがない既存の経路である。
  - `mise run test-go-mutation -- backend/idmanagement/db_postgres`：41 件中 28 件を検出し、7 件は時間切れになった。最初に生き残った `ReadCSVArtifactPage` の `page < 0` → `page <= 0` は先頭ページを読むテストがない不足だったので、ページ 0 と負のページの表明を足し、この変異で失敗することを確かめた。残りは `page > math.MaxInt32` → `>=` と `n > 0` → `>=`（空のチャンクを書いても読み出すバイト列は同じ）で、実質的に等価である。`len(payload) > csvArtifactChunkBytes` → `>=` と、`fetch` の失敗を伝える分岐の否定は既存の不足であり、ちょうど 64 KiB のページと問い合わせの失敗を起こすテストがない。
  - `mise run test-go-mutation -- backend/jobs/db_postgres`：40 件中 35 件を検出した。今回の変更にかかる生き残りは `limit > math.MaxInt32` → `>=` だけで、実質的に等価である。ほかの 3 件は変更していない行にある。
- **Verification Results**:
  - `mise run check-repository` - 成功（`ok  976 Go file(s), 9 //sql:raw exception(s)`）
  - `mise run sqlc-generate` を再実行しても差分なし
  - `mise run lint-go`、`mise run lint-tools` - 成功
  - `mise run test-go-changed`（サンドボックスの外で実行し、DB テストを実行させた） - 成功
  - `mise run verify`（サンドボックスの外で実行） - 成功
