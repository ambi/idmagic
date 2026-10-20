---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-21
priority: p1
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/ws-federation/features/active-sts.md, requirement: REQ-WSFEDERATION-004 }
  - { path: docs/modules/application/features/assignment.md, requirement: REQ-APPLICATION-011, impact: conforms }
---

# WS-Trust のトークン発行を Application の割り当てとサインインポリシーで判定する

## 動機

REQ-APPLICATION-011 は、アプリケーションに関連付けたプロトコル設定へのフェデレーションで、どのプロトコルでも同じ関門で割り当てを確かめると定める。
WS-Fed のパッシブサインインは `ApplicationGate.EvaluateApplicationAccess` を通るが、能動的な STS（`/trust/usernamemixed`）は、RP を引いてパスワードを照合すると、関門を通らずにトークンを発行する。
そのため、割り当てのない利用者も、アプリケーションに関連付けた RP のトークンを WS-Trust で得られる。
アプリケーションのサインインポリシーが MFA を求める場合も、パスワードだけで発行する。

この判断はコードの読解によるもので、テストでは未確認である。
WS-Trust の割り当てとポリシーを検査するテストはない。

## 対象範囲

- WS-Trust の発行の前に、パッシブサインインと同じ `ApplicationGate` で、認証の事実（`amr` は `pwd`）と発信元の IP を渡して判定する。
- 拒否されたときは、403 と `text/plain` で拒否し、`WsTrustTokenRejected` を発行し、トークンを発行しない。
- REQ-WSFEDERATION-004 に、この拒否を書く。

## 対象外

- WS-Trust のパスワード認証の委譲（[wi-88033](wi-88033-authenticate-ws-trust-passwords-through-authentication.md)）。
- WS-Trust での MFA の対応。MFA を求めるポリシーは拒否とする。

## 設計

関門の判定は Application が所有し、WsFederation はパッシブサインインと同じ判定を呼ぶ。
アプリケーションに関連付けていない RP では、REQ-APPLICATION-011 のとおり割り当てを求めない。
ポリシーが追加の認証を求める判定は、WS-Trust では満たせないので拒否に写す。

## 計画

1. 割り当てのない利用者と、MFA を求めるポリシーの RP で、WS-Trust がトークンを発行する故障を RED にして確認する。
2. 仕様を書き、関門を呼んで GREEN にする。

## タスク

- [ ] T001 [Acceptance] 割り当てのない利用者と MFA を求めるポリシーで、WS-Trust が発行する故障を RED にする。
- [ ] T002 [Spec] REQ-WSFEDERATION-004 に関門による拒否を書く。
- [ ] T003 [App] WS-Trust の発行の前に関門を呼ぶ。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run test-go-changed`、`mise run lint-go`、`mise run check-spec`。
- REQ-APPLICATION-011 を引くテストを WS-Trust に加える。

## リスク

Microsoft 365 型のクライアントが割り当てのない利用者で WS-Trust を使っていた場合、導入後は拒否される。
仕様どおりの結果だが、拒否の理由を `WsTrustTokenRejected` に残して調べられるようにする。
