---
status: pending
authors: [tn]
risk: medium
created_at: 2026-11-27
priority: p3
group: federation-protocols
depends_on: [wi-65625-verify-upstream-oidc-identity-providers-with-recordings]
change_kind: feature
affected_spec:
  - { path: docs/modules/authentication/standards.md, requirement: OIDC-CORE-CLIENT-SECRET-BASIC }
  - { path: docs/modules/authentication/features/federation.md, requirement: REQ-AUTHENTICATION-041 }
  - { path: spec/modules/authentication/models.tsp, symbol: IdMagic.Contract.IdentityProviderConnection }
  - { path: spec/modules/authentication/models.tsp, symbol: IdMagic.Contract.IdentityProviderConnectionRequest }
  - { path: spec/modules/authentication/models.tsp, symbol: IdMagic.Contract.IdentityProviderMetadataPreview }
---

# 上流の OIDC 接続ごとにトークンエンドポイントのクライアント認証の方式を選べるようにする

## 動機

[wi-65625](../done/wi-65625-verify-upstream-oidc-identity-providers-with-recordings.md) で、上流のトークン要求の認証を常に `client_secret_basic` にした（OIDC-CORE-CLIENT-SECRET-BASIC）。
Discovery §3 のデフォルトであり、RFC 6749 §2.3.1 で認可サーバーが必ず受け付ける方式だからである。
しかし Basic を受け付けず `client_secret_post` だけを受け付ける IdP もあり（Auth0 のアプリケーションのデフォルトの設定など）、その IdP とは連携ログインが `invalid_client` で失敗する。
仕様に厳密でない相手の問題ではあるが、利用者が接続を作れないのは望ましくないので、接続ごとに方式を選べるようにする。

## 対象範囲

- 上流の OIDC 接続に `token_endpoint_auth_method`（`client_secret_basic` または `client_secret_post`、デフォルトは `client_secret_basic`）を持たせ、作成、更新、表示の契約と管理画面に加える。
- トークン要求は接続の方式で認証する。
- Discovery の候補の取得（REQ-AUTHENTICATION-041）は、`token_endpoint_auth_methods_supported` から方式の候補も示す。省略時は Discovery §3 のデフォルトどおり `client_secret_basic` を示す。管理者が適用して保存するまで接続を変えない。
- 方式の変更を信頼の根拠の更新として扱うか（`Active` の接続を `Disabled` に戻すか）を決める。

## 対象外

- `private_key_jwt`、`client_secret_jwt`、`tls_client_auth`。必要になった相手が出てから起票する。
- ログインのたびに Discovery から方式を自動で決めること。REQ-AUTHENTICATION-041 の判断（取得処理を保存の書き手にしない）と衝突し、上流の文書の変化でログインの挙動を黙って変えるので採らない。
- `invalid_client` を受けたときに別の方式で送り直すこと。シークレットを二つの方式で送ることになり、失敗の原因も隠す。

## 設計

既存製品との比較は着手時に書く（Keycloak の Identity Provider の「Client authentication」、Okta と Entra ID の外部 IdP の設定を予定する）。
TokenVault の接続は同じ二つの値の `token_endpoint_auth_method` を持つ（`spec/modules/token-vault/models.tsp`）ので、値の名前とデフォルトを揃える。
模倣は `testing_oidctwin.Twin.AuthMethods` で受け付ける方式を絞れるので、両方の方式の成功と、合わない方式の `invalid_client` を模倣で確かめる。

## 計画

1. `spec-change` で TypeSpec、OIDC-CORE-CLIENT-SECRET-BASIC、REQ-AUTHENTICATION-041 を更新する。
2. 接続の保存と移行、トークン要求、Discovery の候補、管理画面を実装する。

## タスク

- [ ] T001 [Spec] 契約と標準の宣言を更新する。
- [ ] T002 [Adapter] 保存とトークン要求を接続の方式に従わせる。
- [ ] T003 [UI] 作成と編集のフォームと Discovery の候補の適用を加える。
- [ ] T004 [Verify] 実際の差分から選んだ検査を実行する。

## 検証

- Basic だけ、Post だけを受け付ける模倣のそれぞれに、方式を合わせた接続でログインが通り、合わない方式では `federation_failed` になる。
- Discovery の候補が `token_endpoint_auth_methods_supported` とその省略から方式を示し、保存までは接続を変えない。

## リスク

デフォルトを `client_secret_post` に戻すと、Basic だけを受け付ける IdP で再び失敗する。
デフォルトは仕様のデフォルトの `client_secret_basic` に保つ。
