# IdManagement

テナント単位のプリンシパル台帳 — 人間の `User`、`Group`、非人間の `Agent` — と、そのプロフィール、ロール、ライフサイクル、管理 API、セルフサービス API を所有する。

資格情報の検証、MFA、ログインセッションは `Authentication` が、OAuth2 クライアントの資格情報とトークン発行は `OAuth2` が持つ。この Context が所有するのは、それらが認証とトークン発行の対象にするプリンシパルの記録そのものである。

`User`、`Group`、`Agent` は `user/`、`group/`、`agent/` に置く別々の機能単位であり、それぞれが自身のドメイン、ポート、ユースケース、アダプターを持つ。

| File | Content |
|---|---|
| [glossary.md](glossary.md) | この Context での語義 |
| [states.md](states.md) | 状態と遷移 |
| [decisions.md](decisions.md) | 設計判断 |
| [internals.md](internals.md) | 機構の説明 |
| [scenarios.md](scenarios.md) | 受け入れシナリオ |
