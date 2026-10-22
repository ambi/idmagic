---
status: completed
---

# 一意と定義したユーザー属性の値を、どの書き込みでも一意に保つ

REQ-TENANCY-002、REQ-IDMANAGEMENT-089、090、051、004 と、`UserAttributeDef`、`UpdateAdminUser`、`UpdateUserProfile`、`UpdateTenantUserAttributeSchema` を改め、`AttributeValueConflictError` と `AttributeValuesNotUniqueError` を足した。
テナントの属性スキーマで、単一の値の文字列の属性を `unique` と定義できる。
IdManagement は、User の保存と同じトランザクションで一意性の表（`user_unique_attribute_values`）を書き直し、同じテナントの削除されていないほかの User と値が重なる保存を拒否する。
どの書き込みの経路も保存を通るので、管理者と本人の更新は 409、CSV の適用は行の拒否になり、経路ごとの検査を持たない。
属性を `unique` に変えるスキーマの保存は既存の値を表へ写し、重なりがあれば 409 で拒否する。
User の保存とスキーマの保存は、テナントごとの勧告的ロックで交差しない。
値は保存した文字列の完全一致で比べ、表記の正規化は wi-14713 が Entra の sourceAnchor について行う。

全文は `git log -1 -- work-items/active/wi-90369-keep-unique-user-attribute-values-unique-on-every-write.md` が示すコミットの親、またはそのコミットで読む。
