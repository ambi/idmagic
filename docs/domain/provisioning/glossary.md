# Provisioning の用語集

| 用語 | 定義 | 別名 |
|---|---|---|
| ProvisioningConnection | Application 1 件に対して最大 1 件だけ存在する外向きプロビジョニングの設定。接続先 `base_url`、認証、機能トグル、スコープ、属性の対応付け、プロビジョニング解除ポリシー、信頼性設定をまとめる。 | connection, 接続 |
| RemoteResourceLink | IdMagic の `User` または `Group` と、下流の SCIM サービスプロバイダー上のリソース（リモート ID、`externalId`、`etag`）との対応を保持するエンティティ。HTTP 409（既存リソースとの衝突）では照合属性を使って既存リソースへ関連付け、HTTP 404（リソースの消失）では再作成して関連付けを更新する。 | remote link, 相関 |
| ProvisioningTask | 内部のライフサイクルイベント 1 件を下流へ反映するために作る、実行待ちの仕事の単位。操作の種類（`create`、`update`、`deactivate`、`delete`）とステータスを保持する。冪等キー（`tenant_id`、`connection_id`、`source_type`、`source_id`、`source_version`）により、重複する投入を no-op とする。実行は Jobs の `Job` に委譲する。 | task, プロビジョニングタスク |
| Deprovision | 割り当て解除、無効化、削除という内部のライフサイクルイベントを、下流に対する無効化、削除、無操作のいずれかへ変換すること。変換規則は `DeprovisionPolicy` で定める。 |  |
| Grace Period | 削除操作を直ちに実行せず、`DeprovisionPolicy.grace_period_days` の経過後に完全削除するまでの猶予期間。期間内に対象が適用範囲へ戻れば取り消す。 | grace period, 猶予期間 |
| Accidental Deletion Guard | 1 回の同期で無効化または削除の対象が `accidental_deletion_count_threshold` を超えた場合に、実行せず接続を Quarantine へ移す誤削除防止策。 | 誤削除ガード |
| Quarantine | 連続失敗または誤削除ガードの超過によってプロビジョニングを停止した `ProvisioningConnection.health` の状態。管理者が `ResumeProvisioningConnection` で解除するまで再開しない。 | quarantine, 隔離 |
| On-Demand Provision | 管理者が対象を 1 件だけ指定し、即時に試験的にプロビジョニングする手動運用。 |  |
| Full Resync | 管理者が接続の適用範囲に含まれるすべての対象を再走査し、下流の状態を IdMagic の状態へ収束させる手動運用。 |  |
| Mirror | 下流の SaaS 上のリソースが IdMagic を記録の正とする写しであり、下流での手動変更は次回のプロビジョニングタスクで上書きされること。 |  |
| Push Groups | `ProvisioningFeatureFlags.push_groups` が有効なとき、Group とメンバーシップを下流へプロビジョニングする機能。 |  |
| System | プロビジョニングタスクを作り、実行する Provisioning の仕組みそのもの。人間の操作者を伴わない技術的主体を指す。 |  |
