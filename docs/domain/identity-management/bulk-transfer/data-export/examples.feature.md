# Feature: データエクスポートの例

## Rule: REQ-IDMANAGEMENT-039 エクスポートの開始は、許可した列と絞り込みだけを受け付ける

### Example: EX-IDMANAGEMENT-039-01 重複した列

- When 管理者が列 [`email`, `email`] で User のエクスポートを開始する
- Then 開始は `invalid_columns` で拒否され、エクスポートは作られない

### Example: EX-IDMANAGEMENT-039-02 許可していない絞り込み

- When 管理者が絞り込み `{"email": "a@example.test"}` で User のエクスポートを開始する
- Then 開始は `invalid_filter` で拒否され、エクスポートは作られない

### Example: EX-IDMANAGEMENT-039-03 大文字の状態の絞り込み

- When 管理者が絞り込み `{"status": " Active "}` で User のエクスポートを開始する
- Then 開始は受け付けられる

## Rule: REQ-IDMANAGEMENT-040 エクスポートの一覧は新しい順に、テナントの直近 200 件から返す

### Example: EX-IDMANAGEMENT-040-01 新しい順

- Given テナントに User のエクスポートが二つあり、二つ目のほうが新しい
- When 管理者が User のエクスポートの一覧を取得する
- Then 二つ目、一つ目の順に返る

### Example: EX-IDMANAGEMENT-040-02 別の種類が新しい 200 件を占める

- Given テナントに古い User のエクスポートが一つあり、その後に Group のエクスポートが 200 件ある
- When 管理者が User のエクスポートの一覧を取得する
- Then 一覧は空である

## Rule: REQ-IDMANAGEMENT-041 終了したエクスポートは取り消せない

### Example: EX-IDMANAGEMENT-041-01 取り消し済みのエクスポートをもう一度取り消す

- Given 管理者が User のエクスポートを取り消している
- When 管理者が同じエクスポートをもう一度取り消す
- Then 409 と `data_export_not_cancelable` を返し、`DataExportCanceled` は再発行されない
