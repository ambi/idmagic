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

## 実現方式

### アーキテクチャ

#### イベント同期

`IdManagement` の User の変更の処理と `Application` の割り当ての変更の処理は、自身の変更を確定した後で `ProvisioningCapture` を呼ぶ。
ポートは、一致する有効な接続ごとに `pending` のタスクを 1 件挿入する。
イベント同期の失敗や、イベント同期を呼ばない経路による変更は、次のインクリメンタル同期が回収する。

#### インクリメンタル同期

`worker` の周期のループは、`PROVISIONING_RECONCILE_INTERVAL`（デフォルト 5 分）ごとに、有効で隔離されていない接続をすべて処理する。

| 読むもの | 意味 |
| --- | --- |
| テナントの User | あるべき状態。有効かどうか、`updated_at` によるバージョン |
| 接続のスコープ | `all_users` なら全 User、`assigned_only` なら割り当てのある User |
| `RemoteResourceLink` | 反映済みの状態。リンクの有無、下流で有効として反映したかを表す `active`、最後に反映したバージョン |
| `succeeded` 以外のタスク | 決着を待つべき User |

| あるべき状態 | 反映済みの状態 | 作るもの |
| --- | --- | --- |
| スコープ内で有効 | リンクなし | `create` |
| スコープ内で有効 | リンクが無効 | `update`（再有効化） |
| スコープ内で有効 | リンクが有効、User のバージョンがリンクより新しい | `update` |
| スコープ内で無効 | リンクが有効 | `deactivate` |
| スコープ外 | リンクあり | `on_unassign` に従う。`deactivate` はリンクが有効なときだけ作る |
| 削除済み | リンクあり | `on_delete` に従う。`delete` で猶予期間があれば予約を作る |

いずれも、接続の機能のフラグが有効なときだけ作る。
`pending` または `in_flight` のタスクがある User は飛ばす。イベント同期や前回のインクリメンタル同期が、同じ変更を扱っているためである。
`dead_letter` のタスクがあり、そのバージョンが User のバージョン以上である User も飛ばす。前回の失敗から User が変わっていないので、作り直しても同じく失敗するためである。

リンクは、下流へ送った `active` を記録する。
割り当ての解除では User の `updated_at` が変わらないので、バージョンだけでは、無効化を反映済みかも、再割り当てで戻すべきかも見分けられない。
User の `delete` を下流へ送ったら、リンクを消す。リンクなしは「下流に何もない」を表す。

1 回のインクリメンタル同期で 1 接続に作るものは 500 件までとし、User の ID の順に打ち切る。
残りは次のインクリメンタル同期が拾う。
インクリメンタル同期が作るタスクのバージョンはインクリメンタル同期の時刻とし、イベント同期と同じ時間の軸に乗せる。

#### 猶予期間つきの削除の予約

`grace_period_days` が 1 以上の接続では、`on_delete=delete` へ変換した User の削除からタスクを作らず、`ScheduledDeprovision` を保存する。
タスクのレコードを作らないので、`worker` が期限の前に下流へ DELETE を送ることはない。

| 状態 | 意味 | 遷移の契機 |
| --- | --- | --- |
| `scheduled` | 期限の到来を待っている。同じ接続と User に一件だけ置き、再度の削除の通知は期限を延ばさない | 猶予期間を持つ接続への User の削除 |
| `materialized` | 期限に達し、同じバージョンの `delete` のタスクを作った | ディスパッチャーの周期の処理が、期限に達した予約を見つけた |
| `cancelled` | 猶予期間の中に対象が適用範囲へ戻ったので取り消した | 同じ Application への User の割り当て。予約より新しいバージョンの割り当てだけが取り消す |

ディスパッチャーは、未関連付けのタスクへジョブを関連付ける前に、期限に達した予約をタスクに変える。
予約を `materialized` にする更新とタスクの挿入は一つの SQL の文で行い、更新は予約が `scheduled` のときだけ成り立つ。
取消と実体化が競合しても、取り消した予約からタスクが作られることはない。
予約は削除の時点の接続の設定で判断し、期限の到来の時に接続の状態や `on_delete` を評価し直さない。
接続を削除した場合だけ、外部キーの連鎖で予約も消える。

#### 誤削除ガード

インクリメンタル同期は、1 回で作るものを計画する途中で deprovision を数え、`DeprovisionPolicy` の閾値と比べる。
数えるのは計画した `deactivate` と `delete` で、猶予期間つきの削除の予約を含む。
決着を待つタスクがあるために飛ばした User は数えない。

| 閾値 | 超過の条件 |
| --- | --- |
| `accidental_deletion_count_threshold` | deprovision の件数が閾値より大きい |
| `accidental_deletion_percent_threshold` | deprovision の件数が、接続の User のリンクの数に対する割合で閾値より大きい |

どちらかの閾値を超えたら、そのインクリメンタル同期では何も作らず、接続を隔離して `ConnectionQuarantined` を発行する。
deprovision 以外も作らないのは、隔離が接続のプロビジョニング全体を止める状態だからである。
1 回あたりの上限で打ち切る前の件数を比べるのは、打ち切った後の件数では、上限を超える一括の deprovision を見逃すためである。

ガードはインクリメンタル同期だけに置く。
インクリメンタル同期は、イベント同期を呼ばない経路の変更も、スコープや方針の設定の変更による一括の差分も、1 回の計画としてまとめて扱う唯一の経路だからである。
イベント同期と猶予期間つきの削除の実体化は 1 件ずつ作るので、数える単位を持たない。
フル同期は管理者が明示的に起動する操作であり、ガードの対象にしない。

### 信頼性

#### 隔離からの回復

止めたものはどこにも残らないので、管理者が原因を正して隔離を解除した後は、次のインクリメンタル同期があるべき状態から作り直す。
実体化に失敗した予約は `scheduled` のまま残り、次の周期で再び処理される。

#### 隔離の通知

連続の失敗と誤削除ガードのどちらで接続を隔離した場合も、隔離を保存して `ConnectionQuarantined` を発行した後、接続の `notification_email` へ通知テンプレート `provisioning_connection_quarantined` のメールを送る。
`notification_email` が未設定または空白だけなら送らない。
隔離はプロビジョニングを止める状態なので、管理者が気付かなければ、下流の反映は止まったままになるためである。

| 差し込み変数 | 値 |
| --- | --- |
| `application_id` | 接続が属する Application の ID |
| `quarantine_reason` | 接続に保存した隔離の理由 |
| `quarantined_at` | 隔離した時刻（UTC） |

送信は共有の `Notifier` を通り、配送の失敗は隔離を取り消さず、再送もしない。
隔離はすでに隔離された接続には再び起きないので、同じ隔離について通知は一度だけ送られる。
