# IdManagement

## 責務と境界

テナント単位のプリンシパルの台帳を扱う。
台帳に載るのは、人間の `User`、`User` を束ねる `Group`、非人間の `Agent` であり、そのプロフィール、ロール、ライフサイクル、管理 API、セルフサービス API を扱う。

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

## 公開する契約

HTTP の操作とモデルの形は TypeSpec の `Identity Management` のタグが定める。
次の表は、それ以外にほかの Context と結ぶ契約である。

| 契約 | 相手 | 向き | 内容 |
| --- | --- | --- | --- |
| `UserMutationCommitter` | `IdGovernance` が実装する | この Context が呼ぶ | User の変更と、その変更から生まれるワークフローの実行を同時に確定する |
| `ProvisioningNotifier`（User と Group） | `Provisioning` が実装する | この Context が呼ぶ | User と Group の変更を下流の宛先へ通知する |
| `UserSourceOwnershipGuard`、`GroupSourceOwnershipGuard` | `Sourcing` が実装する | この Context が呼ぶ | 外部の権威が管理する User と Group を、CSV が上書きしないように判定する |
| `ProvisionFederatedUser` | `Authentication` が呼ぶ | この Context が提供する | フェデレーションのログインで User を作る |
| `UserRepository`、`GroupRepository` の読み取り | `Authentication`、`OAuth2`、`Authorization` などが呼ぶ | この Context が提供する | プリンシパルと実効ロールを読む |
| ドメインイベント | 監査と下流が購読する | この Context が発行する | `User…`、`Group…`、`DynamicGroupRule…`、`Agent…`、`DataExport…`、`EmailChange…` |

## 機能

| 機能群 | 機能 |
| --- | --- |
| [プリンシパル](principals/README.md) | [ユーザー](principals/user/README.md)、[アカウントのセルフサービス](principals/account/README.md)、[エージェント](principals/agent/README.md) |
| [グループ](groups/README.md) | [グループ](groups/group/README.md)、[動的グループ](groups/dynamic-group/README.md) |
| [一括転送](bulk-transfer/README.md) | [CSV の転送](bulk-transfer/csv-transfer/README.md)、[ユーザー CSV](bulk-transfer/user-csv/README.md)、[グループ CSV](bulk-transfer/group-csv/README.md)、[データエクスポート](bulk-transfer/data-export/README.md) |
| [共通](common/README.md) | [管理 API の認可](common/admin-access/README.md)、[ロール](common/roles/README.md) |

| 文書 | 内容 |
| --- | --- |
| [IdManagement の用語集](glossary.md) | この Context での語義 |
| [IdManagement の品質要件](quality.md) | この Context に割り当てた品質要件 |
| [IdManagement の設計](design/README.md) | 話題ごとの設計と重要な判断 |
