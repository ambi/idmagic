---
context: application
updated_at: 2026-08-11
---

# Application Specification

## Overview

運用者が「接続する業務アプリケーション」として扱う上位概念を所有する。OIDC client /
SAML SP / WS-Fed RP は Application の protocol binding であり、表示名、アイコン、
ライフサイクル、割当はここに集約する。割当はポータル可視性と
フェデレーション利用可否を fail-closed で制御する。protocol binding の wire 挙動は
各 protocol context が所有し、Application は binding を opaque key で参照する。

`Application` owns the ApplicationCatalog: the tenant-scoped registry of applications an identity
provider issues tokens or assertions for, the sign-in policy (per-application and tenant-default) that
gates each login, and the relation between a catalog entry and its concrete protocol configuration
(OAuth2 client, SAML service provider, or WS-Fed relying party). `domain` holds the aggregate and policy
evaluation rules, `ports`/`usecases` the catalog and policy operations, and `handlers_http` the HTTP
adapter; `module.go` composes them for the router.

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| Application | 運用者が接続・割当・監査する業務アプリケーション。federated / service Application は最大1個の protocol 設定を持つ。 | アプリケーション, Application |
| ApplicationProtocol | Application が利用する単一の protocol 設定への型付き参照。OAuth2Client、SamlServiceProvider、WsFedRelyingParty のいずれか1個を指す。 | application_protocol |

## Authorization Boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.

## Design

### Internal Interfaces

#### AssignApplicationDesiredState
呼び出し元 bounded context (IdManagement の LifecycleWorkflow 等) が Application への
user 割当を desired-state で付与する内部インターフェース。HTTP には公開せず、同一プロセス内の
Go 呼び出しとして各 context の usecase から使う。既に同じ (id, user_id) の割当が
指定どおりの visibility で存在する場合は変更せず changed=false を返す (冪等)。呼び出し元は
同一テナントの id / user_id だけを渡す。

#### UnassignApplicationDesiredState
呼び出し元 bounded context が Application への user 割当を desired-state で解除する内部
インターフェース。割当が存在しない場合も no-op (changed=false) で正常終了する (冪等)。

### Sign-in policy evaluation

`AppSignInPolicy` is a tenant/application-scoped ordered set of `SignInRule`s that ApplicationCatalog
owns and evaluates on every federation start (OIDC authorize, SAML SSO, WS-Fed sign-in) before a token or
assertion is issued — the same gate point as the existing per-application protocol-binding check, so
policy evaluation cannot be bypassed by choosing a different protocol entry point. An earlier
version accepted free-text ACR/factor strings and free-text network/device conditions but only ever
enforced them as a fail-closed rejection, producing configuration fields that looked functional but were
never actually evaluated; the model was replaced with values the evaluator can and does check.

Required authentication strength is a constrained `RequiredAuthnStrength` enum — `Password` or `Mfa` —
mapped 1:1 to internal ACR URNs and AMR values, rather than free text, since only two ACR values exist in
practice and an unconstrained string invites misconfiguration. Of the original free-text conditions,
only the two that the evaluator can actually check were kept structured: `reauth_max_age_seconds`
(evaluated against authentication/step-up recency) and `network_allow_cidrs` (the request's client IP
checked against admin-supplied, save-time-validated CIDRs); free-text device conditions were dropped
entirely rather than kept as an unenforced input.

Evaluation is fail-closed throughout. OIDC can route an insufficient-strength result to the existing
step-up flow; SAML and WS-Fed instead halt the protocol transaction outright with an explicit rejection
reason, since neither has a step-up mechanism to redirect to yet. A non-empty CIDR allowlist that the
client IP doesn't match, or a request where the client IP can't be determined at all, is a hard rejection
rather than a step-up opportunity.

### Tenant default policy composition

`TenantDefaultSignInPolicy` lets a tenant set one baseline sign-in policy for every application that
doesn't define its own, using the same `SignInRule` vocabulary and evaluator as per-application policy so
no second policy language exists. It is owned by ApplicationCatalog rather than `Tenancy`, since it is
conceptually about how applications are signed into, not about the tenant aggregate itself, keeping
sign-in policy ownership in one place.

The relationship between default and per-application policy is **override, not composition**: if an
application defines any enabled rules, those rules entirely replace the tenant default for evaluation
purposes; otherwise the default applies as-is. `EffectiveSignInRules(default, app)` selects one side or
the other before handing rules to the same fail-closed evaluator used for per-application policy. An
initial design composed the two as a floor the application couldn't weaken, but that made the effective
policy hard for admins to read at a glance and required a separate exemption flag for legitimate low-risk
relaxation; override was chosen for a single, directly-inspectable effective policy per application,
consistent with the principle that the per-application policy has final say.

Because override lets an application go below the tenant default, `AppSignInPolicyResponse` carries a
`weaker_than_default` flag — set when the override reduces required strength, loosens or drops
re-auth recency, or widens the allowed network — computed by `AppPolicyWeakerThanDefault(default, app)`
and surfaced as a UI warning rather than a block, since the underlying design goal was to make deliberate
relaxation easy, not to forbid it. New tenants start with an empty (allow-all) default so introducing
this feature changes no existing tenant's behavior until an admin opts in; auto-applying a strict default
such as mandatory MFA at migration time was rejected as too likely to cause a mass lockout. Because the
default lives as an ordinary table row, clearing its rules or deleting the row is an immediate,
reversible rollback to allow-all with no schema change involved.

### Application/protocol relation

An Application has at most one protocol configuration, fixed at creation time: a `weblink` application
has none, and a `federated`/`service` application has exactly one of OAuth2 client, SAML service
provider, or WS-Fed relying party. Reconnecting, detaching, or changing protocol type afterward is not
supported, reflecting that no real creation/edit flow ever needed an application to carry more than one
protocol binding, even though the original JSON-array binding model was built to allow it.

Each protocol table (`oauth2_clients`, `saml_service_providers`, `wsfed_relying_parties`) keeps its own
existing primary key and gains a nullable, unique `application_id`; a non-`NULL` value is a composite
foreign key that also pins tenant and a fixed protocol discriminator, so the database itself rejects two
protocol rows claiming the same Application, a cross-table duplicate claim, or a tenant/type mismatch —
none of which the prior JSON-array-of-bindings representation could express as a constraint, which had
instead required a full per-tenant scan to resolve. `NULL` represents a legitimate catalog-external
record: protocol configurations created through Dynamic Client Registration or trust-management APIs
that were never meant to appear in the Application catalog, which is why every protocol config isn't
required to carry an Application.

Because catalog creation spans two records (the Application row and the protocol row's `application_id`
relation), both commit in one transaction: if the second half fails, no orphaned catalog-visible
Application is left behind, and the protocol row that does exist is still valid as a catalog-external
record. Deleting an Application cascades to delete its owned protocol configuration, but a protocol
config that's Application-owned rejects direct deletion through the lower-level protocol management API
as a conflict — deletion has to go through the Application it belongs to.

The OAuth2 protocol table was renamed from the generic `clients` to `oauth2_clients` to match the SAML
and WS-Fed table names (`saml_service_providers`, `wsfed_relying_parties`), which already used
protocol-specific, domain-standard terms rather than a name generic enough to be confused with any other
kind of client.

### Design Decisions

- Application sign-in policy (`AppSignInPolicy`) evaluates a structured, evaluator-checkable rule set —
  a constrained `RequiredAuthnStrength` enum, `reauth_max_age_seconds`, and `network_allow_cidrs` — fail-
  closed at the same federation gate for every protocol, replacing an earlier free-text ACR/network/device
  model whose fields were never actually enforced
  ([ADR-079](../../../decisions/ADR-079-application-sign-in-policy-evaluation.md)).
- The tenant-wide default sign-in policy overrides, rather than composes with, an application's own
  policy, keeping a single, directly-inspectable effective policy per application
  ([ADR-081](../../../decisions/ADR-081-tenant-default-sign-in-policy-composition.md)).
- An Application has at most one protocol configuration, fixed at creation time and enforced by a
  composite foreign key from the owning protocol table rather than a JSON-array-of-bindings model
  ([ADR-138](../../../decisions/ADR-138-relate-single-application-protocol-by-foreign-key.md)).

## Scenarios

### REQ-APPLICATION-001: 管理者はアプリ詳細でIdMagic側の連携設定を確認できる
- ACTOR TenantAdministrator
- GIVEN OIDC または SAML protocol を持つ Application が存在する
- WHEN 管理者が Application の詳細画面を開く
- THEN 画面は IdMagic に登録済みの RP / SP 情報と、相手側へ投入する IdMagic の discovery または metadata を区別して表示する
- THEN OIDC application には OpenID Discovery URL と client_id を表示する
- THEN SAML application には IdP metadata URL、entityID、SSO URL、SLO URL、署名証明書を表示する
- THEN client secret は作成・互換ローテーション・追加発行の成功応答以外では表示しない

### REQ-APPLICATION-002: 管理者は通常設定と独立したセクションでclient secretを管理できる
- ACTOR TenantAdministrator
- GIVEN secret-based OIDC protocol を持つ Application が存在する
- GIVEN 期限なし legacy credential が1件 Active である
- WHEN 管理者が Application の編集画面を開く
- THEN client_id は通常の OIDC 設定カード内に参照項目として表示される
- THEN credential 一覧と追加発行・個別失効操作は通常設定の保存 form 外にある専用トップレベルセクションへ表示される
- WHEN 管理者が90日の期限を選んで新 secret を追加発行する
  - ALT Active credential が既に2件存在する → 追加発行操作は利用不可となり、先に既存 credential を失効する案内を表示する
- THEN 新 secret は一度だけ表示され、一覧には作成日・有効期限・Active 状態が表示される
- WHEN 管理者が旧 credential を個別失効する
  - ALT credential が Expired または Revoked である → 個別失効操作は表示しない
- THEN その credential だけが Revoked 状態になる

### REQ-APPLICATION-003: API token発行者はaccount scope内で自身のportal applicationだけを操作できる
- ACTOR SelfApiClient
- GIVEN client は対象 tenant の active User に固定された有効な API access token を提示している
- WHEN client が account:read scope で自身に割り当てられた application と保存済み順序を要求する
  - ALT token の tenant または user_id が操作対象と一致しない → 操作は AccessDeniedError で拒否される
- THEN client 自身の application と保存済み順序だけが返る
- WHEN client が account:write scope で自身の application 順序の保存を要求する
  - ALT client が account:read scope だけを持つ → 操作は AccessDeniedError で拒否される
- THEN client 自身の application 順序が保存される

### REQ-APPLICATION-004: management API clientはApplication scope内の操作だけを実行できる
- ACTOR ManagementApiClient
- GIVEN client は対象 tenant の有効な API access token を提示している
- WHEN client が Application、category、assignment、または tenant default sign-in policy の操作を要求する
  - ALT applications:read だけで Application の変更を要求する → 操作は AccessDeniedError で拒否される
  - ALT token の tenant と request tenant が一致しない → 操作は AccessDeniedError で拒否される
- THEN applications:read scope は Application、category、assignment の参照だけを許可する
- THEN applications:write scope は Application の作成・protocol 設定更新・削除を許可する
- THEN settings:read または settings:write scope は tenant default sign-in policy の対応する操作種別だけを許可する

### REQ-APPLICATION-005: 管理者はApplicationのSAML protocol設定を更新できる
- ACTOR TenantAdministrator
- GIVEN 管理者が SAML protocol を持つ Application の編集画面を開いている
- WHEN 管理者が ACS URL、署名方針、claim 規則、IdP profile割当を更新する
  - ALT AuthnRequest 署名必須だが検証可能な証明書がない → 更新は InvalidRequestError で拒否される
- THEN SAML service provider 設定だけが同一テナント内で更新される

### REQ-APPLICATION-006: 管理者はApplication単位でclaim releaseを絞り込める
- ACTOR TenantAdministrator
- GIVEN 同一テナントに OIDC Application "payroll" と "directory" が存在し、いずれも employee_number (visibility=SelfReadable) を含む同じ User 属性を参照できる
- WHEN 管理者が "payroll" の claim release rule に claim_type="employee_number"、source=user_attribute、source_key="employee_number" を追加して保存する
  - ALT 管理者が visibility=Private の属性 (例 password 関連の内部属性) を source_key に指定する → 更新は InvalidRequestError で拒否される (claim_release_rules_within_floor)
  - ALT 管理者が reserved claim type (例 \"sub\"、\"iss\") を claim_type に指定する → 更新は InvalidRequestError で拒否される (claim_release_rules_within_floor)
- THEN "ApplicationClaimMappingUpdated" が発行される
- THEN "payroll" 向けに発行される ID Token / assertion には employee_number claim が含まれる
- THEN "directory" は自身の rule を更新していないため employee_number claim を含まない

### REQ-APPLICATION-007: 管理者は管理画面でアプリケーションと単一protocolを構成できる
- ACTOR TenantAdministrator
- GIVEN roles=["admin"] のユーザー "operator" が管理画面のアプリケーション一覧を開いている
- WHEN 管理者 "operator" が confidential なアプリケーション "portal" (type=oidc) を作成する
- THEN 作成応答だけが生成された client_secret を一度だけ含む
- WHEN 管理者 "operator" がアプリケーション "portal" の OIDC 設定 (redirect_uris / scope) を編集する
- THEN OIDC 設定が保存される
- WHEN 管理者 "operator" がアプリケーション "portal" をユーザー "alice" に割り当てる
  - ALT 別テナントまたは存在しない subject を指定する → InvalidRequestError で拒否される
- THEN "alice" への割当が保存される
- WHEN 管理者 "operator" がアプリケーション "portal" を取得する
  - ALT 別テナントの管理者が同じ id を指定する → InvalidRequestError で拒否される
- THEN 同一テナントのアプリケーションだけが返る
- WHEN 管理者 "operator" がアプリケーション "portal" を削除する
- THEN "ApplicationCreated"、"ApplicationAssigned"、"ApplicationDeleted" が発行される

### REQ-APPLICATION-008: 管理者はApplicationアイコンをアップロード削除できる
- ACTOR TenantAdministrator
- GIVEN 管理者が Application 編集画面を開いている
- WHEN 管理者が PNG / JPEG / WebP / GIF の 256KiB 以下の画像をアップロードする
  - ALT 非画像または上限超過ファイルをアップロードする → InvalidRequestError で拒否され、既存アイコンは置き換わらない
- THEN Application は icon_object_key と内部 icon_url を持つ
- WHEN 管理一覧・詳細・利用者ポータルが icon_url を取得する
  - ALT 別テナントの application_id と id で同じアイコンを取得する → アセットは存在しないものとして扱われ InvalidRequestError で拒否される
- THEN icon_url は IdP の配信 URL を指す
- WHEN 管理者がアイコンを削除する
- THEN icon_object_key と icon_url は空になる

### REQ-APPLICATION-009: 管理者はアプリケーション別サインインポリシーを設定できる
- ACTOR TenantAdministrator
- GIVEN 管理者が Application 編集画面を開いている
- GIVEN Application は OIDC / SAML / WS-Fed のいずれか単一の protocol を持つ
- WHEN 管理者が MFA 必須と再認証を求めるまでの時間 (秒) を要求する sign-in policy を保存する
  - ALT 管理者以外が policy を更新する → AccessDeniedError で拒否される
- THEN AppSignInPolicyUpdated が発行される
- WHEN 単要素セッションの利用者が対象 Application にアクセスする
- THEN システムは token / assertion 発行前に policy を評価する (強制点は OAuth2.Authorize)
  - ALT 許可 CIDR に含まれないクライアント IP、またはクライアント IP を取得できない → federation を拒否し、AppAccessDeniedByPolicy を発行する
- THEN step-up が可能な経路では step-up を要求し、昇格後に federation を完了する

### REQ-APPLICATION-010: 管理者はテナントデフォルトサインインポリシーを設定し全アプリに適用できる
- ACTOR TenantAdministrator
- GIVEN roles=["admin"] のユーザー "operator" がサインインポリシー画面を開いている
- GIVEN テナントに OIDC protocol を持つ複数の Application が存在し、いずれも個別 sign-in policy を持たない
- WHEN 管理者が MFA 必須、将来の強制開始日時、enrollment bypass 猶予、管理者承認を要求するテナントデフォルトサインインポリシーを保存する
  - ALT 管理者がルールを空にして保存する → TenantDefaultSignInPolicyUpdated を発行し、独自ポリシーを持たない Application の federation に追加要件を課さない
- THEN TenantDefaultSignInPolicyUpdated が発行される
- THEN 画面は active user の MFA 未登録人数と強制時のロックアウト影響を表示する
- WHEN 単要素セッションの利用者が個別ポリシーを持たない Application にアクセスする
- THEN システムはデフォルトポリシーを適用し step-up を要求する
- WHEN 管理者が対象 Application にデフォルトより弱い sign-in policy を保存する
- THEN システムはデフォルトより弱い旨の警告を表示するが保存を許可する
- THEN 当該 Application では弱い policy を適用し、他の Application ではデフォルトの MFA 必須を引き続き適用する
- WHEN 管理者が Application の編集画面を開く
- THEN 画面はテナントデフォルト・この Application の上書き・最終的に適用される policy を区別して表示する

### REQ-APPLICATION-011: 未割当のsubjectはprotocol経由でアプリケーションへフェデレーションできない
- ACTOR TenantAdministrator
- GIVEN アプリケーション "portal" にユーザー "alice" は割り当てられていない
- WHEN "alice" が "portal" への federation を試みる (強制点は OAuth2.Authorize)
  - ALT 管理者が事前に "portal" へ "alice" を visibility=visible で割り当てる → "alice" は federation を完了できる
- THEN 未割当のため federation は拒否される

### REQ-APPLICATION-012: hidden割当はポータル一覧から除外されるがprotocol利用は許可される
- ACTOR TenantAdministrator
- GIVEN 管理者が "portal" にユーザー "alice" を visibility=hidden で割り当てている
- WHEN "alice" が自分のポータルアプリ一覧 (ListMyApplications) を取得する
- THEN 一覧に "portal" は含まれない
- WHEN "alice" が "portal" への federation を試みる (強制点は OAuth2.Authorize)
- THEN hidden 割当があるため federation を完了できる

### REQ-APPLICATION-013: adminロールを持たない利用者はApplicationを操作できない
- ACTOR AuthenticatedSelf
- GIVEN "alice" は admin ロールを持たない認証済みユーザーである
- WHEN "alice" が ListAdminApplications を呼び出す
- THEN AccessDeniedError で拒否される

### REQ-APPLICATION-014: desired-state割当はgroup経由の割当を変更しない
- ACTOR TenantAdministrator
- GIVEN "alice" は dynamic group 経由で "portal" への group assignment (subject_type=group) を既に持つ
- WHEN IdManagement の LifecycleWorkflow が "alice" に対して AssignApplicationDesiredState を呼び出す
  - ALT 個人の direct assignment が既に指定どおりの visibility で存在する → 変更を行わず changed=false を返す
- THEN "alice" 個人への direct user assignment (subject_type=user) が作成される
- THEN group assignment (subject_type=group) の行は変更されない
- WHEN LifecycleWorkflow が後から UnassignApplicationDesiredState を呼び出す
- THEN direct user assignment だけが削除され、group assignment は残る
- THEN federation は引き続き許可される
