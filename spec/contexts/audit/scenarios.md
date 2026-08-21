# Audit Scenarios

### REQ-AUDIT-001: 管理者は監査ログを期間で絞り込み参照・エクスポートできる
- ACTOR TenantAdministrator
- GIVEN ロール=["admin"] のユーザー "operator" が管理画面の監査ログを開いている
- WHEN 管理者 "operator" が直近 24 時間で監査イベントを絞り込む
- THEN 一覧に所属テナントの監査イベントだけが表示される
- WHEN 管理者 "operator" が絞り込み結果をエクスポートする
- THEN 所属テナントの絞り込み結果がエクスポートデータとして返る

### REQ-AUDIT-002: worker プロセスが発行した業務イベントも管理者は監査ログで参照できる
- ACTOR TenantAdministrator
- GIVEN 適用対象の CSV インポートが存在する
- WHEN worker プロセスが CSV インポートを適用し `UserCreated` を発行する
- WHEN 管理者が `ListAdminAuditEvents` で `type=UserCreated` を検索する
- THEN 発行元プロセス (`idmagic-api` / `idmagic-worker`) にかかわらずイベントが監査ログに含まれる

### REQ-AUDIT-003: 管理者は workflow_id と run_id で LifecycleWorkflow の監査イベントを検索できる
- ACTOR TenantAdministrator
- GIVEN LifecycleWorkflow "leaver-offboarding" の WorkflowRun "run-1" が実行中である
- WHEN WorkflowRun "run-1" の 1 つのステップが失敗する
- THEN "LifecycleWorkflowRunPartiallyFailed" と "LifecycleWorkflowStepFailed" が発行される
- WHEN 管理者がフィルターに `workflow_run.id="run-1"` を指定して監査ログを検索する
- THEN "run-1" に紐づくイベントだけが返り、属性値やメール本文は含まれない

### REQ-AUDIT-004: 管理者は監査ログをページ単位で閲覧でき、絞り込みを変えるとカーソルが無効になる
- ACTOR TenantAdministrator
- GIVEN 所属テナントに `limit` を超える件数の監査イベントが存在する
- WHEN 管理者が `ListAdminAuditEvents` に `limit` だけを指定して先頭ページを取得する
  - ALT 絞り込みに一致するイベントが 0 件である → 空のイベント一覧と、総件数 / 総ページ数 / 現在ページとして 0 / 0 / 0 を返す → first / prev / next / last の Link は返さない
  - ALT 正確な件数の取得に失敗する → 0 件として成功させず、リクエスト全体をサーバーエラーで失敗させる
  - ALT 実行者が TenantAdministrator ロールを持たない → ListAdminAuditEvents は AccessDeniedError で拒否される
- THEN レスポンスは絞り込みに一致する正確な総件数、総ページ数、現在のページ、ページサイズを返す
- THEN レスポンスの `Link` ヘッダー (`rel="next"`) にコンパクトなカーソルが含まれる
- WHEN 管理者が取得済みのカーソルで次ページを取得する
  - ALT category や filter などを変更し、元の絞り込み条件で発行されたカーソルを送る → InvalidRequestError を返し、管理者は先頭ページから検索し直す
  - ALT カーソルが別テナントで発行された、改ざんされた、または旧方式の有効期限を超過している → InvalidRequestError を返す
- THEN 直前のページと重複や欠落なく後続のイベントが返る
- WHEN 管理者が `Link` ヘッダー (`rel="prev"`) のカーソルで前ページを取得する
- THEN 前ページのイベントが正規の時系列降順で返る
- WHEN 管理者が `rel="last"` の終端カーソルで最終ページを取得する
- THEN 端数を含む最終ページが返る
- WHEN 管理者が `rel="first"` のカーソルを含まない URL で先頭ページを取得する
- THEN 正規の先頭ページが返る
