---
status: in_progress
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-23
priority: p2
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/tenancy/features/attribute-schema.md, requirement: REQ-TENANCY-002 }
  - { path: docs/modules/identity-management/features/user.md, requirement: REQ-IDMANAGEMENT-089 }
  - { path: docs/modules/identity-management/features/user.md, requirement: REQ-IDMANAGEMENT-090 }
  - { path: docs/modules/identity-management/features/account.md, requirement: REQ-IDMANAGEMENT-051 }
  - { path: docs/modules/identity-management/features/user-csv.md, requirement: REQ-IDMANAGEMENT-004 }
  - { path: spec/modules/claim-mapping/models.tsp, symbol: IdMagic.Contract.UserAttributeDef }
  - { path: spec/modules/identity-management/models.tsp, symbol: IdMagic.Contract.AttributeValueConflictError }
  - { path: spec/modules/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.UpdateAdminUser }
  - { path: spec/modules/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.UpdateUserProfile }
  - { path: spec/modules/tenancy/models.tsp, symbol: IdMagic.Contract.AttributeValuesNotUniqueError }
  - { path: spec/modules/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.UpdateTenantUserAttributeSchema }
primary_use_cases:
  - id: unique-attribute-write
    requirement: REQ-IDMANAGEMENT-089
    observable_result: 一意の属性にほかの User と同じ値を書く管理者の作成と更新、本人の更新、CSV の適用が拒否され、User が変わらない。
    boundary: acceptance
    test: { path: backend/idmanagement/internal/user/usecases/unique_attribute_examples_test.go, name: TestUniqueAttributeValuesStayUniqueOnEveryWrite, task: test-go-race }
    fault_model: 書き込みの経路のどれかが一意性の表を通らずに保存するか、一意性の失敗を別の失敗に写す。
  - id: unique-attribute-schema
    requirement: REQ-TENANCY-002
    observable_result: 既存の User の値が重なる属性を一意にするスキーマの保存は 409 で拒否され、重ならなければ以後の重複を止める。
    boundary: acceptance
    test: { path: backend/tenancy/internal/handlers_http/admin_user_attribute_schema_handler_test.go, name: TestUserAttributeSchemaPutRejectsUniqueOverSharedValues, task: test-go-race }
    fault_model: 一意にした時点の既存の値を表へ写さず、既存の重複や以後の重複を見逃す。
---

# 一意と定義したユーザー属性の値を、どの書き込みでも一意に保つ

## 動機

Entra のドメインフェデレーションは、sourceAnchor の属性の値から ImmutableID を導き、Entra はその値で Entra のユーザーを決める。
二人の User が同じ値を持つと、片方がもう一方の Microsoft 365 のアカウントでサインインする。
この一意性は値を保存する IdManagement の不変条件だが、IdManagement はユーザー名とメールアドレスのほかに、属性の値の一意性を守る仕組みを持たない。
WsFederation が発行のたびにテナントの User を走査して確かめる案は、サインインの経路に User 数に比例する処理を入れ、値の持ち主でないモジュールに不変条件の点検を負わせるので採らない。

## 対象範囲

- テナントのユーザー属性の定義に `unique` を足す。一意にできるのは、単一の値の文字列の属性だけである。
- IdManagement は、`unique` の属性の値を、User の保存と同じトランザクションで一意性の表に書く。同じテナントのほかの削除されていない User と同じ値を保存しようとした書き込みは失敗し、User を変えない。
- 管理者による更新と本人によるプロフィールの更新は 409 と `attribute_value_conflict` で拒否し、CSV の適用は行を拒否する。管理者による作成は属性を受け取らないので重ならない。
- 属性を `unique` に変えるスキーマの保存は、既存の User の値を一意性の表へ写し、重複があれば 409 と `attribute_values_not_unique` で拒否する。`unique` でなくなった属性の値は表から消す。

## 対象外

- 値の表記の正規化。一意性は保存した文字列の完全一致で判定する。Entra の sourceAnchor の表記は wi-14713 で一つに限る。
- SCIM。SCIM はテナント定義の属性を書かない。
- 組み込みの属性の一意性。組み込みの属性の定義は変えられない。

## 設計

### 境界の判断

入力：[境界を選ぶ判断手順](../../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)、[IdManagement のデータ](../../docs/modules/identity-management/data.md#排他と一意性)。

| 候補 | 制約への適合 | 変更の波及 | 採否 |
| --- | --- | --- | --- |
| IdManagement が、User の保存と同じトランザクションで一意性の表を保つ | 不変条件の持ち主が守る。User の書き込みはすべて `internalSaveUser`（メモリ版は同じ排他）を通るので、経路ごとの検査が要らない。同時の書き込みは表の一意索引が止める | IdManagement の公開する型（`UserAttributeDef` と保存の失敗の型）が広がる。呼び出し側は新しい失敗を 409 に写す | 採る |
| 各ユースケースが保存の前に値で検索する | 経路ごとに検査を書く。同時の書き込みを止められない | 経路の追加のたびに漏れうる | 採らない |
| WsFederation が発行のたびに確かめる | 持ち主でないモジュールが不変条件を点検する。サインインに User 数に比例する処理が入る | Entra 以外の用途に使えない | 採らない |

モジュール間の依存は増えない。Tenancy は属性の定義の保存を、すでに使っている IdManagement の保存先を通して行い、新しい失敗を受け取る。

### データと操作

```go
// UserAttributeDef に足す。
Unique bool `json:"unique"`

// ports に足す。保存した値が、同じテナントのほかの User の一意の属性の値と重なったことを表す。
type UniqueAttributeConflictError struct{ Key string }
```

| 表 | 列 | 制約 |
| --- | --- | --- |
| `user_unique_attribute_values` | `tenant_id`、`attribute_key`、`value`、`user_id` | 主キー（`tenant_id`、`attribute_key`、`value`）、`user_id` の索引、`users` への外部キー（削除で連鎖） |

- User の保存は、テナントの属性スキーマの行を `FOR SHARE` で読み、その User の行を消してから、削除されていない User の `unique` の属性の空でない値を入れる。主キーの違反は `UniqueAttributeConflictError` にする。
- 属性スキーマの保存は、行を固定して新旧の `unique` の集合を比べ、新たに一意にした属性の既存の値を入れ、外した属性の行を消す。
- `FOR SHARE` と固定により、スキーマの変更と User の保存が交差しても、変更の前の集合で保存した値が写しから漏れない。

## タスク

- [x] T001 [Spec] REQ-TENANCY-002、REQ-IDMANAGEMENT-089、090、051、004、`UserAttributeDef`、二つの失敗の Problem Details、三つの操作の応答、データの文書と表の説明を改めた。管理者による作成は属性を受け取らないので、作成の拒否は仕様に足さなかった。
- [x] T002 [Acceptance] 共有の契約 `runUniqueAttributeValues` はメモリと PostgreSQL の両方で `create with a taken value: err = <nil>` で失敗し、CSV の行は `apply_failed` で失敗した。
- [x] T003 [Domain] `UserAttributeDef.Unique` の形式の検査、`UniqueKeys`、`UniqueAttributeValues` を実装した。
- [x] T004 [Infra] PostgreSQL は `user_unique_attribute_values` と、テナントごとの勧告的ロック（User の保存は共有、スキーマの保存は排他）で実装した。メモリは `NewUserStore` が User とスキーマの保存先を一つのロックの順序でつなぐ。
- [x] T005 [App] 管理者と本人の更新を 409 `attribute_value_conflict`、CSV の行を `attribute_value_conflict`、スキーマの保存を 409 `attribute_values_not_unique` に写した。ユースケースは保存先の失敗の型をそのまま返し、HTTP の層が写す。
- [x] T006 [UI] 属性スキーマの画面に、文字列の属性だけに出る `unique` の切り替えと、重なりによる拒否の辞書の文言を足した。
- [x] T007 [Verify] 変更を検証した。結果は検証の節に書いた。

## 検証

- `mise run test-go-changed -- main`（wi-11577 を含む main へ rebase した後）：5,003 件が成功、失敗とスキップは 0 件。PostgreSQL の契約テストを含む。
- `typecheck-ui`、`lint-ui`、`format-check-ui`、`test-ui-unit-file`（属性スキーマの画面と正規化）：成功。属性スキーマの画面を通る e2e テストはない。
- `mise run check`、`lint-go`、`check-spec-impact`、`check-boundaries`、`check-schema-tables`、`check-operation-references`：成功。
- `check-schema`（PostgreSQL のスキーマの収束）は、この環境で Docker が動いていないので実行していない。
- ミューテーションテスト（変更行、`main` と比較）：`user/domain` は 5 件、`db_memory` は 13 件、`db_postgres` は 19 件をすべて検出した。最初の `user/domain` の調査で、`unique` の形式の検査と値の取り出しの観測の不足が見つかり、単体テストを足した。
- 手動の故障注入：管理者の 409、本人の 409、CSV の行のコード、Tenancy への失敗の変換、`unique` の写しをそれぞれ外し、対応するテストが六つとも検出した。

## 要約

REQ-TENANCY-002、REQ-IDMANAGEMENT-089、090、051、004 と、`UserAttributeDef`、`UpdateAdminUser`、`UpdateUserProfile`、`UpdateTenantUserAttributeSchema` を改め、`AttributeValueConflictError` と `AttributeValuesNotUniqueError` を足した。
テナントの属性スキーマで、単一の値の文字列の属性を `unique` と定義できる。
IdManagement は、User の保存と同じトランザクションで一意性の表（`user_unique_attribute_values`）を書き直し、同じテナントの削除されていないほかの User と値が重なる保存を拒否する。
どの書き込みの経路も保存を通るので、管理者と本人の更新は 409、CSV の適用は行の拒否になり、経路ごとの検査を持たない。
属性を `unique` に変えるスキーマの保存は既存の値を表へ写し、重なりがあれば 409 で拒否する。
User の保存とスキーマの保存は、テナントごとの勧告的ロックで交差しない。
値は保存した文字列の完全一致で比べ、表記の正規化は wi-14713 が Entra の sourceAnchor について行う。

## リスク

- User の保存ごとに、スキーマの行の読み取りと一意性の表の更新が増える。主キーと `user_id` の索引で引く。
