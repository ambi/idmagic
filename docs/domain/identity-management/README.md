# IdManagement

## 責務と境界

テナント単位の `User`、`Group`、`Agent` の台帳を扱う。
`User` は人間、`Group` は `User` を束ねてロールをまとめて付与する単位、`Agent` は非人間の行為者である。
この Context は、それぞれのプロフィール、ロール、ライフサイクル、管理 API、セルフサービス API、CSV による一括の取り込みと書き出しを扱う。

| 扱わないもの | 担当 |
| --- | --- |
| 資格情報の検証、多要素認証、ログインセッション | `Authentication` |
| OAuth2 クライアントの資格情報とトークンの発行 | `OAuth2` |
| ライフサイクルワークフローによる自動化 | `IdGovernance`。この Context は、そこから呼ばれる冪等なコマンドの側であり、誰がいつ変更したかの記録はこの Context に残る |
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
次の表は、それ以外にほかの Context と結ぶ契約である。

| 契約 | 相手 | 向き | 内容 |
| --- | --- | --- | --- |
| `UserMutationCommitter` | `IdGovernance` が実装する | この Context が呼ぶ | User の変更と、その変更から生まれるワークフローの実行を同時に確定する |
| `ProvisioningNotifier`（User と Group） | `Provisioning` が実装する | この Context が呼ぶ | User と Group の変更を下流の宛先へ通知する |
| `UserSourceOwnershipGuard`、`GroupSourceOwnershipGuard` | `Sourcing` が実装する | この Context が呼ぶ | 外部の権威が管理する User と Group を、CSV が上書きしないように判定する |
| `ProvisionFederatedUser` | `Authentication` が呼ぶ | この Context が提供する | フェデレーションのログインで User を作る |
| `UserRepository`、`GroupRepository` の読み取り | `Authentication`、`OAuth2`、`Authorization` などが呼ぶ | この Context が提供する | User と Group、および実効ロールを読む |
| ドメインイベント | 監査と下流が購読する | この Context が発行する | `User…`、`Group…`、`DynamicGroupRule…`、`Agent…`、`DataExport…`、`EmailChange…` |

## 機能

| 機能 | 内容 |
| --- | --- |
| [ユーザー](user/README.md) | 管理者による `User` の作成、一覧、更新、無効化、削除、属性 |
| [アカウントのセルフサービス](account/README.md) | 本人によるプロフィール、メールアドレス、データエクスポートの操作 |
| [エージェント](agent/README.md) | `Agent` の登録、資格情報のバインド、ライフサイクル |
| [グループ](group/README.md) | `Group` の作成と更新、連絡先とカスタム属性、手動のメンバーシップ、実効ロール |
| [動的グループ](dynamic-group/README.md) | CEL の規則の保存、プレビュー、全件の再評価による所属 |
| [CSV の転送](csv-transfer/README.md) | User と Group の CSV が共有する、見出し、行、セルの読み書きと転送の上限 |
| [ユーザー CSV](user-csv/README.md) | User の CSV のインポートとエクスポート |
| [グループ CSV](group-csv/README.md) | Group と、一つの Group のメンバーシップの CSV のインポートとエクスポート |
| [データエクスポート](data-export/README.md) | 非同期のエクスポートのジョブの開始、一覧、ダウンロード、取り消し、保持期限 |
| [管理 API の認可](admin-access/README.md) | 管理者のセッションと API アクセストークンによる管理 API の呼び出しを認める条件 |
| [ロール](roles/README.md) | ロールの正規化と、予約ロール `system_admin` を割り当てられる対象 |

| 文書 | 内容 |
| --- | --- |
| [IdManagement の用語集](glossary.md) | この Context での語義 |
| [IdManagement の品質要件](quality.md) | この Context に割り当てた品質要件 |
| [IdManagement の設計](design/README.md) | 話題ごとの設計と重要な判断 |
