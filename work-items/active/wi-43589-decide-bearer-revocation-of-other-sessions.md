---
status: pending
authors: [tn]
risk: medium
created_at: 2026-11-14
change_kind: bugfix
priority: p1
group: spec-conformance
depends_on: []
affected_spec:
  - { path: docs/modules/authentication/features/session.md, requirement: REQ-AUTHENTICATION-013 }
---

# Bearer で受けた「他のセッションの一括の失効」の対象を決める

## 動機

`POST /api/account/v1/sessions/revoke-others` は、残すセッションを要求のログインセッションの ID で決める。
アクセストークンはセッションを持たないので、Bearer で受けると残すセッションの ID が空になり、その利用者のすべてのセッションを失効させると読める（2026-11-14 のコード読解）。
ポータルはこの操作をセッションの Cookie で呼ぶので影響しないが、アカウントのスコープを持つ OAuth のトークンや管理発行の API アクセストークンからは、この挙動になる。
[[wi-52890-step-up-bearer-requests-with-rfc-9470]] の着手中に見つけた。

## 対象範囲

- Bearer で受けたときの意味を決める。候補は、すべてのセッションの失効として明文化する、Bearer では拒否する、の二つである。
- 決めた意味を REQ-AUTHENTICATION-013 に書き、HTTP 境界のテストで固定する。

## 対象外

- ポータルの呼び方。ポータルは Cookie で呼び続ける。

## 計画

1. 管理発行の API アクセストークンの `account:sessions:write` が想定する使い方を確かめ、意味を決める。
2. 規範とテストを更新する。

## タスク

- [ ] T001 [Decision] Bearer で受けたときの意味を決める。
- [ ] T002 [Spec] REQ-AUTHENTICATION-013 を更新する。
- [ ] T003 [Acceptance] HTTP 境界で、決めた意味を観測する。

## 検証

- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

意味を変えると、API アクセストークンでセッションを一括で失効させている利用者の結果が変わる。
