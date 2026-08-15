---
context: audit
updated_at: 2026-08-15
---

# Audit Specification

## Overview

Authentication、IdManagement、OAuth2、Tenancy、SigningKeys、Application、SAML、WS-Federation など、複数の Bounded Context が発行するセキュリティ監査イベントの横断的な Read Model を所有する。監査イベントの検索とエクスポートを行う管理 API、検索属性レジストリ、個人識別情報の変換方針、保持期間をまとめる。イベントの発行元は各 Context のままとし、Audit はそれらを横断して読む窓口を所有する。

`DomainEvent` として蓄積される監査の記録は、管理者向けの `AdminAuditEventResponse` として公開する。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| AuditEventSearchRegistry | 監査イベントで使用できる検索属性（`AuditEventSearchAttribute`）の定義一覧。属性ごとにフィールド、演算子、変換方法を宣言し、任意の SQL や JSONPath を受け付けない閉じた検索文法を定める。Go の `AuditSearchRegistry` マップを唯一の正とする。 | search-attribute-registry, 検索属性定義 |
| AuditEventFilterExpression | `AuditEventQuery.filter` の 1 項。レジストリの許可リストにあるフィールド、演算子、値の並びからなる論理積 (AND) の 1 要素。個人識別情報の属性は平文入力をサーバー側で変換してから照合する。 | filter-expression, フィルター式 |
| AuditActor | 操作を行った本人。ペイロードの `userId` に相当する。認証、OAuth2 フロー、本人による操作では、常に本人が操作者になる。`AuditEventQuery.user_id`、検索時に `user_id` へ解決する `username`、フィルターの `actor.id` は、いずれも操作者を指す。UI では「ユーザー ID（操作者）」「ユーザー名（操作者）」「ログイン試行のユーザー名」と表記する。 | actor, 操作者 |
| AuditTargetUser | 別のユーザーに対する操作の対象。ペイロードの `targetUserId` に相当する。管理者が実行する UserCreated / UserDisabled / GroupMemberAdded の対象ユーザーがこれにあたる。フィルターの `target.id` が指すのはこちらであり、操作者とは区別する。UI では「対象ユーザー」と表記する。 | target, 対象ユーザー |
| ResolvableUserEventPayloadPolicy | 実在するアカウントが必ず特定できるイベント (UserAuthenticated、ConsentGranted、AuthorizationCodeIssued、AuthorizationCodeRedeemed、AccessTokenIssued、RefreshTokenIssued など) のペイロードには、平文かハッシュかを問わずユーザー名を含めないという方針。各 Context のイベントモデルにユーザー名相当のフィールドを設けないことで構造的に保証する。ユーザー名による検索は `AuditEventQuery.username` を User Repository で `user_id` へ解決し、既存の `user_id` 検索に帰着させる。実在しないアカウント名も追跡する必要がある認証失敗イベントに限り、平文の検索属性 `actor.username` を持たせる。 | resolvable-user-event-payload-policy |
| TenantAdministrator | 所属テナントまたは (system_admin の場合) 全テナント横断で監査イベントを読み出す権限を持つ管理者。 |  |

## Authorization Boundary

監査イベントの参照とエクスポートは `AdminAuditEventsRead` 権限 (AuthZEN action `admin:audit_events_read`) を要する。`admin` または `system_admin` ロールを持つ、有効かつ認証済みのユーザーだけが行える。書き込み経路は公開せず、イベントの追加は各 Context の発行経路だけが行う。API アクセストークンでは、ロールに加えて次のスコープをそれぞれの操作に要求する。

| スコープ | 許可する操作 |
|---|---|
| `audit:read` | ListAdminAuditEvents、GetAdminAuditEvent、ExportAdminAuditEvents、GetAdminAuditEventSearchOptions |

変更系のスコープは存在しない。書き込み経路そのものを公開しないためである。

問い合わせは既定で呼び出し元のテナントに閉じる。全テナント横断の参照は、`system_admin` ロールを持つユーザーが制御面 (default テナント) の経路から明示的に要求した場合に限る。ページングのカーソルはテナントと絞り込み条件を束縛するため、別テナントで発行されたカーソルや条件が変わったカーソルは拒否する。

## Design

### Retention

監査レコードは GDPR 第 30 条 (処理活動の記録) を根拠に、追記のみで 7 年間保持する。削除やアーカイブのインターフェースは提供しない。これは SLO ではなく運用上の固定値なので、製品要件ではなく Design に記載する。

### Search attribute registry

検索文法は `AuditSearchRegistry` が宣言する属性、演算子、変換方法の閉じた集合に限る。任意の SQL や JSONPath は受け付けない。個人識別情報にあたる属性は、登録簿が宣言した変換 (要約値化や丸め) を適用してから照合するため、検索用のインデックスに平文は残らない。平文が残るのは `audit_events.payload` だけであり、それも失敗イベントに限って短い保持期間の下で保持する。

## Scenarios

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
