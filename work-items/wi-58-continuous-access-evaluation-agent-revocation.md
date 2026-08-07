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
- [ ] T003 [Enforcement] KillAgent/credential unbind/policy revoke と epoch 更新を同一 transaction/event log に接続し、token issue/introspection/denylist path で評価する。
- [ ] T004 [Transmitter] SET builder/signer、outbox projector、retry/backoff/dead-letter delivery と stream status endpoint を実装する。
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
