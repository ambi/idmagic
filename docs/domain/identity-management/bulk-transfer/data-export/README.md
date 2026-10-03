# データエクスポート

## 概要

この文書は、管理者が User、Group、Group のメンバーシップを CSV のファイルとして非同期にエクスポートする機能の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | エクスポートの開始、一覧、参照、ダウンロード、取り消し、保持期限 |
| 行為者 | 管理者 |
| 扱わないもの | 各種別の列とセルの書き方は[ユーザー CSV](../user-csv/README.md)、[グループ CSV](../group-csv/README.md)、[CSV の転送](../csv-transfer/README.md)が、ジョブの実行は `Jobs` が、本人によるデータエクスポートは[アカウントのセルフサービス](../../principals/account/README.md)が扱う |

## モデル

エクスポートは、種類 `data_export` の `Jobs` のジョブとして記録する。
対象（`user`、`group`、`group_membership`）、要求した列、絞り込み、要求した管理者をジョブのパラメーターに、ファイル名、行数、バイト数、成果物の参照、SHA-256 をジョブの結果に置く。
CSV の本体はジョブに置かず、テナント単位の不変な成果物ストアに置く。

## 状態遷移

### DataExportLifecycle

`queued` で受理され、`worker` が `running` で CSV を生成する。
成功するとダウンロードできる `succeeded`、失敗すると不完全なファイルをダウンロードできない `failed` で終わる。
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
| queued | DataExportStarted | — | running |  |
| running | DataExportSucceeded | — | succeeded |  |
| running | DataExportFailed | — | failed |  |
| queued | DataExportCanceled | — | canceled |  |
| running | DataExportCanceled | — | canceled |  |
| succeeded | DataExportExpired | duration_since(completed_at) >= duration('2592000s') | expired |  |

## 操作

### エクスポートの開始

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者 |
| 入力 | 対象、列、絞り込み。メンバーシップのエクスポートでは、パスの `group_id` |
| 成功時の作用 | `queued` のジョブを作り、`DataExportRequested` を発行する |
| 拒否 | 不正な列（`invalid_columns`）、許可していない絞り込み（`invalid_filter`）、実行中のジョブの上限の超過（429 `active_job_quota_exceeded`）。拒否した開始はジョブを作らず、イベントを発行しない |

#### REQ-IDMANAGEMENT-039 エクスポートの開始は、許可した列と絞り込みだけを受け付ける

- 列は一つ以上を指定し、同じ列を二度含めない。違反する開始は `invalid_columns` で拒否する。
- 絞り込みのキーは、User のエクスポートでは `status` だけ、Group のエクスポートでは受け付けない。それ以外のキーを含む開始は `invalid_filter` で拒否する。
- `status` の値は前後の空白と大文字と小文字を区別せず、User の状態のどれでもない値を `invalid_filter` で拒否する。
- メンバーシップのエクスポートの対象 Group は、パスの `group_id` だけで決める。本文の絞り込みは対象を変えない。
- 拒否した開始はジョブを作らず、`DataExportRequested` を発行しない。
- 実行中のジョブの上限を超える開始は、429 と `active_job_quota_exceeded` で拒否する。
- **担保手段**：`usecases.StartDataExport`、`handlers_http.HandleStartUserExport`

### エクスポートの一覧

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者 |
| 入力 | 対象の種類。メンバーシップでは対象の Group |
| 成功時の作用 | テナントの直近 200 件のうち、要求した種類と Group のエクスポートを新しい順に返す |

#### REQ-IDMANAGEMENT-040 エクスポートの一覧は新しい順に、テナントの直近 200 件から返す

- 一覧は、テナントのエクスポートを作成の新しい順に並べる。
- 一覧は、テナントのすべての種類のエクスポートのうち新しい 200 件を読み、そこから要求した種類と Group に属するものだけを返す。
- **担保手段**：`usecases.ListDataExports`
- **要判断**：別の種類のエクスポートが新しい 200 件を占めると、要求した種類の古いエクスポートは、保持期限内でも一覧に現れない。種類ごとに 200 件を返すか、ページングにするかを決める。

### エクスポートの取り消し

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者 |
| 入力 | エクスポートの ID |
| 成功時の作用 | `canceled` にし、`DataExportCanceled` を発行する |
| 拒否 | 終了したエクスポート（409 `data_export_not_cancelable`）。状態を変えず、イベントを発行しない |

#### REQ-IDMANAGEMENT-041 終了したエクスポートは取り消せない

- `succeeded`、`failed`、`canceled`、`expired` のエクスポートの取り消しは、409 と `data_export_not_cancelable` で拒否する。
- 拒否した取り消しは状態を変えず、`DataExportCanceled` を発行しない。
- **担保手段**：`usecases.CancelDataExport`

## セキュリティ上の考慮

| 観点 | 内容 |
| --- | --- |
| 対象ごとの分離 | 参照、ダウンロード、取り消しは、パスの対象の種類（と、メンバーシップでは Group）に属するエクスポートだけを解決する。別の種類や別の Group のエクスポートの ID は、存在しないものとして扱う |
| 個人情報 | エクスポートの参照の応答とジョブの記録は CSV の本体を含まない。ダウンロードは `DataExportDownloaded` を発行する |
| 認可 | エクスポートは対象を変えないので、参照のスコープで許可する。規則は[管理 API の認可](../../common/admin-access/README.md)が定める |
