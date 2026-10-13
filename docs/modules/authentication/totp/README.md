# TOTP

## 概要

この文書は、TOTP の認証の要素の登録、照合、解除の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | TOTP の要素の登録と有効化、ログインでの照合、ステップアップ認証のうえでの解除 |
| 行為者 | 本人、EndUser |
| 扱わないもの | どの場面で第二要素を求めるかは[多要素認証](../mfa/README.md)が扱う |

## モデル

TOTP は RFC 6238 の標準のパラメーター（SHA1、30 秒のステップ、6 桁、前後 1 ステップの許容、160 ビットの seed）を使う。

- **判断**：TOTP の seed を含め、アプリケーションのデータベースに残る可逆なシークレットは平文で保存せず、`DataKeys` のモジュールと `EnvelopeCrypto` のポートによるエンベロープ暗号化で保存する。

## 操作

### 本人による TOTP の登録

#### REQ-AUTHENTICATION-011 ユーザーは TOTP 認証要素を登録して有効化できる

- 本人が TOTP の登録を始めたとき、Authentication は、160 ビットの seed のシークレットとアカウント名を返し、seed をエンベロープ暗号化で保存する。
- 本人がそのシークレットに対する正しいコード（SHA1、30 秒のステップ、6 桁、前後 1 ステップの許容）で登録を確認したとき、Authentication は、要素を有効にし、`mfa_enrolled` を `true` にし、`MfaFactorEnrolled` を発行する。
- 誤ったコードで登録の確認を受けた場合、Authentication は、400 と `invalid_request` で拒否し、要素を有効にしない。
- **例**：EX-AUTHENTICATION-011-01

### 利用者による TOTP の照合

#### REQ-AUTHENTICATION-017 TOTP が必須のユーザーは正しいコードで認証を継続できる

- 第二要素の照合を待つログインセッションの間、正しい TOTP のコードをブラウザーの TOTP の API に受けたとき、Authentication は、`amr` に `otp` を加えて認証を成立させ、`factorType=totp` の `MfaChallengeSucceeded` と `UserAuthenticated` を発行し、認可を続けさせる。
- 誤った TOTP のコードを受けた場合、Authentication は、401 と `invalid_totp` で拒否し、`factorType=totp` の `MfaChallengeFailed` を発行し、認証を成立させない。
- 照合を待つログインセッションがない場合、Authentication は、401 と `authentication_required` で拒否する。
- TOTP をすでに照合したセッションで要求を受けた場合、Authentication は、403 と `access_denied` で拒否する。
- 要素の保管先へ到達できない場合、Authentication は、503 と `mfa_unavailable` で拒否する。
- **例**：EX-AUTHENTICATION-017-01、EX-AUTHENTICATION-017-02

### 本人による TOTP の解除

#### REQ-AUTHENTICATION-012 ユーザーはステップアップ再認証のうえで TOTP 認証要素を解除する

- ステップアップ認証が有効な間、本人が現在の TOTP のコードで要素の解除を要求したとき、Authentication は、要素を消し、`mfa_enrolled` を計算し直し、`MfaFactorRemoved` を発行する。
- ステップアップ認証なしで解除を要求された場合、Authentication は、403 と `step_up_required` で拒否し、要素を残す。
- **例**：EX-AUTHENTICATION-012-01、EX-AUTHENTICATION-012-02

## 実現方式

### データ

`mfa_factors.secret` は以前からある平文の TOTP の seed のカラムであり、既存のレコードを読めるようにするためだけに残している（二重の読み取り）。
新しい書き込みは `secret_key_version` と `secret_ciphertext` を埋め、`secret` は `NULL` のままにする。
残りの平文のレコードは、保留中の埋め戻しで移し、その後 `secret` を削除する。

### 検証

TOTP の照合は Go のネイティブのファジングの対象である。
導いた符号が宣言した時間の窓の内側で受理されることと、桁数と文字の種類の検査が効いていることを表明する。
