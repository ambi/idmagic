# Provisioning

## 責務と境界

下流の SaaS へユーザーとグループを反映する、外向きのプロビジョニングを扱う。
情報の正は IdMagic の側の User と Group であり、下流のリソースはその複製である。
接続は Application 1 件につき最大 1 件とし、プロビジョニングの対象の範囲には既存の `ApplicationAssignment` を使う。

| 扱わないもの | 担当 |
| --- | --- |
| 外部からの取り込み | `Sourcing` |
| User と Group と割り当ての記録の正 | `IdManagement`、`Application` |
| タスクを実行する Job のキューと再試行 | `Jobs` |

`Sourcing` が外部から取り込むのに対し、このモジュールは外部へ送り出す。
処理の向き、記録の正の所在、語彙が異なるので、`Tenancy`、`Application`、`IdManagement`、`Jobs` の公開するインターフェースを除いて、コードを共有しない。

## モデル

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `ProvisioningConnection` | 接続先の `base_url`、認証、機能のトグル、スコープ、属性の対応付け、`DeprovisionPolicy`、信頼性の設定、健全性（隔離を含む）、`notification_email` | `Application` を一つ参照する |
| `RemoteResourceLink` | IdMagic の User または Group と、下流のリソース（リモートの ID、`externalId`、`etag`）の対応、下流で有効として反映したか、最後に反映したバージョン | `ProvisioningConnection` と、User または Group を参照する |
| `ProvisioningTask` | 操作の種類（`create`、`update`、`deactivate`、`delete`）、状態、冪等のキー、関連付けた Job | `ProvisioningConnection` を参照する |
| `ScheduledDeprovision` | 猶予期間つきの削除の予約。接続、User、削除のイベントのバージョン、期限、状態 | `ProvisioningConnection` と User を参照する |
| `FullResync` | フル同期の対象の数と状態 | `ProvisioningConnection` を参照し、`ProvisioningTask` を関連付ける |

- **判断**：反映済みの状態を `RemoteResourceLink` に持つ理由は、[反映済みの状態を RemoteResourceLink に保持する](design/decisions.md#反映済みの状態を-remoteresourcelink-に保持する)。

## 公開する契約

HTTP の操作とモデルの形は TypeSpec の `Provisioning` のタグが、下流の SCIM の規則は[Provisioning の標準仕様](standards.md)が定める。
次の表は、それ以外にほかのモジュールと結ぶ契約である。

| 契約 | 相手 | 向き | 内容 |
| --- | --- | --- | --- |
| `ProvisioningCapture`（イベント同期） | `IdManagement` の User の変更、`Application` の割り当ての変更 | このモジュールが提供する | 変更を確定した後に呼ばれ、一致する有効な接続ごとに `pending` のタスクを作る |
| `AttributeSource`、`GroupMemberSource` | `IdManagement` の User と Group を読むアダプター | このモジュールが定める | 実行の時点の属性とメンバーを読む |
| `provisioning_task` の Job | `Jobs` が実行する | このモジュールが定める | タスクを一回試行する |
| ドメインイベント | 監査と下流が購読する | このモジュールが発行する | `ProvisioningConnection…`、`ProvisioningCredentialRotated`、`ProvisioningTaskStarted`、`UserProvisioned`、`UserDeprovisioned`、`UserProvisioningFailed`、`GroupPushed`、`GroupMembershipPushed`、`ConnectionQuarantined`、`FullResyncCompleted` |

## 機能

| 機能 | 内容 |
| --- | --- |
| [接続の管理](connection/README.md) | 接続の登録と接続テスト、On-Demand Provision、フル同期、資格情報のローテーション |
| [同期](synchronization/README.md) | イベント同期、インクリメンタル同期、猶予期間つきの削除、誤削除ガード |
| [プロビジョニングタスクの実行](task/README.md) | 下流への反映、衝突と消失の解決、再試行と配信不能 |

| 文書 | 内容 |
| --- | --- |
| [Provisioning の用語集](glossary.md) | このモジュールでの語義 |
| [Provisioning の標準仕様](standards.md) | 採用する外部標準仕様 |
| [Provisioning の設計](design/README.md) | 話題ごとの設計と重要な判断 |
