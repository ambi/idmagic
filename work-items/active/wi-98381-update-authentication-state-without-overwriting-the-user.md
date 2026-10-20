---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-21
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/identity-management/features/user.lifecycle.md, requirement: REQ-IDMANAGEMENT-010, impact: conforms }
  - { path: docs/modules/authentication/features/password.md, requirement: REQ-AUTHENTICATION-024, impact: conforms }
---

# Authentication は User の変える項目だけを更新する

## 動機

Authentication は、ログイン、パスワードの変更、TOTP の登録のたびに、IdManagement の `UserRepository.Save` で User を丸ごと保存する。
変える値は最終ログイン時刻、パスワードのハッシュと変更時刻、`update_password` の付与と解除、`mfa_enrolled` だけである。
`SaveUser` は全列を上書きし、版も確かめない。
そのため、ログインの処理が User を読んでから保存するまでの間に管理者が User を無効化すると、ログインの保存が `lifecycle` ごと `Active` を書き戻し、REQ-IDMANAGEMENT-010 の無効化が失われ得る。
同じ競合で、管理者による属性とロールの更新も失われ得る。

## 対象範囲

- `backend/authentication/internal/signin/handlers_http/login.go:recordLoginAndRequiredAction`、`internal/password/usecases/change_password.go`、`internal/mfa/usecases/account_mfa.go` の `UserRepo.Save` を、IdManagement が公開する、変える項目だけを確定する操作へ置き換える。
- 置き換えた操作が、同時に行われた User の無効化、属性とロールの更新を書き戻さないことを検査する。

## 対象外

- SCIM による User の保存（[wi-11577](wi-11577-route-scim-writes-through-identity-management-use-cases.md)）、seed の保存（[wi-24943](wi-24943-route-seed-writes-through-context-use-cases.md)）。
- 管理 API の楽観的な排他制御（[wi-605](wi-605-detect-lost-updates-with-etag-and-if-match.md)）。

## 設計

IdManagement が、User の一部の項目だけを書く操作を公開し、Authentication はそれを呼ぶ。
操作は、最終ログインの記録（期限切れによる `update_password` の付与を含む）、パスワードの置き換え（変更時刻と `update_password` の解除を含む）、`mfa_enrolled` の設定の三つを想定し、着手時にシグネチャを決める。
必須操作の付与と解除は、読んだ時点の User から計算した一覧で上書きせず、操作の中で現在の値に対して行う。

採らない案：

| 案 | 採らない理由 |
| --- | --- |
| 現在のパスワードのハッシュを Authentication の集約へ移す | User の作成と CSV の取り込みが、二つのモジュールの保存を同じトランザクションで行う形になる。競合の解消には不要で、境界は単純にならない |
| `SaveUser` に版の比較を加える | 呼び出し元すべてに再試行が要り、関係のない項目どうしの競合でも失敗する |

## 計画

1. 着手時に、Authentication からの User の保存と、各操作が読んで判定する値を列挙し、操作のシグネチャを設計へ記す。
2. ログインの保存の直前に無効化を挟むテストで、競合の故障を RED にする。
3. 操作を実装し、Authentication の呼び出しを置き換えて GREEN にする。

## タスク

- [ ] T001 [Design] 保存と判定を列挙し、操作のシグネチャを記す。
- [ ] T002 [Acceptance] ログイン、パスワードの変更、TOTP の登録が、同時の無効化を書き戻す故障を RED にする。
- [ ] T003 [App] IdManagement の操作を実装し、Authentication の保存を置き換える。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- メモリと PostgreSQL の両方の User の保存で、競合の検査を行う。PostgreSQL の検査はサンドボックスの外で実行する。
- `mise run test-go-changed`、`mise run lint-go`、`mise run check-boundaries`。

## リスク

読んだ User から判定した値（パスワードの期限切れによる必須操作の付与など）を書く処理は、項目を絞っても同じ競合を残し得る。
判定と確定を同じ操作の中で行う。
