# SET の配送

## 概要

この文書は、確定した失効を、送信側のストリームを通じて外部の受信側へ CAEP のイベントとして配送する仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | `AgentAccessRevoked` からの配送の作成、配送の試行、再試行、配信不能への遷移、配信不能の配送のやり直し |
| 行為者 | System（失効への反応と、`worker` の配送のループ）、管理者（やり直し） |
| 扱わないもの | ローカルの失効の確定は[失効エポック](revocation.md)が扱う |

## 状態遷移

### SecurityEventDeliveryLifecycle

生成時に `pending` として作成する。配送の試行に成功すると終端状態の `delivered` へ、失敗すると `failed` へ遷移して再試行を予定する。再試行の時刻が来ると `pending` に戻って試行し、`max_delivery_attempts` 回目の試行に失敗すると `dead_letter` へ遷移する。管理者がやり直すと、`dead_letter` の配送は試行の回数を 0 にして `pending` に戻る。

<!-- spec:state-machine backend/sharedsignals/testdata/SecurityEventDeliveryLifecycle.statemachine.json -->

| State | Kind | Meaning |
| --- | --- | --- |
| pending | initial | 配送待ち。生成直後と再試行の時刻の到来後がこの状態である |
| delivered | terminal | 受信側へ配送できた |
| failed | — | 配送に失敗し、再試行を予定している |
| dead_letter | — | `max_delivery_attempts` を使い切った。管理者がやり直すまで配送しない |

| From | Event | Guard | To | Effects |
| --- | --- | --- | --- | --- |
| pending | SecurityEventTransmitted | — | delivered |  |
| pending | SecurityEventDeliveryFailed | `attempt_count < max_delivery_attempts` | failed |  |
| pending | SecurityEventDeliveryDeadLettered | `attempt_count = max_delivery_attempts` | dead_letter |  |
| failed | SecurityEventDeliveryRetried | — | pending |  |
| dead_letter | SecurityEventDeliveryRedeliveryRequested | — | pending |  |

| State | 配送の試行 | 再試行の時刻の到来 | 管理者によるやり直し |
| --- | --- | --- | --- |
| pending | → delivered（受信側が受理）<br>→ failed（失敗し、試行の回数が上限未満）<br>→ dead_letter（失敗し、試行の回数が上限） | 何もしない | 何もしない |
| delivered | 何もしない | 何もしない | 何もしない |
| failed | 何もしない | → pending | 何もしない |
| dead_letter | 何もしない | 何もしない | → pending |

<!-- /spec:state-machine -->

## 操作

### worker による配送

#### REQ-SHAREDSIGNALS-006 配送失敗は再試行し、上限を超えると dead_letter へ遷移する

- `AgentAccessRevoked` を発行したとき、SharedSignals は、`enabled` で `session-revoked` を購読する送信側のストリームごとに、`pending` の配送を作る。
- `worker` が配送の時刻が来た `pending` の配送を試行し、受信側が受理したとき、SharedSignals は、配送を `delivered` にし、`SecurityEventTransmitted` を発行する。
- `worker` が配送を試行して失敗し、試行の回数が `max_delivery_attempts`（デフォルト 8 回）未満の場合、SharedSignals は、配送を `failed` にし、30 秒から倍々に増えて 30 分を上限とする待ち時間の後に次の試行を予定し、`SecurityEventDeliveryFailed` を発行する。
- `worker` が配送を試行して失敗し、試行の回数が `max_delivery_attempts` に達した場合、SharedSignals は、配送を `dead_letter` にし、`SecurityEventDeliveryFailed` と `SecurityEventDeliveryDeadLettered` を発行し、管理者がやり直すまで配送しない。
- `failed` の配送の再試行の時刻が来たとき、SharedSignals は、配送を `pending` に戻し、`SecurityEventDeliveryRetried` を発行してから試行する。
- 送信側の設定を削除したストリームの配送を試行する場合、SharedSignals は、配送を失敗として扱う。
- **例**：EX-SHAREDSIGNALS-006-01、EX-SHAREDSIGNALS-006-02

### 管理者によるやり直し

#### REQ-SHAREDSIGNALS-013 管理者は配信不能の配送をやり直せる

- 管理者が送信側のストリームの配送のやり直しを要求したとき、SharedSignals は、そのストリームの `dead_letter` の配送をすべて `pending` に戻し、試行の回数を 0 に、次の試行の時刻と最後のエラーを空にし、配送ごとに `SecurityEventDeliveryRedeliveryRequested` を発行し、200 と戻した件数を返す。
- やり直した配送を試行するとき、SharedSignals は、保存した SET を作り直さず、同じ `jti` と同じ署名のまま送る。
- `dead_letter` の配送がないストリームのやり直しを要求された場合、SharedSignals は、配送を変えず、イベントを発行せず、200 と件数 0 を返す。
- ストリームが `disabled` の間にやり直しを要求された場合、SharedSignals は、409 と `ssf_stream_disabled` で拒否し、配送を変えない。
- 存在しないか別のテナントのストリームを指定された場合、SharedSignals は、404 と `ssf_stream_not_found` で拒否する。
- **例**：EX-SHAREDSIGNALS-013-01、EX-SHAREDSIGNALS-013-02

### SET の形

#### REQ-SHAREDSIGNALS-014 送信する SET は最上位に sub_id を持つ

- SET を署名するとき、SharedSignals は、最上位に `format` が `opaque` の `sub_id` を含め、`sub` と `exp` を含めない。
- 失効の SET を署名するとき、SharedSignals は、`sub_id` の `id` を失効した主体の識別子にする。
- **例**：EX-SHAREDSIGNALS-014-01

### 監査イベントの配送

#### REQ-SHAREDSIGNALS-015 監査の記録と配送を同時に確定する

- 監査の記録を新しく追記したとき、SharedSignals は、同じトランザクションで、テナントの `enabled` の送信側のストリームのうち `audit_event_types` が `*` または記録の種類を含むものごとに、`pending` の配送を一つ作る。
- 監査の追記が確定しなかった場合、SharedSignals は、配送も残さない。
- 配送を作れなかった場合、SharedSignals は、失敗を返して監査の追記を確定させない。
- 同じ ID の監査の記録を再び受け取ったとき、SharedSignals は、配送を増やさない。
- `SecurityEvent` で始まる種類の記録を追記したとき、SharedSignals は、配送を作らない。
- テナントを持たない記録を追記したとき、SharedSignals は、配送を作らない。
- **例**：EX-SHAREDSIGNALS-015-01、EX-SHAREDSIGNALS-015-02

#### REQ-SHAREDSIGNALS-016 監査の SET は許可した識別子だけを持つ

- `worker` が監査の配送を試行するとき、SharedSignals は、`urn:idmagic:secevent:audit:<種類>` をイベントの種類とする SET をその時点の鍵で署名して送り、配送の行が持つ `jti` を再試行でも変えない。
- 監査の SET を署名するとき、SharedSignals は、イベントに `event_timestamp`、`audit_event_id`、`tenant_id`、任意の `actor`（`type` と `id`）、任意の `target_id` だけを含め、ユーザー名、メールアドレス、IP アドレス、トークン、資格情報を含めない。
- 監査の SET を署名するとき、SharedSignals は、`sub_id` の `id` を `target_id`、行為者の識別子、テナントの識別子の順で最初にあるものにする。
- 監査の配送が受理されたとき、SharedSignals は、配送を `delivered` にし、`SecurityEventTransmitted` を発行しない。
- `failed` の監査の配送を再試行するとき、SharedSignals は、`SecurityEventDeliveryRetried` を発行しない。
- 監査の配送の試行が失敗したとき、SharedSignals は、REQ-SHAREDSIGNALS-006 と同じ回数と間隔で再試行し、失敗と配信不能のイベントを発行する。
- **例**：EX-SHAREDSIGNALS-016-01

### 配送先と配送の隔離

#### REQ-SHAREDSIGNALS-017 配送先は公開の HTTPS に限り、接続のたびに検査する

- 管理者が送信側のストリームを登録するとき、SharedSignals は、配送先のすべての解決先が公開アドレスであることを確かめる。
- 配送先が `https` でない、userinfo を含む、または公開アドレスへ解決しない場合、SharedSignals は、422 と `ssf_transmitter_delivery_endpoint_invalid` で拒否する。
- 配送を試行するとき、SharedSignals は、配送先を解決し直して全アドレスを検査し、環境のプロキシと接続の再利用を使わずに、検査したアドレスへだけ接続する。
- 受信側がリダイレクトを返した場合、SharedSignals は、転送先へ接続せず、試行を失敗として記録する。
- 試行の失敗を記録するとき、SharedSignals は、`last_error` に内部のアドレスを含めない。
- **例**：EX-SHAREDSIGNALS-017-01、EX-SHAREDSIGNALS-017-02

#### REQ-SHAREDSIGNALS-018 連続失敗でストリームを無効化し、配送を隔離する

- 連続の失敗が 20 回に達し、かつ最初の失敗から 1 時間以上が過ぎたとき、SharedSignals は、ストリームを `disabled` にし、`SsfStreamAutoDisabled` を発行する。
- 配送が受理されたとき、SharedSignals は、連続の失敗の記録を消す。
- 管理者がストリームを再有効化したとき、SharedSignals は、連続の失敗の記録を消し、保留していた配送の送信を再開する。
- ストリームが `disabled` の間、SharedSignals は、配送を送らず、配送の行を残す。
- 一周の配送でストリームの送信が失敗したとき、SharedSignals は、そのストリームの残りの配送をその周では送らず、残りの試行の回数を使わない。
- `delivered` と `dead_letter` の配送の作成から 30 日が過ぎたとき、SharedSignals は、`idmagic-batch` の掃引でその配送を消す。
- **例**：EX-SHAREDSIGNALS-018-01

## 実現方式

### アーキテクチャ

失効エポックを進めた反応が `AgentAccessRevoked` を発行すると、同じプロセスが、有効な送信側のストリームごとに SET を署名し、`pending` の配送を作る。
この投影は 5 秒の期限で行い、失敗してもログに残すだけで、ローカルの失効の結果を変えない。

監査の配送は、[Audit](../../audit/README.md) の追記が組み立て地点から受け取った関数を同じトランザクションで呼び、一文の `INSERT ... SELECT` で作る。
Audit と SharedSignals は互いを import しない。
署名はトランザクションの外で行うために送信時まで遅らせる。

`worker` の配送のループは、一定の間隔で、試行の時刻の来た配送を `FOR UPDATE SKIP LOCKED` で 100 件まで選び、時刻を 2 分先へ進めてリースする。
一つのストリームからは一周に 10 件までとし、ストリームごとに 8 並行で送る。
`worker` が途中で止まっても、時刻の到来で別の周期が拾う。
一件の送信の失敗はその配送の状態に記録し、ほかのストリームの処理を止めない。

### 信頼性

再試行の間隔は、30 秒から始めて失敗のたびに倍にし、30 分で頭打ちにする。
`max_delivery_attempts` を使い切った配送は `dead_letter` になり、管理者がやり直すまで送らない。
受信側を直した後は、管理 API の配送の状況から `dead_letter` の配送を確かめ、ストリームの単位でやり直す。

やり直しは新しい SET を作らず、同じ行を `pending` に戻す。
RFC 8935 の 2 節は同じ SET の再送を認め、RFC 8417 の `jti` で受信側は受理済みの SET を見分けられる。
`event_timestamp` は失効が起きた時刻のままなので、受信側は古い事象であることを判断できる。
一つの更新文で戻すので、重複した要求でも配送ごとのイベントは一度である。

保存した SET の署名は作り直さないので、署名鍵が JWKS から外れた後のやり直しは受信側で検証できず、再び `dead_letter` になることがある（[リスク](../risks.md)）。
