---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-06
priority: p1
group: spec-conformance
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/authentication/features/trusted-device.md, requirement: REQ-AUTHENTICATION-028 }
---

# 記憶済みの端末の失効を、契機となるすべてのユースケースが自身の中で担う

## 動機

[wi-10874](../done/wi-10874-inventory-promises-kept-by-every-entry-path.md) の棚卸しで、[REQ-AUTHENTICATION-028](../../docs/modules/authentication/features/trusted-device.md) の失効に二つの抜けが見つかった。

一つ目は、失効を資格情報を変えるユースケースではなく HTTP ハンドラーが一つずつ呼んでいることである。

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

二つ目は、User を止める二つの操作のうち、無効化（`SetUserDisabled`）だけが端末を失効させ、削除の予約（`SoftDeleteUser`）は失効させないことである（2026-11-14 にも確認）。
[SCIM の設計](../../docs/modules/sourcing/features/scim.md)と [IdManagement の設計](../../docs/modules/identity-management/README.md)は、削除の予約にも端末の失効が伴うと書くが、REQ-AUTHENTICATION-028 は契機に無効化だけを挙げる。
削除を予約した User は `RestoreUser` で `Active` に戻り、予約の前の記憶済みの端末が第二要素を省略させる。
無効化で端末を失効させる理由（wi-91：再有効化で古い信頼を生き返らせない）は、予約からの復元にも当たる。

この二つは同じ要件の契機の表を扱うので、一つの作業で表と担い手を揃える。
片方だけを直すと、もう片方が契機の表を変える前提を崩す。

## 対象範囲

- REQ-AUTHENTICATION-028 の契機に、User の削除の予約を加える。管理 API、SCIM、Directory Connector の取り込みのどの経路で予約しても、その User の `Active` の端末を `Revoked` にし、端末ごとに `TrustedDeviceRevoked` を発行する。
- サインイン中の TOTP の登録でも端末を失効させる。
- 契機となる Authentication のユースケース（パスワードの変更と再設定、TOTP と WebAuthn の登録と解除、管理者のリセット、全セッションの失効）が、失効のポートを必須の依存として持ち、自身の中で失効させる。ハンドラーから失効の呼び出しを外す。
- IdManagement の User を止める共通の手順を一つ置き、端末の失効と所有する Agent の無効化をそこへ集める。`SetUserDisabled` と `SoftDeleteUser` はその手順を呼ぶ。
- 契機の閉じた表を表駆動のテストにし、各ユースケースの後に端末が `Revoked` になることを観測する。

## 対象外

- 既に予約中の User に残った端末の失効。
- 完全削除の匿名化。既に `DeleteAllForSub` が物理削除する。

## 設計

失効の理由は `TrustedDeviceRevokeReason` に削除の予約を表す値を足すか、`account_disabled` を流用するかを着手時に決める。
値を足す場合は `spec/` の列挙と監査の表示を同時に更新する。
ポートが nil の組み立ては構築の時点で失敗させるか、テスト用の記録器を明示して渡す。

## 検証

- `mise run test-go-package -- ./backend/authentication/...`
- `mise run test-go-package -- ./backend/idmanagement/internal/user/usecases`
- `mise run check-spec`

## リスク

- テストの組み立てがユースケースの依存を直接作っている箇所は、失効のポートを渡す形へ直す必要がある。
- 削除の予約の取り消しを想定した運用で、利用者が端末の記憶をやり直す必要が生じる。予約からの復元は稀な操作であり、失効を優先する。
