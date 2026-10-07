---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-30
priority: p2
depends_on: []
change_kind: bugfix
evidence_policy: risk-based-v4
documentation_impact:
  level: release_note
  reason: 状態が未設定の User が、PostgreSQL 構成でも管理者のユーザー一覧の `status=active` の絞り込みに含まれるようになる。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-13199-close-remaining-user-adapter-divergences.md }
initial_context:
  specification: [docs/domain/identity-management/user/README.md#REQ-IDMANAGEMENT-005]
  typespec: []
  source:
    - backend/idmanagement/user/db_memory/users.go
    - backend/idmanagement/user/db_memory/email_change_token.go
    - backend/idmanagement/user/db_memory/tenant_user_attribute_schema.go
    - backend/idmanagement/user/db_postgres/users.go
    - backend/idmanagement/user/db_postgres/users.sql
    - backend/idmanagement/user/db_postgres/email_change_tokens.sql
    - backend/idmanagement/user/domain/users.go
    - infra/schema/postgres.sql
  tests:
    - backend/idmanagement/user/testing_contract/contract.go
    - backend/idmanagement/user/db_memory/contract_test.go
    - backend/idmanagement/user/db_postgres/contract_test.go
  stop_before_reading: [frontend, backend/idmanagement/user/usecases, backend/idmanagement/user/handlers_http]
affected_spec:
  - { path: docs/domain/identity-management/user/README.md, requirement: REQ-IDMANAGEMENT-005, impact: conforms }
primary_use_cases:
  - id: unset-status-filters-as-active
    requirement: REQ-IDMANAGEMENT-005
    observable_result: 状態が未設定の User が、PostgreSQL 構成でも `status=active` の絞り込みの一覧と件数に含まれる。
    boundary: contract
    test: { path: backend/idmanagement/user/db_postgres/contract_test.go, name: TestPersistenceContract, task: test-go-race }
    fault_model: 絞り込みの SQL が、`lifecycle` に保存された空文字の状態を `active` と見なさず、その User を一覧と件数から落とす。
---

# idmanagement/user の契約テストへ、未検証の食い違い候補 6 件を加える

## 動機

wi-16658 は、`idmanagement/user` のメモリ実装と PostgreSQL 実装を `backend/idmanagement/user/testing_contract` の同じ契約へ通した。
その契約は 7 つのサブテストからなり、検出した食い違いは「メモリ User repository が同一テナントで有効な `preferred_username` の重複を受理する」1 件だけだった。

wi-16658 の着手時にコードを読んだところ、この契約が確かめていない食い違いの候補が 6 件見つかった。
いずれもコードを読んで立てた仮説であり、テストでは観測していない。

| 候補 | 対象 | 予想される差 |
|---|---|---|
| C1 | メモリ `UserRepository.Save` | 削除済みの user A と同じ `preferred_username` を、有効な user B が再利用しているとき、A を保存し直すとメモリ実装は `errPreferredUsernameExists` で拒否する。PostgreSQL の部分一意索引は削除済みの行を除外するので受理する。 |
| C2 | メモリ `UserRepository` の読み取り | 保存したポインターをそのまま返し、`Save` の引数も複製せずに保持する。呼び出し側が返り値や引数を後から書き換えると、`Save` を経ずに保存内容が変わる。PostgreSQL 実装では起きない。 |
| C3 | メモリ `UserRepository.Save` の更新 | 既存の user を保存し直すと `CreatedAt` と `TenantID` を上書きする。PostgreSQL の `SaveUser` は `ON CONFLICT (id)` でこの 2 列を更新しない。 |
| C4 | PostgreSQL の状態フィルター | `UserLifecycle.Status` の JSON タグに `omitempty` が無いので、未設定の状態は `"status":""` として保存される。`coalesce(lifecycle->>'status', 'active')` は空文字を置き換えないため、`active` で絞り込むとその user が漏れる。メモリ実装とドメインの `EffectiveStatus` は未設定を `active` として扱う。 |
| C5 | メモリ `EmailChangeTokenStore.Find` | 返すエンベロープは構造体の複製だが、`Payload` の map は保存内容と共有している。返り値の map を書き換えると、次の `Find` の結果が変わる。 |
| C6 | 時刻の分解能 | PostgreSQL は `TIMESTAMPTZ` をマイクロ秒に切り捨てて返す。メモリ実装はナノ秒のまま返す。wi-16658 の設計は「保存した時刻はマイクロ秒で読み戻る」ことを契約として固定するとしたが、契約はマイクロ秒に切り捨てた入力しか与えていない。 |

上位のテストはメモリ実装をフェイクとして使い、`PERSISTENCE=memory` は本番の構成でも選べる。
C2 と C5 の共有があると、`Save` を呼び忘れたユースケースでもメモリ実装のテストは通ってしまう。
C4 が事実なら、管理画面で状態を `active` に絞り込んだときに、PostgreSQL 構成でだけ一部の user が一覧から消える。

## 対象範囲

- C1〜C6 の各候補について、`backend/idmanagement/user/testing_contract` にサブテストを加え、両実装で実行して食い違いが実在するかを観測する。
- 実在した食い違いのうち、既存の PostgreSQL の制約またはドメインの規則に合わせるだけで直せるものは、この項目で直す。wi-16658 と同じ扱いである。
- 実在しなかった候補も、契約のサブテストとして残す。同じ誤りの再発を検出するためである。

## 対象外

- `idmanagement/user` 以外の Context の契約。同じ種類の共有（C2、C5）や時刻の分解能（C6）の差が他の Context にもあり得るが、この項目では扱わない。`idmanagement/user` で観測した結果から、別の項目で扱うかを判断する。
- 規範判断を要する食い違いの修正。bugfix work item に切り出す。
- `userdomain.UserLifecycle` の JSON 表現を変えるデータ移行。製品は未リリースなので、表現を変える場合もスキーマ宣言の更新だけで足りる。

## 設計

契約は wi-16658 の `testing_contract.Run` と `Fixture` をそのまま使い、サブテストを足す。
新しい仕組みは加えない。

各候補が契約として固定する性質は次のとおりである。

| 候補 | 契約として固定する性質 | 直す側の候補 |
|---|---|---|
| C1 | 削除済みの user は、その `preferred_username` を別の有効な user が使っていても保存し直せる。その後も `FindByUsername` は有効な user を返す。 | メモリ実装。重複の判定と名前の索引を、削除済みの user を除外して行う。 |
| C2 | `Find*`、`FindAll`、`ListPage*` の返り値を書き換えても、保存内容は変わらない。`Save` の後で引数を書き換えても、保存内容は変わらない。 | メモリ実装。境界で深く複製する。`TenantUserAttributeSchemaRepository` の `cloneUserAttributeSchema` が前例である。 |
| C3 | 既存の user を保存し直しても、`CreatedAt` と `TenantID` は最初に保存した値のまま読み戻る。 | メモリ実装。`TenantUserAttributeSchemaRepository.Save` が `CreatedAt` を保持するのと同じ形にする。 |
| C4 | `Lifecycle.Status` が未設定の user は、`active` での絞り込み（`ListPage*Filtered`、`CountFiltered`）に含まれる。 | PostgreSQL 実装。SQL で空文字も `active` として扱う。 |
| C5 | `Find` が返すエンベロープの `Payload` を書き換えても、次の `Find` の結果は変わらない。 | メモリ実装。`Payload` を複製して返す。 |
| C6 | ナノ秒を含む時刻を保存すると、マイクロ秒に切り捨てた値として読み戻る。対象は `User.CreatedAt`、`User.UpdatedAt`、エンベロープの `IssuedAt`、`ExpiresAt` である。 | メモリ実装。`Save` で列の時刻をマイクロ秒に切り捨てる。 |

観測の結果、6 件とも実在した。
C1、C2、C3、C5、C6 はメモリ実装だけで、C4 は PostgreSQL 実装だけで落ちた。
いずれも既存の PostgreSQL の制約、ドメインの規則（`EffectiveStatus`）、または規範（REQ-IDMANAGEMENT-005）に合わせるだけで直せるので、すべてこの項目で直し、切り出した項目はない。

| 候補 | 分類 | 修正 |
|---|---|---|
| C1 | (c) 実装を直す | メモリ `UserRepository.Save` は、名前の索引を持つ User が削除済みでなく別の User であるときだけ、有効な User の保存を拒否する。削除済みの User は、その名前を使う有効な User から索引を奪わない。前の名前の索引は、自分を指しているときだけ消す。 |
| C2 | (c) 実装を直す | `cloneUser` で `Save` の引数を複製して保存し、すべての読み取りで複製を返す。 |
| C3 | (c) 実装を直す | 保存し直しでは、既存の `TenantID` と `CreatedAt` を保つ。 |
| C4 | (c) 実装を直す。REQ-IDMANAGEMENT-005 への準拠の回復 | 絞り込みの SQL と `users_tenant_lifecycle_status_active_idx` の式を `coalesce(nullif(lifecycle->>'status', ''), 'active')` にする。索引の式を揃えたので、状態の絞り込みは引き続きこの索引を使う（`TestUserSearchQueryPlanUsesTenantAndTrigramIndex`）。 |
| C5 | (c) 実装を直す | `cloneEnvelope` で `Save` の引数と `Find` の返り値の `Payload` を複製する。 |
| C6 | (c) 実装を直す | メモリ実装を PostgreSQL に合わせ、`User` の `CreatedAt`、`UpdatedAt` とエンベロープの `IssuedAt`、`ExpiresAt` をマイクロ秒に切り捨てて保存する。`lifecycle` の中の時刻は PostgreSQL でも JSONB にナノ秒のまま入るので、切り捨てない。 |

C4 は保存側で空文字を `active` に正規化する案もあった。
読み戻した `Lifecycle.Status` がメモリ実装と食い違い、空文字の行を残したまま検索側の式だけが正しくなるわけでもないので、設計どおり SQL 側で扱った。

採用しない案は次のとおりである。

| 案 | 採用しない理由 |
|---|---|
| 候補ごとに項目を分ける | 6 件とも同じ契約パッケージに同じ形のサブテストを足す作業であり、別々に受け入れられる成果がない。記録を分けると、readiness pass と証拠の固定費が 6 倍になる。 |
| メモリ実装の読み取りを複製せず、「返り値を書き換えない」ことを呼び出し側の規約にする | 規約は検査できず、`Save` を呼び忘れたユースケースのテストが通り続ける。設計ガイドラインの「所有者を曖昧にしない可変値」にも反する。 |

## 計画

1. C1〜C6 のサブテストを書き、両実装で実行して RED を観測する。DB を通すテストはサンドボックスの外で実行し、PostgreSQL 側がスキップされていないことを確かめる。
2. 実在した食い違いを、局所的な修正と規範判断を要するものに分ける。
3. 局所的な修正を 1 候補ずつ行い、GREEN にする。
4. 規範判断を要する食い違いは bugfix work item に切り出し、契約では wi-16658 と同じく、その項目を名指す理由付きで一時的に外す。
5. `docs/development/testing.md` の契約の節に書き足すことがあれば更新する。

着手時に未解決だった 2 つの問いには、手順 1 の観測の後に次のとおり答えた。

- **C4 は製品の振る舞いの修正か。** 修正である。REQ-IDMANAGEMENT-005 は、`status` を指定した一覧が条件に一致する User だけを返し、その件数を `pagination.total_items` に返すと定める。状態が未設定の User はドメインの `EffectiveStatus` で `active` なので、PostgreSQL 構成で漏れるのはこの規範への違反である。`change_kind` を `bugfix` にし、`affected_spec` に REQ-IDMANAGEMENT-005 を `conforms` で記した。
- **C6 はどちらへ合わせるか。** 時刻の分解能を定める規範はないので、メモリ実装を PostgreSQL に合わせてマイクロ秒に切り捨てた。`mise run test-go-changed` で逆依存のパッケージを含めて実行し、ナノ秒の一致に依存して壊れる上位のテストはなかった。

`docs/development/testing.md` の共有契約の節は他の Context にも適用する一般則であり、時刻の分解能と共有の扱いを他の Context へ広げるかは対象外としたので、変えていない。

既存の契約のサブテストは `//spec:covers` 付きの `TestPersistenceContract` から実行され、変更する `Save` と読み取りの往復、一意性、テナント分離、並び順を固定している。そのため特性化テストは足していない。

## タスク

- [x] T001 [Test] C1〜C6 のサブテストを書き、両実装で実行して RED を記録する。境界は contract、検査は `mise run test-go-test -- ./backend/idmanagement/user/db_memory/ TestPersistenceContract` と、サンドボックスの外での同じ検査の `db_postgres` 版。
- [x] T002 [Triage] 実在した食い違いを分類し、未解決の 2 つの問いに答えて設計へ書く。
- [x] T003 [Fix] 局所的な修正を GREEN にする。C4 は REQ-IDMANAGEMENT-005、`TestPersistenceContract/user_repository/unset_status_filters_as_active`。
- [x] T004 [Triage] 規範判断を要する食い違いを bugfix work item に切り出す。該当なし。C4 は既存の規範への準拠の回復なので、この項目で直した。
- [x] T005 [Verify] 変更を検証する。

## 検証

- メモリ実装と PostgreSQL 実装のテストが、追加したサブテストを同じ契約本体から実行する。
- 各修正を戻すと、対応するサブテストのうち、直した実装の側だけが落ちる。
- `mise run test-go-package -- ./backend/idmanagement/user/db_postgres/` をサンドボックスの外で実行し、スキップされずに通る。
- `mise run test-go-changed`
- `mise run verify`

## リスク

- **C2 の複製で上位のテストが落ちる。** メモリ実装の共有に依存して、`Save` を呼ばずに状態を変えている上位のテストやユースケースがあれば、複製によって落ちる。これは製品の誤りの検出であり、テストを緩めて通さない。落ちた箇所が多い場合は、その件数を記録して進め方を決め直す。
- **PostgreSQL 側のテストが黙ってスキップされる。** サンドボックス内では `testing_postgres.Require` がスキップする。検証はサンドボックスの外で行い、PostgreSQL 側でだけ落ちる C4 の RED を観測して、実行されたことを確かめる。
- **候補が実在しない。** 読解だけから立てた仮説なので、観測で否定される候補があり得る。その場合も、サブテストは再発の検出として残す。

## 完了

- **完了日**: 2026-10-08
- **要約**:
  `mise run spec-diff` は規範仕様の差分なしを報告した。REQ-IDMANAGEMENT-005 は変えず、PostgreSQL 構成で状態が未設定の User が `status=active` の絞り込みの一覧と件数から漏れていたのを、規範に合わせて直した。あわせて、`idmanagement/user` の共有契約へ 6 件のサブテストを加え、メモリ実装を PostgreSQL 実装に揃えた。削除済みの User は一意性に数えず、保存と読み取りでは値を共有せず、保存し直しても作成時刻と所属テナントを保ち、時刻の列とトークンの時刻をマイクロ秒に切り捨て、トークンのペイロードを共有しない。
- **主要ユースケースの証拠**:
  - id: unset-status-filters-as-active
    red: 修正前の PostgreSQL 実装で `TestPersistenceContract/user_repository/unset_status_filters_as_active` が `ListPageFiltered(active) = ([], <nil>), want the user without a status` で失敗した。メモリ実装では同じサブテストが通った。
    fault_injection: 絞り込みの SQL から `nullif(..., '')` を外して sqlc を再生成すると、PostgreSQL 実装の同じサブテストだけが同じ表明で失敗し、メモリ実装は通った。
- **受け入れ RED の証拠**:
  - **テスト**: `TestPersistenceContract`（`backend/idmanagement/user/db_memory`）に加えた 6 件のサブテスト。
  - **要件**: N/A: C1〜C3、C5、C6 はアダプター間の永続化の契約であり、規範となる製品要件がない。
  - **観測した失敗**: メモリ実装で、C1 は `Save tombstone again: preferred username already exists`、C2 は `stored user changed after mutating the Save argument`、C3 は `resaved user tenant=tenant-b created_at=...04:04:05...`、C5 は `Find after mutating the Save argument ... new_email:saved-side@example.com`、C6 は User とトークンの両方で `...05.123456789` のナノ秒の読み戻しで失敗した。
  - **検出できる理由**: 各サブテストは PostgreSQL 実装では通る同じ表明であり、メモリ実装だけが落ちることで食い違いそのものを区別する。
- **単体 RED の証拠**:
  - **テスト**: 単体境界は該当しない。契約の境界が最も狭い観測点である。
  - **要件**: N/A: アダプターの永続化の意味であり、単体で切り出せる純粋な判断がない。
  - **観測した失敗**: 上記の契約のサブテストの失敗を代替検査とした。
  - **検出できる理由**: 共有の契約本体が両実装を同じ表明で検査する。
- **変更耐性の結果**:
  `mise run test-go-mutation -- ./backend/idmanagement/user/db_memory/` は 44 件の変異のうち 40 件を検出した。生き残った 2 件と未被覆の 2 件は、変更していない `user_import_committer.go` の条件の反転である。変異器が表現できない故障は手で注入し、すべて対応するサブテストだけが落ちた。名前の判定から削除済みの除外を外す、削除済みの User に索引を上書きさせる、`Save` の複製を外す、`FindByEmail` と `ListPage` の複製を外す、`Roles` の深い複製を外す、`CreatedAt` と `TenantID` の保持をそれぞれ外す、トークンの `Find` と `Save` の複製をそれぞれ外す、User とトークンの時刻の切り捨てをそれぞれ外す、PostgreSQL の `nullif` を外す、の 13 件である。
- **検証結果**:
  - `mise run test-go-changed` - 成功（逆依存を含む、race 付き）
  - `mise run test-go-package -- ./backend/idmanagement/user/db_postgres/` - サンドボックスの外で成功（スキップなし、クエリプランの検査を含む）
  - `mise run check-schema` - 成功
  - `mise run verify` - 成功
  - `mise run test-ui-e2e` - 成功（39 件）
