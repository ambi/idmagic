---
context: audit
updated_at: 2026-08-11
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
| AuditActorVsTarget | 監査イベントでユーザーを関連付ける軸には、操作者と対象ユーザーの 2 種類がある。操作者は操作を行った本人で、ペイロードの `userId` に相当する。認証、OAuth2 フロー、本人による操作では、すべて本人が操作者になる。対象ユーザーは別のユーザーに対する操作の対象で、ペイロードの `targetUserId` に相当する。たとえば、管理者が UserCreated / UserDisabled / GroupMemberAdded を実行した場合のユーザーである。`AuditEventQuery.user_id`、検索時に `user_id` へ解決する `username`、フィルターの `actor.id` は、いずれも操作者を表す。フィルターの `target.id` は対象ユーザーを表すため、これらとは区別する。実在しないアカウント名も追跡する認証失敗イベントだけは、平文の検索属性 `actor.username` を使用する。UI では操作者側を「ユーザー ID（操作者）」「ユーザー名（操作者）」「ログイン試行のユーザー名」、対象側を「対象ユーザー」と表記する。 | actor-vs-target, 操作者と対象ユーザー |
| ResolvableUserEventPayloadPolicy | UserAuthenticated、ConsentGranted、AuthorizationCodeIssued、AuthorizationCodeRedeemed、AccessTokenIssued、RefreshTokenIssued など、実在するアカウントが必ず特定できるイベントのペイロードには、平文かハッシュかを問わずユーザー名を含めない。この制約は、各 Context のイベントモデルにユーザー名相当のフィールドを設けないことで構造的に保証する。たとえば、`Authentication.UserAuthenticated` のペイロードは `userId` だけを持つ。管理 UI でユーザー名による検索が必要な場合は、検索時に `AuditEventQuery.username` を User Repository で `user_id` に解決し、既存の `user_id` 検索を使用する。実在しないアカウント名も追跡する必要がある認証失敗イベントに限り、例外として平文の検索属性 `actor.username` を持たせる。 | resolvable-user-event-payload-policy |
| TenantAdministrator | 所属テナントまたは (system_admin の場合) 全テナント横断で監査イベントを読み出す権限を持つ管理者。 |  |

## Authorization Boundary

認可の意味はアプリケーションとそのテストで強制する。本仕様は API の認証を記述するが、ポリシーの DSL は意図的に定義しない。ポリシー言語を採用する前に、別の作業項目で Cedar を評価する。

## Design

### Retention

監査レコードは GDPR 第 30 条 (処理活動の記録) を根拠に、追記のみで 7 年間保持する。現時点では削除やアーカイブを行うインターフェースを提供しない。これは SLO ではなく運用上の固定値なので、製品要件ではなく Design に記載する。

### Design Decisions

- 監査イベントの保持期間は、GDPR 第 30 条が定める記録の保存に基づき 7 年で固定する。製品目標ではなく、運用上の値としてここに記録する。

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
