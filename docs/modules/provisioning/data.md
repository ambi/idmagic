# Provisioning のデータ

この文書は、Provisioning の Aggregate を扱う。

## 集約

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `ProvisioningConnection` | 接続先の `base_url`、認証、機能のトグル、スコープ、属性の対応付け、`DeprovisionPolicy`、信頼性の設定、健全性（隔離を含む）、`notification_email` | `Application` を一つ参照する |
| `RemoteResourceLink` | IdMagic の User または Group と、下流のリソース（リモートの ID、`externalId`、`etag`）の対応、下流で有効として反映したか、最後に反映したバージョン | `ProvisioningConnection` と、User または Group を参照する |
| `ProvisioningTask` | 操作の種類（`create`、`update`、`deactivate`、`delete`）、状態、冪等のキー、関連付けた Job | `ProvisioningConnection` を参照する |
| `ScheduledDeprovision` | 猶予期間つきの削除の予約。接続、User、削除のイベントのバージョン、期限、状態 | `ProvisioningConnection` と User を参照する |
| `FullResync` | フル同期の対象の数と状態 | `ProvisioningConnection` を参照し、`ProvisioningTask` を関連付ける |

- **判断**：反映済みの状態を `RemoteResourceLink` に持つ理由は、[反映済みの状態を RemoteResourceLink に保持する](decisions.md#反映済みの状態を-remoteresourcelink-に保持する)。
