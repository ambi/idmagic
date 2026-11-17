# 本人のセキュリティ設定

この文書は、本人のセキュリティ設定の操作、入力項目と状態を示す。

> 表示確認用の途中プレビュー。全ページの棚卸しと検証は継続中である。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Authentication](../README.md) / [ページ一覧](README.md) / 本人のセキュリティ設定

分類：[アカウントポータル](../../../design/application/pages/account/README.md) / [セキュリティ](../../../design/application/pages/account/security/README.md)

`account-security`　`/account/security`

認証手段、回復コード、記憶済みデバイスと通知を管理する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [Authentication](../README.md) |
| 利用者 | 認証済みの本人 |
| 入口 | アカウントポータル |
| 利用条件 | 対象テナントの本人のセッションで参照する。CSRF と再認証は共通規則に従う。 |
| 表示内容 | TOTP の登録状態と登録用 QR、パスキー一覧、回復コードの残数と発行直後のコード、記憶済みデバイス、通知設定、リンク済み外部 ID |

利用手順は[Authentication のページ一覧](README.md)にまとめる。

## フォームと入力項目

項目ごとのバリデーション規則は別の作業で定義する。

### TOTP 登録

識別子：account-security/enrollTotp。表示条件：未登録で登録を開始した場合。確定：登録を完了。

| 項目 | 識別子 | 入力の種類 | 型 | 初期値 | 表示条件 | データとの対応 |
| --- | --- | --- | --- | --- | --- | --- |
| 設定キー | account-security/enrollTotp/secret | 読み取り専用 text | string | StartTotpEnrollment の secret | 登録開始後。QR を読み取れない場合に展開する | 確認要求の secret。利用者は変更しない |
| 確認コード | account-security/enrollTotp/code | text | string | 空 | 対象のフォームを操作するとき | 確認要求の code |

### TOTP 解除

識別子：account-security/removeTotp。表示条件：登録済み。確定：解除。

| 項目 | 識別子 | 入力の種類 | 型 | 初期値 | 表示条件 | データとの対応 |
| --- | --- | --- | --- | --- | --- | --- |
| 解除用コード | account-security/removeTotp/code | text | string | 空 | 対象のフォームを操作するとき | 解除要求の code |

### パスキー登録

識別子：account-security/passkey。表示条件：WebAuthn を利用できるブラウザー。確定：登録。

| 項目 | 識別子 | 入力の種類 | 型 | 初期値 | 表示条件 | データとの対応 |
| --- | --- | --- | --- | --- | --- | --- |
| パスキー名 | account-security/passkey/label | text | string | 空 | 対象のフォームを操作するとき | 登録要求の label。資格情報はブラウザーの認証器から取得する |

### 通知設定

識別子：account-security/notifications。表示条件：通知の種別が取得できた場合。確定：切り替え時に更新。

| 項目 | 識別子 | 入力の種類 | 型 | 初期値 | 表示条件 | データとの対応 |
| --- | --- | --- | --- | --- | --- | --- |
| 通知を受け取る（繰り返し） | account-security/notifications/enabled | checkbox | boolean | カテゴリごとの enabled | mandatory の種別は編集不可 | 無効な任意カテゴリを disabled_categories として送信する |

## パラメーター

ページ固有のパラメーターはない。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| startTotp | ready | TOTP の登録を開始 | 未登録 | IdMagic.Authentication.Operations.StartTotpEnrollment | 設定キーと QR を表示する | [本人のセキュリティ設定](account-security.md) / ready | 対象の識別子と操作の入力値 |
| confirmTotp | ready | TOTP の登録を完了 | コードの確認成功 | IdMagic.Authentication.Operations.ConfirmTotpEnrollment | 登録済みの表示へ切り替える | [本人のセキュリティ設定](account-security.md) / ready | 対象の識別子と操作の入力値 |
| cancelTotp | ready | 登録をキャンセル | 登録中 | UI 内の操作 | 未確定の設定キーを破棄する | [本人のセキュリティ設定](account-security.md) / ready | 対象の識別子と操作の入力値 |
| removeTotp | ready | TOTP を解除 | 再認証と解除が成功 | IdMagic.Authentication.Operations.RemoveTotpFactor | 登録状態を更新する | [本人のセキュリティ設定](account-security.md) / ready | 対象の識別子と操作の入力値 |
| registerPasskey | ready | パスキー登録 | ブラウザーの認証器で登録が成功 | IdMagic.Authentication.Operations.StartWebAuthnRegistration、IdMagic.Authentication.Operations.FinishWebAuthnRegistration | 再読込して一覧を更新する | [本人のセキュリティ設定](account-security.md) / ready | 対象の識別子と操作の入力値 |
| removePasskey | ready | パスキー削除 | 再認証と削除が成功 | IdMagic.Authentication.Operations.RemoveWebAuthnCredential | 対象の資格情報を一覧から除く | [本人のセキュリティ設定](account-security.md) / ready | 対象の識別子と操作の入力値 |
| recovery | ready | 回復コードを発行・再発行 | 再認証が成功 | IdMagic.Authentication.Operations.GenerateRecoveryCodes | 新しいコードを一度だけ表示し、旧コードを置き換える | [本人のセキュリティ設定](account-security.md) / ready | 対象の識別子と操作の入力値 |
| revokeRecovery | ready | 回復コードをすべて失効 | 再認証が成功 | IdMagic.Authentication.Operations.RevokeRecoveryCodes | 表示中のコードを消し、残数を更新する | [本人のセキュリティ設定](account-security.md) / ready | 対象の識別子と操作の入力値 |
| devices | ready | デバイスの記憶を解除・すべて解除 | 再認証が成功 | IdMagic.Authentication.Operations.RevokeMyTrustedDevice、IdMagic.Authentication.Operations.RevokeMyTrustedDevices | 対象を記憶済みデバイス一覧から除く | [本人のセキュリティ設定](account-security.md) / ready | 対象の識別子と操作の入力値 |
| notifications | ready | 通知設定を切り替える | 任意の種別で再認証が成功 | IdMagic.Authentication.Operations.UpdateMyNotificationPreferences | サーバーが返した通知設定を表示する | [本人のセキュリティ設定](account-security.md) / ready | 対象の識別子と操作の入力値 |
| failure | ready | 操作 | キャンセル、認証器の拒否、再認証の中止または API の拒否 | UI 内の操作 | 更新を確定せず、失敗理由を表示する | [本人のセキュリティ設定](account-security.md) / error | 対象の識別子と操作の入力値 |

## 表示状態

| 状態 | 表示 |
| --- | --- |
| ready | 取得した内容を表示する。 |
| loading | ページ移動中は直前の表示を保つ。初回の取得はルーターが扱う。 |
| error | 表示できなければ共通エラーを表示し、操作失敗は操作の近くに残す。 |
| forbidden | 権限不足を表示し、未反映の状態を保つ。 |
| expired | 認証が必要なページでは再認可し、復帰できなければログインへ移る。 |
| success | 成功した操作を一度通知し、表示内容を再取得する。 |
| invalid | 入力不備の表示は共通フォーム規則に従う。項目ごとの規則は別に定義する。 |
| submitting | 実行中の操作を示し、二重送信を防ぐ。 |

## 関連する設計

- [TypeSpec の定義](../../../../spec/modules/authentication/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[Authenticationのページ一覧へ戻る](README.md)
