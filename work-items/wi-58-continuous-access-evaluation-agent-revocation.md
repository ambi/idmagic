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
- [ ] T005 [Admin] stream CRUD/key metadata、delivery health/retry と kill 結果表示を追加する。
- [ ] T006 [Verify] kill 前後 token、cache、多 replica、duplicate/out-of-order SET、receiver outage、cross-tenant subject と最大伝播時間を検証する。

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
