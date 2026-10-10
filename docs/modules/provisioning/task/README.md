# プロビジョニングタスクの実行

## 概要

この文書は、`worker` がプロビジョニングタスクを実行して下流へ反映する仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 実行の時点の属性による下流への反映、409 の衝突と 404 の消失の解決、一時的な失敗の再試行、配信不能、属性の対応付けの検証 |
| 行為者 | System（`worker` の Job のハンドラー）、テナント管理者（配信不能のタスクの再試行） |
| 扱わないもの | タスクを作ることは[同期](../synchronization/README.md)が扱う |

## モデル

タスクは、実行の時点の User と Group から属性を読む状態に基づく仕組みである。
下流のリソースは IdMagic を記録の正とする写し（Mirror）であり、下流での手動の変更は、次のタスクで上書きされる。

## 状態遷移

### ProvisioningTaskLifecycle

`ProvisioningTask.status` の状態遷移を表す。`pending` で作成し、ディスパッチャーが Jobs の `Job` を関連付けると `in_flight` に遷移する。Jobs 側で再試行している間は `in_flight` のまま保持し、`succeeded` または `dead_letter` で終了する。管理者が `dead_letter` のタスクを再試行すると、同じタスクを `job_id` のない `pending` に戻す。遷移の表の管理者による再試行は遷移の契機の名前であり、ドメインイベントとしては発行しない。

| State | Kind | Meaning |
|---|---|---|
| pending | initial | 作成直後または再試行の後。ディスパッチャーが Job を関連付けるのを待つ |
| in_flight | — | Job を関連付け、Jobs 側の再試行を含めて実行中である |
| succeeded | terminal | 下流への反映が完了した |
| dead_letter | — | 実行できずに終了した。管理者が再試行できる |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| pending | ProvisioningTaskStarted | — | in_flight |  |
| in_flight | UserProvisioned | `source_type == 'user'` | succeeded |  |
| in_flight | UserDeprovisioned | `source_type == 'user'` | succeeded |  |
| in_flight | GroupPushed | `source_type == 'group'` | succeeded |  |
| in_flight | GroupMembershipPushed | `source_type == 'group'` | succeeded |  |
| in_flight | UserProvisioningFailed | — | dead_letter |  |
| dead_letter | 管理者による再試行 | — | pending |  |

| State | ディスパッチ | 試行の成功 | 試行の失敗 | 管理者による再試行 |
|---|---|---|---|---|
| pending | → in_flight | 何もしない | 何もしない | 拒否：409 provisioning_conflict |
| in_flight | 何もしない | → succeeded | 何もしない（一時的な失敗で、試行が上限未満）<br>→ dead_letter（試行が上限に達したか、必須の属性を解決できない） | 拒否：409 provisioning_conflict |
| succeeded | 何もしない | 何もしない | 何もしない | 拒否：409 provisioning_conflict |
| dead_letter | 何もしない | 何もしない | 何もしない | → pending |

## 操作

### worker によるタスクの実行

#### REQ-PROVISIONING-007 下流の 409 衝突は既存リソースへの関連付けとして解決する

- `worker` が作成のタスクで下流へ POST して 409 を受けたとき、Provisioning は、`conflict_match_attribute`（デフォルト `userName`）で下流の既存のリソースを探し、`RemoteResourceLink` を作り、タスクを `succeeded` にし、`UserProvisioned` を発行する。
- `RemoteResourceLink` がある間、`worker` がタスクを実行したとき、Provisioning は、そのリンクのリソースを PATCH で更新する。
- `worker` がタスクを実行するとき、Provisioning は、実行の時点の User と Group の属性を読み、下流での手動の変更を上書きする。
- **例**：EX-PROVISIONING-007-01

#### REQ-PROVISIONING-008 下流で消失したリソースは 404 の検出後に再作成する

- `worker` が下流へ PATCH して 404 を受けたとき、Provisioning は、下流へ新たに POST し、`RemoteResourceLink.remote_id` を更新し、タスクを `succeeded` にし、`UserProvisioned` を発行する。
- **例**：EX-PROVISIONING-008-01

#### REQ-PROVISIONING-009 下流の一時的な 429 と 5xx はバックオフして再試行し、復旧後に収束する

- 下流が 429 か 5xx を返し、Job の試行が上限未満の場合、Provisioning は、Jobs の再試行を予定し、タスクを `in_flight` のまま残す。
- 下流が `Retry-After` を伴う 429 を返した場合、Provisioning は、`Retry-After` の時間まで次の試行の待ち時間を延ばす。
- 下流の復旧の後の試行が成功したとき、Provisioning は、タスクを `succeeded` にし、`UserProvisioned` を発行する。
- **例**：EX-PROVISIONING-009-01

#### REQ-PROVISIONING-018 必須の属性マッピングを解決できないプロビジョニングタスクはフェイルクローズで失敗する

- 必須の属性マッピングの値を対象から解決できない場合、Provisioning は、下流へ何も送らず、試行の上限を待たずにタスクを `dead_letter` にし、`UserProvisioningFailed` を発行する。
- **例**：EX-PROVISIONING-018-01

#### REQ-PROVISIONING-010 試行上限を超えたプロビジョニングタスクは dead_letter となり、管理者が手動で再試行できる

- 最後の試行も失敗した場合、Provisioning は、タスクを `dead_letter` にし、`UserProvisioningFailed` を発行する。
- 管理者が `dead_letter` のタスクを再試行したとき、Provisioning は、タスクを `pending` に戻し、`job_id` を消し、200 とタスクを返す。
- 管理者がタスクを一覧または取得したとき、Provisioning は、接続のタスクを `status` と `source_type` の絞り込みとカーソルで返す。
- `dead_letter` でないタスクの再試行を要求された場合、Provisioning は、409 と `provisioning_conflict` で拒否する。
- 未知の `status` か `source_type`、別の絞り込みで発行したカーソルを受けた場合、Provisioning は、400 と `invalid_request` で拒否する。
- **例**：EX-PROVISIONING-010-01、EX-PROVISIONING-010-02

## セキュリティ上の考慮

`worker` が下流へ示すのは、接続に保存した資格情報だけである。
