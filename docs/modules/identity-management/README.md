# IdManagement

## 責務と境界

テナント単位の `User`、`Group`、`Agent` の台帳を扱う。
`User` は人間、`Group` は `User` を束ねてロールをまとめて付与する単位、`Agent` は非人間の行為者である。
このモジュールは、それぞれのプロフィール、ロール、ライフサイクル、管理 API、セルフサービス API、CSV による一括の取り込みと書き出しを扱う。

| 扱わないもの | 担当 |
| --- | --- |
| 資格情報の検証、多要素認証、ログインセッション | `Authentication` |
| OAuth2 クライアントの資格情報とトークンの発行 | `OAuth2` |
| ライフサイクルワークフローによる自動化 | `IdGovernance`。このモジュールは、そこから呼ばれる冪等なコマンドの側であり、誰がいつ変更したかの記録はこのモジュールに残る |
| 外部の権威からの取り込み（SCIM など） | `Sourcing` |
| 外部の宛先への送り出し | `Provisioning` |

## モデル

Aggregate root は `User`、`Group`、`Agent` の三つである。

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `User` | 属性（`attributes`）、必須操作 | `Group` のメンバーになる。`Agent` の所有者になる |
| `Group` | `GroupMembership`、`DynamicGroupRule` | メンバーの `User` を ID で参照する |
| `Agent` | `AgentCredentialBinding` | 所有者の `User` または `Group` と、バインドする `OAuth2Client` を ID で参照する |

境界の内側にあるものは、その Aggregate root を経由せずに参照しない。
`User` の実効ロールは、直接付与したロールと、所属する `Group` のロールの和集合である。

### 値オブジェクト

| 値 | 使う項目 | 正規化 | 比較 | 一意性の範囲 |
| --- | --- | --- | --- | --- |
| 名前 | `User` の `preferred_username`、`Group` の `name`、`Agent` の `name` | 前後の空白を除き、残りを入力された表記のまま保存し、表示する | Unicode の case folding をした値どうしで比較し、大文字と小文字を区別しない | 同じテナントの、削除されていない同じ種類の Aggregate |
| メールアドレス | `User` の `email` | 前後の空白を除き、残りを表記のまま保存する。空になる値は設定しない | 名前と同じく、case folding をした値どうしで比較する | 同じテナントの削除されていない `User` |

- **判断**：名前を大文字と小文字を区別せずに比較するのは、主要な IdP（Okta、Entra ID、Google Workspace、AWS IAM）と SCIM の `userName`（`caseExact: false`）に合わせるためである。表記を保つのは、Okta と Entra ID と同じく、利用者が入力した表記を表示するためである。
- **判断**：比較には小文字への変換ではなく case folding（Unicode のデフォルトの caseless matching）を使う。小文字への変換では、`ß` と `SS`、語末の `ς` と `Σ` のように、大文字と小文字だけが異なる表記が一致しない。
- **判断**：メールアドレスをテナントの中で一意にするのは、Google Workspace、Keycloak、Auth0 と同じく、メールアドレスで User を一人に決められるようにするためである。

## 公開する契約

HTTP の操作とモデルの形は TypeSpec の `Identity Management` のタグが定める。
次の表は、それ以外にほかのモジュールと結ぶ契約である。

| 契約 | 相手 | 向き | 内容 |
| --- | --- | --- | --- |
| `UserMutationCommitter` | `IdGovernance` が実装する | このモジュールが呼ぶ | User の変更と、その変更から生まれるワークフローの実行を同時に確定する |
| `ProvisioningNotifier`（User と Group） | `Provisioning` が実装する | このモジュールが呼ぶ | User と Group の変更を下流の宛先へ通知する |
| `UserSourceOwnershipGuard`、`GroupSourceOwnershipGuard` | `Sourcing` が実装する | このモジュールが呼ぶ | 外部の権威が管理する User と Group を、CSV が上書きしないように判定する |
| `ProvisionFederatedUser` | `Authentication` が呼ぶ | このモジュールが提供する | フェデレーションのログインで User を作る |
| `UserRepository`、`GroupRepository` の読み取り | `Authentication`、`OAuth2`、`Authorization` などが呼ぶ | このモジュールが提供する | User と Group、および実効ロールを読む |
| `UserWriter` | `Authentication`（パスワードの再設定）、`IdGovernance`（ワークフローの実行の記録）が呼ぶ | このモジュールが提供する | 呼び出し側のトランザクションに参加して User を保存する。確定と巻き戻しは呼び出し側が決め、組み立て地点がトランザクションから作る構築関数を渡す |
| ドメインイベント | 監査と下流が購読する | このモジュールが発行する | `User…`、`Group…`、`DynamicGroupRule…`、`Agent…`、`DataExport…`、`EmailChange…` |

## 機能

| 機能 | 内容 |
| --- | --- |
| [ユーザー](features/user.md) | 管理者による `User` の作成、一覧、更新、無効化、削除、属性 |
| [アカウントのセルフサービス](features/account.md) | 本人によるプロフィール、メールアドレス、データエクスポートの操作 |
| [エージェント](features/agent.md) | `Agent` の登録、資格情報のバインド、ライフサイクル |
| [グループ](features/group.md) | `Group` の作成と更新、連絡先とカスタム属性、手動のメンバーシップ、実効ロール |
| [動的グループ](features/dynamic-group.md) | CEL の規則の保存、プレビュー、全件の再評価による所属 |
| [CSV の転送](features/csv-transfer.md) | User と Group の CSV が共有する、見出し、行、セルの読み書きと転送の上限 |
| [ユーザー CSV](features/user-csv.md) | User の CSV のインポートとエクスポート |
| [グループ CSV](features/group-csv.md) | Group と、一つの Group のメンバーシップの CSV のインポートとエクスポート |
| [データエクスポート](features/data-export.md) | 非同期のエクスポートのジョブの開始、一覧、ダウンロード、取り消し、保持期限 |
| [管理 API の認可](features/admin-access.md) | 管理者のセッションと API アクセストークンによる管理 API の呼び出しを認める条件 |
| [ロール](features/roles.md) | ロールの正規化と、予約ロール `system_admin` を割り当てられる対象 |

## 文書

| 設計領域 | 区分 | 参照先 | 内容 |
| --- | --- | --- | --- |
| アーキテクチャ設計 | 固有設計 | [担当文書](architecture.md#文脈と範囲) | 文脈と範囲、解決戦略、構成要素、実行時の流れ |
| データ設計 | 固有設計 | [担当文書](data.md#aggregate-と永続化) | Aggregate と永続化、保持と削除、排他と一意性、テナントの属性スキーマ |
| セキュリティ設計 | 委譲 | [管理 API の認可](features/admin-access.md#概要)、[認可設計](../../design/security/authorization.md#主体の種類) | 管理 API の認可は機能 管理 API の認可 が、システム全体の認可は認可設計が扱う |
| 信頼性設計 | 固有設計 | [担当文書](reliability.md#idmanagement-の信頼性) | 障害の症状、原因、直し方 |
| 性能設計 | 固有設計 | [担当文書](performance.md#実効ポリシーを超えるエクスポート) | 実効ポリシーを超えるエクスポート |
| オブザーバビリティ設計 | 固有設計 | [担当文書](observability.md#経路) | ドメインイベント、監査の記録、下流への通知の経路と、確定との関係 |
| 検証設計 | 委譲 | [性能](performance.md#実効ポリシーを超えるエクスポート) | 品質要件の確かめ方は性能に置き、それ以外はシステムの検証設計に従う |

| 文書 | 内容 |
| --- | --- |
| [IdManagement の用語集](glossary.md) | このモジュールでの語義 |
| [IdManagement の品質要件](quality.md) | このモジュールに割り当てた品質要件 |
| [設計判断](decisions.md) | 代替案を比べた重要な判断 |
| [リスク](risks.md) | 既知の欠陥、技術的負債、受容したリスク |
