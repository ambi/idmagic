---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-23
depends_on: [wi-273]
---

# アカウント自己管理 API を API アクセストークンの scope で制御できるようにする

## Motivation

統一 API アクセストークン（[wi-273]）と OAuth 2.0 access token は、管理 API だけでなく、公開できる API は一律 scope を定義して統一的に制御する方針。`/api/admin/*`（管理）と SCIM は [wi-273]（安定リソース）と [wi-274]（アプリ領域）でカバーするが、**アカウント自己管理 API（`/api/account/*`、`AuthenticatedSelf`、約28エンドポイント）** が未カバー。これらもユーザー本人を表す OAuth access token または personal access token で叩けるよう、`account:*` scope を定義・enforcement する。

GitHub PAT の user scope に相当する位置づけで、トークン所有者自身のプロフィール・セッション・MFA・consent・ポータルタイルを read/write できる。security-sensitive な facet（MFA / session / consent の変更）は最小権限のため専用 scope に分ける。

## Scope

- `spec/contexts/api-tokens.yaml` の `models.ApiTokenScope`（account 系 scope 値を追記）。
- `spec/contexts/identity-management.yaml`・`authentication.yaml` 等、`/api/account/*` interface を持つ context の `authorization`（`AuthenticatedSelf` を scope でも満たせるよう `SelfApiClient` principal + policy を追加）と対象 interface の `access.policies`。
- 管理発行 API access token を通常 OAuth access token と同じ RFC 9068 JWT に統一し、既存
  OAuth2 `/introspect`（RFC 7662）と `/revoke`（RFC 7009）へ統合する。
- Authorization Code + PKCE / Device Authorization で `account:*` を要求・同意・発行できるようにし、OAuth access token と PAT を同じ account resource server policy で認可する。User subject を持たない client_credentials には `account:*` を発行しない。

## Out of Scope

- 管理 API・SCIM の scope（[wi-273] / [wi-274]）。
- OAuth2/OIDC の login / consent / par / userinfo 等の既存挙動変更。authorize / token / device は account scope 発行に必要な範囲、`/introspect` と `/revoke` は PAT の RFC 対応範囲だけを対象に含める（userinfo は既存の `openid` scope で制御）。
- 一般ユーザー向けの PAT 発行・一覧・失効 API / UI。既存どおり管理者が発行し、token の `user_id` は発行した管理者本人に固定する。

## Plan

- 適用標準の棚卸し:
  - RFC 6750 — Authorization header の Bearer / DPoP 提示、`WWW-Authenticate` の `invalid_token` / `insufficient_scope`、query token 禁止。
  - RFC 7009 — 既存 `/revoke` で管理発行 JWT の lifecycle record を即時失効し、未知・既失効 token は 200 no-op。
  - RFC 7662 — 既存 `/introspect` で active / scope / sub / aud / iat / exp / jti / cnf を返し、inactive は active=false のみ。
  - RFC 8414 — discovery に既存 introspection / revocation endpoint と対応 auth method を広告。
  - RFC 9700 / RFC 8707 — PAT を realm の IdMagic API audience に固定し、別 realm / resource で拒否。長寿命 bearer の盗用対策として DPoP sender constraint を選択可能にする。
  - RFC 9449 — 発行時に任意の `dpop_jkt` を束縛し、束縛 token は DPoP scheme + htm / htu / iat / jti / replay / key thumbprint を検証する。
  - RFC 9728 — realm の IdMagic API protected resource metadata に account / admin / SCIM scope と bearer method を公開する。
  - RFC 9068 — OAuth grant と管理発行の wire format、署名鍵、claim profile、検証器を統一する。
    RFC 9701 の JWT introspection response、RFC 8705 mTLS sender constraint は optional で本 WI では追加しない。

- 提案する account scope 分割（`<resource>:<action>`、`account:` 名前空間、SCIM の三段 scope に倣う）:
  - `account:read` — summary, profile(GET), security, signin_activity, data_export, applications(GET), applications/order(GET), consents(GET), sessions(GET), `/api/auth/account`
  - `account:write` — profile(PATCH), email/change_request, applications/order(PUT)
  - `account:mfa:write` — TOTP / WebAuthn の enroll・remove、recovery-codes
  - `account:sessions:write` — 自セッションの revoke（{id}/revoke, revoke_others）
  - `account:consents:write` — 自 consent の revoke
- 判断が要る境界:
  - `POST /api/auth/change_password`（自パスワード変更）は sensitive なため専用 `account:password:write` とする。
  - step_up（start/complete/webauthn/challenge）は対話的な再認証フローで PAT 経路にそぐわないため、scope 対象外（認証機構）とする。
  - email verify context / confirm は未認証 token-possession フローのため PAT scope 対象外とする。
- [wi-273] の token は tenant + scope のみで本人を識別できないため、`ApiToken` / `ApiTokenPrincipal` に
  `user_id` を追加し、発行時の管理者本人へ固定する。管理発行 token は tenant + user 一致と active な
  `jti` record を共通認証境界で fail-closed に検証する。
- 管理発行は OAuth client の事前登録を要求しない。各 realm の Authorization Server が所有する
  built-in public client `idmagic-api-token` を自動的に `client_id` とし、個々の credential は `jti` で
  識別する。アプリ固有の actor identity が必要な連携は通常の OAuth grant と専用 client を使う。
- Bearer PAT は cookie 認証ではないため CSRF 対象外とし、正しい PAT だが scope 不足は 403、無効・失効・期限切れ PAT は 401 とする。
- RFC 7662 introspection は認証済み Resource Server に active / scope / sub (`user_id`) / client_id /
  aud / iat / exp / jti / cnf を返し、無効・失効・期限切れ・realm 不一致は `active=false` のみとする。
  RFC 7009 revocation は built-in public client の `client_id` と token を提示して管理 record を即時失効し、
  未知・既失効 token も 200 no-op とする。管理者による既存 DELETE 失効も維持する。
- OAuth Authorization Code + PKCE / Device Authorization の user-bound grant は client の許可 scope と user consent の積集合で `account:*` を発行できる。client_credentials / User subject のない token exchange では `account:*` を fail-closed で拒否する。

## Tasks

- [x] T001 [Design] account scope 分割を確定（password は専用 scope、step_up / email verify は対象外）。
- [x] T001b [Design] 管理発行 access token に適用する OAuth RFC を棚卸しし、SCL standards と ADR に採否・境界を記録。
- [x] T001c [Design] wire format を RFC 9068 JWT に統一し、管理発行は realm built-in public client で完結させる。
- [x] T002 [SCL] `ApiTokenScope` enum と token subject 契約を更新し、account 系 scope を追記。
- [x] T003 [SCL] account interface の `access` に `SelfApiClient` の scope policyを付与（`AuthenticatedSelf` と OR）。
- [x] T004 [Domain] 管理発行 token の subject / client / account scope 語彙を test-first で追加。
- [x] T005 [UseCase] 発行者 subject を token に固定し認証 principal へ返す契約を test-first で追加。
- [x] T006 [Adapter] PostgreSQL / memory 永続化と HTTP 共通認証・CSRF 境界を test-first で結線し、全対象 route の scope 対応を検証。
- [x] T007 [UI] account scope を API token 発行 UI から選択可能にする。
- [x] T008 [OAuth Adapter] 管理発行 JWT を RFC 7662 `/introspect` と RFC 7009 `/revoke` へ test-first で統合する。
- [x] T009 [OAuth UseCase/Adapter] Authorization Code + PKCE / Device Authorization で account scope を発行し、client_credentials では拒否する test-first 契約を追加。
- [x] T010 [Security Adapter] RFC 6750 error、audience、任意 DPoP sender constraint、RFC 9728 metadata を test-first で統合する。
- [x] T011 [Verify] `just yaml-check` / `just scl-render` / `just verify`、実 HTTP クライアント相当の adapter test で scope 別アクセスと introspection / revocation を確認。

## Verification

- `just yaml-check` / `just scl-render`（account scope が OpenAPI に反映）。
- 実クライアント: `account:read` トークンで profile 取得可・更新不可、`account:mfa:write` で MFA 変更可、scope 外が 403。
- RFC client: `/introspect` で active PAT の scope / sub を取得でき、失効後は `active=false` のみ。`/revoke` は初回・再実行・未知 PAT のいずれも 200。

## Risk Notes

- **self 判定の結線（中）**: PAT を「ユーザー本人」として `AuthenticatedSelf` に通す設計を誤ると、他人の account を触れる横断アクセスになりうる。tenant + subject 一致を fail-closed で担保する。
- sensitive facet（MFA/session/consent/password）の scope 分割を誤ると最小権限が崩れる。
- **fuzz/property test 判断**: 不採用。追加する入力は既存の固定 enum scope と UUID subject であり、再帰文法や組み合わせ爆発はない。scope-route 対応の表駆動 adapter test と tenant + subject の境界 test で攻撃面を直接検証する。

## Completion

- **Completed At**: 2026-07-23
- **Summary**: 管理画面発行 token と OAuth grant 発行 token の wire format を RFC 9068 JWT に統一した。管理画面発行では OAuth client の事前登録を不要とし、realm 組み込み public client `idmagic-api-token`、`sub=user_id`、credential 識別子 `jti` を使用する。account scope を resource server の route 単位で enforce し、管理発行 token の lifecycle record を RFC 7662 introspection と RFC 7009 revocation に統合した。Authorization Code + PKCE / Device Authorization の user-bound grant は account scope を発行でき、client credentials と user subject のない token exchange は拒否する。RFC 6750 error、realm API audience、任意 DPoP、RFC 9728 metadata も共通境界へ結線した。
- **Affected Guarantees State**: tenant・subject・audience・active lifecycle record の一致を fail-closed で検証する。管理発行 token の subject は発行管理者本人から変更できず、account API は本人の resource のみを操作する。sensitive facet は専用 scope を要求し、対話的 step-up と email verify token-possession flow は API token 対象外のまま維持する。
- **Verification Results**:
  - `just yaml-check-scl` / `just scl-render` — passed（23 SCL file、派生 HTML / JSON Schema / OpenAPI を再生成）
  - `just verify-go` — passed（lint 0 issues、全 Go package の race test green）
  - `just verify-ui` — passed（77 test files / 425 tests、typecheck・lint・production build green）
  - SCIM・管理 API・account API の RFC 6750 adapter tests — passed（invalid token 401、insufficient scope 403、`WWW-Authenticate` を確認）
  - introspection / revocation、管理発行 JWT lifecycle、OAuth grant rejection、DPoP / protected resource metadata tests — passed
  - `just yaml-check` — WI-275/SCL は valid。既存 `work-items/done/wi-216-dynamic-group-rule-builder-ui.md` の completion metadata 不足のみで repository-wide check は失敗（本 WI の変更対象外）。
- **Evidence**:
  - 実行日: 2026-07-23
  - 実行環境: macOS local workspace
  - 保存先: repository 内の SCL、ADR-137、実装・adapter test、生成済み specification artifacts
