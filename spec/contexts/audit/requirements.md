# Audit Requirements

> This Markdown file is the normative, language-independent home for product requirements. Models and API contracts live in the adjacent TypeSpec source.

## Requirements

### REQ-AUDIT-001: 管理者は監査ログを期間で絞り込み参照・エクスポートできる
- Actor: TenantAdministrator
- Given: roles=["admin"] のユーザー "operator" が管理画面の監査ログを開いている
- Then: 管理者 "operator" が直近 24 時間で監査イベントを絞り込む
- Then: 一覧に所属テナントの監査イベントだけが表示される
- Then: 管理者 "operator" が絞り込み結果をエクスポートする

### REQ-AUDIT-002: workerプロセスが発行した業務イベントも管理者は監査ログで参照できる
- Actor: TenantAdministrator
- Given: CSV インポートの apply が worker プロセスで実行される
- Then: worker プロセスが CSV インポートの apply を実行し UserCreated を発行する
- Then: 管理者が ListAdminAuditEvents で type=UserCreated を検索する
- Then: 発行元プロセス (idmagic-api / idmagic-worker) に関わらずイベントが監査ログに含まれる

### REQ-AUDIT-003: 管理者はworkflow_idとrun_idでLifecycleWorkflowの監査イベントを検索できる
- Actor: TenantAdministrator
- Given: LifecycleWorkflow "leaver-offboarding" の WorkflowRun "run-1" が1つの step で失敗した
- Then: "LifecycleWorkflowRunPartiallyFailed" と "LifecycleWorkflowStepFailed" が発行される
- Then: 管理者が filter に workflow_run.id="run-1" を指定して監査ログを検索する
- Then: "run-1" に紐づくイベントだけが返り、attribute value や email 本文は含まれない

### REQ-AUDIT-004: 管理者は監査ログをページングしながら閲覧でき絞り込み変更でcursorが無効化される
- Actor: TenantAdministrator
- Given: 所属テナントに limit を超える監査イベントが存在する
- Then: 管理者が ListAdminAuditEvents を limit のみで実行して先頭ページを取得する
- Then: 応答は絞り込みに一致する exact total items / total pages / current page / page size を返す
- Then: 応答の Link response header (rel="next") から compact cursor を取得する
- Then: 管理者が取得済みの cursor で次ページを取得する
- Then: 直前のページと重複や欠落なく後続のイベントが返る
- Then: 応答の Link response header (rel="prev") の cursor で前ページへ戻る
- Then: 前ページのイベントが canonical な時系列降順で返る
- Then: 管理者が rel="last" の end anchor cursor で端数を含む最終ページへ移動する
- Then: 管理者が rel="first" の cursor を含まない URL で先頭ページへ戻る
- Alternative (次ページ取得前に category や filter などの絞り込み条件を変更する): cursor は元の絞り込み条件に紐づくため ListAdminAuditEvents は InvalidRequestError を返す → 管理者は先頭ページへ戻って検索し直す
- Alternative (cursor が別テナントで発行された、改ざんされた、または legacy expiry を超過している): ListAdminAuditEvents は InvalidRequestError を返す
- Alternative (絞り込みに一致する event が 0 件である): events は空で total items / total pages / current page は 0 / 0 / 0 を返す → first / prev / next / last Link は返さない
- Alternative (exact count の取得に失敗する): 0 件として成功させず request 全体を server error で失敗させる
- Alternative (実行者が TenantAdministrator ロールを持たない): ListAdminAuditEvents は AccessDeniedError で拒否される

### REQ-AUDIT-005: ListAdminAuditEvents
管理者が所属テナントの監査用 DomainEvent を時系列降順の双方向 keyset pagination で取得する。
SOC / コンプライアンス用途を想定し、期間絞り込み (after / before) と件数上限 (limit)、継続取得
(cursor、Link response header の rel="prev" / rel="next" 経由) と type / user_id / username
(操作者) による絞り込みを提供する。認証イベント (ログイン成功 / 失敗 / MFA / セッション / 集約) も
同一ストアに属し、category (authentication / success / fail / aggregated / user / group / client /
consent / token / tenant / key) で絞り込んでセキュリティ調査や運用調査に使える。さらに q (raw text
部分一致) と filter (registry allowlist の field/operator による構造化式) で汎用検索できる。filter の
field は AuditEventSearchAttribute registry で許可されたもののみで、任意 SQL / JSONPath は受け付けない。
user_id と filter の actor.id は操作者を指し、username は操作者のログイン名を検索時に user_id へ
解決する (hash 化せず、該当ユーザーが無ければ 0 件)。filter の actor.username は実在しないアカウント名も
追跡する認証失敗イベント専用の平文一致検索。filter の target.id は管理操作の対象ユーザーを指す別概念
(AuditActorVsTarget)。system_admin であれば default tenant 経路から全テナント横断、admin は所属テナント内に
閉じる (クエリが tenant_id 入力を取らないため所属テナントに構造的に閉じ、system_admin だけが専用経路から
横断できる)。

### REQ-AUDIT-006: ExportAdminAuditEvents
管理者が監査イベント (認証イベントを含む) を期間 / フィルタ指定でエクスポートする。
SIEM への streaming は別 WI で、本 interface は 1 回分のダウンロードを返すのみ。
フィルタは ListAdminAuditEvents と同じ AuditEventQuery (q / filter / category / type / 期間)。
filter の field は registry allowlist のみ。

### REQ-AUDIT-007: GetAdminAuditEventSearchOptions
管理者が監査イベント検索フォームの event.type / outcome を選択式にするための選択肢一覧を取得する。
値は AuditSearchRegistry / auditEventCategoryTypes と同じ Go 側の単一の正から導出し、UI 側で
ハードコードした allowlist との drift を防ぐ。

### REQ-AUDIT-008: GetAdminAuditEvent
管理者が監査イベント 1 件を ID で取得する。別テナントのイベントは未存在として扱う。
- Postcondition: output.event.tenant_id == context.tenant_id

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| AuditEventSearchRegistry | 監査イベント検索で許可される検索軸 (AuditEventSearchAttribute) の allowlist。field/operator/transform を宣言し、任意 SQL / JSONPath ではなく閉じた検索文法を定める。Go 側の AuditSearchRegistry map が単一の正。 | search-attribute-registry, 検索属性 registry |
| AuditEventFilterExpression | AuditEventQuery.filter の 1 項。registry allowlist の field と operator、値の並びからなる連言 (AND) の 1 要素。PII 属性は平文入力をサーバ側で transform してから照合する。 | filter-expression, フィルタ式 |
| AuditActorVsTarget | 監査イベントのユーザー相関には actor (操作者) と target (対象ユーザー) の 2 つの独立した軸がある。<br>actor は操作を行った当人 (payload の userId 相当。認証・OAuth2 フロー・自身の操作はすべて actor)。<br>target は操作の対象になった別ユーザー (payload の targetUserId 相当。例: 管理者が UserCreated /<br>UserDisabled / GroupMemberAdded を実行した際の対象ユーザー)。AuditEventQuery.user_id / username<br>(検索時に user_id へ解決) と filter の actor.id は同じ actor 概念の別表現であり、filter の target.id とは<br>別概念である。実在しないアカウント名も追跡する認証失敗イベントだけは、別の平文検索属性<br>actor.username で扱う。UI 語彙では actor 系を「ユーザー ID (操作者)」「ユーザー名 (操作者)」<br>「ログイン試行のユーザー名」、target 系を「対象ユーザー」と表記して区別する。 | actor-vs-target, 操作者と対象ユーザー |
| ResolvableUserEventPayloadPolicy | 実アカウントが常に確定するイベント (UserAuthenticated、ConsentGranted、<br>AuthorizationCodeIssued、AuthorizationCodeRedeemed、AccessTokenIssued、RefreshTokenIssued 等) は<br>payload に username (平文・hash とも) を持たない。この制約はイベントを発行する各 context<br>自身の event model が username 相当の field を持たないことで構造的に保証される<br>(例: Authentication.UserAuthenticated の payload は userId のみを持つ)。管理 UI がユーザー名で<br>検索したい場合は AuditEventQuery.username が検索時に UserRepo で user_id へ解決し、既存の<br>user_id 経路で絞り込む。実在しないアカウント名も追跡する必要がある認証失敗イベントだけが、<br>例外として actor.username 検索属性に平文 username を持つ。 | resolvable-user-event-payload-policy |
| TenantAdministrator | 所属テナントまたは (system_admin の場合) 全テナント横断で監査イベントを読み出す権限を持つ管理者。 |  |

## Authorization boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.
