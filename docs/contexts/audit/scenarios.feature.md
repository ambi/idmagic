# Feature: Audit Scenarios

## Rule: REQ-AUDIT-001 管理者は監査ログを期間で絞り込み参照・エクスポートできる

Primary actor: `TenantAdministrator`

### Example: EX-AUDIT-001-01 通常経路

- Given ロール=["admin"] のユーザー "operator" が管理画面の監査ログを開いている
- When 管理者 "operator" が直近 24 時間で監査イベントを絞り込む
- Then 一覧に所属テナントの監査イベントだけが表示される
- When 管理者 "operator" が絞り込み結果をエクスポートする
- Then 所属テナントの絞り込み結果がエクスポートデータとして返る

## Rule: REQ-AUDIT-002 worker プロセスが発行した業務イベントも管理者は監査ログで参照できる

Primary actor: `TenantAdministrator`

### Example: EX-AUDIT-002-01 通常経路

- Given 適用対象の CSV インポートが存在する
- When worker プロセスが CSV インポートを適用し `UserCreated` を発行する
- When 管理者が `ListAdminAuditEvents` で `type=UserCreated` を検索する
- Then 発行元プロセス (`idmagic-api` / `idmagic-worker`) にかかわらずイベントが監査ログに含まれる

## Rule: REQ-AUDIT-003 管理者は workflow_id と run_id で LifecycleWorkflow の監査イベントを検索できる

Primary actor: `TenantAdministrator`

### Example: EX-AUDIT-003-01 通常経路

- Given LifecycleWorkflow "leaver-offboarding" の WorkflowRun "run-1" が実行中である
- When WorkflowRun "run-1" の 1 つのステップが失敗する
- Then "LifecycleWorkflowRunPartiallyFailed" と "LifecycleWorkflowStepFailed" が発行される
- When 管理者がフィルターに `workflow_run.id="run-1"` を指定して監査ログを検索する
- Then "run-1" に紐づくイベントだけが返り、属性値やメール本文は含まれない

## Rule: REQ-AUDIT-004 管理者は監査ログをページ単位で閲覧でき、絞り込みを変えるとカーソルが無効になる

Primary actor: `TenantAdministrator`

### Example: EX-AUDIT-004-01 通常経路

- Given 所属テナントに `limit` を超える件数の監査イベントが存在する
- When 管理者が `ListAdminAuditEvents` に `limit` だけを指定して先頭ページを取得する
- Then レスポンスは絞り込みに一致する正確な総件数、総ページ数、現在のページ、ページサイズを返す
- Then レスポンスの `Link` ヘッダー (`rel="next"`) にコンパクトなカーソルが含まれる
- When 管理者が取得済みのカーソルで次ページを取得する
- Then 直前のページと重複や欠落なく後続のイベントが返る
- When 管理者が `Link` ヘッダー (`rel="prev"`) のカーソルで前ページを取得する
- Then 前ページのイベントが正規の時系列降順で返る
- When 管理者が `rel="last"` の終端カーソルで最終ページを取得する
- Then 端数を含む最終ページが返る
- When 管理者が `rel="first"` のカーソルを含まない URL で先頭ページを取得する
- Then 正規の先頭ページが返る

### Example: EX-AUDIT-004-02 絞り込みに一致するイベントが 0 件である

- Given 所属テナントに `limit` を超える件数の監査イベントが存在する
- When 管理者が `ListAdminAuditEvents` に `limit` だけを指定して先頭ページを取得する
- But 絞り込みに一致するイベントが 0 件である
- Then 空のイベント一覧と、総件数 / 総ページ数 / 現在ページとして 0 / 0 / 0 を返す
- And first / prev / next / last の Link は返さない

### Example: EX-AUDIT-004-03 正確な件数の取得に失敗する

- Given 所属テナントに `limit` を超える件数の監査イベントが存在する
- When 管理者が `ListAdminAuditEvents` に `limit` だけを指定して先頭ページを取得する
- But 正確な件数の取得に失敗する
- Then 0 件として成功させず、リクエスト全体をサーバーエラーで失敗させる

### Example: EX-AUDIT-004-04 実行者が TenantAdministrator ロールを持たない

- Given 所属テナントに `limit` を超える件数の監査イベントが存在する
- When 管理者が `ListAdminAuditEvents` に `limit` だけを指定して先頭ページを取得する
- But 実行者が TenantAdministrator ロールを持たない
- Then ListAdminAuditEvents は AccessDeniedError で拒否される

### Example: EX-AUDIT-004-05 category や filter などを変更し、元の絞り込み条件で発行されたカーソルを送る

- Given 所属テナントに `limit` を超える件数の監査イベントが存在する
- When 管理者が `ListAdminAuditEvents` に `limit` だけを指定して先頭ページを取得する
- Then レスポンスは絞り込みに一致する正確な総件数、総ページ数、現在のページ、ページサイズを返す
- Then レスポンスの `Link` ヘッダー (`rel="next"`) にコンパクトなカーソルが含まれる
- When 管理者が取得済みのカーソルで次ページを取得する
- But category や filter などを変更し、元の絞り込み条件で発行されたカーソルを送る
- Then InvalidRequestError を返し、管理者は先頭ページから検索し直す

### Example: EX-AUDIT-004-06 カーソルが別テナントで発行された、改ざんされた、または旧方式の有効期限を超過している

- Given 所属テナントに `limit` を超える件数の監査イベントが存在する
- When 管理者が `ListAdminAuditEvents` に `limit` だけを指定して先頭ページを取得する
- Then レスポンスは絞り込みに一致する正確な総件数、総ページ数、現在のページ、ページサイズを返す
- Then レスポンスの `Link` ヘッダー (`rel="next"`) にコンパクトなカーソルが含まれる
- When 管理者が取得済みのカーソルで次ページを取得する
- But カーソルが別テナントで発行された、改ざんされた、または旧方式の有効期限を超過している
- Then InvalidRequestError を返す

## Rule: REQ-AUDIT-005 管理者はエージェントが代行した操作を本人の操作と区別して検索できる

Primary actor: `TenantAdministrator`

### Example: EX-AUDIT-005-01 通常経路

- Given Agent "A1" が User "alice" を subject とする委任トークンを得ている
- When "A1" がそのトークンで操作し、監査イベントが発行される
- Then イベントは行為者の種別、エージェントの識別子、委譲の深さ、委譲モードを検索軸として持つ
- Then 行為者の識別子は代行した側のものであり、"alice" へ読み替えられない。"alice" は対象として残る
- When 管理者がフィルターに `actor.type="agent"` と `agent.id="A1"` を指定して監査ログを検索する
- Then "A1" が代行した操作だけが返る
- When 管理者がフィルターに `actor.id="alice"` を指定して監査ログを検索する
- Then "alice" 本人の操作だけが返り、"A1" が代行した操作は含まれない

### Example: EX-AUDIT-005-02 管理者が "A1" を登録・無効化した操作である

- Given Agent "A1" が User "alice" を subject とする委任トークンを得ている
- When "A1" がそのトークンで操作し、監査イベントが発行される
- Then イベントは行為者の種別、エージェントの識別子、委譲の深さ、委譲モードを検索軸として持つ
- Then 管理者が "A1" を登録・無効化した操作である
- Then 行為者は管理者であり、"A1" はエージェントの識別子としてだけ残る

### Example: EX-AUDIT-005-03 追加した軸を持たない過去のイベントである

- Given Agent "A1" が User "alice" を subject とする委任トークンを得ている
- When "A1" がそのトークンで操作し、監査イベントが発行される
- Then イベントは行為者の種別、エージェントの識別子、委譲の深さ、委譲モードを検索軸として持つ
- Then 行為者の識別子は代行した側のものであり、"alice" へ読み替えられない。"alice" は対象として残る
- When 管理者がフィルターに `actor.type="agent"` と `agent.id="A1"` を指定して監査ログを検索する
- But 追加した軸を持たない過去のイベントである
- Then どの値にも一致せず、結果に混ざらない

### Example: EX-AUDIT-005-04 許可リストに無い軸を指定する

- Given Agent "A1" が User "alice" を subject とする委任トークンを得ている
- When "A1" がそのトークンで操作し、監査イベントが発行される
- Then イベントは行為者の種別、エージェントの識別子、委譲の深さ、委譲モードを検索軸として持つ
- Then 行為者の識別子は代行した側のものであり、"alice" へ読み替えられない。"alice" は対象として残る
- When 管理者がフィルターに `actor.type="agent"` と `agent.id="A1"` を指定して監査ログを検索する
- But 許可リストに無い軸を指定する
- Then フィルターの解析で拒否され、問い合わせは発行されない

## Rule: REQ-AUDIT-006 管理者は委譲チェーンの参加者から代行の連なりを横断検索できる

Primary actor: `TenantAdministrator`

### Example: EX-AUDIT-006-01 通常経路

- Given User "alice" のトークンをクライアント "app-a" が交換し、それをさらにクライアント "app-b" が交換した委任トークンがある
- When "app-b" がそのトークンを発行させ、監査イベントが発行される
- Then イベントは "alice"、"app-a"、"app-b" のいずれからも引ける委譲チェーンの参加者を検索軸として持つ
- When 管理者がフィルターに `delegation.actor="app-a"` を指定して監査ログを検索する
- Then チェーンの中間にいる "app-a" が関与したイベントが返り、参加者には識別子だけが含まれユーザー名は含まれない
- When 管理者がフィルターに `delegation.mode` を指定して絞り込む
- Then 絞り込みに使う値は、同じ交換について REQ-OAUTH2-049 がイントロスペクションへ返すモードと一致する
