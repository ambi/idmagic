---
depends_on: []
status: completed
authors: [tn]
risk: low
created_at: 2026-08-16
change_kind: bugfix
priority: p2
initial_context:
  specification:
    - spec/contexts/tenancy/SPECIFICATION.md#REQ-TENANCY-002
    - spec/contexts/tenancy/SPECIFICATION.md#REQ-TENANCY-020
    - spec/SPECIFICATION.md
  typespec:
    - Product.ClaimMapping.UserAttributeDef
    - Product.ClaimMapping.AttrVisibility
    - Product.IdManagement.GroupAttributeDef
    - Product.Tenancy.TenantUserAttributeSchemaResponse
    - Product.Tenancy.TenantUserAttributeSchemaUpdateRequest
  source:
    - backend/idmanagement/user/domain/users.go
    - backend/idmanagement/group/domain/groups.go
    - backend/idmanagement/domain/enums.go
    - backend/tenancy/handlers_http/admin_user_attribute_schema_handler.go
    - backend/shared/spec/length.go
    - frontend/src/types.ts
  tests:
    - backend/idmanagement/user/domain
  stop_before_reading:
    - backend/oauth2
    - backend/sourcing
affected_spec:
  - { path: spec/contexts/claim-mapping/models.tsp, symbol: Product.ClaimMapping.UserAttributeDef }
  - { path: spec/contexts/identity-management/models.tsp, symbol: Product.IdManagement.GroupAttributeDef }
---

# UserAttributeDef の契約を実際の線上表現に合わせる

## Motivation
`GET` / `PUT /api/admin/v1/tenant/user_attribute_schema` の handler は `userdomain.UserAttributeDef` をそのまま JSON 化して返す（`toUserAttributeSchemaResponse`）。この構造体は 10 個のフィールドを持つ。

```go
type UserAttributeDef struct {
    Key            string
    Label          string                   `json:"label,omitempty"`
    Type           idmdomain.AttributeType
    MultiValued    bool
    Required       bool
    EditableByUser bool
    ClaimName      *string                  `json:"claim_name,omitempty"`
    OIDCScope      *string                  `json:"oidc_scope,omitempty"`
    Visibility     idmdomain.AttrVisibility
    PII            bool
}
```

一方 `spec/contexts/claim-mapping/models.tsp` の `UserAttributeDef` は `key` と `visibility` しか宣言していない。生成される OpenAPI は、このエンドポイントの要求本体と応答本体を 8 フィールド分すくなく記述している。契約だけを読む利用者は `claim_name`、`oidc_scope`、`pii`、`editable_by_user`、`multi_valued`、`required`、`type`、`label` の存在を知りえない。

`just check-spec` はこれを検出しない。TypeSpec のモデルと Go の構造体を突き合わせる仕組みが無く、モデルは単体で妥当なためである。`wi-128-string-length-limits-policy` は、`Label` / `ClaimName` / `OIDCScope` に長さ上限を宣言しようとしてこの欠落に当たり、長さとは独立した問題として分離した。

`pii` と `visibility` は claim 露出のフェイルクローズ判定に関わるので、契約に出ていないことは単なる記述漏れ以上の意味を持つ。

## Scope
- `Product.ClaimMapping.UserAttributeDef` に、実際に線上へ出ている残り 8 フィールドを宣言する。既定値と省略可否は Go の JSON タグに合わせる。
- 同時に、`wi-128` の String length limits に従って `label`（100）、`claim_name`（100）、`oidc_scope`（64）へ `@maxLength` を付ける。
- `Product.IdManagement.GroupAttributeDef` にも同じ突き合わせを行う。こちらは `key` / `label` / `type` / `multi_valued` / `required` を宣言済みなので、差分の有無を確認する。
- `just check-api-compat` で、フィールド追加が破壊的変更にならないことを確認する。

## Out of Scope
- TypeSpec のモデルと Go の構造体を機械的に突き合わせる検査の追加。有用だが独立した道具立てであり、この work item では行わない。
- 属性スキーマの振る舞いそのものの変更。契約の記述を実装に合わせるだけで、受理する値も返す値も変えない。

## Design
突き合わせの基準は Go の JSON タグである。`omitempty` の付いたフィールドだけを TypeSpec の optional (`?`) にし、それ以外は必須にする。真偽値は Go の zero value がそのまま線上の既定になるので `= false` を宣言する。長さ上限は `backend/shared/spec` の区分定数から写す (`LengthName` = 100、`LengthHandle` = 64)。`ClaimName` / `OIDCScope` は zog 側が `Chars(1, N)` なので `@minLength(1)` も併せて宣言する。

`visibility` だけは既存の宣言も実装と食い違っていた。TypeSpec は `= AttrVisibility.Private` を宣言していたが、`userAttributeDefSchema` の `Visibility` は `Required()` なので、値を省いた定義は既定値に落ちるのではなく 400 で拒否される。契約から既定値を落として必須のみを残す。応答では常に出るフィールドなので、生成される OpenAPI の `required` からは外れない。

`pii` の既定は `false` である。管理画面の新規属性フォームは `pii: true` を初期値にするが、それは UI の初期値であって契約の既定ではない。契約を読んで組む利用者が `false` を安全側だと誤解しないよう、明示的に送るべきことを `@doc` に書く。

## Plan
`UserAttributeDef` に不足フィールドと長さ上限を宣言し、`GroupAttributeDef` は差分の有無だけを確認する。実装には触れない。

## Tasks
- [x] T001 [Inventory] 両モデルについて、TypeSpec の宣言と Go の JSON タグを 1 対 1 で突き合わせる。

  `UserAttributeDef` (`backend/idmanagement/user/domain/users.go`) — 変更前は `key` / `visibility` の 2 個だけが宣言されていた。

  | JSON | Go | 検証 | 変更前の TypeSpec | 変更後 |
  | --- | --- | --- | --- | --- |
  | `key` | `string` | snake_case 必須 | `@minLength(1) key: string` | 変更なし |
  | `label` | `string,omitempty` | `CharsAtMost(LengthName)` | 未宣言 | `@maxLength(100) label?: string` |
  | `type` | `AttributeType` | enum 必須 | 未宣言 | `type: AttributeType` |
  | `multi_valued` | `bool` | なし | 未宣言 | `multi_valued: boolean = false` |
  | `required` | `bool` | なし | 未宣言 | `required: boolean = false` |
  | `editable_by_user` | `bool` | なし | 未宣言 | `editable_by_user: boolean = false` |
  | `claim_name` | `*string,omitempty` | `Chars(1, LengthName)` | 未宣言 | `@minLength(1) @maxLength(100) claim_name?: string` |
  | `oidc_scope` | `*string,omitempty` | `Chars(1, LengthHandle)` | 未宣言 | `@minLength(1) @maxLength(64) oidc_scope?: string` |
  | `visibility` | `AttrVisibility` | enum 必須 | `visibility: AttrVisibility = AttrVisibility.Private` | `visibility: AttrVisibility` (既定値を削除) |
  | `pii` | `bool` | なし | 未宣言 | `pii: boolean = false` |

  `GroupAttributeDef` (`backend/idmanagement/group/domain/groups.go`) — `key` / `label,omitempty` / `type` / `multi_valued` / `required` の 5 個で、TypeSpec の宣言 (`label` は `?` + `@maxLength(100)`、真偽値は `= false`) と一致していた。差分なし。
- [x] T002 [Spec] 欠けているフィールドと `@maxLength` を宣言し、`just spec-render` で再生成する。
- [x] T003 [Verify] 生成された OpenAPI の要求／応答本体が実際の JSON と一致することを確認する。

  `Contract.UserAttributeDef` の `properties` は 10 個になり、`required` は `key` / `type` / `multi_valued` / `required` / `editable_by_user` / `visibility` / `pii` の 7 個で、Go の `omitempty` が付かないフィールドと一致する。`spec/generated/docs/models/idmagic-contract-userattributedef.html` も同じ 10 フィールドを同じ要否で並べる。

## Verification
- `just check-spec`
- `just check-api-compat`
- `just verify`
- 手動確認: `spec/generated/docs` の該当モデルが、実際のレスポンス本体と同じフィールドを列挙している。

## Risk Notes
契約へフィールドを足すだけなので、受理範囲も応答も変わらない。ただし `pii` と `visibility` の既定値を実装と違えて宣言すると、契約を読んで組んだ利用者が安全側でない既定を前提にしうる。既定値は Go の実装から写す。

## Completion
- **Completed At**: 2026-08-18
- **Summary**:
  `Product.ClaimMapping.UserAttributeDef` が、`GET` / `PUT /api/admin/v1/tenant/user_attribute_schema` と `GET /api/account/profile` の `editable_attributes` に実際に出ている 10 フィールドすべてを宣言するようになった。追加したのは `label` / `type` / `multi_valued` / `required` / `editable_by_user` / `claim_name` / `oidc_scope` / `pii` の 8 個で、省略可否は Go の `omitempty` に、真偽値の既定は Go の zero value に一致する。`label` に 100、`claim_name` に 1〜100、`oidc_scope` に 1〜64 の長さ上限が契約に出るようになり、String length limits の Name / Handle 区分が公開契約側にも現れる。`visibility` からは実装が持たない既定値 `Private` を落とし、値の省略が拒否されることを `@doc` に書いた。`GroupAttributeDef` は突き合わせの結果、既に実装と一致していたので変更していない。
  `just spec-diff` は「no normative specification change」と報告する。normative scenario・状態遷移・宣言名のいずれも増減していないためで、モデルのフィールド追加はこの道具の対象外である。意味の差分は生成される OpenAPI の `Contract.UserAttributeDef` にのみ現れる。実装・受理する値・返す値は変更していない。
  突き合わせの過程で、`AttrVisibility` と `AttributeType` の enum 値が TypeSpec では PascalCase (`Private` / `String`)、線上では snake_case (`private` / `string`) であることが分かった。これは TypeSpec 全体に共通する食い違い (`UserStatus` なども同じ) なので、この work item では触れていない。
- **Verification Results**:
  - `just check-spec` - passed
  - `just check-api-compat` - passed (no breaking changes vs baseline)
  - `just verify` - passed
  - 手動確認: `spec/generated/docs/models/idmagic-contract-userattributedef.html` が実際のレスポンス本体と同じ 10 フィールドを同じ要否で列挙している。
