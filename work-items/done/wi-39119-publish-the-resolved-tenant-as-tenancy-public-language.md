---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-09
priority: p2
depends_on: [wi-92970-adopt-information-hiding-modular-design]
change_kind: refactor
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 解決済みのテナントを運ぶ Go の関数の置き場所と、クォータの操作を呼ぶ経路だけを変えるので、HTTP の応答、テナントの解決規則、発行するイベントは変わらず、リリースの読者へ知らせることがない。
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - backend/tenancy/ports/resolved_tenant.go
    - backend/tenancy/module.go
    - backend/tenancy/ports/quota_repository.go
    - backend/tenancy/db_postgres/quota_repository.go
    - backend/idmanagement/user/db_postgres/user_import_committer.go
    - backend/idmanagement/group/db_postgres/group_import_committer.go
    - backend/cmd/internal/bootstrap/postgres.go
    - tools/check/src/boundary-fitness.ts
    - tools/check/src/check-boundaries.ts
    - tools/check/boundary-debt.json
  tests:
    - backend/tenancy/ports/resolved_tenant_test.go
    - backend/idmanagement/user/db_postgres/user_import_committer_test.go
    - backend/idmanagement/group/db_postgres/import_committers_test.go
  stop_before_reading: [frontend, spec, docs/modules]
spec_impact: { kind: none, reason: "解決済みのテナントを運ぶ Go のパッケージの置き場所と、Tenancy の非公開パッケージへの直接の依存だけを変える。テナントの解決規則、HTTP の応答、発行するドメインイベント、永続状態は変えない。" }
---

# 解決済みのテナントを Tenancy の公開契約として公開する

## 動機

`tools/check/boundary-debt.json` の `private-import` のうち 54 件は、ほかのモジュールから Tenancy の非公開パッケージへの依存である。

| 参照先 | 件数 | 原因 |
| --- | --- | --- |
| `backend/tenancy`（ルートパッケージ） | 41 | 解決済みのテナントを `context.Context` に載せて読む `backend/tenancy/context.go` が、組み立て地点にだけ公開するルートパッケージにある |
| `backend/tenancy/usecases` | 11 | Tenancy のユースケースを直接呼ぶ |
| `backend/tenancy/db_postgres` | 2 | Tenancy のアダプターを直接使う |

テナントの境界は、ほぼすべてのモジュールが前提にする。
それにもかかわらず、その前提を受け取る経路が公開パッケージにないため、正しい前提の受け取りまでが非公開の実装への依存になっている。
共有ライブラリ（`shared/http/support_http`、`shared/storage/db_memory` など）から Tenancy への `shared-dependency` も、同じ受け渡しを原因に含む。

## 対象範囲

- `backend/tenancy/context.go` の `WithTenant`、`Tenant`、`TenantID`、`Issuer`、`URLPrefix` を、Tenancy の公開パッケージへ移し、呼び出し元を書き換える。
- ほかのモジュールが Tenancy の `usecases` と `db_postgres` を直接使う箇所を、公開操作を経由する形へ直す。
- 解消した違反 ID を `tools/check/boundary-debt.json` から消す。

## 対象外

- テナントの解決規則そのものの変更。
- Tenancy の `internal/` への移行。モジュールごとの `internal/` 移行の項目で扱う。

## 設計

境界を選ぶ判断手順の記録は、[モジュール設計の項目](wi-92970-adopt-information-hiding-modular-design.md)の「判断手順の適用」にある。
要点は次のとおりである。

- D1：解決済みのテナントの状態と、テナントのない文脈を default テナントへ落とさない規則の所有者は Tenancy なので、Tenancy を担当とする。
- D3：非公開の関数をそのまま公開するのではなく、取得と付与を Tenancy の公開契約として公開パッケージに置く。
- D5：値の型が `tenancy/domain.Tenant` であり、規則がテナント分離の保証そのものなので、共有ライブラリの条件を満たさない。

置き場所は `ports` を第一候補にする。
`context.Context` に値を載せる処理は計算ではなく実行時の受け渡しなので、`structure.md` が決定論的な計算だけを置くと定める `domain` には合わない。

採らない案は次のとおり。

| 案 | 採らない理由 |
| --- | --- |
| テナントの受け渡しを `backend/shared` へ移す | Tenancy の型と規則を運ぶので D5 を満たさず、`shared-dependency` へ形を変えるだけである |
| 境界の検査で Tenancy を例外にする | 検査にモジュール固有の例外を持ち込むと、責務表という宣言が検査の判断を説明しなくなる |

### 解決済みのテナントの置き場所

`context.go` の関数を `backend/tenancy/ports` へそのまま移し、ファイル名を `resolved_tenant.go` にする。
`ports` には、ほかのモジュールがすでに `tenantports` の別名で import する `QuotaRepository` などがあるので、呼び出し側は同じ別名で受け取れる。
`ports` が import する `idmanagement/user/domain`、`idmanagement/group/domain`、`shared/notification/ports` は Tenancy の受け渡しを使わないので、移しても Go の import の循環は生じない。
組み立て地点だけが使う `tenancy.Module` はルートパッケージに残す。

### Tenancy の usecases と db_postgres への依存

| 参照先 | 使っている操作 | 直し方 | 理由 |
| --- | --- | --- | --- |
| `tenancy/usecases`（11 パッケージ） | `CheckQuotaAndIncrement`、`DecrementQuota` | 呼び出し側が受け取っている `tenantports.QuotaRepository` の `CheckAndIncrement`、`Decrement` を直接呼び、2 関数を消す | 2 関数は port の同名の操作へ引数を渡すだけで、隠す規則がない。上限の判定と使用量の加減算は port の契約（`EX-TENANCY-036-*`、`EX-TENANCY-037-*`）がすでに公開している |
| `tenancy/db_postgres`（`idmanagement/user/db_postgres`、`idmanagement/group/db_postgres`） | `NewQuotaRepository(tx)` | 取り込みの committer の構築関数が `func(pgx.Tx) tenantports.QuotaRepository` を必須の引数として受け取り、組み立て地点が Tenancy の `QuotaRepositoryInTx` を渡す。フィールドは非公開にし、構造体リテラルで渡し忘れる経路をなくす | 取り込みの 1 行は、使用量の加減算と行の書き込みを同じトランザクションで確定する。所有者の公開操作を呼び出し側のトランザクションに参加させれば、原子性を保ったまま実装への依存だけを外せる（D2、D7）。別のトランザクションで呼ぶ形は、行の確定と使用量がずれる窓を作るので採らない |

`shared` から Tenancy への `shared-dependency` は、参照先が `ports` へ移っても共有ライブラリがモジュールへ依存する点は変わらないので、本項目では消えない。
共有ライブラリ側の整理は、[境界の負債の順位付け](wi-33994-reinventory-the-remaining-boundary-debt.md)で扱う。

### 証拠

テストの import と呼び出しの修飾子を書き換えるので、[振る舞いを保つ変更](../../docs/development/specification-first-workflow.md#振る舞いを保つ変更)には当たらない。

| 証拠 | 内容 |
| --- | --- |
| 受け入れ RED | N/A: 製品の振る舞いを変えない依存の向きの変更であり、対応する REQ がない。代替として、`tools/check/boundary-debt.json` から Tenancy への `private-import` の項目を先に消し、`mise run check-boundaries` が未記録の違反として失敗することを確かめる |
| 単体 RED | N/A: 振る舞いを変えないので、実装前に失敗する単体テストはない。移す関数のテスト（`TestWithTenantAndAccessors`、`TestTenantIDPanicsWithoutResolvedTenant`）は表明を変えずに移す |
| 特性化 | PostgreSQL の取り込みの committer が使用量を加減算する経路は、アダプターの水準で固定するテストがない。変更の前に `TestCharacterizeUserImportRowQuota` と `TestCharacterizeGroupImportRowQuota` で、上限内の加算、上限での拒否、行の書き込みの失敗による使用量の巻き戻し、削除による減算を固定する |
| 変更耐性 | `mise run test-go-mutation -- backend/tenancy/ports` で、移した関数への変異を移したテストが検出することを確かめる。committer は、使用量の加減算を呼び出し側のトランザクションの外で行う変異を手で入れ、特性化テストが失敗することを確かめる。組み立て地点での渡し忘れは、構築関数の必須引数にしたのでコンパイルが拒否する |

## 計画

1. `context.go` を公開パッケージへ移し、呼び出し元を一括で書き換える。
2. 残った `private-import` を一件ずつ公開操作の経由へ直す。

## タスク

- [x] T001 [App] 解決済みのテナントの受け渡しを Tenancy の公開パッケージへ移す。
  - `backend/tenancy/context.go` を `backend/tenancy/ports/resolved_tenant.go` へ、テストを `resolved_tenant_test.go` へ移した。テストの表明は変えていない
  - 呼び出し側 195 ファイルの修飾子を `tenancy.` から `tenantports.` へ書き換え、ルートパッケージの import は `tenancy.Module` を使う組み立て地点とテストだけに残した
  - 検査: `mise run lint-go`、`mise run test-go-changed`、`mise run check-boundaries`（残りは T002 で扱う 13 件だけになった）
- [x] T002 [App] Tenancy の `usecases` と `db_postgres` への直接の依存を公開操作の経由へ直す。
  - 特性化: `TestCharacterizeUserImportRowQuota`（`backend/idmanagement/user/db_postgres`）と `TestCharacterizeGroupImportRowQuota`（`backend/idmanagement/group/db_postgres`）を、変更前の `main` の実装に対して書いて固定した。誤った期待値を置いた実行で実際の使用量（1、0、1）を読み、その値を期待値にした
  - 特性化の検出能力: `mise run test-go-mutation` で、変更前の committer の該当行の変異はすべて検出された。削除の経路の 94 行目（`Decrement` のエラー判定の反転）だけが生き残ったので、削除の監査記録の表明を足し、`mise run test-go-mutation-mutant` で検出を確かめた
  - `CheckQuotaAndIncrement` と `DecrementQuota` の 11 か所の呼び出しを、`tenantports.QuotaRepository` の操作の直接の呼び出しへ ast-grep で書き換え、2 関数とそのテストを消した。テストは repository の契約テスト（`EX-TENANCY-036-*`、`EX-TENANCY-037-*`）と同じことを確かめていた
  - committer は `NewUserImportRowCommitter` と `NewGroupImportRowCommitter` で `func(pgx.Tx) tenantports.QuotaRepository` を受け取り、組み立て地点が `tenancypostgres.QuotaRepositoryInTx` を渡す。特性化テストは構築の 1 行だけを変え、期待値を変えずに通った
  - 検査: `mise run test-go-test -- <package> TestCharacterize*`、`mise run lint-go`、`mise run test-go-changed`
- [x] T003 [Tooling] 解消した違反 ID を `tools/check/boundary-debt.json` から消す。
  - Tenancy への `private-import` の 10 項目（54 件）を消した。台帳は 211 件から 157 件になり、`private-import` は 111 件から 57 件になった
  - `module-cycle` の X->Tenancy は、公開パッケージへの依存でもモジュールの依存として残るので消えない。`shared-dependency:shared->Tenancy` も残る
  - 検査: `mise run check-boundaries`、`mise run check-boundary-debt-ratchet -- main`
- [x] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`

## リスク

- 公開パッケージへ移した操作が、テナントの境界ではない用途にも使われるおそれがある。
  移す操作は解決済みのテナントの受け渡しに限り、Tenancy の管理操作は公開しない。

## 完了

- **完了日**: 2026-10-10
- **要約**:
  `mise run spec-diff` は規範の差分なしを報告する。
  解決済みのテナントを `context.Context` で受け渡す `WithTenant`、`Tenant`、`TenantID`、`Issuer`、`URLPrefix` を、Tenancy のルートパッケージから公開パッケージ `backend/tenancy/ports` へ移した。
  クォータの加減算は、素通しだった `tenancy/usecases` の 2 関数を消し、呼び出し側が `tenantports.QuotaRepository` の操作を直接呼ぶ形にした。
  CSV 取り込みの PostgreSQL の committer は、構築関数の必須引数として Tenancy の `QuotaRepositoryInTx` を受け取り、行の書き込みと使用量の加減算を同じトランザクションで確定する。
  境界の負債の台帳から Tenancy への `private-import` の 54 件を消し、台帳は 211 件から 157 件になった。
  HTTP の応答、テナントの解決規則、発行するイベント、永続状態は変えていない。
- **受け入れ RED の証拠**:
  - **テスト**: `mise run check-boundaries`
  - **要件**: N/A: 規範上の振る舞いを変えない、依存の向きの変更であり、対応する REQ がない。
  - **観測した失敗**: 台帳から Tenancy への `private-import` の 10 項目を先に消すと、`mise run check-boundaries` が 54 件を `violation(s) are absent from boundary-debt.json` として失敗した。T001 の後は 13 件、T002 の後は 0 件になり、検査が通った。
  - **検出できる理由**: 依存の向きを変える作業なので、製品の振る舞いの RED は存在しない。代わりに、リファクタリングを必要にした構造ゲートが変更の前後の違いを観測した。
- **単体 RED の証拠**:
  - **テスト**: `TestCharacterizeUserImportRowQuota`、`TestCharacterizeGroupImportRowQuota`、`TestWithTenantAndAccessors`、`TestTenantIDPanicsWithoutResolvedTenant`
  - **要件**: N/A: 規範上の振る舞いを変えない、依存の向きの変更であり、対応する REQ がない。移したテストのうち `TestTenantIDPanicsWithoutResolvedTenant` は `REQ-TENANCY-006` を引く。
  - **観測した失敗**: 該当なし。振る舞いを変えないので、実装前に失敗する単体テストはない。代替として、書き換える committer の経路を変更前の実装で特性化し、使用量の加算を別のトランザクションで確定させる変異を手で入れると、ユーザーとグループの両方の `failed_row_write_rolls_the_count_back` が「使用量 2、期待 1」で失敗することを確かめた。
  - **検出できる理由**: 別のトランザクションでの加算は、本項目が採らなかった案（D2、D7）の誤りそのものであり、行の書き込みが失敗したときに使用量だけが残ることで観測できる。
- **変更耐性の結果**:
  `mise run test-go-mutation -- backend/tenancy/ports` は 3 件中 3 件を検出した。
  変更前の `backend/idmanagement/user/db_postgres` と `backend/idmanagement/group/db_postgres` への `test-go-mutation` では、committer の使用量の経路にある変異はすべて検出された。削除の経路の 94 行目（`Decrement` のエラー判定の反転）だけが生き残り、削除の監査記録の表明を足して検出した。
  `saveGroupInTx` の nil の正規化の 2 件（103 行目と 111 行目）は今回変えていない行であり、分類しない。
  組み立て地点での渡し忘れは、committer のフィールドを非公開にして構築関数の必須引数にしたので、コンパイルが拒否する。変更前の構造体リテラルの形で組み立て地点から渡す値を外しても、`backend/cmd/...` のテストはすべて通ったので、この故障はテストではなく型で塞いだ。
- **検証結果**:
  - `mise run check-boundaries` - 成功
  - `mise run check-boundary-debt-ratchet -- main` - 成功（private-import 111 → 57）
  - `mise run lint-go` - 成功
  - `mise run test-go-changed` - 成功
  - `mise run verify` - 成功
  - `mise run test-ui-e2e` - 成功（39 件）
