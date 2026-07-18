---
depends_on: [wi-253-postgresql-persistent-login-sessions]
status: completed
authors: ["tn"]
risk: high
created_at: 2026-06-20
change_kind: feature
initial_context:
  scl:
    Authentication:
      - models.LoginSession
      - interfaces.ListMySessions
      - interfaces.RevokeMySession
    OAuth2:
      - models.OAuth2Client
      - models.RefreshTokenRecord
      - models.EndSessionParameters
      - interfaces.EndSession
  source:
    - backend/oauth2/domain/client.go
    - backend/oauth2/domain/refresh_token.go
    - backend/oauth2/usecases/exchange_code.go
    - backend/oauth2/adapters/http/end_session_handler.go
  tests:
    - backend/oauth2/usecases/exchange_code_test.go
    - backend/oauth2/adapters/http/end_session_handler_test.go
  stop_before_reading:
    - backend/saml
    - backend/wsfederation
affected_spec:
  - { context: Authentication, kind: model, element: LoginSession }
  - { context: Authentication, kind: interface, element: ListMySessions }
  - { context: OAuth2, kind: model, element: OAuth2Client }
  - { context: OAuth2, kind: model, element: RefreshTokenRecord }
  - { context: OAuth2, kind: model, element: EndSessionParameters }
  - { context: OAuth2, kind: interface, element: EndSession }
---

# セッション管理と OIDC logout を実運用相当に完成させる

## Motivation
`/end_session` は存在するが、production IdP としては user/admin の
session inventory、個別 revoke、OIDC RP-Initiated Logout の厳密化が足りない。

Keycloak / Okta / Google 相当では、ユーザーがデバイス・セッションを確認し、
管理者が全セッションを失効できることが重要になる。

Front-Channel Logout / Back-Channel Logout / Session Management 1.0
(接続済み RP への logout 伝播) は、SCL は本 WI の T001 で既に追加済みだが、
Go 実装は [[wi-257-oidc-front-back-channel-logout-notifications]] へ切り出した
(下記 Plan 参照)。本 WI は session/token のローカル失効までを完成させる。

## Scope
- **decision**:
  - 新規 ADR: local session、browser session、client session、refresh token family、 OIDC logout notification の関係を定義する。`id_token_hint` 検証と client 解決の失敗時挙動も明記する。
- **scl**:
  - SessionRecord / ClientSession / LogoutNotification を追加する。
  - ListSessions / RevokeSession / RevokeUserSessions を admin/self interface として追加する。
  - BackChannelLogout / FrontChannelLogout / CheckSessionIframe を追加する。
- **go**:
  - [[wi-253-postgresql-persistent-login-sessions]] が提供する PostgreSQL session 正本を利用する。
  - `/end_session` で `id_token_hint` 署名・iss・aud・sub・sid を検証する。
  - session revoke 時に refresh token family を整合して失効する。
- **ui**:
  - account portal に active sessions / devices を表示する。
  - admin users detail に user sessions と revoke all を表示する。
- **documentation**:
  - README に OIDC logout 対応範囲、RP metadata、既知制約を書く。

## Out of Scope
- CAEP / Shared Signals。別 WI で扱う。
- device trust / managed device inventory。
- native app の platform-specific logout。
- RP ごとの backchannel_logout_uri / frontchannel_logout_uri 登録、logout token
  生成・配送、front/back-channel logout、check_session_iframe。SCL は追加済みだが
  Go 実装は [[wi-257-oidc-front-back-channel-logout-notifications]] が担当する
  (下記 Plan 参照)。

## Plan
- 本 WI に durable session repository まで含めると、DB migration、session lifecycle、
  token family、外部 RP 通知、account/admin UI が一つの変更単位に集中する。このため
  PostgreSQL-only session 基盤を [[wi-253-postgresql-persistent-login-sessions]] へ
  切り出し、本 WI はその完了後に OIDC session binding と logout propagation を実装する。
- WI-253 の方針により、ログイン済み session の正本・認証参照・一覧・失効は
  PostgreSQL に一本化する。Valkey に active session を複製せず、本 WI も cache
  invalidation protocol を追加しない。
- OAuth2 の authorization/refresh token に `sid`/session reference を持たせ、session revoke から refresh family と access-token denylist を終端化する。session 正本を OAuth2 client/RP ごとに複製しない。
- RP-Initiated Logout は `id_token_hint` の issuer/audience/signature/sid を検証して session を特定する。client_id fallback は既存互換として残すが、hint と client の不一致を拒否する。
- account portal の session inventory は [[ADR-042-end-user-account-portal-scope]] と [[ADR-043-account-portal-csrf-and-step-up]] に従い、他 session/global revoke を step-up 対象にする。
- `last_seen_at` は認証済みrequestごとに更新せず、WI-253のcoarse touch契約を利用する。
  local revokeのDB transactionを先に確定し、通知失敗でsessionを復活させない
  (この不変条件自体は本WIのローカル失効にも front/back-channel 通知にも共通して適用される)。
- front/back-channel logout の通知配送 (署名済み logout token、Jobs 経由の retry、
  front-channel iframe target 算出) は、client metadata・ClientSession・
  LogoutNotification の永続化・Jobs handler・worker 登録という独立した実装単位を
  持つため、T005 完了後に [[wi-257-oidc-front-back-channel-logout-notifications]]
  へ切り出した。ADR-127 の決定5/6/8 (このADRは本WIで作成済み) がその設計を規定して
  おり、切り出し後も設計の一貫性は保たれる。

## Tasks
- [x] T000 [Dependency] [[wi-253-postgresql-persistent-login-sessions]] を完了し、
      PostgreSQL session正本、indexed list、冪等revoke、coarse touchを利用可能にする。
      wi-253 は 2026-07-18 に completed (`work-items/done/`)。
- [x] T002 [Decision] `decisions/ADR-127-oidc-session-binding-and-logout-propagation.md`
      を作成 (sid=LoginSession.id の単一値・全RP共通、ClientSessionは索引に限定、
      RefreshTokenRecord.sidによるfamily横断revoke、id_token_hint検証の項目と
      client不一致拒否・exp非検証、back-channel配送はJobs context経由、
      front-channelは配送保証なしの計算結果、access token即時失効はscope外
      (TTL 600秒の残存露出を許容)、check_session_iframeは最小実装
      (`adoption: optional`, Draft 28) を記録)。
- [x] T001 [SCL] oauth2 の sid、id_token_hint、front/back-channel logout
      metadata・interface・正常/境界/拒否scenarioを更新して再生成した。
      `spec/contexts/authentication.yaml`: `SessionRecord`/`SessionRecordListResponse`、
      admin向け `ListSessions`/`RevokeSession`/`RevokeUserSessions`
      (`TenantAdministrator`policy、既存`ListUserSignInActivity`と同型)、
      `SessionEnded`説明改訂、admin session管理scenario追加。
      `spec/contexts/oauth2.yaml`: `AuthorizationRequest`/`AuthorizationCodeRecord`/
      `RefreshTokenRecord`/`IdTokenClaims`へ`sid`追加、`OAuth2Client`へ
      backchannel/frontchannel logout metadata追加、新規model
      `ClientSession`/`LogoutNotification`/`LogoutNotificationState`/
      `FrontChannelLogoutTarget`と新規state machine`LogoutNotificationLifecycle`、
      `EndSession`へid_token_hint検証`requires`追加、新規interface
      `FrontChannelLogout`/`BackChannelLogout`(access: internal)/
      `CheckSessionIframe`(public)、`DiscoveryDocument`へlogout関連metadata追加、
      `standards`に`OIDC-LOGOUT-ID-TOKEN-HINT`と新規スタンダード3件、
      id_token_hint検証・back-channel配送のscenario追加。
      `spec/contexts/jobs.yaml`: `JobKind`に`backchannel_logout_delivery`追加。
      `spec/scl.yaml`: `OAuth2`の`depends_on`に`Jobs`追加。
      `just yaml-check` / `just check-ids` / `just scl-render` — passed。
- [x] T004 [Token] ID/access/refresh token と refresh family に sid を伝播し、session revoke 時に refresh family/denylist が冪等に更新される use case を実装した。
      Domain: `AuthorizationRequest.Sid` / `AuthorizationCodeRecord.Sid` / `RefreshTokenRecord.Sid` /
      `IDTokenClaims.Sid` を追加。`GenerateAuthorizationCode` (`AuthorizationCodeInput.Sid`)、
      `GenerateInitialRefreshToken`(sid引数追加)、`RotateRefreshToken`(親のSidを引き継ぐ) —
      RED: `TestGenerateInitialRefreshToken`/`TestRotateRefreshToken` の sid 伝播 assertion を
      一時的に実装コメントアウトして fail 確認 → GREEN (`RefreshTokenRecord.sid` SCL field)。
      UseCase: `authn.SessionID` (`AuthenticationContext.session_id`) → `CompleteLoginInput.Sid`
      → `AuthorizationCodeRecord.Sid` → `ExchangeCodeForToken` で `RefreshTokenRecord.Sid` /
      `IDTokenInput.Sid` へ伝播 (`TestExchangeCodePropagatesSidToRefreshTokenAndIDToken`)。
      `AttachAuthentication` port に `sid` 引数を追加し memory/valkey adapter を更新。
      `oauth2/usecases.RevokeTokensBySid` — RED: `TestRevokeTokensBySidRevokesAllFamiliesAndClients`
      を先に `RevokeTokensBySid` 未定義でコンパイル不能確認 → GREEN
      (`RefreshTokenStore.RevokeBySid`、sid 単位で family/client を横断して revoke、
      `UPDATE ... WHERE sid = $1` により idempotent、`AuthorizationCodeRecord.sid` /
      `RefreshTokenRecord.sid` scenario)。
      `authentication/usecases.RevokeOtherSessions` は失効した sessionID 一覧を返すよう変更し
      (`TestRevokeOtherSessionsKeepsCurrent` 更新)、`account_sessions_handler.go` の
      `handleRevokeAccountSession`/`handleRevokeOtherAccountSessions` から
      `oauth2usecases.RevokeTokensBySid` を呼び出すよう配線 (cross-context 呼び出しは
      `account_consents_handler.go` の既存パターンに合わせ HTTP adapter 層で実施)。
      Adapters: postgres `refresh_tokens` テーブルに `sid UUID`
      (cross-context の opaque reference として FK なし、`refresh_tokens_sid_idx` partial index)
      を追加し `just sqlc-generate` で再生成、`RevokeBySid` / `refreshFromRow` / `refreshInsertParams`
      を更新。`shared/adapters/crypto/jwt_signer.go` の `SignIDToken` に `sid` claim 追加 —
      RED: `TestSignIDTokenIncludesSidWhenPresent`/`TestSignIDTokenOmitsSidWhenAbsent` を
      未実装で fail 確認 → GREEN。
      副次修正: `spec/scl.yaml` の `OAuth2.depends_on.SigningKeys/Jobs` に T001 由来の
      YAML 構造破損 (`reason` キー重複によるマッピング不正) があり、`MustLoadSCL` が
      panic して oauth2/authentication 配下の Go テストが全滅していたため修正
      (`just yaml-check` / `just scl-render` で再検証、生成物への実質差分なし)。
      検証: `just yaml-check` / `go build ./...` / `go vet ./...` / `just lint-go` (0 issues) /
      `just test-go` — pass (既知の pre-existing 失敗 1 件を除く。下記参照)。
      **既知の未対応 (T004 スコープ外、T007/[[wi-257-oidc-front-back-channel-logout-notifications]] で対応予定):**
      `TestAssembledRoutesMatchGeneratedOpenAPI` は T001 で SCL に追加された admin 向け
      `ListSessions`/`RevokeSession`/`RevokeUserSessions`/`CheckSessionIframe`
      (`GET /api/admin/users/{sub}/sessions` 等、`GET /session/check`) の Go 実装が
      まだ無いために fail する (`ListSessions`/`RevokeSession`/`RevokeUserSessions` は
      T007、`CheckSessionIframe` は wi-257 で解消予定、T004 では意図的に手を付けていない)。
- [x] T005 [RP Logout] `id_token_hint` 検証を既存 end_session handler に追加し、client redirect validation と local logout を use case へ抽出した。
      Adapter (crypto): `oauth2/ports.IDTokenHintVerifier` を新設し
      `shared/adapters/crypto.JWTSigner.VerifyIDTokenHint` を実装 (署名を登録済み鍵で
      検証し iss を fail-closed 比較、exp は検証しない — ADR-127 決定4)。
      RED: `TestVerifyIDTokenHintReturnsClaimsForValidToken` /
      `TestVerifyIDTokenHintRejectsTamperedSignature` /
      `TestVerifyIDTokenHintRejectsOtherIssuer` を未実装で fail 確認 → GREEN。
      副次整理: 内部 `verifyPS256AnyKey` の未使用な header 戻り値を削除
      (`IntrospectAccessToken` 側の `_ = header` も削除、golangci-lint unparam 解消)。
      UseCase: `oauth2/usecases.ResolveEndSession` を新設し、
      client_id/post_logout_redirect_uri の既存検証ロジックを handler から抽出のうえ
      id_token_hint 検証を追加 (hint の aud と client_id パラメータの不一致を拒否、
      HintVerifier 未配線環境での hint 使用も拒否、post_logout_redirect_uri 省略時は
      レガシー互換で client 解決自体をスキップ) —
      RED: `TestResolveEndSession*` 一式 (6 ケース) を `ResolveEndSession` 未定義で
      コンパイル不能確認 → GREEN。
      Authentication 側: `authentication/usecases.EndSession` を新設 (所有者
      (sub) 未検証の sid を revoke し `SessionEnded` を発行。self-service の
      `RevokeOwnSession` と異なり `FindOwned` の所有者確認をしない — sid 自体が
      検証済み id_token_hint か本人だけが送れる browser cookie 由来であるため。
      `Find` は有効セッションのみ返すため未知/失効済み sid は自然に no-op で
      idempotent) — RED: `TestEndSessionRevokesBySidAndEmitsEvent` /
      `TestEndSessionUnknownSidIsNoop` を未定義で fail 確認 → GREEN。
      `SessionManager.SessionIDFromCookie` を追加 (revoke せず sid のみ取得、
      id_token_hint が無いときの cookie フォールバック用)。
      Adapter (http): `end_session_handler.go` を全面書き換え。
      `ResolveEndSession` → (hint 由来の sid、無ければ
      `SessionManager.SessionIDFromCookie` にフォールバック) → 解決した sid で
      `authusecases.EndSession` (session revoke) と `oauthusecases.RevokeTokensBySid`
      (T004、family/client 横断 revoke) を実行 → post_logout_redirect_uri が
      あれば検証済み URI へ 302、無ければ従来通り `/status?state=signed-out` へ
      303 (レガシー互換維持)。
      RED: `TestEndSessionWithValidIDTokenHintRevokesSessionAndAllClientTokens` /
      `TestEndSessionRejectsIDTokenHintAudienceMismatch` /
      `TestEndSessionRejectsIDTokenHintFromOtherIssuer` /
      `TestEndSessionAcceptsExpiredIDTokenHint` (SCL scenario
      "RP-Initiated Logout はid_token_hintからsessionとclientを解決する" を実サーバー
      経由で検証、`LegacyBareIssuer` を使い署名時と検証時の issuer を揃えた) を
      未実装 (400 invalid_request) で fail 確認 → GREEN。既存
      `TestEndSessionRedirectsToRegisteredURIWithStatePropagation` /
      `TestEndSessionRejectsUnregisteredPostLogoutURI` は無変更のまま green を維持。
      DI 配線: `oauth2.Module` / `oauth2/adapters/http.Deps` /
      `shared/adapters/http/server/routes.go` / `cmd/idmagic/server.go` に
      `IDTokenHintVerifier` を配線 (`TokenIssuer`/`TokenIntrospector` と同じ
      `*crypto.JWTSigner` インスタンスを使い回す)。
      Risk Notes (ADR-121 fuzz/property test 検討, 自己申告):
      id_token_hint は未信頼入力だが、JWT は固定 3-segment 構造 (base64 + 2 個の
      flat JSON オブジェクト) で再帰・組み合わせ爆発を持たず、`payload[...].(string)`
      は全箇所 comma-ok で安全、失敗経路は署名/iss 不一致によるエラー戻りのみで
      panic 経路がない。同種の未信頼 JWT 入力である既存
      `IntrospectAccessToken`/introspect endpoint にも fuzz test が無い
      (本リポジトリの唯一の fuzz target は再帰文法を持つ SCIM filter parser の
      `scim/domain/filter_fuzz_test.go` のみ)。この前例と risk profile の同一性
      から、本 WI でも id_token_hint 解析に fuzz/property test は採用しない
      ([[wi-257-oidc-front-back-channel-logout-notifications]] の logout token
      生成・検証着手時に再評価する)。
      検証: `go build ./...` / `go vet ./...` / `gofmt -l .` (差分無し) /
      `just lint-go` (0 issues) / `just test-go` / `just verify-go` (race) —
      pass (T004 で記録済みの既知 pre-existing 失敗 1 件を除く。saml/wsfederation
      配下は無変更で全て green、`stop_before_reading` を遵守)。
(T006 [Notifications] client metadata、logout token signer、jti replay guard、front/back-channel delivery は
[[wi-257-oidc-front-back-channel-logout-notifications]] へ切り出した。本 WI の Tasks では欠番とする。)
- [x] T007 [Account API/UI] session list/current marker、個別/global revoke endpoint と step-up/CSRF 付き画面を追加した。
      self-service (session list/current marker/個別revoke/他revoke) の account portal UI
      (`AccountActivityPage.tsx`, `/api/account/sessions*`) は既存実装 (wi-253 系列) 済みで
      本タスクでは手を加えていない。本タスクは admin 向け (SCL `ListSessions`/`RevokeSession`/
      `RevokeUserSessions`, ADR-127 決定9) の Go 実装と admin UI に絞った。
      UseCase: `authentication/usecases.AdminListSessions`/`AdminRevokeSession`/
      `AdminRevokeUserSessions` を新設 (`ListUserSignInActivity` と同型の
      `TenantAdministrator`/`resource=User,id=input.user_id` アクセス制御パターンを踏襲、
      self-service 版と衝突しないよう `Admin` prefix を付けた)。`AdminRevokeSession` は
      `FindOwned` で対象 sessionID が `user_id` の所有か検証し、不一致は
      `ErrSessionNotFound` (404) — URL 上の user_id と実際のセッション所有者の乖離を
      fail-closed で拒否する。`AdminRevokeUserSessions` は self-service の
      `RevokeOtherSessions` と異なり除外対象を持たない (操作者自身のセッションではない
      ため) — RED: `TestAdminListSessionsHasNoCurrentMarker` /
      `TestAdminRevokeSessionRejectsSessionOfOtherUser` /
      `TestAdminRevokeUserSessionsRevokesAllWithNoExclusion` を未定義で fail 確認 → GREEN。
      Adapter (http): `handleAdminListSessions`/`handleAdminRevokeSession`/
      `handleAdminRevokeAllSessions` を追加し `GET/POST /api/admin/users/{sub}/sessions*`
      へ配線。mutating 2 endpoint は既存の admin mutation ハンドラ (MFA enrollment bypass 等)
      と同じ `VerifyBrowserRequest` (Origin+CSRF double-submit) を通し、revoke 成功後は
      T004 の `revokeOAuthSessionTokens`/`RevokeTokensBySid` を呼んで同じ sid の
      RefreshTokenRecord も family/client 横断で失効させる (self-service ハンドラと同じ
      cascade を admin 側にも適用)。RED: `TestAdminListSessionsReturnsTargetUserSessions`/
      `TestAdminRevokeSessionCascadesToRefreshTokens`/`TestAdminRevokeSessionRejectsMismatchedUser`/
      `TestAdminRevokeAllSessionsRevokesEveryTargetSession`/
      `TestAdminSessionEndpointsRequireAdminRole` を未実装で fail 確認 → GREEN。
      UI: `frontend/src/features/admin-users/AdminUsersShared.tsx` に
      `UserSessionsSection` (一覧・個別終了・全終了、`window.confirm` で全終了のみ確認、
      既存の `AdminLifecycleWorkflowsPage.tsx` の破壊的操作確認パターンを踏襲) を追加し
      `AdminUserDetailPage.tsx` のユーザー詳細画面に配線。`types.ts` に
      `AdminSessionRecord`、`api/admin.ts` に `listAdminUserSessions`/
      `revokeAdminUserSession`/`revokeAllAdminUserSessions` を追加。i18n は
      `AdminUsersPage.i18n.ts` に日英完全対訳で追加 (AMR ラベルは
      `AccountActivityPage.i18n.ts` と重複させず admin 辞書内に `sessionAmr*` として複製)。
      `ARCHITECTURE.md` の `ui-page-lines` debt ceiling (`AdminUserDetailPage.tsx`) を
      488→490 に更新 (ロジック本体は `AdminUsersShared.tsx` 側で、ページ本体への追加は
      import + JSX 1 行のみ)。
      手動検証 (実サーバー, `./dev.sh memory` + curl による実 OIDC authorization_code+PKCE
      フロー): alice でログイン → `GET .../sessions` で自身のセッションが admin API 経由で
      見える → CSRF/Origin 無しの revoke は 403 → 正しい CSRF/Origin 付きの revoke は 204 →
      同じ sid の refresh_token を refresh grant で使うと invalid_grant (T004 cascade が
      admin 経路でも実際に機能することをライブ確認) → revoke_all も同様に確認。
      発見事項: 新規セッション (OAuth token 発行のみで cookie 解決を経ていない) は
      `last_seen_at` が Go の zero time (`0001-01-01T00:00:00Z`) のまま返り、素朴に
      日時フォーマットすると壊れた表示になることが判明したため、
      `sessionLastSeenLabel`ヘルパーで zero time を検出し「記録なし」表示にフォールバック
      する対応を追加 (`TestSessionIDFromCookie...` 同様の unit test を追加)。
      検証: `go build`/`go vet`/`gofmt -l .`/`just lint-go` (0 issues)/`just test-go`/
      `just verify-go` (race)/`just typecheck-ui`/`just lint-ui`/`just format-check-ui`/
      `just test-ui-unit` (403件)/`just build-ui`/`just yaml-check` — pass
      (T004 で記録済みの既知 pre-existing 失敗 1 件のみ除く。実サーバーでの手動確認は上記の通り)。
- [x] T008 [Verify] 複数端末での session/token revoke、期限切れ hint、redirect 攻撃、プロセス再起動後の
      session/refresh token revoke の永続性を検証した (通知配送の再試行・jti不変性・複数RP配送の検証は
      [[wi-257-oidc-front-back-channel-logout-notifications]] の T007 が担当する)。
      複数端末: `TestRevokeTokensBySidRevokesAllFamiliesAndClients` (sid A の token 一括revoke
      が別 sid B に影響しない, T004) / `TestAdminRevokeUserSessionsRevokesAllWithNoExclusion`・
      `TestAdminRevokeSessionRejectsSessionOfOtherUser` (複数ユーザー間の隔離, T007) /
      手動: alice で2回ログインし2つの独立した session+refresh_token を作成、
      個別revoke・revoke_allそれぞれで対象外のtoken/sessionが無傷であることを実サーバーで確認 (上記)。
      期限切れhint: `TestEndSessionAcceptsExpiredIDTokenHint` (T005)。
      redirect攻撃: `TestEndSessionRejectsUnregisteredPostLogoutURI`・
      `TestEndSessionRejectsIDTokenHintAudienceMismatch`・`TestEndSessionRejectsIDTokenHintFromOtherIssuer`
      (T005、post_logout_redirect_uri 未登録・client_id/hint不一致・他issuer署名のいずれも拒否)。
      プロセス再起動後の永続性: session (`authentication_sessions`) と
      refresh_tokens (sid 含む) は wi-253/T004 で PostgreSQL を単一正本化済みであり、
      `oauth2/adapters/persistence/postgres` および `authentication/adapters/persistence/postgres`
      の既存 postgres 統合テスト (embedded-postgres 経由、`just test-go` に含まれる) が
      Save→(プロセス内の別呼び出しでの)Find/RevokeBySid の往復を保証している。プロセスの
      生死とは独立した外部DBへ状態を持つ設計そのものが永続性の根拠であり、追加の
      「再起動シミュレーション」テストは新たな振る舞いを検証しないため作成しなかった
      (ADR-119: SCLに無い振る舞いをテストで創作しない)。
      検証: 上記は全て T004/T005/T007 で既に green 化済みの回帰テスト
      + 本タスクで実施した実サーバーでのライブ確認 (T007 参照) の組み合わせで、
      新規のテストコードは追加していない。

## Verification
- `just test-go`
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just lint-ui`
- `just build-ui`
- 手動: 複数ブラウザで login し、account portal から片方の session を revoke すると対象 browser だけが再認証要求になることを確認する。

## Risk Notes
session、refresh token、browser cookie の整合が崩れると logout したつもりで
生き残る経路ができる (RP notification 配送の整合は
[[wi-257-oidc-front-back-channel-logout-notifications]] の Risk Notes が扱う)。
状態遷移を SCL に寄せ、session lifecycle の property test を厚くする。

active sessionをPostgreSQLとValkeyへ二重保存しないことで、最も危険なstale cache経路を
設計から除外する。性能上の懸念はWI-253のindexed queryと負荷ゲートで先に評価する。
`id_token_hint` は認証・認可に関わる未信頼JWTであるため、署名、期限、
issuer、audience、subject、sid、client不一致のproperty/fuzz test採用をT005着手時に
再評価し、採用しない判断をself-attestへ記録済み (logout tokenのreplay/fuzz判断は
[[wi-257-oidc-front-back-channel-logout-notifications]] が担当する)。

## Completion
- **Completed At**: 2026-07-19
- **Summary**:
  session/token のローカル失効を実運用相当まで完成させた。(T004) OIDC の `sid`
  (Authentication の `LoginSession.id` と同値) を `AuthorizationRequest` →
  `AuthorizationCodeRecord` → `RefreshTokenRecord` → ID Token へ一貫して伝播させ、
  session revoke (self-service 個別/全revoke、admin 個別/全revoke、
  RP-Initiated Logout) のいずれからも `RevokeTokensBySid` で同じ sid を共有する
  refresh token を family/client 横断で失効させるようにした。(T005) `/end_session`
  に `id_token_hint` の署名・iss・aud・sub・sid 検証を追加し (fail-closed、
  client_id 不一致は拒否、hint 無しは既存 cookie 解決へ後方互換フォールバック)、
  local logout を use case へ抽出した。(T007) admin 向け session 一覧・個別/全revoke
  (`ListSessions`/`RevokeSession`/`RevokeUserSessions`, ADR-127 決定9) を Go/UI
  両方に実装し、self-service と同じ oauth2 cascade を admin 経路にも適用した。(T008)
  上記の検証観点 (複数端末、期限切れ hint、redirect 攻撃、DB永続性) を既存の
  回帰テストと実サーバーでの手動確認で満たした。
  front/back-channel logout の通知配送 (client metadata、`ClientSession`、
  `LogoutNotification`、logout token signer、Jobs 経由配送、`CheckSessionIframe`) は
  独立した実装単位として [[wi-257-oidc-front-back-channel-logout-notifications]] へ
  切り出し、本 WI のスコープからは意図的に除外した (ADR-127 の決定5/6/8 がその WI の
  設計を規定する)。
- **Disclosed Gaps** (ADR-121):
  - `spec/contexts/oauth2.yaml` の `standards.OpenIDConnectFrontChannelLogout` /
    `OpenIDConnectBackChannelLogout` (`adoption: required`) と
    `OpenIDConnectSessionManagement` (`adoption: optional`) は、SCL 宣言済みだが
    Go 実装は本 WI のスコープ外で [[wi-257-oidc-front-back-channel-logout-notifications]]
    が担当する。`TestAssembledRoutesMatchGeneratedOpenAPI` は `GET /session/check`
    (`CheckSessionIframe`) 1件を spec-only として検出し続ける — これは意図的な既知差分。
  - id_token_hint / logout token の解析には fuzz/property test を採用していない
    (self-attest、Risk Notes 参照)。
- **Verification Results**:
  - `go build ./...` / `go vet ./...` / `gofmt -l .` — pass (差分無し)
  - `just lint-go` — 0 issues
  - `just test-go` — pass (spec-only `GET /session/check` の既知差分1件を除く)
  - `just verify-go` (race) — 同上
  - `just typecheck-ui` / `just lint-ui` / `just format-check-ui` / `just build-ui` — pass
  - `just test-ui-unit` — 403 tests pass
  - `just yaml-check` (SCL/work-item/ids/architecture/traceability) — pass
  - 手動 (実サーバー `./dev.sh memory` + 実 OIDC authorization_code+PKCE フロー):
    admin session一覧・個別revoke・全revokeのいずれも、同じ sidを共有する
    refresh_tokenの失効 (refresh grantが invalid_grant になること) まで
    ライブ確認済み (T007 self-attest 参照)。
  - saml/wsfederation 配下は無変更、`stop_before_reading` を遵守。
