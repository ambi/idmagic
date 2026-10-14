---
status: completed
archived_from: 31a913b6f668d53bbf9270bbf047473f91415398
---

# authentication コンテキストへバックエンド・コンテキストローカリティを横展開する

authentication 固有の業務型を `backend/authentication/domain` へ移し、MFA・password history・WebAuthn・recovery code・email change token・authentication event bucket の memory / postgres adapter を同 context へ同居させた。postgres 側の 24 クエリはすべて sqlc 生成へ置換した。OAuth2 / WsFederation / SAML は authentication が公開する `AuthenticationContext` を直接 import しており、共有 kernel は新設不要と判断した。`authentication.Module` に context 固有の port と実行時依存を集約し、bootstrap の通常経路と server の route 組立は Module 経由へ統一した。既存の直接 `server.Deps` 指定はテスト移行用の deprecated 互換入力として Module へ正規化する。

全文は `git show 31a913b6f668d53bbf9270bbf047473f91415398:<このファイルのパス>` で読む。
