# IdManagement

テナント単位のプリンシパル台帳 — 人間の `User`、`Group`、非人間の `Agent` — と、そのプロフィール、ロール、ライフサイクル、管理 API、セルフサービス API を担う。

資格情報の検証、MFA、ログインセッションは `Authentication` が、OAuth2 クライアントの資格情報とトークン発行は `OAuth2` が持つ。この Context が受け持つのは、それらが認証とトークン発行の対象にするプリンシパルの記録そのものである。

ライフサイクルワークフローによる自動化そのものは `IdGovernance` が持つ。この Context は、そこから呼ばれる冪等なコマンドの側であり、誰がいつ変更したかの記録はここに残る。

| 文書 | 内容 |
|---|---|
| [IdManagement の用語集](glossary.md) | この Context での語義 |
| [IdManagement の状態遷移](states.md) | 状態と遷移 |
| [IdManagement の設計判断](decisions.md) | 設計判断 |
| [IdManagement の内部設計](internals.md) | 機構の説明 |
| [IdManagement Scenarios](scenarios.feature.md) | 受け入れシナリオ |
