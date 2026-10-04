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

`ProvisioningTask.status` の状態遷移を表す。`pending` で作成し、ディスパッチャーが Jobs の `Job` を関連付けると `in_flight` に遷移する。Jobs 側で再試行している間は `in_flight` のまま保持し、`succeeded` または `dead_letter` で終了する。`RetryProvisioningTask` は、管理者の操作によって `dead_letter` のプロビジョニングタスクを基に新しい `pending` のプロビジョニングタスクのレコードを作る。このため、同じプロビジョニングタスクのレコードを戻す状態遷移ではなく、ユースケースの事後条件として扱う。

| State | Kind | Meaning |
|---|---|---|
| pending | initial | 作成直後。ディスパッチャーが Job を関連付けるのを待つ |
| in_flight | — | Job を関連付け、Jobs 側の再試行を含めて実行中である |
| succeeded | terminal | 下流への反映が完了した |
| dead_letter | terminal | 実行できずに終了した。管理者の再試行は新しい `pending` のレコードを作る |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| pending | ProvisioningTaskStarted | — | in_flight |  |
| in_flight | UserProvisioned | source_type == 'user' | succeeded |  |
| in_flight | UserDeprovisioned | source_type == 'user' | succeeded |  |
| in_flight | GroupPushed | source_type == 'group' | succeeded |  |
| in_flight | GroupMembershipPushed | source_type == 'group' | succeeded |  |
| in_flight | UserProvisioningFailed | — | dead_letter |  |

## 操作

### worker によるタスクの実行

#### REQ-PROVISIONING-007 下流の 409 衝突は既存リソースへの関連付けとして解決する

#### REQ-PROVISIONING-008 下流で消失したリソースは 404 の検出後に再作成する

#### REQ-PROVISIONING-009 下流の一時的な 429 と 5xx はバックオフして再試行し、復旧後に収束する

#### REQ-PROVISIONING-018 必須の属性マッピングを解決できないプロビジョニングタスクはフェイルクローズで失敗する

#### REQ-PROVISIONING-010 試行上限を超えたプロビジョニングタスクは dead_letter となり、管理者が手動で再試行できる

## セキュリティ上の考慮

`worker` が下流へ示すのは、接続に保存した資格情報だけである。
