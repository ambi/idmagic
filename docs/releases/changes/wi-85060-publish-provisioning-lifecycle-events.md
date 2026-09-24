# wi-85060-publish-provisioning-lifecycle-events

Provisioning が、宣言済みのライフサイクルイベントを状態の保存の後に発行するようになった。イベントは監査に記録される。

- 管理 API で接続を登録、資格情報を更新、隔離を解除すると、`ProvisioningConnectionRegistered`、`ProvisioningCredentialRotated`、`ProvisioningConnectionQuarantineCleared` を記録する（[`REQ-PROVISIONING-002`](../../domain/provisioning/scenarios.feature.md)、[`REQ-PROVISIONING-014`](../../domain/provisioning/scenarios.feature.md)）。
- ディスパッチャーが配信にジョブを関連付けると、配信は `in_flight` になり `ProvisioningDeliveryStarted` を記録する。これまでは関連付けの後も `pending` と表示されていた（[`REQ-PROVISIONING-017`](../../domain/provisioning/scenarios.feature.md)）。
- 下流への反映に成功した配信は `UserProvisioned`、`UserDeprovisioned`、`GroupPushed` のいずれかを、最後の試行で失敗した配信は `UserProvisioningFailed` を記録する。連続失敗で接続が隔離されたときは `ConnectionQuarantined` を一度だけ記録する（[`REQ-PROVISIONING-003`](../../domain/provisioning/scenarios.feature.md)、[`REQ-PROVISIONING-010`](../../domain/provisioning/scenarios.feature.md)）。
- Application の割り当て解除による無効化は、User が有効なままでも下流へ `active=false` を送る。これまでは `active=true` が送られ、下流のアカウントが有効なまま残っていた。この変更の後、割り当てを解除済みで下流に残っているアカウントは、次にそのアカウントへの無効化が配信されたときに無効になる（[`REQ-PROVISIONING-005`](../../domain/provisioning/scenarios.feature.md)）。
- 下流にまだ作られていない User への無効化は、下流へ何も送らずに成功する。これまでは無効化のために下流へ User を作成していた。
- 必須の属性マッピングを解決できない配信は、再試行せずに最初の試行で `dead_letter` になり、`UserProvisioningFailed` を記録する。この失敗は接続の連続失敗には数えない（[`REQ-PROVISIONING-018`](../../domain/provisioning/scenarios.feature.md)）。
- イベントの項目名を TypeSpec の宣言（`tenantId`、`applicationId` など）に合わせた。これにより、監査のテナント絞り込みで Provisioning のイベントを検索できるようになる。
