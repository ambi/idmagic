---
status: completed
---

# 連携先の OIDC IdP への前提を、記録の再生と模倣で確かめる

上流の OIDC OP への前提を、OpenID Connect Core 1.0、Discovery 1.0、RFC 7517 の例を転記した記録の再生と、状態を持つ模倣で確かめるようにした。
記録と模倣は、呼ぶのが Authentication の連携だけなので `backend/authentication/internal/federation/testing_oidctwin` に置いた。
フォームの照合は `testing_httprecord.CompareForm` へ移して OAuth の記録と共有した。
Core の例の ID トークンは付録 A.7 の鍵で署名を検証できる（§3.1.3.3）ので、再生では検証の時刻を例の `iat` と `exp` の間に置いて実際に署名を検証する。
模倣は PKCE の S256、一度だけの認可コード、鍵の回転、`kid` の欠落、回転に遅れた JWK Set、経路ごとの 5xx と不正な本文を作れる。照合では自分で署名した `id_token` と鍵の `kid`、`n` を型だけ比べ、否定の検査で照合が空虚でないことを確かめる。

見つけた誤りを一つ直した（利用者に確認した）。

- OIDC-CORE-CLIENT-SECRET-BASIC（追加）：上流のトークン要求は常に `client_secret_post` で送っており、Basic だけを受け付ける OP とは連携ログインが `invalid_client` で失敗していた。シークレットのある接続は `client_secret_basic`（資格情報は RFC 6749 付録 B の符号化の後）で認証し、シークレットのない接続は `client_id` だけを本文に入れる。

接続ごとに `client_secret_post` を選ぶオプションと、Discovery の候補がその方式を示す入力支援は wi-63565 に起票した。
