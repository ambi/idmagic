---
status: completed
archived_from: bb8b8175200d963c2b68946eff1c6aa4d4a57935
---

# `system_admin` を制御面 User のための予約ロールにする

`mise run spec-diff` の規範差分は `added scenarios: REQ-IDMANAGEMENT-032` の 1 件である。
`system_admin` が予約ロールになり、新しく割り当てられる対象は制御面テナントの `User` と `Group` だけになった。`Agent` はどのテナントでも割り当てられない。
判定は `backend/idmanagement/usecases/role_assignment.go` の `ValidateRoleAssignment` が持ち、`User` と `Group` の作成と更新、`Agent` の登録と更新、`User` CSV と `Group` CSV の計画器の 8 箇所が保存前に呼ぶ。CSV の適用も同じ計画器を通るため、保存済みプレビューから予約ロールが入る経路は残らない。
判定は保存後のロール集合ではなく書き込みが新しく加える予約ロールだけを見るため、既存値の再送、無編集エクスポートの再適用、`roles` を明示した除去のいずれも従来どおり通る。
公開契約は増えていない。6 操作の 422 union は既に `InvalidRoleError` を含んでおり、拒否は既存の `invalid_role` と CSV の `invalid_roles` を使う。`mise run check-api-compat` は破壊的変更なしで通った。

全文は `git show bb8b8175200d963c2b68946eff1c6aa4d4a57935:<このファイルのパス>` で読む。
