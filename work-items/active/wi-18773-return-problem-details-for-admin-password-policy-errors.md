---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-03
priority: p3
depends_on: [wi-96676-transcribe-implicit-specifications-of-identity-management]
change_kind: bugfix
affected_spec:
  - { path: spec/modules/identity-management/main.tsp, symbol: IdMagic.IdManagement.CreateAdminUserError400Body, impact: conforms }
---

# 管理者による User の作成のパスワードポリシー違反を Problem Details で返す

## 動機

API ガイドラインは、汎用 API のエラーを RFC 9457 Problem Details で返し、独自形式を二つに限ると定める。
管理者による User の作成は、パスワードがテナントのポリシーに違反すると、400 と `{"error": "password_policy", "message": ..., "violations": [...]}` を返す。
この形式は、ガイドラインが認める独自形式のどちらにも当たらず、TypeSpec が宣言する `application/problem+json` の `InvalidRequestError` とも異なる。

IdManagement の既存コードを書き起こしたとき、この応答の形はガイドラインに反するため規則として書き起こさず、この work item で扱うことにした。

## 対象範囲

- 応答を TypeSpec の宣言どおり Problem Details にし、違反の一覧を拡張メンバーとして返す。
- 管理コンソールの読み取りを合わせる。

## 対象外

- パスワードポリシーの内容の変更。

## 計画

1. 応答の形を決め、TypeSpec を先に変える。
2. ハンドラーと管理コンソールを合わせる。

## タスク

- [ ] T001 [Spec] TypeSpec の応答を Problem Details にする。
- [ ] T002 [App] ハンドラーと管理コンソールを合わせる。
- [ ] T003 [Verify] 変更を検証する。

## 検証

- `mise run check-contract-drift`
- `mise run verify`

## リスク

- 現在の形を読む外部の利用者がいれば、応答の変更で読み取りが壊れる。リリースノートで知らせる。
