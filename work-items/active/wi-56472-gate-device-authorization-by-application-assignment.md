---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-11-06
priority: p1
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/application/features/assignment.md, requirement: REQ-APPLICATION-011 }
---

# デバイス認可の承認も、アプリケーションの割り当てとサインインポリシーの関門を通す

## 動機

[wi-10874](../done/wi-10874-inventory-promises-kept-by-every-entry-path.md) の棚卸しで、利用者のトークンを発行する入口のうち、デバイス認可だけが `EvaluateApplicationAccess` を呼ばないことが分かった。

| 入口 | 関門 |
| --- | --- |
| OIDC の認可（`authorize_completion.go`、`signin_continuation.go`） | あり |
| SAML の SSO、WS-Fed のサインイン | あり |
| WS-Trust | あり |
| ID-JAG | あり |
| デバイス認可の承認（`handleDeviceAPI` の `ApproveUserCode`）とデバイスコードの交換 | なし |

[REQ-APPLICATION-011](../../docs/modules/application/features/assignment.md) は、どのプロトコルでも同じ関門で割り当てを確かめ、無効なアプリケーションへのフェデレーションを拒否すると定める。
デバイス認可では、割り当てのない利用者も無効なアプリケーションのクライアントのトークンを得られ、アプリケーションのサインインポリシー（MFA の要求など）も評価されない。

## 対象範囲

- デバイス認可の承認で関門を評価し、拒否したら承認しない。
- 利用者のトークンを発行する承認の入口の閉じた表を作り、全入口で関門を観測する。

## 対象外

- 割り当てを解除した後のリフレッシュの扱い。規範はフェデレーションの開始を対象にする。
- CIBA の承認。人間の承認を Agent の承認として扱う別の規範に従う。

## 設計

### 所有の形

候補 A に B を組み合わせる。
`ApproveUserCode` の依存に関門のポートを必須で持たせ、ユースケースの中で評価する。
関門を必要とする承認のユースケースの一覧を表で定義し、表駆動のテストで、割り当てのない利用者と無効なアプリケーションをすべての入口で拒否することを観測する。
入口が OAuth2、SAML、WS-Fed にまたがるので、A だけでは新しいプロトコルの入口の追加を止められない。

拒否の応答は、承認の画面で既存のフェデレーションの拒否と同じ表示にする。
サインインポリシーが追加の認証を求めた場合の、デバイスの承認画面からの誘導を着手時に決める。

## タスク

- [ ] T001 [Test] 割り当てのない利用者がデバイス認可を承認できることを RED にする。
- [ ] T002 [App] 承認のユースケースで関門を評価する。
- [ ] T003 [Test] 入口の表を表駆動のテストにする。

## 検証

- `mise run test-go-package -- ./backend/oauth2/internal/device/...`
- `mise run test-go-package -- ./backend/oauth2/internal/handlers_http`

## リスク

- 既存のデバイスのクライアントの利用者で、割り当てを持たない者が承認できなくなる。
  リリースの告知で、割り当ての付与を案内する。
