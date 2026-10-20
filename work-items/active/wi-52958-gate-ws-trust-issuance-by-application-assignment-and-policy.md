---
status: in_progress
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
primary_use_cases:
  - id: ws-trust-refuses-unassigned-user
    requirement: REQ-APPLICATION-011
    observable_result: アプリケーションに関連付けた RP へ、割り当てのない利用者が正しい UsernameToken で RST を送ると、403 で拒否され、トークンも WsTrustTokenIssued も出ない。
    boundary: acceptance
    test: { path: backend/wsfederation/internal/handlers_http/wstrust_examples_test.go, name: TestWsTrustIssue_RefusesAUserTheApplicationDoesNotAssign, task: test-go-race }
    fault_model: WS-Trust の発行が Application の割り当ての関門を呼ばない。
  - id: ws-trust-refuses-mfa-policy
    requirement: REQ-WSFEDERATION-004
    observable_result: 割り当てのある利用者でも、アプリケーションのサインインポリシーが MFA を求めると、403 で拒否され、トークンが出ない。
    boundary: acceptance
    test: { path: backend/wsfederation/internal/handlers_http/wstrust_examples_test.go, name: TestWsTrustIssue_RefusesWhenTheSignInPolicyRequiresMFA, task: test-go-race }
    fault_model: 関門に MFA で認証した事実を渡すか、ポリシーの追加認証の要求を許可として扱う。
---

# WS-Trust のトークン発行を Application の割り当てとサインインポリシーで判定する

## 動機

REQ-APPLICATION-011 は、アプリケーションに関連付けたプロトコル設定へのフェデレーションで、どのプロトコルでも同じ関門で割り当てを確かめると定める。
WS-Fed のパッシブサインインは `ApplicationGate.EvaluateApplicationAccess` を通るが、能動的な STS（`/trust/usernamemixed`）は、RP を引いてパスワードを照合すると、関門を通らずにトークンを発行する。
そのため、割り当てのない利用者も、アプリケーションに関連付けた RP のトークンを WS-Trust で得られる。
アプリケーションのサインインポリシーが MFA を求める場合も、パスワードだけで発行する。

着手時に、受け入れテストで割り当てのない利用者と MFA を求めるポリシーの両方に 200 でトークンが発行されることを確かめた。
起票時には、WS-Trust の割り当てとポリシーを検査するテストはなかった。

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

着手時の確認で、能動的な STS のハンドラーは RP を引いて資格情報を照合した後、関門を呼ばずに `WsTrustService.IssueToken` へ進むことをコードで確かめた。
TypeSpec の `WsTrustIssue` はすでに 403 を宣言しているので、TypeSpec は変えない。

拒否のイベントの発行をパッシブサインインと共有するため、`backend/wsfederation/internal/usecases/signin.go` に次の関数を置き、パッシブサインインもこれを使う。

```go
// enforceApplicationAccess は wtrealm の RP が属するアプリケーションの関門で判定する。
// 拒否したときは AppStepUpRequired と AppAccessDeniedByPolicy を発行し、理由を返す。
// 許可したときは空の理由を返す。
func enforceApplicationAccess(
    ctx context.Context, gate ApplicationGate, emit func(spec.DomainEvent),
    tenantID, wtrealm, sub string, authn *authdomain.AuthenticationContext, clientIP string, now time.Time,
) (denyReason string, err error)
```

`WsTrustService` に `Gate` と `Emit` を持たせ、`TokenRequest` に発信元の IP と認証の時刻を加える。
`IssueToken` はクレームの発行の前に、`AMR` を `pwd`、`ACR` をその導出値、認証の時刻を照合の時刻とした認証の事実で関門を呼ぶ。
拒否は `TokenDecision{RejectReason: <理由>, RejectStatus: 403}` に写し、ハンドラーの既存の拒否の経路が `WsTrustTokenRejected` と `text/plain` の 403 を返す。

仕様の漏れの観点で答えた問い：

| 観点 | 答え | 分類 |
| --- | --- | --- |
| 順序 | 関門は利用者を特定した後にしか判定できない。資格情報の誤りは従来どおり 401、関門の拒否は 403 | 要件 |
| 同じ種類の操作 | パッシブサインインと同じアプリケーションのイベントを発行する | 要件（既存のイベントの再利用） |
| 認可 | MFA を求めるポリシーと、無効なアプリケーションは拒否する | 要件 |
| 記録 | 拒否の理由を `WsTrustTokenRejected` に残す | 要件 |

## 計画

1. 割り当てのない利用者と、MFA を求めるポリシーの RP で、WS-Trust がトークンを発行する故障を RED にして確認する。
2. 仕様を書き、関門を呼んで GREEN にする。

## タスク

- [x] T001 [Acceptance] 割り当てのない利用者と MFA を求めるポリシーで、WS-Trust が発行する故障を RED にする。
- [x] T002 [Spec] REQ-WSFEDERATION-004 に関門による拒否を書く。
- [x] T003 [App] WS-Trust の発行の前に関門を呼ぶ。
- [x] T004 [Verify] 変更を検証する。

## 検証

- `mise run test-go-changed`、`mise run lint-go`、`mise run check-spec`。
- REQ-APPLICATION-011 を引くテストを WS-Trust に加える。

## リスク

Microsoft 365 型のクライアントが割り当てのない利用者で WS-Trust を使っていた場合、導入後は拒否される。
仕様どおりの結果だが、拒否の理由を `WsTrustTokenRejected` に残して調べられるようにする。

## 要約

REQ-WSFEDERATION-004 に、WS-Trust の発行がパッシブサインインと同じアプリケーションの関門で判定され、関門の拒否とサインインポリシーの追加認証の要求を 403 で拒否する二文を加えた。
TypeSpec はすでに 403 を宣言していたので変えていない。

能動的な STS は資格情報を照合した後、関門を呼ばずにトークンを発行していた。
受け入れテストで、割り当てのない利用者にも、MFA を求めるポリシーのアプリケーションにも 200 でトークンが出ることを確かめてから直した。
`WsTrustService` が、パスワードだけで認証した事実と発信元の IP で関門を呼び、拒否を 403 と `WsTrustTokenRejected` に写す。
拒否時の `AppStepUpRequired` と `AppAccessDeniedByPolicy` の発行はパッシブサインインと同じ関数に寄せた。

変更行のミューテーションは、関門を検査する受け入れテストが別のパッケージにあるため到達を数えられなかった。
代わりに、認証の事実を MFA にする、拒否を 400 にする、アプリケーションのイベントを出さない、拒否の理由を捨てて発行する、の四つを手で注入し、すべて受け入れテストが検出することを確かめた。
WS-Trust のパスワード認証の委譲は wi-88033 に残る。
