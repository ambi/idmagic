---
context: audit
updated_at: 2026-08-11
---

# Audit Specification

## Overview

authentication、identity-management、oauth2、tenancy、signing-keys、application、
saml / wsfederation など複数 bounded context から発火されるセキュリティ監査イベントの
横断的な read model を所有する。監査イベントの検索・エクスポート管理 API、検索属性
registry、PII 変換方針、保持期間を集約する。イベントの発火元は各 context のままであり、
Audit はそれらを横断して読む窓口を所有する。

The `Audit` context exposes the admin-facing `AdminAuditEventResponse` view over the platform's
audit trail of `DomainEvent`s.

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| AuditEventSearchRegistry | 監査イベント検索で許可される検索軸 (AuditEventSearchAttribute) の allowlist。field/operator/transform を宣言し、任意 SQL / JSONPath ではなく閉じた検索文法を定める。Go 側の AuditSearchRegistry map が単一の正。 | search-attribute-registry, 検索属性 registry |
| AuditEventFilterExpression | AuditEventQuery.filter の 1 項。registry allowlist の field と operator、値の並びからなる連言 (AND) の 1 要素。PII 属性は平文入力をサーバ側で transform してから照合する。 | filter-expression, フィルタ式 |
| AuditActorVsTarget | 監査イベントのユーザー相関には actor (操作者) と target (対象ユーザー) の 2 つの独立した軸がある。<br>actor は操作を行った当人 (payload の userId 相当。認証・OAuth2 フロー・自身の操作はすべて actor)。<br>target は操作の対象になった別ユーザー (payload の targetUserId 相当。例: 管理者が UserCreated /<br>UserDisabled / GroupMemberAdded を実行した際の対象ユーザー)。AuditEventQuery.user_id / username<br>(検索時に user_id へ解決) と filter の actor.id は同じ actor 概念の別表現であり、filter の target.id とは<br>別概念である。実在しないアカウント名も追跡する認証失敗イベントだけは、別の平文検索属性<br>actor.username で扱う。UI 語彙では actor 系を「ユーザー ID (操作者)」「ユーザー名 (操作者)」<br>「ログイン試行のユーザー名」、target 系を「対象ユーザー」と表記して区別する。 | actor-vs-target, 操作者と対象ユーザー |
| ResolvableUserEventPayloadPolicy | 実アカウントが常に確定するイベント (UserAuthenticated、ConsentGranted、<br>AuthorizationCodeIssued、AuthorizationCodeRedeemed、AccessTokenIssued、RefreshTokenIssued 等) は<br>payload に username (平文・hash とも) を持たない。この制約はイベントを発行する各 context<br>自身の event model が username 相当の field を持たないことで構造的に保証される<br>(例: Authentication.UserAuthenticated の payload は userId のみを持つ)。管理 UI がユーザー名で<br>検索したい場合は AuditEventQuery.username が検索時に UserRepo で user_id へ解決し、既存の<br>user_id 経路で絞り込む。実在しないアカウント名も追跡する必要がある認証失敗イベントだけが、<br>例外として actor.username 検索属性に平文 username を持つ。 | resolvable-user-event-payload-policy |
| TenantAdministrator | 所属テナントまたは (system_admin の場合) 全テナント横断で監査イベントを読み出す権限を持つ管理者。 |  |

## Authorization Boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.

## Design

### Retention

Audit records are kept append-only for 7 years, on the basis of GDPR Article 30 (records of
processing activities); no deletion or archival interface exists for them today. This is a fixed
operational value rather than an SLO, so it lives in Design rather than as a product requirement.

### Design Decisions

- Audit event retention is fixed at 7 years under GDPR Article 30 record-keeping, documented as an
  operational value here rather than as a product objective.

## Scenarios

### REQ-AUDIT-001: 管理者は監査ログを期間で絞り込み参照・エクスポートできる
- ACTOR TenantAdministrator
- GIVEN roles=["admin"] のユーザー "operator" が管理画面の監査ログを開いている
- WHEN 管理者 "operator" が直近 24 時間で監査イベントを絞り込む
- THEN 一覧に所属テナントの監査イベントだけが表示される
- WHEN 管理者 "operator" が絞り込み結果をエクスポートする
- THEN 所属テナントの絞り込み結果が export data として返る

### REQ-AUDIT-002: workerプロセスが発行した業務イベントも管理者は監査ログで参照できる
- ACTOR TenantAdministrator
- GIVEN apply 対象の CSV インポートが存在する
- WHEN worker プロセスが CSV インポートの apply を実行し UserCreated を発行する
- WHEN 管理者が ListAdminAuditEvents で type=UserCreated を検索する
- THEN 発行元プロセス (idmagic-api / idmagic-worker) に関わらずイベントが監査ログに含まれる

### REQ-AUDIT-003: 管理者はworkflow_idとrun_idでLifecycleWorkflowの監査イベントを検索できる
- ACTOR TenantAdministrator
- GIVEN LifecycleWorkflow "leaver-offboarding" の WorkflowRun "run-1" が実行中である
- WHEN WorkflowRun "run-1" の1つの step が失敗する
- THEN "LifecycleWorkflowRunPartiallyFailed" と "LifecycleWorkflowStepFailed" が発行される
- WHEN 管理者が filter に workflow_run.id="run-1" を指定して監査ログを検索する
- THEN "run-1" に紐づくイベントだけが返り、attribute value や email 本文は含まれない

### REQ-AUDIT-004: 管理者は監査ログをページングしながら閲覧でき絞り込み変更でcursorが無効化される
- ACTOR TenantAdministrator
- GIVEN 所属テナントに limit を超える監査イベントが存在する
- WHEN 管理者が ListAdminAuditEvents を limit のみで実行して先頭ページを取得する
  - ALT 絞り込みに一致する event が 0 件である → events は空で total items / total pages / current page は 0 / 0 / 0 を返す → first / prev / next / last Link は返さない
  - ALT exact count の取得に失敗する → 0 件として成功させず request 全体を server error で失敗させる
  - ALT 実行者が TenantAdministrator ロールを持たない → ListAdminAuditEvents は AccessDeniedError で拒否される
- THEN 応答は絞り込みに一致する exact total items / total pages / current page / page size を返す
- THEN 応答の Link response header (rel="next") に compact cursor が含まれる
- WHEN 管理者が取得済みの cursor で次ページを取得する
  - ALT category や filter などを変更して元の絞り込み条件の cursor を送る → InvalidRequestError を返し、管理者は先頭ページから検索し直す
  - ALT cursor が別テナントで発行された、改ざんされた、または legacy expiry を超過している → InvalidRequestError を返す
- THEN 直前のページと重複や欠落なく後続のイベントが返る
- WHEN 管理者が Link response header (rel="prev") の cursor で前ページを取得する
- THEN 前ページのイベントが canonical な時系列降順で返る
- WHEN 管理者が rel="last" の end anchor cursor で最終ページを取得する
- THEN 端数を含む最終ページが返る
- WHEN 管理者が rel="first" の cursor を含まない URL で先頭ページを取得する
- THEN canonical な先頭ページが返る
