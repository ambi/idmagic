---
depends_on: []
status: pending
authors: [tn]
risk: low
created_at: 2026-08-16
change_kind: bugfix
priority: p2
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

## Tasks
- [ ] T001 [Inventory] 両モデルについて、TypeSpec の宣言と Go の JSON タグを 1 対 1 で突き合わせる。
- [ ] T002 [Spec] 欠けているフィールドと `@maxLength` を宣言し、`just spec-render` で再生成する。
- [ ] T003 [Verify] 生成された OpenAPI の要求／応答本体が実際の JSON と一致することを確認する。

## Verification
- `just check-spec`
- `just check-api-compat`
- `just verify`
- 手動確認: `spec/generated/docs` の該当モデルが、実際のレスポンス本体と同じフィールドを列挙している。

## Risk Notes
契約へフィールドを足すだけなので、受理範囲も応答も変わらない。ただし `pii` と `visibility` の既定値を実装と違えて宣言すると、契約を読んで組んだ利用者が安全側でない既定を前提にしうる。既定値は Go の実装から写す。
