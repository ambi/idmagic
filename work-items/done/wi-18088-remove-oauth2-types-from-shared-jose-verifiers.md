---
status: completed
---

# 共有の JOSE 検証器が OAuth2 の型に依存しないようにする

規範仕様の差分はない。
JWKS 取得器は URI と JWKS 文書だけを受け取り、OAuth2Client からの変換を OAuth2 のクライアント認証に置いた。
リプレイ検証器は共有側の最小の ReplayStore を使い、既存の保存先が構造的に適合する。
共有の JOSE パッケージからモジュール依存がなくなり、境界台帳の OAuth2 依存を除いた。
変更行は型とシグネチャだけなので自動 mutation は0件であり、検出能力の新しい証拠には数えない。
既存の取得、リプレイ拒否、HTTP 認証のテストを維持した。

全文は `git log -1 -- work-items/active/wi-18088-remove-oauth2-types-from-shared-jose-verifiers.md` が示すコミットの親、またはそのコミットで読む。
