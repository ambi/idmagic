# 同期

## 概要

この文書は、IdMagic の User と割り当ての変化を、下流へ反映するタスクに変える仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | イベント同期、インクリメンタル同期、ディスパッチャーによる回収、猶予期間つきの削除の予約、誤削除ガードと隔離、隔離の通知 |
| 行為者 | System（変更を確定したモジュールのユースケースと、`worker` の周期の処理） |
| 扱わないもの | タスクの実行は[プロビジョニングタスクの実行](../task/README.md)が、管理者が起動するフル同期は[接続の管理](../connection/README.md)が扱う |

## モデル

| 同期 | 契機 | 役割 |
| --- | --- | --- |
| イベント同期 | User や割り当ての変更の確定の直後 | 反映の遅延を短くする近道。失敗しても、発火元の変更は確定したまま残る |
| インクリメンタル同期 | `worker` の周期ごと | 接続ごとに、User のあるべき状態と `RemoteResourceLink` が記録する反映済みの状態を突き合わせ、差分をタスクにする |

タスクの冪等のキーは `(tenant, connection, source_type, source_id, source_version)` である。

`DeprovisionPolicy` は、割り当ての解除、無効化、削除を、下流に対する無効化、削除、無操作のいずれかへ変換する。
`grace_period_days` が 1 以上の接続では、削除をすぐに送らず、猶予期間の後に送る予約にする。

- **判断**：反映を書き込みの時点の原子性ではなく同期で保証する理由は、[下流への反映をインクリメンタル同期で保証する](../design/decisions.md#下流への反映をインクリメンタル同期で保証する)。

## 操作

### User と割り当ての変更によるイベント同期

#### REQ-PROVISIONING-003 割り当て済みユーザーの作成は下流への作成としてプロビジョニングされる

- 有効な接続（`create_users=true`）の適用範囲の中の User を作成または割り当てたとき、Provisioning は、変更の確定の直後に `operation=create` の `pending` のプロビジョニングタスクを作る。
- ディスパッチャーがタスクに Job を関連付けたとき、Provisioning は、タスクを `in_flight` にし、`ProvisioningTaskStarted` を発行する。
- `worker` が作成のタスクで下流へ POST して成功したとき、Provisioning は、タスクを `succeeded` にし、`UserProvisioned` を発行する。
- `scope=assigned_only` の接続で、User がそのアプリケーションに割り当てられていない場合、Provisioning は、タスクを作らない。
- イベント同期に失敗した場合、Provisioning は、発火元の変更を確定したまま残し、インクリメンタル同期に差分を回収させる。
- **例**：EX-PROVISIONING-003-01、EX-PROVISIONING-003-02

#### REQ-PROVISIONING-004 ユーザーの無効化は下流への無効化としてプロビジョニングされる

- 下流にリンクのある User を無効化したとき、Provisioning は、`DeprovisionPolicy` の無効化の扱いに従って `operation=deactivate` のタスクを作る。
- `worker` が無効化のタスクを実行したとき、Provisioning は、下流へ `active=false` の PATCH を送り、`UserDeprovisioned` を発行する。
- **例**：EX-PROVISIONING-004-01

#### REQ-PROVISIONING-005 Application からの割り当て解除はデフォルトで下流の無効化としてプロビジョニングされる

- 下流にリンクのある User の割り当てを解除したとき、Provisioning は、`DeprovisionPolicy` の割り当ての解除の扱い（デフォルトは無効化）に従って `operation=deactivate` のタスクを作る。
- `DeprovisionPolicy` が無操作の間、割り当ての解除、無効化、削除があったとき、Provisioning は、タスクを作らない。
- **例**：EX-PROVISIONING-005-01

#### REQ-PROVISIONING-006 ユーザーの削除は猶予期間の経過後に下流への削除としてプロビジョニングされる

- `DeprovisionPolicy.on_delete=delete` で `grace_period_days` が 1 以上の接続では、User を削除したとき、Provisioning は、直ちには下流へ削除を送らず、猶予期間の後に `operation=delete` のタスクを作る予約をする。
- `worker` が削除のタスクを実行したとき、Provisioning は、下流へ DELETE を送り、`action=delete` の `UserDeprovisioned` を発行する。
- 猶予期間の間に User が同じアプリケーションへ再び割り当てられたとき、Provisioning は、削除の予約を取り消す。
- 猶予期間の間に User が再び有効になったとき、Provisioning は、すべての接続でその User の削除の予約を取り消す。
- **例**：EX-PROVISIONING-006-01、EX-PROVISIONING-006-02、EX-PROVISIONING-006-03

#### REQ-PROVISIONING-016 同じ冪等キーの重複したプロビジョニングタスクは既存のレコードに収束する

- `(tenant, connection, source_type, source_id, source_version)` が一致するタスクがある間、同じ変更を再び受けたとき、Provisioning は、新しいタスクを作らず、既存のレコードを使う。
- **例**：EX-PROVISIONING-016-01

#### REQ-PROVISIONING-017 イベント同期直後のキュー投入に失敗しても、定期ディスパッチャーが未関連付けのプロビジョニングタスクを回収する

- イベント同期の直後の Jobs への投入に失敗した場合、Provisioning は、タスクを `job_id` のない `pending` のまま残す。
- `worker` の定期のディスパッチャーが `job_id` のない `pending` のタスクを走査したとき、Provisioning は、冪等のキーを `dedup_key` にしてジョブを投入し、`job_id` を関連付け、`ProvisioningTaskStarted` を発行する。
- **例**：EX-PROVISIONING-017-01

### worker によるインクリメンタル同期

#### REQ-PROVISIONING-019 インクリメンタル同期は、あるべき状態と下流へ反映済みの状態の差分をプロビジョニングタスクにする

- `worker` のインクリメンタル同期が接続を処理するとき、Provisioning は、適用範囲の User のあるべき状態と `RemoteResourceLink` が記録する反映済みの状態を突き合わせ、差分ごとにタスクを作る。
- 適用範囲の中で有効だがリンクのない User があるとき、Provisioning は、`operation=create` のタスクを作る。
- 割り当てを解除された User のリンクが残っているとき、Provisioning は、`operation=deactivate` のタスクを作る。
- 対象に未完了のタスクがある間、インクリメンタル同期が接続を処理するとき、Provisioning は、その対象に新しいタスクを作らない。
- 猶予期間つきの削除の対象では、インクリメンタル同期が接続を処理するとき、Provisioning は、`operation=delete` のタスクを作らず、猶予期間の後に削除する予約を作る。
- **例**：EX-PROVISIONING-019-01、EX-PROVISIONING-019-02、EX-PROVISIONING-019-03、EX-PROVISIONING-019-04

#### REQ-PROVISIONING-011 誤削除ガードの閾値を超えると接続を隔離し、管理者が解除する

- インクリメンタル同期が計画した無効化と削除の件数が `accidental_deletion_count_threshold` を超えるか、リンクのある User に対する割合が `accidental_deletion_percent_threshold` を超える場合、Provisioning は、タスクを 1 件も作らず、下流へ何も送らず、接続の `health` を `quarantined` にし、`ConnectionQuarantined` を発行し、`notification_email` へ通知する。
- 無効化と削除の件数が閾値ちょうどの間、インクリメンタル同期が接続を処理するとき、Provisioning は、タスクを作り、接続の `health` を `ok` のまま残す。
- 管理者が隔離した接続を再開したとき、Provisioning は、`health` を `ok` に戻し、200 と接続を返し、`ProvisioningConnectionQuarantineCleared` を発行する。
- 隔離していない接続の再開を要求された場合、Provisioning は、400 と `invalid_request` で拒否し、接続を変えない。
- **例**：EX-PROVISIONING-011-01、EX-PROVISIONING-011-02、EX-PROVISIONING-011-03、EX-PROVISIONING-011-04

## セキュリティ上の考慮

タスクそのものは、管理者の権限を借りない。
`worker` が下流へ示すのは接続に保存した資格情報だけであり、IdMagic の側の管理者の権限が下流へ伝わることはない。
