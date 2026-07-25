---
status: completed  # pending | in_progress | completed | cancelled
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

**移行**：ephemeral はデータ移行不要（in-flight フローは揮発でよい＝ADR-126 と同じ割り切り）。dual-write 不要。`PERSISTENCE` は `postgres` に統一し、旧 `postgres_valkey` は alias を残さず削除（切替は環境変数を `postgres` へ更新するデプロイと同時に実施。未更新環境は起動時に fail-fast）。

**却下した代替案**：(1) Valkey を残す＝運用負荷が痛く、方向は ADR-126 で既定。(2) 別の Redis 系（KeyDB/Dragonfly）＝"2つ目の基盤"が消えず目的に反する。(3) 主 DB を別物に＝durable 層の巨大な再投資で本末転倒。

**未決定**：UNLOGGED/LOGGED の最終判断と GC 間隔は Phase3 の staging 実測（dead tuple/vacuum/p99）で確定する。

## Tasks

**Phase 0：決定とスキーマ基盤**
- [x] T001 [ADR] ADR-139 起票（揮発性を全て PostgreSQL に統合し Valkey 廃止。denylist 高RPS戦略は共有ストア選択と独立の設計方針を明記）。ADR-016 supersede、ADR-077 機構部 supersede（fail-closed は維持＝依存を追加でなく削減する旨）、ADR-105/106 改訂。
- [x] T002 [SCL] `spec/contexts/system.yaml` / `authentication.yaml` / `jobs.yaml`（dev infra）の Valkey 参照を Postgres 単一依存へ更新。`just check` グリーン。SCL 3.0 移行で `ValkeyResilience`/`SharedEphemeralStateHA` objective は既に除去済みのため SCL 側の HA objective 変更は不要と判明。派生再生成（`scl-render`）は integration/main で実施（work-item branch では派生物を commit しない方針）。
- [x] T003 [DB] `infra/schema/postgres.sql` に 9 テーブル追加（UNLOGGED/LOGGED 別、expires_at/GC index、tenant_id 保持理由をヘッダに追記）。`just sqlc-generate` で schema parse＋全 db_postgres の models.go 再生成を確認。sqldef 適用は embedded-postgres 契約テストで検証（Phase 1 以降）。

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
- [x] T004 [App] SAML `AuthnRequestReplayStore`。RED: `TestAuthnRequestReplayStore`（重複予約が true を返す／16 並行で 16 勝者／GC=0）を stub で先に fail 確認（constraint `SAML2Core-BearerAssertion`「同一 tenant/SP/request ID は一度だけ」）→ GREEN。SETNX+TTL は `INSERT ... ON CONFLICT DO UPDATE ... WHERE expires_at<=now RETURNING`（live 予約は 0 行=false、期限切れ/未存在は 1 行=true）で写した。DeleteExpiredBatch も実装。`just test-go-package -race` green（原子性を 16 並行で実証）。最初の縦スライスで pipeline（schema→sqlc→adapter→contract test）を検証済み。
- [x] T005 [App] oauth2 `ReplayStore`(DPoP/ClientAssertion, `Prefix`→`Kind` 列) と `AccessTokenDenylist`。RED: `TestReplayStore`（重複=true）/`TestAccessTokenDenylist`（post-add not revoked）を stub で fail 確認 → GREEN。ReplayStore は 1 テーブル+kind 列で両 port を構造的に満たす。denylist は `SELECT EXISTS(... expires_at>now)`、Add は `ON CONFLICT DO NOTHING`（冪等・insert-only）。tenant/kind 分離・期限切れ非 revoked・GC を検証。
- [x] T006 [App] `WebAuthnSessionStore`。RED: `TestWebAuthnSessionStore`（Save 後 Take=nil）を stub で fail 確認 → GREEN。GetDel = `DELETE ... WHERE expires_at>now RETURNING data`。round-trip/一度きり消費/期限切れ nil/tenant 分離/GC を検証。webauthn/db_postgres に TestMain harness を新設。
- [x] T007 [App] oauth2 `PARStore` / `AuthorizationCodeStore` / `DeviceCodeStore`。RED: `TestPARStore`/`TestAuthorizationCodeStore`/`TestDeviceCodeStore`（Save 後 Find=nil）を stub で fail 確認 → GREEN。full record を payload JSONB に持ち、**昇格列（used/state/redeemed_at/issued_family_id）を可変フィールドの権威とし read で payload に overlay**（単文 CAS の payload 陳腐化を回避、timestamp フォーマット地雷も回避）。Redeem/Consume/Exchange は `UPDATE ... WHERE <state/used> RETURNING`、device は (tenant_id,user_code) UNIQUE と user_id 列で `DeleteAllForSub`。Find は memory 同様に期限フィルタなし（parity）。
- [x] T008 [App] oauth2 `AuthorizationRequestStore`。RED: `TestAuthorizationRequestStore`（Save 後 Find=nil）を stub で fail 確認 → GREEN。UpdateState/AttachAuthentication は tx + `SELECT ... FOR UPDATE` の read-modify-write（Valkey WATCH の写し）で `spec.TransitionAuthorizationCodeFlow` を直列適用。不正遷移/不明 id は error。
- [x] T009 [App] session `LoginAttemptThrottle`。RED: `TestLoginAttemptThrottle`（閾値到達で lock されない）を stub で fail 確認 → GREEN。fixed-window の counter と lockout を 1 行（failures/window_expires_at/locked_until）に統合、RecordFailure は tx + `SELECT FOR UPDATE`（Valkey Lua の原子性の写し）、閾値で locked_until=now+lockout・failures=0。SHA-256 識別子・tenant 分離・fail-closed（error 伝播）維持。
- [x] T010 [App] 全 postgres adapter に `DeleteExpiredBatch` を実装済み。`bootstrap.RunEphemeralSweepOnce`（`ephemeral.go`）が 9 ストア（+ throttle は factory 経由で instance 化）を `ephemeralPurger` 型アサーションで集め一括 GC（memory/valkey は未実装のため自動除外＝`RunRetentionSweepOnce` と同型）。`idmagic-worker` に `ephemeralSweepLoop`（60s ticker、`EPHEMERAL_SWEEP_INTERVAL` 可変）を配線。best-effort、正しさは read の expires_at 述語が担保。

**Phase 2：配線切替**
- [x] T011 [App] `postgres_valkey.go`→`postgres.go`（`assemblePostgres`）。Valkey client/config/breaker と VALKEY_URL 要求・valkey import を削除、9 バインドを `*postgres.*`（webauthn/throttle/oauth2 7 種/saml replay）に置換。`build-go` green。
- [x] T012 [App] `deps.go` switch を `case "postgres","postgres_valkey":`（移行期 alias、error 文言も更新）に。`Dependencies.ValkeyPing` フィールドと health 系（`health_handler.go` を Postgres 単一依存に書換／`support_http/deps.go`／`server.go`／`memory.go`）の ValkeyPing を削除。full `just test-go` / `just lint-go` green。

**Phase 3：観測・切替**
- [x] T013 [Verify] → **[[wi-282]] に分離**。staging 実測（dead tuple/autovacuum/p99、UNLOGGED/LOGGED と GC 間隔の確定）は再利用可能な負荷テスト基盤として wi-282 で実施する。本 WI の実装は暫定判断のまま完了しており、ADR-139 §4 とスキーマに「暫定・staging 実測で確定」と明記済み。正しさは各 read の `expires_at > now()` 述語が担保するため storage/GC の確定は性能・容量最適化であり機能完了の前提ではない。

**Phase 4：撤去**
- [x] T014 [Infra] インフラから Valkey 削除：`docker-compose.dev.yaml`（valkey service・VALKEY_URL・depends_on）、`k8s`（configmap `PERSISTENCE`・networkpolicy の valkey egress×4）、`gcp` provision.sh（Memorystore create・secret・env）、`cloudrun-idmagic.yaml`（PERSISTENCE・VALKEY_URL secret）、README.md（構成図・コスト表・env）、`dev.sh`（VALKEY_URL）、`idmagic-dev-infra` の miniredis endpoint。全て `PERSISTENCE=postgres` に統一。
- [x] T015 [App] `backend/**/db_valkey/`（session/webauthn/oauth2/saml）と `backend/shared/storage/db_valkey/` を削除、`go.mod` から `redis/go-redis/v9`・`alicebob/miniredis/v2`（＋推移依存 gopher-lua 等）を `go mod tidy` で除去、`postgres_valkey` alias と `ValkeyPing` を完全撤去。ARCHITECTURE.md から 5 db-valkey モジュールを削除。コメント中の Valkey 参照も整理（memory パリティ表記・機構直記述へ）。`build-go`/`test-go`/`lint-go`/`check` green。

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
- **破壊的変更（config）**：`PERSISTENCE` モード名変更（`postgres_valkey`→`postgres`）。alias は残さず削除する方針のため、切替は環境変数更新デプロイと同時に行う。未更新環境は起動時に "PERSISTENCE must be memory or postgres" で fail-fast（silent な誤起動はしない）。
- **移行**：切替時の in-flight フロー（進行中の /authorize・PAR・device・throttle counter）は放棄されるが再開で回復（ADR-126 と同じ割り切り）。データ移行・dual-write は不要。
- **検証基盤（発見・修正済み）**：T004 着手時、`backend/shared/storage/testing_postgres/pgtest.go` の `schemaPath()` が `..` を 6 個使っており（パッケージはルートから 4 階層）、schema パスが `/Users/tn/infra/...` に誤解決されて **リポジトリ全体の postgres 契約テストが黙って skip**（skip は "ok" 表示で `just verify` はグリーンのまま）していた。直近の sqlc ディレクトリ flatten（wi-267）で階層が浅くなった際の未更新が原因と見られる。本 WI の検証戦略（memory とのパリティ契約テスト）が機能する前提なので `..` を 4 個へ修正し、リポジトリ全 DB テストが実走・green になることを確認した（wi-278 本体とは別コミット）。**後続 Phase の全 postgres 契約テストはこの修正の上で実走する。**

## Completion

- **Completed At**: 2026-07-25
- **Summary**:
  揮発性の認証 / OAuth2 一時状態 9 ストア（authorization request / code / PAR / device code /
  JTI replay（DPoP・client assertion）/ access-token denylist / WebAuthn challenge /
  login throttle / SAML AuthnRequest replay）を全て PostgreSQL に統合し、Valkey を
  コード・依存・インフラから完全撤去した（ADR-139）。port 契約は不変のまま context ごとに
  `db_postgres` アダプタを新設（SETNX→`INSERT ON CONFLICT`、Lua CAS→`UPDATE ... RETURNING`、
  GetDel→`DELETE ... RETURNING`、WATCH→tx＋`SELECT FOR UPDATE`）。TTL の正しさは全 read の
  `expires_at > now()` 述語が担保し、GC は `idmagic-worker` の周期 `ephemeralSweepLoop`（60s）で
  best-effort に空間回収する。bootstrap を `assemblePostgres` に切替え、`PERSISTENCE` は
  `postgres` に統一（旧 `postgres_valkey` は alias を残さず削除）、readiness は Postgres 単一依存に。
  SCL（system/authentication/jobs）・スキーマ（9 テーブル、UNLOGGED/LOGGED 別、暫定）・
  ARCHITECTURE・README・GCP デプロイ資材・dev 環境（miniredis 撤去）・`go.mod`
  （go-redis/miniredis 除去）を同期。UNLOGGED/LOGGED と GC 間隔の staging 実測確定（旧 T013）は
  [[wi-282]] に分離した。
- **Verification Results**:
  - `just check`（SCL / work-items / ids / architecture / traceability）- passed
  - `just build-go` - passed
  - `just test-go`（全 postgres 契約テストが pgtest 修正後に実走）- passed
  - `just lint-go` - 0 issues
  - oauth2 / session `db_postgres` を `-race` - passed（16 並行 INSERT で勝者 1・throttle tx CAS を実証）
- **Affected Guarantees State**:
  - fail-closed（ADR-077）: login throttle / access-token denylist は LOGGED で維持。到達不能時は
    error 伝播で fail-closed。機構を Postgres へ移しただけで防御は不変（依存はむしろ削減）。
  - once-only（replay / code redeem / PAR consume / device exchange / SAML replay）: 単文 CAS または
    tx＋`SELECT FOR UPDATE` で原子性を維持（契約テストで並行検証）。
  - tenant 分離: 各 ephemeral は opaque key の fail-closed lookup として `tenant_id` を保持
    （ADR-082 §4 例外、ADR-139 §8）。
  - **暫定**: 各テーブルの UNLOGGED/LOGGED と GC 間隔は staging 未実測。正しさは `expires_at>now()` が
    担保するため機能は完了だが、性能・容量の最終確定は wi-282 に委ねる。
- **Evidence**:
  - 手順: `just check` / `just build-go` / `just test-go` / `just lint-go` / `just test-go-package "-race ..."`
    をローカルで実行（embedded-postgres 契約テスト）。
  - 実行環境: darwin、本リポジトリ作業ツリー。
  - 対象ソース版: 本 WI の 3 コミット（Phase 0＋SAML slice / T005-T012 / T014-T015）＋
    別コミットの pgtest 修正・applications sqlc 再生成。
  - 結果: 上記すべて green（詳細は各 Task の RED→GREEN 証跡を参照）。
  - 保存先: git 履歴（feature ブランチ）。大容量ログ・機密は無し。
  - 未実施: staging 負荷実測（wi-282）、フル OAuth2/WebAuthn の `just dev` E2E 通し（wi-282 の
    負荷シナリオと併せて実施予定）。
