---
status: pending  # pending | in_progress | completed | cancelled
authors: [tn]
risk: high        # low | medium | high | critical
created_at: 2026-07-24  # YYYY-MM-DD
depends_on: []
---

# 揮発性の認証/OAuth2 一時状態を全て PostgreSQL に統合し、Valkey を廃止する

## Motivation

idmagic は現在 2 つのステートフル基盤に依存する。durable state 全部を持つ **PostgreSQL**（必須）と、揮発性の認証/OAuth2 一時状態だけを持つ **Valkey**（本番は GCP Memorystore STANDARD_HA、月 $150-200 想定）。Valkey が持つのは 9 ストア（認可中間状態・認可コード・PAR・device code・JTIリプレイ(DPoP/client assertion)・トークン失効リスト・WebAuthnチャレンジ・ログインスロットル・SAMLリプレイ）だけで、**pub/sub もキューも永続データも無い**（イベントは Kafka/PubSub、ジョブキューは既に PostgreSQL）。

PostgreSQL はどのみち必須依存であり、Valkey は "2 つ目" の基盤にすぎない。その運用負荷（コスト・HA運用・セルフホスト導入障壁・critical path が 2 本）は実際に負担であり、削減したい。全ストアは ports で抽象化済み（memory/valkey/postgres が同一インターフェース）で、ADR-126 が既に login session を Valkey→PostgreSQL に移した前例がある。この WI はその軌道の完遂として、残る揮発性ストアを全て PostgreSQL に統合し Valkey を撤去する。

スケーラビリティ判断：9ストアは (A) 対話レート律速・短命の 8 ストア＝PGで容易、(B) `AccessTokenDenylist.IsRevoked`（introspection でリクエスト毎に走り得る唯一のホット読取）に割れる。PG はスケールアップ（RAMにホットセット常駐）と読取レプリカが得意で、本ワークロードは PG が苦手な水平書込シャーディングを踏まない。高RPS の introspection は共有ストア選択と独立に「per-request lookup を避ける設計（短命トークン／インスタンス内キャッシュ／push 失効）」で解くのが定石であり、「denylist のために Valkey が要る」は偽の前提。

## Scope

機能変更ではなく永続化アダプタの差し替え＋非機能（依存基盤削減）だが、SCL の運用記述・objective に触れるため以下を更新する。

- `spec/contexts/system.yaml`（ReadinessProbe 記述・「PostgreSQL または Valkey へ到達できない」scenario extension・`postgres_valkey` モード名参照 → Postgres 単一依存へ）
- `spec/contexts/authentication.yaml`（`LoginSession` の「Valkey 全損を跨いで」表現・login throttle objective の Valkey 参照除去）
- 派生成果物再生成：`spec/idmagic.full.html` / `spec/idmagic.html` / `spec/idmagic.models.schema.json` / `spec/idmagic.openapi.json`（`scl-render`）
- スキーマ：`infra/schema/postgres.sql`（9 テーブル追加、宣言的スキーマ＋sqldef 適用）
- 各 context の PostgreSQL アダプタ実装：`backend/oauth2/db_postgres/`、`backend/authentication/session/db_postgres/`、`backend/authentication/webauthn/db_postgres/`、`backend/saml/db_postgres/`
- bootstrap：`backend/cmd/internal/bootstrap/postgres_valkey.go`（→ `postgres.go`）、`deps.go`、health 系
- ADR：ADR-139 新規、ADR-016 supersede、ADR-077 機構部 supersede、ADR-105 / ADR-106 改訂

## Out of Scope

- introspection/denylist の高RPS向けキャッシュ層・短命トークン化・push 失効の実装（ADR-139 に設計方針として記載するのみ。本 WI は per-request の共有ストア参照を PG に置き換えるところまで）。
- denylist の時間パーティション＋`DROP PARTITION` GC（高RPS時の将来最適化として記載のみ）。
- ジョブキュー・イベント配信（既に PostgreSQL / Kafka で Valkey 非依存）。
- durable ストアの DB 変更（PostgreSQL 継続）。

## Plan

**方針**：ADR-139 先行で決定を固め、その後 context 単位で段階実装（単純→複雑）。`backend/authentication/session/db_postgres/sessions.go` ＋ `authentication_sessions.sql` を sqlc アダプタのテンプレとする。

**TTL の正しさ**：全 read クエリに `AND expires_at > now()` を付ける（`authentication_sessions.sql:27` と同型）。期限切れ行は絶対に返らない。GC は空間回収のみで cadence は正しさに無関係。

**原子性の写し替え**（各ストア対応表は Tasks 参照）：
- Lua CAS → `UPDATE ... WHERE <cas条件> RETURNING *`（行が返れば成功、`ErrNoRows`→`nil`＝Valkey の `goredis.Nil`/`false` と同義）
- SetNX → `INSERT ... ON CONFLICT DO NOTHING`＋挿入行数（rows==1 で新規）
- GetDel → `DELETE ... RETURNING`
- WATCH 楽観ロック → tx 内 `SELECT ... FOR UPDATE`

**性能ハイブリッド**：揮発性が高く消えても再開で済むテーブル（auth_request/code/par/device/webauthn/replay）は **UNLOGGED**（WAL を書かず Valkey の揮発性に最も近い）。失効リストと throttle は failover で消えるとセキュリティ後退/fail-closed 崩れになるため **LOGGED**（throttle は fillfactor 80＋HOT update で dead tuple 抑制）。

**tenant scoping**：これら ephemeral は全て opaque token key の高頻度 fail-closed lookup なので `tenant_id` 列保持の例外に該当（ADR-082 §4／[[tenant-id-key-policy-adr083]] に整合）。既存 `authentication_sessions` に倣い `tenant_id UUID NOT NULL` + `tenants(id) ON DELETE CASCADE`。

**GC**：各ストアに `DeleteExpiredBatch(ctx, cutoff, limit)`（`session/ports/session_store.go:44` と同一シグネチャ）を追加。常駐 `idmagic-worker` に周期 ticker（60s 目安）で `ephemeralSweep` を配線（`worker.go` の既存 `time.NewTicker` パターン）。usecase は `EphemeralPurger`（`retention.go:150` `SessionPurger` に倣う）新設。best-effort。

**移行**：ephemeral はデータ移行不要（in-flight フローは揮発でよい＝ADR-126 と同じ割り切り）。dual-write 不要。`postgres_valkey` を 1 リリースだけ `postgres` の alias として残し、infra 切替後に alias 削除。

**却下した代替案**：(1) Valkey を残す＝運用負荷が痛く、方向は ADR-126 で既定。(2) 別の Redis 系（KeyDB/Dragonfly）＝"2つ目の基盤"が消えず目的に反する。(3) 主 DB を別物に＝durable 層の巨大な再投資で本末転倒。

**未決定**：UNLOGGED/LOGGED の最終判断と GC 間隔は Phase3 の staging 実測（dead tuple/vacuum/p99）で確定する。

## Tasks

**Phase 0：決定とスキーマ基盤**
- [ ] T001 [ADR] ADR-139 起票（揮発性を全て PostgreSQL に統合し Valkey 廃止。denylist 高RPS戦略は共有ストア選択と独立の設計方針を明記）。ADR-016 supersede、ADR-077 機構部 supersede（fail-closed は維持＝依存を追加でなく削減する旨）、ADR-105/106 改訂。
- [ ] T002 [SCL] `spec/contexts/system.yaml` / `authentication.yaml` の Valkey 参照・モード名を更新し、`scl-render` で派生再生成。
- [ ] T003 [DB] `infra/schema/postgres.sql` に 9 テーブル追加（下表、UNLOGGED/LOGGED 別、expires_at index、tenant_id 保持列挙コメント追記）。sqldef 適用確認。

| テーブル | PK | 主要列 | storage |
|---|---|---|---|
| `oauth2_authorization_requests` | id | tenant_id, expires_at, payload JSONB(state) | UNLOGGED |
| `oauth2_authorization_codes` | code | tenant_id, state, redeemed_at, issued_family_id, payload | UNLOGGED |
| `oauth2_par_requests` | request_uri | tenant_id, used, payload | UNLOGGED |
| `oauth2_device_codes` | device_code_hash | tenant_id, user_code UNIQUE, user_id, state, payload | UNLOGGED |
| `oauth2_replay_jtis` | (tenant_id,kind,jti) | expires_at | UNLOGGED |
| `oauth2_access_token_denylist` | (tenant_id,jti) | expires_at | LOGGED |
| `webauthn_sessions` | (tenant_id,session_key) | data JSONB, expires_at | UNLOGGED |
| `login_throttle_counters` | (tenant_id,kind,identifier_hash) | failures, window_expires_at, locked_until | LOGGED, fillfactor 80 |
| `saml_authnrequest_replays` | (tenant_id,entity_id,request_id) | expires_at | UNLOGGED |

**Phase 1：context 単位アダプタ実装（単純→複雑、各: sqlc→adapter→DeleteExpiredBatch→contract test）**
- [ ] T004 [App] SAML `AuthnRequestReplayStore`（SetNX → INSERT ON CONFLICT DO NOTHING）。最初の縦スライスで pipeline 検証。
- [ ] T005 [App] oauth2 `ReplayStore`(DPoP/ClientAssertion, `Prefix`→`Kind` 列) と `AccessTokenDenylist`（INSERT / `SELECT EXISTS(... expires_at>now)`）。
- [ ] T006 [App] `WebAuthnSessionStore`（GetDel → `DELETE ... WHERE expires_at>now RETURNING data`）。
- [ ] T007 [App] oauth2 `PARStore` / `AuthorizationCodeStore` / `DeviceCodeStore`（単一列 CAS：`UPDATE ... WHERE <state> RETURNING *`、device は user_code UNIQUE＋user_id 列で `DeleteAllForSub`）。
- [ ] T008 [App] oauth2 `AuthorizationRequestStore`（tx＋`SELECT FOR UPDATE`→`spec.TransitionAuthorizationCodeFlow`→UPDATE）。
- [ ] T009 [App] session `LoginAttemptThrottle`（tx＋`SELECT FOR UPDATE`→window 再計算→閾値で locked_until=now+lockout,failures=0。fail-closed 維持）。
- [ ] T010 [App] 各 context に `DeleteExpiredBatch` を追加し `EphemeralPurger` usecase 経由で `idmagic-worker` の周期 sweep に配線。

**Phase 2：配線切替**
- [ ] T011 [App] `postgres_valkey.go`→`postgres.go`（`assemblePostgres`）。Valkey client/config/breaker/ValkeyPing 削除、9 バインドを `*postgres.*` に置換、valkey import 削除。
- [ ] T012 [App] `deps.go` switch を `case "postgres","postgres_valkey":`（移行期 alias）に。health 系（`health_handler.go`/`support_http/deps.go`/`server.go`/`memory.go`）の ValkeyPing 削除。

**Phase 3：観測・切替**
- [ ] T013 [Verify] staging を `PERSISTENCE=postgres` に切替、dead tuple/autovacuum/p99 latency を実測し UNLOGGED/LOGGED 判断と GC 間隔を確定。

**Phase 4：撤去**
- [ ] T014 [Infra] インフラから Valkey 削除（`docker-compose.dev.yaml` valkey service、`k8s` configmap/networkpolicy、`gcp` provision.sh Memorystore/secret、`cloudrun-idmagic.yaml`、`dev.sh`、`idmagic-dev-infra` の miniredis）。`PERSISTENCE=postgres` に統一。
- [ ] T015 [App] `backend/**/db_valkey/`・`backend/shared/storage/db_valkey/` 削除、`go.mod` から `redis/go-redis/v9`・`alicebob/miniredis/v2` 削除、`postgres_valkey` alias と `ValkeyPing` 完全撤去。

## Verification

- 層ごと：`just test-go`（新アダプタの contract test が memory/valkey とパリティ）。
- 統合（miniredis 撤去後、PostgreSQL 単体起動）：`just dev` で `/authorize`→code→token 交換、device flow、PAR、WebAuthn 登録/認証、ログイン失敗連続でロックアウト（fail-closed）、トークン失効→introspection で revoked 確認、を実際に走らせる。
- readiness：health エンドポイントが Postgres 単一依存になることを確認。
- 全体：`just verify` / `just yaml-check` / `just check-ids`。
- 性能（Phase3）：staging 実測メトリクスを Evidence に記録。

## Risk Notes

- **security（fail-closed）**：throttle と denylist は failover で状態が消えると防御後退。→ この 2 つは LOGGED（物理 standby へ複製）で担保。ADR-077 の fail-closed ポリシーは維持し、機構のみ Postgres へ（Postgres は既に hard dependency で依存を追加でなく削減）。
- **正しさ（TTL）**：GC 遅延で肥大しても正しさは `expires_at>now()` フィルタが担保。GC は best-effort。
- **性能（write 増幅/vacuum）**：高churn の denylist/replay/throttle は autovacuum 負荷。→ UNLOGGED（可能なもの）＋ fillfactor/HOT update＋per-table autovacuum チューニング。Phase3 で実測してから prod 切替。
- **破壊的変更（config）**：`PERSISTENCE` モード名変更。→ 移行期 alias で 1 リリース吸収。
- **移行**：切替時の in-flight フロー（進行中の /authorize・PAR・device・throttle counter）は放棄されるが再開で回復（ADR-126 と同じ割り切り）。データ移行・dual-write は不要。
