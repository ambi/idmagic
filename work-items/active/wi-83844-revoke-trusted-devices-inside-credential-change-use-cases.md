---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-06
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/authentication/features/trusted-device.md, requirement: REQ-AUTHENTICATION-028 }
---

# 資格情報を変えるユースケースが、記憶済みの端末の失効を担う

## 動機

[wi-10874](../done/wi-10874-inventory-promises-kept-by-every-entry-path.md) の棚卸しで、[REQ-AUTHENTICATION-028](../../docs/modules/authentication/features/trusted-device.md) の失効を、資格情報を変えるユースケースではなく HTTP ハンドラーが一つずつ呼んでいることが分かった。

| 経路 | 失効を呼ぶ箇所 |
| --- | --- |
| パスワードの変更 | `change_password_handler.go` |
| パスワードの再設定 | `password_reset_handler.go` |
| アカウントの TOTP の登録と解除 | `account_totp_enrollment_handler.go` |
| WebAuthn の登録と解除 | `account_webauthn_handler.go` |
| 管理者による認証器のリセット | `admin_reset_handler.go` |
| 全セッションの失効 | `account_sessions_handler.go` |
| サインイン中の TOTP の登録（`signin/handlers_http/enrollment.go`） | なし |

サインイン中の登録は、管理者が発行した登録の迂回で到達する。
WebAuthn を持つ User がこの経路で TOTP を登録しても、記憶済みの端末は残る。
REQ-AUTHENTICATION-028 は「認証の要素を登録」したときに失効させると定めるので、これは違反である。
同じ `ConfirmTOTPEnrollment` を呼ぶ二つの入口のうち、片方だけが失効を足していた。

## 対象範囲

- サインイン中の TOTP の登録でも、記憶済みの端末を失効させる。
- REQ-AUTHENTICATION-028 の契機となる全ユースケースが、自身の中で失効させる形へ移す。

## 対象外

- 失効の契機の追加と削除。規範の契機は変えない。

## 設計

### 所有の形

候補 A を採る。
`ChangePassword`、`ResetPasswordWithToken`、`ConfirmTOTPEnrollment`、`RemoveTOTPFactor`、`FinishWebAuthnRegistration`、`RemoveWebAuthnCredential`、管理者のリセット、全セッションの失効の各ユースケースの依存に、端末の失効のポートを必須で持たせる。
ハンドラーから `Deps.RevokeTrustedDevices` を外し、ハンドラーが失効を選べないようにする。
ポートが nil の組み立ては構築の時点で失敗させるか、テスト用の記録器を明示して渡す。

B を組み合わせ、契機の閉じた表を表駆動のテストにして、各ユースケースの後に端末が `Revoked` になることを観測する。

## タスク

- [ ] T001 [Test] サインイン中の TOTP の登録の後に端末が残ることを RED にする。
- [ ] T002 [App] 失効をユースケースへ移し、ハンドラーの呼び出しを消す。
- [ ] T003 [Test] 契機の表を表駆動のテストにする。

## 検証

- `mise run test-go-package -- ./backend/authentication/...`

## リスク

- テストの組み立てがユースケースの依存を直接作っている箇所は、失効のポートを渡す形へ直す必要がある。
