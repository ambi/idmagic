# IdManagement

テナント単位のプリンシパル台帳 — 人間の `User`、`Group`、非人間の `Agent` — と、そのプロフィール、ロール、ライフサイクル、管理 API、セルフサービス API を担う。

資格情報の検証、MFA、ログインセッションは `Authentication` が、OAuth2 クライアントの資格情報とトークン発行は `OAuth2` が担う。この Context では、それらが認証とトークン発行の対象にするプリンシパルの記録を扱う。

ライフサイクルワークフローによる自動化そのものは `IdGovernance` が担う。この Context は、そこから呼ばれる冪等なコマンドの側であり、誰がいつ変更したかの記録はここに残る。

| 文書 | 内容 |
|---|---|
| [IdManagement の用語集](glossary.md) | この Context での語義 |
| [IdManagement の状態遷移](states.md) | 複数の機能が共有するエクスポートの状態と遷移 |
| [IdManagement の設計判断](decisions.md) | 複数の機能にまたがる設計判断 |
| [IdManagement の内部設計](internals.md) | 複数の機能にまたがる機構の説明 |
| [IdManagement のシナリオ](scenarios.feature.md) | 管理 API の認可と予約ロールのように、複数の機能にまたがる受け入れシナリオ |

一つの機能だけの規則、状態遷移、設計判断、機構の説明は、次の機能ノードに置く。

| 機能ノード | 内容 |
|---|---|
| [ユーザー](user/README.md) | `User` の作成、一覧、無効化、削除、属性 |
| [アカウントのセルフサービス](account/README.md) | 本人によるプロフィール、メールアドレス、データエクスポートの操作 |
| [ユーザー CSV](user-csv/README.md) | `User` の CSV インポートとエクスポート |
| [グループ](group/README.md) | `Group` の作成と更新、手動のメンバーシップ、実効ロール |
| [動的グループ](dynamic-group/README.md) | CEL の規則によるメンバーシップの評価 |
| [グループ CSV](group-csv/README.md) | `Group` とメンバーシップの CSV インポートとエクスポート |
| [エージェント](agent/README.md) | `Agent` の登録、資格情報のバインド、ライフサイクル |
