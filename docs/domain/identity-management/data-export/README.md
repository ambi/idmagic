# データエクスポート

## 概要

この文書は、管理者が User、Group、Group のメンバーシップを CSV のファイルとして非同期にエクスポートする機能の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | エクスポートの開始、生成、一覧、参照、ダウンロード、取り消し、保持期限 |
| 行為者 | 管理者、エクスポートを生成する `worker` |
| 扱わないもの | 各種別の列とセルの書き方は[ユーザー CSV](../user-csv/README.md)、[グループ CSV](../group-csv/README.md)、[CSV の転送](../csv-transfer/README.md)が、ジョブの実行と再試行は `Jobs` が、本人によるデータエクスポートは[アカウントのセルフサービス](../account/README.md)が扱う |

## モデル

エクスポートは、種類 `data_export` の `Jobs` のジョブとして記録する。
対象（`user`、`group`、`group_membership`）、要求した列、絞り込み、要求した管理者をジョブのパラメーターに、ファイル名、行数、バイト数、成果物の参照、SHA-256 をジョブの結果に置く。
CSV の本体はジョブに置かず、テナント単位の不変な成果物ストアに置く。

## 状態遷移

### DataExportLifecycle

`queued` で受理され、`worker` が `running` で CSV を生成する。
成功するとダウンロードできる `succeeded` になる。
失敗すると、`Jobs` の試行の上限まで `queued` に戻して生成をやり直し、上限に達すると不完全なファイルをダウンロードできない `failed` で終わる。
終わる前は `canceled` で取り消せる。
`succeeded` は保持期限を過ぎると `expired` になる。
保持期限の 30 日は、Jobs のデフォルトの記録の保持期間に合わせた。

| State | Kind | Meaning |
|---|---|---|
| queued | initial | 受理済み。`worker` の実行を待つ |
| running | — | `worker` が CSV を生成している |
| succeeded | — | 生成が完了し、保持期限までダウンロードできる |
| failed | terminal | 生成に失敗した。不完全なファイルはダウンロードできない |
| canceled | terminal | 終了前に取り消した |
| expired | terminal | 保持期限を過ぎ、ファイル本体を完全削除した。メタデータと監査記録だけが残る |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| queued | DataExportStarted | — | running | DataExportStarted |
| running | DataExportSucceeded | — | succeeded | DataExportSucceeded |
| running | JobRetried | job.attempts < job.max_attempts | queued | DataExportFailed |
| running | DataExportFailed | job.attempts >= job.max_attempts | failed | DataExportFailed |
| queued | DataExportCanceled | — | canceled | DataExportCanceled |
| running | DataExportCanceled | — | canceled | DataExportCanceled |
| canceled | DataExportSucceeded | — | canceled | DataExportSucceeded |
| canceled | DataExportFailed | — | canceled | DataExportFailed |
| succeeded | DataExportDownloaded | — | succeeded | DataExportDownloaded |
| succeeded | DataExportExpired | duration_since(completed_at) >= duration('2592000s') | expired |  |

| State | 生成の開始 | 生成の終了 | ダウンロード | 取り消し | 保持期限の経過 |
|---|---|---|---|---|---|
| queued | → running | 何もしない | 拒否：409 data_export_not_downloadable | → canceled | 何もしない |
| running | 何もしない | → succeeded（生成が成功した）<br>→ queued（生成が失敗し、試行が上限に達していない）<br>→ failed（生成が失敗し、試行が上限に達した） | 拒否：409 data_export_not_downloadable | → canceled | 何もしない |
| succeeded | 何もしない | 何もしない | → succeeded | 拒否：409 data_export_not_cancelable | → expired |
| failed | 何もしない | 何もしない | 拒否：409 data_export_not_downloadable | 拒否：409 data_export_not_cancelable | 何もしない |
| canceled | 何もしない | → canceled（取り消す前に始めた生成が終わった） | 拒否：409 data_export_not_downloadable | 拒否：409 data_export_not_cancelable | 何もしない |
| expired | 何もしない | 何もしない | 拒否：409 data_export_not_downloadable | 拒否：409 data_export_not_cancelable | 何もしない |

## 操作

### エクスポートの開始

#### REQ-IDMANAGEMENT-039 エクスポートの開始は `queued` のジョブを作り、許可した列と絞り込みだけを受け付ける

- 管理者がエクスポートを開始したとき、IdManagement は、202 とエクスポートの ID を返し、`queued` のジョブを作り、`DataExportRequested` を発行する。
- 管理者がメンバーシップのエクスポートを開始したとき、IdManagement は、パスの `group_id` だけで対象の Group を決め、本文の絞り込みで対象を変えない。
- 管理者が `status` で絞り込んだエクスポートを開始したとき、IdManagement は、値を前後の空白と大文字と小文字を区別せずに読む。
- 列を一つも指定しない開始、または同じ列を二度含む開始を要求された場合、IdManagement は、422 と `invalid_columns` で拒否する。
- User のエクスポートで `status` 以外の絞り込みのキーを指定されたか、Group のエクスポートで絞り込みを指定された場合、IdManagement は、422 と `invalid_filter` で拒否する。
- User の状態のどれでもない `status` の値を指定された場合、IdManagement は、422 と `invalid_filter` で拒否する。
- 実行中のジョブの上限を超える開始を要求された場合、IdManagement は、429 と `active_job_quota_exceeded` で拒否する。
- 開始を拒否した場合、IdManagement は、ジョブを作らず、`DataExportRequested` を発行しない。
- **例**：EX-IDMANAGEMENT-039-01、EX-IDMANAGEMENT-039-04

### エクスポートの生成

#### REQ-IDMANAGEMENT-086 エクスポートの生成は、成功すると `succeeded` に、失敗すると `failed` にして、それぞれイベントを発行する

- `worker` が生成を始めたとき、IdManagement は、`DataExportStarted` を発行する。
- 生成が完了したとき、IdManagement は、`succeeded` にし、`downloadable` を `true` にし、行数とバイト数を記録して、`DataExportSucceeded` を発行する。
- 生成が失敗したとき、IdManagement は、`DataExportFailed` を発行する。
- 生成が失敗し、試行の回数が `Jobs` の上限に達していない場合、IdManagement は、`queued` に戻し、生成をやり直す。
- 生成が失敗し、試行の回数が `Jobs` の上限に達した場合、IdManagement は、`failed` にし、`downloadable` を `false` にし、`error_code` を記録し、不完全なファイルをダウンロードさせない。
- 取り消した後に、取り消す前に始めた生成が終わった場合、IdManagement は、`canceled` のまま変えず、生成の結果に応じて `DataExportSucceeded` または `DataExportFailed` を発行する。
- **例**：EX-IDMANAGEMENT-086-01、EX-IDMANAGEMENT-086-02

### エクスポートの一覧

#### REQ-IDMANAGEMENT-040 エクスポートの一覧は新しい順に、テナントの直近 200 件から返す

- 管理者がエクスポートの一覧を取得したとき、IdManagement は、テナントのエクスポートを作成の新しい順に並べて返す。
- 管理者がエクスポートの一覧を取得したとき、IdManagement は、テナントのすべての種類のエクスポートのうち新しい 200 件を読み、そこから要求した種類と Group に属するものだけを返す。

### エクスポートの参照とダウンロード

#### REQ-IDMANAGEMENT-079 エクスポートの保持期限は、完了の時刻から 30 日である

- 管理者がエクスポートを参照したとき、IdManagement は、状態、行数、バイト数、保持期限を返し、CSV の本体を返さない。
- エクスポートが `succeeded` の間、管理者がエクスポートを参照したとき、IdManagement は、完了の時刻（`completed_at`）に 30 日を加えた時刻を `expires_at` として返す。
- エクスポートが `succeeded` でない間、管理者がエクスポートを参照したとき、IdManagement は、`expires_at` を返さない。
- `succeeded` のエクスポートの `expires_at` より前の間、IdManagement は、`downloadable` を `true` として返し、ダウンロードを受け付ける。
- `succeeded` のエクスポートの `expires_at` 以降に参照されたとき、IdManagement は、状態を `expired`、`downloadable` を `false` として返す。
- **判断**：保持期限を作成の時刻から数えると、生成に時間のかかったエクスポートほど、ダウンロードできる期間が短くなる。
- **例**：EX-IDMANAGEMENT-079-01

#### REQ-IDMANAGEMENT-087 エクスポートのダウンロードは、選んだ列の見出しを持つ CSV を添付として返し、`DataExportDownloaded` を発行する

- 管理者が `succeeded` のエクスポートをダウンロードしたとき、IdManagement は、成果物の SHA-256 とバイト数を照合してから、選んだ機械キーと一致する見出しの RFC 4180 の CSV を `Content-Disposition: attachment` で返し、`DataExportDownloaded` を発行する。
- `succeeded` でないエクスポートと `expired` のエクスポートのダウンロードを要求された場合、IdManagement は、409 と `data_export_not_downloadable` で拒否し、CSV を返さない。
- 成果物の SHA-256 またはバイト数がジョブの結果と一致しない場合、IdManagement は、409 と `data_export_not_downloadable` で拒否し、CSV を返さない。
- **例**：EX-IDMANAGEMENT-087-01、EX-IDMANAGEMENT-087-02

#### REQ-IDMANAGEMENT-088 エクスポートの参照、ダウンロード、取り消しは、種類またはテナントの異なるパスで指定した ID を拒否する

- エクスポートの ID を、別の種類のパス（例：User のエクスポートを `/groups/exports`）、別の Group のパス、または別のテナントで指定した参照、ダウンロード、取り消しを要求された場合、IdManagement は、存在しない ID と同じく 404 と `data_export_not_found` で拒否し、CSV を返さず、状態を変えない。
- **例**：EX-IDMANAGEMENT-088-01

### エクスポートの取り消し

#### REQ-IDMANAGEMENT-041 エクスポートの取り消しは、終了前のエクスポートを `canceled` にし、終了したエクスポートを拒否する

- 管理者が `queued` または `running` のエクスポートを取り消したとき、IdManagement は、`canceled` にし、`DataExportCanceled` を発行する。
- `succeeded`、`failed`、`canceled`、`expired` のエクスポートの取り消しを要求された場合、IdManagement は、409 と `data_export_not_cancelable` で拒否し、状態を変えず、`DataExportCanceled` を発行しない。

## セキュリティ上の考慮

| 観点 | 内容 |
| --- | --- |
| 対象ごとの分離 | 参照、ダウンロード、取り消しは、パスの対象の種類（と、メンバーシップでは Group）に属するエクスポートだけを解決する。別の種類や別の Group のエクスポートの ID は、存在しないものとして扱う |
| 個人情報 | エクスポートの参照の応答とジョブの記録は CSV の本体を含まない。ダウンロードは `DataExportDownloaded` を発行する |
| 認可 | エクスポートは対象を変えないので、参照のスコープで許可する。規則は[管理 API の認可](../admin-access/README.md)が定める |
