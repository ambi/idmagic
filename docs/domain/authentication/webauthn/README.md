# WebAuthn

## 概要

この文書は、WebAuthn の資格情報による第二要素とステップアップ認証のセレモニーの仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | WebAuthn の資格情報の登録、第二要素とステップアップ認証のアサーションの検証 |
| 行為者 | 本人 |
| 扱わないもの | パスワードレス認証と Discoverable Credential の流れは範囲の外である |

## モデル

WebAuthn は、パスワードと組み合わせるフィッシング耐性の高い第二要素として扱う。
一つのアカウントへ複数の認証器を登録できる。

| 項目 | 値 |
| --- | --- |
| RP ID と許可するオリジン | デプロイの設定（`WEBAUTHN_RP_ID`、`WEBAUTHN_RP_ORIGINS`）。起動時に検証し、セレモニーごとに確かめ直す |
| アテステーション | `none`（端末の機種の強制よりプライバシーを優先する） |
| ユーザーの検証 | `preferred` |
| 常駐の鍵 | `discouraged` |
| challenge | 32 バイト、120 秒 |

返された `sign_count` が保存値以下の場合（0 から 0 を除く）は、認証器が複製された証拠とみなし、アサーションをその場で拒否する。
真正な認証器のカウンターは増える一方だからである。
第二要素の検証に成功すると、`acr` は `urn:idmagic:acr:mfa` へ上がり、`amr` に `webauthn`（RFC 8176 の登録値）が加わる。

## 操作

### 本人によるステップアップ認証のチャレンジ

#### REQ-AUTHENTICATION-006 ユーザーは WebAuthn でステップアップ認証のチャレンジを開始できる

- 本人が WebAuthn の資格情報を登録した認証済みのセッションで、正しい CSRF トークンとともにステップアップ認証のチャレンジを要求したとき、Authentication は、200 と、現在のセッションに束縛した 120 秒の challenge の `PublicKeyCredentialRequestOptions` を返す。
- 本人がそのチャレンジへのアサーションの検証に成功したとき、Authentication は、`amr` に `webauthn` を加え、`acr` を `urn:idmagic:acr:mfa` にし、`step_up_at` を進める。
- 保存値以下の `sign_count`（0 から 0 を除く）のアサーションを受けた場合、Authentication は、認証器の複製の証拠としてアサーションを拒否する。
- WebAuthn を構成していない場合、Authentication は、503 と `webauthn_unavailable` で拒否し、チャレンジを発行しない。
- CSRF トークンが一致しない場合、Authentication は、403 と `csrf_failed` で拒否し、チャレンジを発行しない。
- セッションが未認証か認証の途中の場合、Authentication は、401 と `authentication_required` で拒否する。
- **例**：EX-AUTHENTICATION-006-01、EX-AUTHENTICATION-006-02

### 本人による WebAuthn の資格情報の登録と解除

#### REQ-AUTHENTICATION-038 本人は WebAuthn の資格情報を登録し、ステップアップ認証のうえで解除できる

- 本人が認証済みのセッションで登録を始めたとき、Authentication は、200 と、アテステーション `none`、ユーザーの検証 `preferred`、常駐の鍵 `discouraged` の `PublicKeyCredentialCreationOptions` を返す。
- 本人が登録の challenge へのアテステーションの検証に成功したとき、Authentication は、204 を返し、資格情報を保存し、`mfa_enrolled` を計算し直し、本人のすべての信頼済みデバイスを失効させ、`WebAuthnCredentialRegistered` を発行する。
- 本人がステップアップ認証を経たセッションで自分の資格情報の解除を要求したとき、Authentication は、204 を返し、資格情報を消し、`mfa_enrolled` を計算し直し、本人のすべての信頼済みデバイスを失効させ、`WebAuthnCredentialRemoved` を発行する。
- 登録の challenge がないか期限を過ぎた確定を受けた場合、Authentication は、400 と `webauthn_challenge_expired` で拒否する。
- アテステーションを検証できない場合、Authentication は、400 と `invalid_webauthn` で拒否し、資格情報を保存しない。
- 本人のものでないか存在しない資格情報の解除を受けた場合、Authentication は、404 と `webauthn_not_found` で拒否する。
- ステップアップ認証を経ていないセッションで解除を受けた場合、Authentication は、403 と `step_up_required` で拒否し、資格情報を残す。
- WebAuthn を構成していない場合、Authentication は、503 と `webauthn_unavailable` で拒否する。
