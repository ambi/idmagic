# 多要素認証

## 概要

この文書は、第二要素を求める条件、MFA の強制と登録の専用の流れ、管理者による認証器のリセットの仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 第二要素を求めるかの判定、MFA の強制の前後の未登録のユーザーの扱い、登録の専用の流れ、管理者による認証器の全リセットと一部リセット |
| 行為者 | EndUser、テナント管理者 |
| 扱わないもの | 個々の認証の要素の登録と照合は、[TOTP](../totp/README.md)、[WebAuthn](../webauthn/README.md)、[復旧コード](../recovery/README.md)が扱う |

## モデル

`User.mfa_enrolled` は、TOTP の要素または WebAuthn の資格情報が一つ以上あることから導き、どちらかを削除するたびに計算し直す。
復旧コードは数えない。

| 時期 | 未登録のユーザー |
| --- | --- |
| MFA の強制の開始の前 | 通常のパスワードのセッションを得て、ステップアップ認証で保護されたアカウントのセキュリティ設定の画面から、認証の要素を先に登録するよう促される |
| MFA の強制の開始の後 | 管理者が発行した、未消費、未失効、期限内の `MfaEnrollmentBypass` がある場合に限り、登録の専用の流れに到達できる |

強制の開始日と猶予期間は運用の上の時刻にすぎず、誰が登録しているかを信頼する根拠にはならない。

- **判断**：登録の専用の流れを管理者の許可に限る理由は、[MFA の強制の後の登録を管理者が発行した単回限りの許可に限る](../design/decisions.md#mfa-の強制の後の登録を管理者が発行した単回限りの許可に限る)。

## 操作

### 利用者による MFA の登録

#### REQ-AUTHENTICATION-018 MFA 未登録のユーザーは管理者が承認した登録を終えて同じ認可処理を継続できる

- MFA の強制の開始の後、未消費、未失効、期限内の登録の許可（`MfaEnrollmentBypass`）を持つ未登録の User が正しいパスワードを送ったとき、Authentication は、許可を消費し、同じログインセッションを `pending_purpose=Enrollment` の未完了の状態にし、`MfaEnrollmentRequired` と `MfaEnrollmentBypassConsumed` を発行し、登録の専用の画面へ進ませる。
- 登録の途中の User が TOTP のシークレットに対する正しいコードで登録を確定したとき、Authentication は、認証の要素を保存し、同じログインセッションの `amr` に `otp` を加えて保留を解き、`MfaEnrollmentCompleted` と `UserAuthenticated` を発行し、元の認可のトランザクションを続けさせる。
- 管理者が未登録の User に登録の許可を発行したとき、Authentication は、201 と許可を返し、`MfaEnrollmentBypassIssued` を発行する。
- 管理者が登録の許可を取り消したとき、Authentication は、204 を返し、`MfaEnrollmentBypassRevoked` を発行する。
- 登録の許可が期限を過ぎたとき、Authentication は、`MfaEnrollmentBypassExpired` を発行し、その許可を使わせない。
- MFA の強制の開始の後、登録の許可がないか、取り消されたか、消費されたか、期限を過ぎた未登録の User が正しいパスワードを送った場合、Authentication は、ログインを完了させずにアクセスを拒否する。
- 登録の期限を過ぎた確定を受けた場合、Authentication は、403 と `mfa_enrollment_expired` で拒否し、認証の要素を保存しない。
- 誤った TOTP のコードで登録の確定を受けた場合、Authentication は、400 と `invalid_totp` で拒否し、認証の要素を保存しない。
- すでに TOTP を登録した User の登録を受けた場合、Authentication は、409 と `mfa_already_enrolled` で拒否する。
- 登録を始められない User の登録か許可の発行を要求された場合、Authentication は、`mfa_enrollment_not_allowed`（ブラウザーの経路では 403、管理 API では 422）で拒否する。
- すでに MFA を登録した User への許可の発行を要求された場合、Authentication は、409 と `mfa_already_enrolled` で拒否する。
- **例**：EX-AUTHENTICATION-018-01、EX-AUTHENTICATION-018-02、EX-AUTHENTICATION-018-03、EX-AUTHENTICATION-018-04

#### REQ-AUTHENTICATION-019 MFA の強制開始前は、未登録のユーザーもログインできるが登録を促される

- MFA の強制の開始の前に、未登録の User が正しいパスワードでログインしたとき、Authentication は、パスワードだけのセッションを成立させ、強制の開始の日時と事前の登録を促す警告を表示させる。
- MFA の強制の開始の前に、未登録の User がステップアップ認証を経てアカウントのセキュリティの設定を開いたとき、Authentication は、認証の要素の事前の登録を受け付ける。
- **例**：EX-AUTHENTICATION-019-01

#### REQ-AUTHENTICATION-020 登録待ちのセッションは通常のリソースへアクセスできない

- ログインセッションが登録待ちの間、アカウント、管理、Application のリソースを要求されたとき、Authentication は、未認証として拒否する。
- ログインセッションが登録待ちの間、Authentication は、登録の開始と確定の API と、元の認可のトランザクションだけを許可する。
- **例**：EX-AUTHENTICATION-020-01

### 利用者によるログインでの第二要素の判定

#### REQ-AUTHENTICATION-015 MFA 登録済みでも、ポリシーが要求しない限り第二要素は求めない

- 実効のサインインポリシーが MFA を求めない間、MFA を登録した User がユーザー名とパスワードを送ったとき、Authentication は、`authentication_pending=false` のログインセッションを作り、第二要素の画面へ進めずに同意か認可コードの発行へ進ませる。
- 対象の Application の実効のサインインポリシーが `Mfa` の場合、Authentication は、ログインセッションを `authentication_pending=true` にし、第二要素の画面へ進ませる。
- **例**：EX-AUTHENTICATION-015-01、EX-AUTHENTICATION-015-02

### 管理者による認証器のリセット

#### REQ-AUTHENTICATION-022 管理者は認証器を全リセットしたユーザーに次回ログインで再登録を強制できる

- 管理者が User の認証器のリセットを要求したとき、Authentication は、`AuthenticatorResetRequested` を発行し、指定した種類（`Totp`、`Webauthn`、`RecoveryCode`）の要素を消し、`mfa_enrolled` を計算し直し、`AuthenticatorResetCompleted` を発行する。
- リセットの結果、TOTP の要素も WebAuthn の資格情報も残らないとき、Authentication は、`reenrollment_required=true` と、自動で発行した一度だけ使える登録の許可を 200 で返し、`MfaEnrollmentBypassIssued` を発行する。
- 再登録が必要な User が次に正しいパスワードでログインしたとき、Authentication は、同じログインセッションを `pending_purpose=Enrollment` にする。
- リセットできない要求を受けた場合、Authentication は、422 と `authenticator_reset_not_allowed` で拒否する。
- 別のテナントの管理者か `admin` のロールを持たない操作者が要求した場合、Authentication は、403 と `access_denied` で拒否し、要素を消さない。
- **例**：EX-AUTHENTICATION-022-01、EX-AUTHENTICATION-022-02

#### REQ-AUTHENTICATION-023 管理者が一部の認証器のみリセットした場合は残存要素でログインを継続できる

- リセットの後に TOTP の要素か WebAuthn の資格情報が残るとき、Authentication は、`mfa_enrolled` を `true` のまま残し、`reenrollment_required=false` を返し、登録の許可を発行しない。
- 残った要素で第二要素の照合を終えたとき、Authentication は、ログインを完了させる。
- **例**：EX-AUTHENTICATION-023-01

## セキュリティ上の考慮

`pending_purpose=Enrollment` の保留中のセッションは、登録の専用の API と元の認可トランザクション以外のすべての場所で、未認証として扱う。
アカウント、管理、アプリケーションのいずれのリソースにも到達できない。
期限切れ、発行不可、失効済み、消費済みの許可は、より弱い経路へ退避せず失敗する。
