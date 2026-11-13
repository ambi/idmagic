---
status: pending
authors: [tn]
risk: medium
reversibility: irreversible
created_at: 2026-11-14
change_kind: feature
priority: p3
depends_on: [wi-52890-step-up-bearer-requests-with-rfc-9470]
affected_spec:
  - { path: docs/modules/authentication/features/account-portal.md, requirement: REQ-AUTHENTICATION-039 }
  - { path: docs/modules/authentication/features/webauthn.md, requirement: REQ-AUTHENTICATION-006 }
  - { path: spec/modules/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.CompleteStepUpAuthentication }
---

# セッションのステップアップ認証の API を廃止するかを決める

## 動機

[[wi-52890-step-up-bearer-requests-with-rfc-9470]] で、ポータルのステップアップは RFC 9470 の再認可に置き換わり、ステップアップ認証の開始と完了の API、WebAuthn のチャレンジの API をポータルから呼ばなくなった。
Cookie の経路の 403 `step_up_required` もポップアップの再認可で解消するので、セッションの `step_up_at` を進める経路は、ポータル以外の利用者がいなければ使われない。
使われない認証の入口は、攻撃面と保守の対象だけを残す。

## 対象範囲

- ポータル以外にこれらの API を呼ぶ利用者がいるかを確かめる。
- いなければ、API、`step_up_at` の判定、関連する規範を廃止する。Cookie の経路の判定は `auth_time` だけにする。

## 対象外

- RFC 9470 の判定と応答。

## 計画

1. 利用者の有無を確かめ、廃止するかを決める。
2. 廃止するなら、規範を `superseded` にし、TypeSpec と実装を取り除く。

## タスク

- [ ] T001 [Decision] 利用者の有無を確かめ、採否を決める。
- [ ] T002 [Spec] 規範と TypeSpec を更新する。
- [ ] T003 [App] API と判定を取り除く。

## 検証

- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

ポータル以外の利用者が見つかれば、その利用者の再認証の手段を失う。T001 で確かめてから廃止する。
