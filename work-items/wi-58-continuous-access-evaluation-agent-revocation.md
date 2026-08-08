---
depends_on: [wi-49-agent-identity-first-class-principal, wi-50-token-exchange-delegation-actor-chain]
status: pending
authors: ["tn"]
risk: high
created_at: 2026-06-22
---

# エージェント向け継続的アクセス評価 (CAEP/SSF) と即時失効・kill-switch 伝播

## Motivation
エージェントは長時間・高頻度にトークンを使い続けるため、リスク変化 (所有者の
オフボード、kill-switch、異常検知) を検知したら近リアルタイムでセッション /
トークンを失効させる必要がある。これを標準化するのが OpenID の Shared Signals
Framework (SSF) と Continuous Access Evaluation Profile (CAEP) で、Security Event
Token (RFC 8417) を transport にイベントを push / receive し、access の継続評価と
即時失効を可能にする。

idmagic の README ロードマップ (Phase 3) は CAEP / SSF を汎用機能として挙げて
いるが、エージェント固有の「所有者オフボードで配下エージェントを一括失効」
「kill-switch ([[wi-49-agent-identity-first-class-principal]]) を全トークンへ伝播」
という観点では未着手である。本 WI は SSF の transmitter / receiver と CAEP イベントを
実装し、エージェントのセッション・トークン・委譲チェーンを継続評価して即時失効
できるようにする。これは導入したエージェント・委譲・vault のすべてに失効の網を被せる。

## Scope
- **decision**:
  - 新規 ADR [[ADR-057]]: SSF の transmitter / receiver 双方向の範囲、CAEP イベント種別 (session-revoked / token-claims-change / credential-change / assurance-level-change)、 Security Event Token (RFC 8417) の署名と配送 (push delivery)、エージェント kill-switch と 所有者オフボードのイベント化、受信側でのローカル失効反映を確定する。
- **scl**:
  - 新規 model: SecurityEventToken / SsfStream / CaepEvent / SsfReceiverConfig / SsfTransmitterConfig。
  - 新規 event: SsfStreamConfigured / SecurityEventTransmitted / SecurityEventReceived / AgentAccessRevoked。
  - 新規 interface: SSF stream 管理、event push / receive。permission AdminSharedSignalsManage。
- **go**:
  - SET の署名・検証・配送、CAEP イベント生成、受信イベントによるローカル token / session の失効反映を fail-closed で実装する。
  - エージェントの kill-switch / 所有者無効化を配下トークンへ伝播する。
- **http**:
  - SSF transmitter (push) / receiver エンドポイント、stream 管理 API。

## Out of Scope
- 異常検知 (impossible travel 等) のシグナル源そのものの実装 (イベント transport が対象)。
- 外部 receiver / transmitter との相互運用認証取得。
- リスクスコアリングエンジンの構築。

## Plan
- 既存 `KillAgent` は新規 token 発行を止めるが既発行 token を即時無効化しないため、agent credential/token family と `revoked_after` epoch を関連付け、introspection・保護 API の denylist check で即時停止する。
- [[ADR-057-continuous-access-evaluation-and-agent-revocation]] の CAEP/SSF event type、subject identifier、delivery semantics を確定する。kill、credential unbind、owner/policy change を security event transmitter の入力にする。
- SSF stream configuration は tenant/receiver ごとに delivery endpoint、audience、verification key、event type、status を持つ。SET は署名済みJWTで、jti/iat/iss/audを固定し、outbox→retryable delivery とする。
- local revocation commit を外部配送より先に行う。receiver outage は local kill を遅らせず、delivery は at-least-once + receiver jti dedup 前提で再送する。
- cache を使う token validation path は revoke epoch より長く active 判定を保持しない。性能目標と最大伝播時間を SCL objective に置く。

## Tasks
- [x] T001 [ADR/SCL] ADR-057 の supported events/subject format/SET profile を確定し、revocation objective、stream/delivery lifecycle、events/scenarios を再生成する。
  - ADR-057 を `suggested` → `accepted` に移行。
  - 新規 SCL bounded context `spec/contexts/sharedsignals.yaml` を追加: `AgentRevocationEpoch`・`SsfStream`/`SsfTransmitterConfig`/`SsfReceiverConfig`・`SecurityEventToken`/`SecurityEventDelivery`/`ReceivedSecurityEvent`・`CaepEvent`/`SsfSubject`、`SsfStreamLifecycle`/`SecurityEventDeliveryLifecycle` 状態機械、stream CRUD + `ReceiveSecurityEvent` (inbound push, `access: public`) + `CheckRevocationEpoch`/`AdvanceRevocationEpoch` (internal) interface、8 scenario。
  - `spec/scl.yaml` に `SharedSignals` context_map entry を追加し、`OAuth2` の `depends_on` に `SharedSignals.CheckRevocationEpoch` を追加。
  - `spec/contexts/oauth2.yaml` の `Introspect` に revocation epoch 判定の `ensures` とシナリオ「kill-switch後のAgentトークンはintrospectionでactive=falseになる」を追加。
  - `spec/contexts/api-tokens.yaml` に `shared-signals:read`/`shared-signals:write` scope を追加。
  - `architecture.yaml` (root) と `ARCHITECTURE.md` に `SharedSignals` を登録 (Go module 未実装のため `backend/sharedsignals` は "planned" と明記)。
  - 検証: `just check` (SCL/work-items/ids/architecture/traceability) が green。
  - Go 実装 (Domain/Persistence/Enforcement/Transmitter/Admin/UI) は T002 以降、別セッションで継続する。
- [x] T002 [Domain/Persistence] AgentRevocationEpoch、SSFStream、SecurityEventDelivery を実装し、memory/PostgreSQL の tenant-scoped store を追加する。
  - **Valkey は対象外に変更**: wi-278/ADR-139 でエフェメラルストアは全て PostgreSQL に統合され Valkey は撤去済み (本 WI 作成時点でのタスク記述が古い)。`AccessTokenDenylist` (`backend/oauth2/db_postgres/access_token_denylist.go`) と同じ、キャッシュ層を持たない直接 PostgreSQL クエリの方針を踏襲した。
  - `backend/sharedsignals/domain`: Domain: RED (`*_test.go` を先に書き、`Int`→スキーマ不備等で fail 確認) → GREEN。`AgentRevocationEpoch.Supersedes`、`SsfStream.Validate/IsEnabled/Subscribes`、`SsfReceiverConfig`/`SsfTransmitterConfig`/`SecurityEventDelivery`/`ReceivedSecurityEvent` の `Validate()` を実装 (spec/contexts/sharedsignals.yaml の constraints に対応)。
  - `backend/sharedsignals/ports`: 6 repository interface。
  - `backend/sharedsignals/db_memory`: 6 repository の in-memory 実装。`AgentRevocationEpochRepository.Advance` の単調増加ガード、`ReceivedSecurityEventRepository.ExistsByJTI` の replay 検知、`SecurityEventDeliveryRepository.ListDue` を RED→GREEN で実装・検証。
  - `backend/sharedsignals/db_postgres`: `infra/schema/postgres.sql` に 6 table 追加 (`agent_revocation_epochs`・`ssf_streams`・`ssf_transmitter_configs`・`ssf_receiver_configs`・`security_event_deliveries`・`received_security_events`)。`AdvanceAgentRevocationEpoch` は条件付き `ON CONFLICT ... WHERE EXCLUDED.epoch >= 既存` で epoch 単調増加を DB 制約として fail-closed に強制 (行が更新されなければ 0 行 → `ErrEpochNotAdvancing`)。`received_security_events` の `UNIQUE (stream_id, set_jti)` で replay を DB 制約として吸収。sqlc 生成 (`sqlc.yaml` に新規ブロック追加)。embedded-postgres 契約テストで monotonic guard・cascade delete・replay 制約・ListDue を検証。
  - `backend/sharedsignals/architecture.yaml` を新規追加し、root `architecture.yaml`/`ARCHITECTURE.md` の `backend/sharedsignals` (planned) を実装済みへ更新。
  - SCL 修正: `type: Int` → `type: Integer` (誤記、`just test-go` の coherence test で検出・修正)。
  - 検証: `just check` green、`go build ./...` green、`go test -race ./backend/sharedsignals/...` green、`just test-go` (repo 全体) green、`golangci-lint run ./backend/sharedsignals/...` 0 issues。
  - usecases 層 (KillAgent 等との接続、OAuth2 Introspect への実配線) は T003 で行う。本タスク時点では repository 単体としては動作するが、まだどこからも呼ばれていない。
- [x] T003 [Enforcement] KillAgent/credential unbind/owner offboard と epoch 更新を接続し、OAuth2 Introspect で評価する。
  - **設計: 個別 notifier 呼び出しではなく汎用 EventReactor に収束させた。** 当初 `AdminAgentDeps`/`AdminUserDeps` に `RevocationNotifier` port を追加し KillAgent 等から直接呼ぶ実装をしたが、レビューで「通知の仕組みが増え続けるのは良くないのでは」という指摘を受け設計を見直した。KillAgent/SetAgentDisabled/UnbindCredential/SetUserDisabled/SoftDeleteUser/DeleteUser は元々 `deps.Emit(...)` で対応イベント (`AgentKilled`/`AgentDisabled`/`AgentCredentialUnbound`/`UserDisabled`/`UserSoftDeleted`/`UserDeleted`) を無条件に emit している。これを trigger として使う `sharedsignals/usecases.AgentRevocationReactor.React(ctx, event)` を実装し、`idmanagement/deps_http.EventReactor` という (context.Context, spec.DomainEvent) だけに依存する汎用 interface 経由で合成する (`Deps.ReactiveEmit()` が best-effort な `LegacyEmit` と fail-closed な Reactor を合成)。結果、`AdminAgentDeps`/`AdminUserDeps` および業務ロジック本体は無変更のまま (Emit 呼び出しがそのまま trigger)、wi-323 (CAEP/SSF の User セッションへの拡張) が `SessionEnded` 等を追加する際も reactor 側に `case` を足すだけで済む。cross-context Postgres transaction (`UserMutationCommitter`/`UserWorkflowCapture` 相当) は採用しなかった: wi-184 で一度「業務更新 + event log を同一 transaction」機構を実装し wi-190 で撤去した経緯があり、この repo は audit-only event を best-effort・eventual に倒す方針を既に選んでいる。fail-closed が必要な revocation epoch advance だけは `EventReactor.React` のエラーを `ReactiveEmit` 経由で呼び出し元 (KillAgent 等) へ伝播させ、audit trail 用の `Emit` 本体は従来どおり best-effort のままにした。
  - `backend/idmanagement/deps_http/deps.go`: `EventReactor` interface (`React(ctx, spec.DomainEvent) error`)、`Deps.Reactor` field、`Deps.ReactiveEmit()` (best-effort Emit + fail-closed Reactor の合成、Reactor は `context.Background()+5s timeout` を使う — 既存 `NewEmitFunc` と同じ「request cancellation に追従しない」規約)。
  - `backend/sharedsignals/usecases/revocation.go` (新規 package): `AdvanceRevocationEpoch`/`CheckRevocationEpoch` (SCL internal interface の実体、`ErrEpochNotAdvancing` は冪等 no-op として吸収)、`AgentRevocationReactor.React` が `AgentKilled`/`AgentDisabled`/`AgentCredentialUnbound`/`UserDisabled`/`UserSoftDeleted`/`UserDeleted` を type-switch し epoch を前進させる。owner イベントは `AgentRepository.ListByTenant` で `OwnerUserID` 一致の Agent を都度解決 (専用の owner-index は持たない、admin 操作のみでホットパスではないため)。`EpochRepo`/`AgentRepo` が nil の lightweight wiring では no-op (既存の `QuotaRepo` 等 nil-skip 規約に合わせる)。RED→GREEN: `TestAgentRevocationReactor_ReactsToAgentEvents`/`_ReactsToOwnerEvents`/`_OwnerWithNoAgentsIsNoop`/`_IgnoresUnrelatedEvents`/`_NilEpochRepoIsNoop`。
  - `backend/oauth2/token/usecases/introspect_token.go`: `IntrospectDeps` に `AgentRepo`/`RevocationEpochRepo` を追加。access token 分岐で `AgentRepo.FindByClientID` → `RevocationEpochRepo.FindByAgent` → `epoch.Supersedes(iat)` を判定し `active=false` を返す (spec/contexts/oauth2.yaml Introspect の `ensures` を実装)。RED→GREEN: `TestIntrospectToken_AgentRevocationEpoch` (issued-before-epoch は inactive、issued-after-epoch は active のまま、non-agent client は無影響)。token issue path (client_credentials) は ADR-048 の `agent.IsActive()` gate が既に Killed/Disabled を fail-closed で拒否しており、epoch は「既発行 token」専用のため追加変更なし。AccessTokenDenylist との関係は変更なし (別チェックとして共存)。
  - composition root 配線: `backend/cmd/internal/bootstrap/{memory,postgres}.go` に `sharedsignals.Module` (6 repository) を追加、`Dependencies.SharedSignals` field。`backend/shared/http/server_http/routes.go` の `registerTenantRoutes` で `AgentRevocationReactor` を1個構築し `idmhttp.Deps.Reactor` と `oauth2http.Deps.RevocationEpochRepo` に配線。`backend/cmd/idmagic-worker/worker.go` の `newAdminUserDeps` にも同様の reactor 合成 (`PurgeExpiredSoftDeleted` の auto-purge 経路)。統合テスト `TestAdminAgentKill_AdvancesRevocationEpoch` (idmanagement/handlers_http) で HTTP → ReactiveEmit → Reactor → epoch repo の配線全体を検証。
  - architecture 同期: `backend/sharedsignals/architecture.yaml` に `sharedsignals-usecases` module を追加。`backend/cmd/architecture.yaml` (bootstrap/worker)・`backend/shared/architecture.yaml` (http-server)・`backend/oauth2/architecture.yaml` (oauth2-adapters/oauth2-token-usecases) の depends_on を実 import に合わせて更新。`ARCHITECTURE.md` の SharedSignals 行を「Introspect 実装済み・AgentRevocationReactor 実装済み」に更新。
  - **Out of Scope として残したもの**: 「policy revoke」に対応する既存ユースケースが見当たらず (role_policies は OAuth2 側の別概念)、trigger を実装していない。所有者オフボード時に配下 Agent の新規 token 発行そのものを止める仕組みは未実装 (Agent.Status は owner offboard で変化しないため、client_credentials は引き続き成功しうる — 既発行 token は epoch で revoke されるが、新規発行は防がない。SCL の `ensures` にこの要件が無いため実装していない。必要なら別途 SCL 拡張が要る)。
  - 検証: `just check` (SCL/architecture/work-items) green。`just build-go` green。`just verify-go` (`lint-go` 0 issues + race-enabled `test-go`) green。
- [x] T004 [Transmitter] SET builder/signer、outbox projector、retry/backoff/dead-letter delivery を実装する。
  - **stream status endpoint (`ListSecurityEventDeliveries`) は T005 へ統合**: stream を作る手段 (`RegisterSsfTransmitterStream`) が T005 側にしかなく、単体で提供しても使えないため、admin API 一式 (stream CRUD + delivery 一覧) として T005 でまとめて実装する方針にした (ユーザー確認済み)。
  - **SET builder/signer**: `backend/sharedsignals/usecases/transmit.go` の `BuildAndSignSecurityEventToken` が RFC 8417 claims (`iss`/`jti`/`iat`/`aud`/`events`) を組み立てる。署名は新規鍵管理を作らず SigningKeys の既存ローテーション/JWKS に相乗りする (ADR-057 決定3/7)。ただし `usecases` (use_cases 層) が `tokens_jose` (adapters 層) を直接 import すると `just check-architecture` の層規則違反になったため、`ports.SecurityEventTokenSigner` port を新設し、実装 (`tokens_jose.SignPS256` + `signingports.KeyStore` を wrap) を新規 adapter package `backend/sharedsignals/sign_jose` に切り出した。RED→GREEN: `TestBuildAndSignSecurityEventToken`/`_OmitsReasonWhenNil`。
  - **outbox projector**: `backend/sharedsignals/usecases/project.go` の `ProjectAgentAccessRevoked` が、SharedSignals 自身が emit する `AgentAccessRevoked` (T003で実装済み) を trigger に、テナント内の有効な Transmit stream (`session-revoked` 購読) へ SET を構築・署名し `SecurityEventDelivery` (pending) を enqueue する。**LocalRevocation (epoch 前進) とは分離した best-effort 経路**: `AdvanceRevocationEpoch` 本体には組み込まず、composition root (`routes.go`/`worker.go`) で「epoch 前進成功後に呼び、失敗してもログするだけで無視する」形にした (ADR-057 決定6)。RED→GREEN: 有効/無効/未購読/Receive方向/config欠落の各 skip 条件、複数 stream への fan-out、nil StreamRepo (lightweight wiring) の no-op。
  - **retry/backoff/dead-letter**: `backend/sharedsignals/usecases/deliver.go` の `ProcessDueDeliveries` が `SecurityEventDeliveryRepository.ListDue` (T002で実装済み) から due な配送を取り出し、`ports.SecurityEventPusher` で push、成功なら delivered、失敗なら attempt_count++ と exponential backoff (30秒基点、上限30分、jobs.domain.NextRetryRunAt と同じ形だが 5 行程度なので import はせず sharedsignals 内に持つ) で failed へ、max_delivery_attempts 到達で dead_letter へ遷移させる。直前が failed だった配送を再試行する際は結果イベントの前に `SecurityEventDeliveryRetried` を emit し、SecurityEventDeliveryLifecycle の状態遷移をイベント上でも表現する。RED→GREEN: 成功/backoff付き失敗/SCLシナリオ「3回失敗でdead_letter」/retry時のRetried emit/transmitter config欠落時のfail-closed。
  - **push adapter**: `backend/sharedsignals/push_http` (新規 adapter package) が SSF push-based delivery (`Content-Type: application/secevent+jwt`, POST, 2xx成功) を実装。`delivery_endpoint` は admin 設定値 (SSRF対象になりうる) のため、DNS-rebinding safe な dial・redirect 再検証を実装 — 一元化された共通クライアントは使わず、`tokens_jose.JWKResolver`・`provisioning/client_scim` と同じ「per-context に SSRF-safe transport を持つ」既存の前例を踏襲した (このリポジトリでは意図的に非一元化)。
  - **配線**: `backend/cmd/internal/bootstrap/{memory,postgres}.go` に signing/push 依存を追加配線。`routes.go`/`worker.go` の `AgentRevocationReactor.Emit` を拡張し、`AgentAccessRevoked` を横取りして best-effort に projector を呼ぶ。`idmagic-worker` に `sharedSignalsDeliveryLoop` (5秒間隔ポーリング、`SHARED_SIGNALS_DELIVERY_INTERVAL` で調整可) を追加し `ProcessDueDeliveries` を定期実行する。Jobs durable queue には乗せない: `SecurityEventDelivery` 自身が attempt_count/next_attempt_at/status を持つ独立した状態機械であり、二重の retry 機構にしないため。
  - architecture 同期: `backend/sharedsignals/architecture.yaml` に `sharedsignals-usecases`（更新）/`sharedsignals-push-http`/`sharedsignals-sign-jose` の3 module を追加・整理。`backend/cmd/architecture.yaml`（bootstrap/worker）・`backend/shared/architecture.yaml`（http-server）・`backend/oauth2/architecture.yaml` の依存を実 import に合わせて更新。ルート `ARCHITECTURE.md` の SharedSignals 行を SET transmitter 実装済みに更新。
  - 検証: `just check`（architecture 層規則違反を1周目で検出・修正）green。`just build-go` green。`just verify-go`（lint-go 0 issues + race-enabled test-go）green。
- [x] T005 [Admin] stream CRUD/key metadata、delivery health を追加する (kill 結果表示は delivery health API に統合)。
  - **フロントエンド UI は対象外** (ユーザー確認済み): 直近の同型前例 wi-54 (WorkloadIdentity federation) が Admin HTTP API のみを実装し UI を作らなかった前例に倣う。「kill 結果表示」は `ListSecurityEventDeliveries` (配送状況 pending/delivered/failed/dead_letter の一覧 API) を通じて管理者が kill の伝播結果を確認できるようにする、という API レベルの意味として実装した。
  - **ManagementApiClient (Agent principal の `shared-signals:read`/`write` scope) の実 enforcement は未実装**: このリポジトリ全体で `ManagementApiClient` policy を Go 実装している既存 context が無い (wi-274 の Risk Notes に明記された既知の保留事項と同じ)。T005 は他 context と同様 `TenantAdministrator` (`RequireAdmin`) のみを実装し、この慣行に合わせた。
  - `backend/sharedsignals/usecases/admin_streams.go`: `AdminStreamDeps` と `ListSsfStreams`/`GetSsfStream`/`RegisterSsfTransmitterStream`/`RegisterSsfReceiverStream`/`UpdateSsfStream`/`DisableSsfStream`/`EnableSsfStream`/`DeleteSsfStream`/`ListSecurityEventDeliveries`。`workloadidentity/usecases/admin_trust_bundles.go` と同型のパターン (tenancy 境界、必須項目を usecase 層で pre-check してから `domain.Validate()` — raw な Validate() 失敗は 500 に落ちてしまうため、既存の established convention に合わせた)。`DeleteSsfStream` は付随する Transmitter/ReceiverConfig を cascade 削除する。RED→GREEN: 必須項目欠落の分岐、登録成功、event_types 省略時の no-op update、disable/enable の冪等性、cascade delete、cross-tenant は ErrStreamNotFound。
  - **Hard Quota (`QuotaExceededError`) は未実装**: SCL は T001 時点で `RegisterSsfTransmitterStream`/`RegisterSsfReceiverStream` に `QuotaExceededError` を宣言しているが、実装するには Tenancy 側に新規 quota resource (`tenant_usages`/`tenant_quotas` テーブルへの新規カラム、`ResourceSsfStreams` 定数、SCL `TenantQuota` model 更新) を追加する必要があり、stream CRUD の範囲を大きく超える cross-context 変更になるため見送った。将来必要になった時点で別途対応する。
  - `backend/sharedsignals/handlers_http/routes.go` (新規 package): `RegisterRoutes` が SCL の 8 admin binding (`GET/POST/PATCH/DELETE /api/admin/shared-signals/streams...`) を登録。`workloadidentity/handlers_http/routes.go` と同型 (`RequireAdmin` + 書き込み系は `VerifyBrowserRequest` (CSRF))。
  - 配線: `backend/shared/http/server_http/routes.go` の `registerTenantRoutes` に `sharedsignalshttp.RegisterRoutes` を追加。
  - architecture 同期: `backend/sharedsignals/architecture.yaml` に `sharedsignals-adapters` (handlers_http) module を新規追加、`sharedsignals-usecases` に `tenancy-public` 依存を追加。`backend/shared/architecture.yaml` (http-server) の依存を更新。
  - **発見事項 (T001-T004 に遡る欠落)**: `just scl-render` を実行したところ (T001 以降ずっと未実行で `spec/idmagic.openapi.json` が stale だった)、`ReceiveSecurityEvent` interface (T001 で SCL に宣言済み) の Go 実装が一度も存在しないことが `TestAssembledRoutesMatchGeneratedOpenAPI` 契約テストで露見した。ユーザーと相談の上、T007 として本セッション内で追加実装した (下記)。
  - 検証: `just check` green。`just build-go` green。`just verify-go` green。
- [x] T006 [Verify] kill 前後 token、cache、多 replica、duplicate/out-of-order SET、receiver outage、cross-tenant subject と最大伝播時間を検証する。
  - **SCL 欠落の是正 (Plan に遡る欠落)**: Plan は「性能目標と最大伝播時間を SCL objective に置く」としていたが T001 では未実装だった。`spec/contexts/sharedsignals.yaml` に `objectives.RevocationPropagationLatency` (`AgentAccessRevoked` から `SecurityEventDelivery` が `delivered` になるまで 99% が 30 秒未満、LocalRevocation 自体はこの目標を待たない旨を明記) を追加し、`just scl-render` で派生物を再生成した。このリポジトリの他 objective (`oauth2.yaml`/`provisioning.yaml`) と同様、値は observability 側 (Prometheus) で継続的に測るドキュメント化された SLO であり、自動テストで直接アサートする対象ではない (既存の慣行と同じ)。worker の delivery loop は 5 秒間隔ポーリング (T004) + 初回試行は backoff 無し、という設計で 30 秒目標に対して十分な余裕がある。
  - **kill 前後 token**: 既存 `TestIntrospectToken_AgentRevocationEpoch` (`backend/oauth2/token/usecases/introspect_token_agent_revocation_test.go`) が issued-before-epoch は inactive、issued-after-epoch は active のままであることを検証済み。統合テスト `TestAdminAgentKill_AdvancesRevocationEpoch` (T003) が HTTP kill → epoch 前進の配線全体を検証済み。追加テストは不要と判断した。
  - **cache**: 専用 subagent でトークン検証/introspection 経路全体を調査した。`IntrospectToken` (epoch を見る唯一の経路) は毎回 `RevocationEpochRepo.FindByAgent` を直接読んでおり、キャッシュは存在しない。JWKS 系のキャッシュ (`JWKResolver`・`WorkloadJWKSCache`) は鍵素材のキャッシュであり token の active 判定とは無関係。**発見した構造的な穴 (今回は未修正)**: `backend/shared/http/support_http/auth.go` の `resolveAuthnContext` (admin/account portal の Bearer 認証) は `TokenIntrospector.IntrospectAccessToken` を直接呼び `oauth2/token/usecases.IntrospectToken` を経由しない (= epoch 判定と `AccessTokenDenylist` の双方をバイパスする)。ただし調査の結果、悪用可能性は無いと判断した: この経路は `sub` claim を `UserRepository.FindBySub` に通すが、Agent の client_credentials token の `sub` は `OAuth2Client.ClientID` (独立生成の UUIDv4)、User の `sub` は `users.id` (別テーブル、別生成) であり、2つの ID 空間は交差しない (`token_handler.go`/`register_client.go`/`admin_users.go` で確認)。つまり Agent token でこの Bearer 経路を通っても `FindBySub` が非 nil を返すことは (UUIDv4 衝突以外) 構造的に起こり得ず、admin/account API になりすまし認証できない。とはいえ epoch/denylist を通らない経路が存在すること自体は defense-in-depth の観点で望ましくないため、必要なら別 work item で `resolveAuthnContext` を `IntrospectToken` 経由に統一することを検討されたい (今回は verify のみで修正はスコープ外とした)。
  - **多 replica**: 新規 `TestAgentRevocationEpochRepository_MultiReplicaConsistency` (`backend/sharedsignals/db_postgres/revocation_epochs_test.go`) を追加。同一 PostgreSQL を共有する2つの独立した repository インスタンス (replica を模す) の一方が `Advance` した直後、もう一方の `FindByAgent` が同じ epoch を読めることを検証した。`AgentRevocationEpochRepository` は process-local な状態を一切持たず (毎呼び出しで SQL を直接発行)、`db_memory` (process-local state を持つ) は本番配線で使われない (`bootstrap/postgres.go` のみが本番経路) ことを合わせて確認済み。
  - **duplicate/out-of-order SET**: duplicate (replay) は既存 `TestReceiveSecurityEvent_RejectsReplay` (JTI dedup) で検証済み。out-of-order (古いイベントに由来する epoch 前進要求が後から届く) は既存 `TestAdvanceRevocationEpoch_AlreadyRevokedIsIdempotent` および `TestAgentRevocationEpochRepository_AdvanceIsMonotonic` (DB 制約レベルの `ON CONFLICT ... WHERE`) で、epoch が後退しないことを既に検証済み。追加テストは不要と判断した。
  - **receiver outage**: 既存 `TestProcessDueDeliveries_FailureBelowMaxSchedulesBackoffRetry`/`_ExhaustingMaxAttemptsDeadLetters`/`_RetryEmitsRetriedBeforeOutcome` (T004) が backoff・dead-letter・retry イベントを検証済み。
  - **cross-tenant subject**: 既存 `TestReceiveSecurityEvent_RejectsUnresolvedSubject` の subtest (unknown agent・cross-tenant subject) で検証済み。
  - 検証: `just check` green (新規 `objectives` block 含む)。`just build-go` green。`just verify-go` (lint-go 0 issues + race-enabled test-go、新規 `TestAgentRevocationEpochRepository_MultiReplicaConsistency` を含む) green。
- [x] T007 [Receiver] (新規、当初の Task 内訳に無かった欠落を今回追加) ReceiveSecurityEvent (inbound SET 受理) を実装する。
  - `backend/shared/security/tokens_jose/security_event_token_verifier.go`: `VerifySecurityEventToken` (PS256/ES256/RS256、iss/aud/jti/iat 検証)。既存の `VerifyWorkloadSVID` と同型だが、SET は RFC 8417 上 `exp`/`maxTTL` を持たない point-in-time 通知のため、その部分は持たない別関数として新設した (`pickJWK`/`verifyJWTSignature`/`verifyAudience` は既存の内部ヘルパーを再利用)。RED→GREEN: 有効署名 (RS256/PS256)・署名偽装・iss不一致・aud不一致・jti欠落・malformedの各拒否。
  - `backend/sharedsignals/ports/verifier.go` + `backend/sharedsignals/verify_jose` (新規 adapter package): `SecurityEventTokenVerifier` port と、`tokens_jose.VerifySecurityEventToken` + 既存の `tokens_jose.JWKResolver` (SSRF-safe, WorkloadIdentity/OAuth2 と共有) を wrap する実装。`sign_jose` と対称に、use_cases 層を tokens_jose (adapters 層) から隔離する。
  - `backend/sharedsignals/usecases/receive.go`: `ReceiveSecurityEvent` が SCL の `requires` を順に評価する: (1) `ssf_receiver_stream_enabled` (2) `security_event_signature_and_claims_valid` (3) `!security_event_replayed` (`ReceivedSecurityEventRepository.ExistsByJTI`) (4) `security_event_subject_resolves_to_tenant_local_principal`。**subject 解決は idmagic 自身の transmitter (T004) が生成する `events` claim 内の `subject` オブジェクト形式を前提とする自前ワイヤーフォーマットとして実装**した (RFC 9493 Subject Identifiers のフル相互運用は対象外、外部 transmitter が現れた時点で拡張する設計判断、ADR-057 の範囲内)。subject_type は `Agent` のみ対応 (`User` は wi-323 の範囲)。検証を通過したイベントは `AdvanceRevocationEpoch` (reason=`InboundSecurityEvent`) で LocalRevocation へ収束させる (ADR-057 決定5)。拒否時は理由ごとに `SecurityEventVerificationResult` を分類して `ReceivedSecurityEvent` に記録し `SecurityEventRejected` を emit する。RED→GREEN: stream 無効/不在、署名検証失敗の分類別拒否、replay 拒否、subject 未解決 (unknown agent・cross-tenant) 拒否、正常受理時の epoch 前進と `SecurityEventReceived` emit。
  - `backend/sharedsignals/handlers_http/routes.go`: `POST /ssf/streams/{stream_id}/events` (`access: public`、`RequireAdmin`/CSRF なし — 認可は `SsfReceiverConfig` の `trusted_issuer`/JWKS 検証そのものが担う)。request body は SET の compact serialization をそのまま受理 (JSON ではない、64KiB 上限)。成功時 202、拒否時 400。
  - 配線: `backend/shared/http/server_http/routes.go` に `verify_jose.Verifier{JWKResolver: d.JWKResolver}` を構築して配線。
  - architecture 同期: `backend/sharedsignals/architecture.yaml` に `sharedsignals-verify-jose` module を新規追加、`sharedsignals-adapters` に `idmanagement-agent-ports` 依存を追加。`backend/shared/architecture.yaml` を更新。
  - 検証: `just check` (契約テスト `TestAssembledRoutesMatchGeneratedOpenAPI` を含め green) 。`just build-go` green。`just verify-go` green。

## Verification
- `just test-go`
  - reason: SET 署名 / 検証、CAEP イベント反映、kill-switch 伝播による失効、改竄イベント拒否の境界。
- `just lint-go`
- `just build-go`
- 手動: エージェントへ token 発行 → kill-switch / 失効イベント送出 → 当該 token が即時に無効化されることを確認する。

## Risk Notes
継続評価と即時失効は侵害時の被害局限の要であり、イベントの取りこぼしや改竄反映は
失効漏れ / 誤失効を招く。Security Event Token は署名検証必須とし、検証を通った
イベントのみ反映する (fail-closed)。kill-switch / 所有者オフボードは確実に配下トークンへ
伝播し、失効は「迷ったら無効化」側に倒す。
