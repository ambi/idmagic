---
status: completed
---

# 上流の OAuth 2.0 トークンエンドポイントへの前提を、記録の再生と模倣で確かめる

上流の OAuth 2.0 トークンエンドポイントへの前提を、RFC 6749 と RFC 6750 の例を転記した記録の再生と、状態を持つ模倣で確かめるようにした。
記録の形式、再生、照合は `backend/shared/testing_httprecord` へ移し、SCIM の記録と模倣はそれを使う形にした（SCIM のテストは変えず、同じ食い違いを報告する）。
トークンエンドポイントの記録と模倣は、Provisioning と TokenVault の両方が使うので `backend/shared/testing_oauthtwin` に置いた。
RFC 6749 の応答の例は `token_type` がすべて `example` なので拒否の検査に使い、受け付ける応答には RFC 6750 §4 の例を使う。

見つけた二つの誤りを直した（どちらも利用者に確認した）。

- RFC7644-OUT-AUTHENTICATION：Provisioning のクライアント資格情報フローは `token_type` を読まず、欠落や `example` でも Bearer として下流へ送っていた。大文字小文字を区別せず `Bearer` だけを受け、ほかは取得の失敗にする。
- REQ-TOKENVAULT-012、REQ-TOKENVAULT-004：TokenVault は `expires_in` を秒のまま `time.Duration` へ掛けていた。約 292 年を超える値は負に回り、0 以下は取得時点で期限切れになっていた。0 以下は期限なし、365 日を超える値は 365 日にする。コードの交換の応答も同じ規則に従う。

記録の例の値が秘密情報スキャンに検出されたので、`.betterleaks.toml` に理由付きの許可を足した。
