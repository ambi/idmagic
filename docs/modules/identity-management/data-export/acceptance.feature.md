# Feature: データエクスポートの例

## Rule: REQ-IDMANAGEMENT-039 エクスポートの開始は `queued` のジョブを作り、許可した列と絞り込みだけを受け付ける

### Example: EX-IDMANAGEMENT-039-01 重複した列

- When 管理者が列 [`email`, `email`] で User のエクスポートを開始する
- Then 開始は `invalid_columns` で拒否され、エクスポートは作られない

### Example: EX-IDMANAGEMENT-039-02 許可していない絞り込み

- When 管理者が絞り込み `{"email": "a@example.test"}` で User のエクスポートを開始する
- Then 開始は `invalid_filter` で拒否され、エクスポートは作られない

### Example: EX-IDMANAGEMENT-039-03 大文字の状態の絞り込み

- When 管理者が絞り込み `{"status": " Active "}` で User のエクスポートを開始する
- Then 開始は受け付けられる

### Example: EX-IDMANAGEMENT-039-04 User のエクスポートの開始

- When 管理者が列 [`preferred_username`, `email`] と `status` の絞り込みを指定して `/users/exports` へエクスポートを開始する
- Then エクスポートは 202 とエクスポート ID を返し、ジョブは `queued` である

## Rule: REQ-IDMANAGEMENT-079 エクスポートの保持期限は、完了の時刻から 30 日である

### Example: EX-IDMANAGEMENT-079-01 作成の 1 日後に完了したエクスポート

- Given 管理者は 1 月 1 日に User のエクスポートを開始し、エクスポートは 1 月 2 日に完了した
- When 管理者が 1 月 31 日にエクスポートを参照する
- Then `expires_at` は 2 月 1 日であり、エクスポートは `succeeded` のままダウンロードできる
- When 管理者が 2 月 1 日にエクスポートを参照する
- Then エクスポートは `expired` であり、ダウンロードできない

## Rule: REQ-IDMANAGEMENT-040 エクスポートの一覧は新しい順に、テナントの直近 200 件から返す

### Example: EX-IDMANAGEMENT-040-01 新しい順

- Given テナントに User のエクスポートが二つあり、二つ目のほうが新しい
- When 管理者が User のエクスポートの一覧を取得する
- Then 二つ目、一つ目の順に返る

### Example: EX-IDMANAGEMENT-040-02 別の種類が新しい 200 件を占める

- Given テナントに古い User のエクスポートが一つあり、その後に Group のエクスポートが 200 件ある
- When 管理者が User のエクスポートの一覧を取得する
- Then 一覧は空である

## Rule: REQ-IDMANAGEMENT-041 エクスポートの取り消しは、終了前のエクスポートを `canceled` にし、終了したエクスポートを拒否する

### Example: EX-IDMANAGEMENT-041-01 取り消し済みのエクスポートをもう一度取り消す

- Given 管理者が User のエクスポートを取り消している
- When 管理者が同じエクスポートをもう一度取り消す
- Then 409 と `data_export_not_cancelable` を返し、`DataExportCanceled` は再発行されない

### Example: EX-IDMANAGEMENT-041-02 終了前のエクスポートの取り消し

- Given 管理者が User のエクスポートを開始し、ジョブは `queued` である
- When 管理者がエクスポートを取り消す
- Then ステータスは `canceled` となり、`DataExportCanceled` が発行される

## Rule: REQ-IDMANAGEMENT-086 エクスポートの生成は、成功すると `succeeded` に、失敗すると `failed` にして、それぞれイベントを発行する

### Example: EX-IDMANAGEMENT-086-01 生成の完了

- Given 管理者が User のエクスポートを開始し、ジョブは `queued` である
- When `worker` プロセスが生成を開始する
- Then `DataExportStarted` が発行される
- When 生成が完了する
- Then ステータスは `succeeded`、`downloadable` は `true` となり、`total_rows` と `byte_size` が記録される
- And `DataExportSucceeded` が発行される

### Example: EX-IDMANAGEMENT-086-02 生成の失敗

- Given 管理者が User のエクスポートを開始し、ジョブは `queued` である
- When `worker` プロセスが生成を開始し、生成が失敗する
- Then ステータスは `failed`、`downloadable` は `false` となり、`error_code` が記録される
- And `DataExportFailed` が発行される
- And 不完全なファイルはダウンロードできない

## Rule: REQ-IDMANAGEMENT-087 エクスポートのダウンロードは、選んだ列の見出しを持つ CSV を添付として返し、`DataExportDownloaded` を発行する

### Example: EX-IDMANAGEMENT-087-01 完了したエクスポートのダウンロード

- Given 列 [`preferred_username`, `email`] の User のエクスポートが `succeeded` である
- When 管理者がファイルをダウンロードする
- Then 選択した機械可読キーと一致するヘッダーを持つ RFC 4180 CSV が、`Content-Disposition: attachment` で返る
- And `DataExportDownloaded` が発行される

### Example: EX-IDMANAGEMENT-087-02 保持期限を経過したエクスポートのダウンロード

- Given User のエクスポートは `succeeded` で、保持期限を経過している
- When 管理者がファイルをダウンロードする
- Then ステータスは `expired`、`downloadable` は `false` となる
- And ファイル本体は完全削除され、ダウンロードは 409 と `data_export_not_downloadable` で拒否される

## Rule: REQ-IDMANAGEMENT-088 エクスポートの参照、ダウンロード、取り消しは、種類またはテナントの異なるパスで指定した ID を拒否する

### Example: EX-IDMANAGEMENT-088-01 User のエクスポートの ID を `/groups/exports` または別テナントで指定する

- Given User のエクスポートが `succeeded` である
- When 管理者がその ID を `/groups/exports` または別テナントで指定して取得、ダウンロード、取り消しを行う
- Then 種類とテナントの境界により、取得、ダウンロード、取り消しは 404 と `data_export_not_found` で拒否される
