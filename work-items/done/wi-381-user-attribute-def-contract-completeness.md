---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# UserAttributeDef の契約を実際の線上表現に合わせる

`Product.ClaimMapping.UserAttributeDef` が、`GET` / `PUT /api/admin/v1/tenant/user_attribute_schema` と `GET /api/account/profile` の `editable_attributes` に実際に出ている 10 フィールドすべてを宣言するようになった。追加したのは `label` / `type` / `multi_valued` / `required` / `editable_by_user` / `claim_name` / `oidc_scope` / `pii` の 8 個で、省略可否は Go の `omitempty` に、真偽値の既定は Go の zero value に一致する。`label` に 100、`claim_name` に 1〜100、`oidc_scope` に 1〜64 の長さ上限が契約に出るようになり、String length limits の Name / Handle 区分が公開契約側にも現れる。`visibility` からは実装が持たない既定値 `Private` を落とし、値の省略が拒否されることを `@doc` に書いた。`GroupAttributeDef` は突き合わせの結果、既に実装と一致していたので変更していない。
`just spec-diff` は「no normative specification change」と報告する。normative scenario・状態遷移・宣言名のいずれも増減していないためで、モデルのフィールド追加はこの道具の対象外である。意味の差分は生成される OpenAPI の `Contract.UserAttributeDef` にのみ現れる。実装・受理する値・返す値は変更していない。
突き合わせの過程で、`AttrVisibility` と `AttributeType` の enum 値が TypeSpec では PascalCase (`Private` / `String`)、線上では snake_case (`private` / `string`) であることが分かった。これは TypeSpec 全体に共通する食い違い (`UserStatus` なども同じ) なので、この work item では触れていない。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
