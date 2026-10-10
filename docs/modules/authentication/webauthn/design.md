# WebAuthn の設計

この文書は、[WebAuthn](README.md)の要件を保証する仕組みを扱う。

## アーキテクチャ

CBOR と COSE の解析、署名の検証といったセレモニーの論理は自作せず、`go-webauthn/webauthn` に全面的に委ねる。
自作のアテステーションとアサーションの検証器では、わずかな誤りがそのままセキュリティの回避につながるからである。

登録と認証のチャレンジには新しいストアを設けず、既存の一時的な `SessionStore` を使う。
登録では `sub`、認証では保留中のログインセッションの ID を鍵とする。
チャレンジは、ほかのセッションのデータと同じライフサイクルを持つ、短命なサーバーの側の値だからである。

## データ

WebAuthn の資格情報は `mfa_factors` へ押し込めず、`credential_id` を鍵とする専用のテーブル `webauthn_credentials` に置く。
`mfa_factors` の `(user_id, type)` という同一性では、ユーザーごと、種類ごとに一つの要素しか持てないが、WebAuthn の価値は一つのアカウントへ複数の認証器を登録できることにあるためである。
`public_key` は COSE の公開鍵（base64url）を持つ。

`webauthn_sessions` は WebAuthn の手続きの challenge のストアであり、`GetDel` は `DELETE ... WHERE expires_at > now() RETURNING data` である。
