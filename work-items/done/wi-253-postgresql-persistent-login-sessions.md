---
status: completed
authors: [tn]
risk: high
created_at: 2026-07-18
depends_on: []
change_kind: feature
initial_context:
  scl:
    Authentication:
      - models.LoginSession
      - models.AccountSession
      - interfaces.ListMySessions
      - interfaces.RevokeMySession
      - interfaces.RevokeMyOtherSessions
  source:
    - backend/authentication/domain/authentication.go
    - backend/authentication/ports/session_store.go
    - backend/authentication/usecases/session_manager.go
    - backend/authentication/usecases/sessions.go
    - backend/authentication/adapters/persistence/memory/sessions.go
    - backend/authentication/adapters/persistence/valkey/stores.go
    - backend/cmd/internal/bootstrap/postgres_valkey.go
  tests:
    - backend/authentication/usecases/sessions_test.go
    - backend/authentication/adapters/persistence/memory/sessions_test.go
  stop_before_reading:
    - backend/oauth2
    - frontend
affected_spec:
  - { context: Authentication, kind: model, element: LoginSession }
  - { context: Authentication, kind: model, element: AccountSession }
  - { context: Authentication, kind: interface, element: ListMySessions }
  - { context: Authentication, kind: interface, element: RevokeMySession }
  - { context: Authentication, kind: interface, element: RevokeMyOtherSessions }
---

# ログイン済みセッションを PostgreSQL の単一正本として永続化する

## Motivation

現在の production `SessionStore` は Valkey の `session:*` にログイン済み
セッションを保存し、ユーザー別一覧と全失効では `SCAN` 後に `user_id` を
アプリケーション側で絞り込む。この構造はセッション総数に比例して探索量が増え、
Valkey の再起動・eviction・全損時には有効なログイン状態も失われる。

PostgreSQL と Valkey に同じ active session を二重保存すると、失効時の更新順序、
stale cache、cache invalidation、再試行 outbox まで必要になり、セキュリティ上重要な
logout 経路が複雑になる。まず PostgreSQL の indexed query に性能を委ね、測定で
必要性が示されるまで active session cache を導入しない。

この判断は、Keycloak 26 が persistent user sessions を既定にし、database を
session source of truth としている実例を参考にする。Keycloak の cache-on-demand
まで初期実装で模倣せず、idmagic では単一ストアから開始する。

- https://www.keycloak.org/2024/12/storing-sessions-in-kc26
- https://www.keycloak.org/server/caching

## Scope

- `spec/contexts/authentication.yaml` の `models.LoginSession` /
  `models.AccountSession`、session lifecycle、一覧・失効 interface/scenario。
- PostgreSQL `authentication_sessions` table、migration、sqlc query、repository。
- session id による認証解決、ユーザー別 active session 一覧、個別失効、
  current 以外の全失効、ユーザー全セッション失効。
- memory repository を PostgreSQL と同じ contract に同期する。
- production bootstrap の active `SessionStore` を Valkey から PostgreSQL へ変更する。
- Valkey からログイン済み session 保存・`SCAN session:*` を除去する。
- 期限切れ・失効済みセッションを小さな batch で削除する housekeeping query。

## Out of Scope

- OAuth2 authorization code / ID token / access token / refresh token family への
  `sid` 伝播。[[wi-28-session-management-and-oidc-logout-completion]] で扱う。
- RP-Initiated Logout、Front-Channel Logout、Back-Channel Logout。
- PostgreSQL session lookup の Valkey cache。計測で必要性が確認された場合だけ
  別 WI で扱う。
- table partitioning、read replica、sharding。単一 primary と index で測定した後の
  拡張手段とする。
- account/admin UI の変更。

## Plan

### 1. 単一正本

ログイン済み session の状態は PostgreSQL にだけ保存する。browser cookie は十分な
entropy を持つ不透明な session id のみを保持する。認証解決は次の indexed query を
直接実行し、失効・期限切れ・別 tenant を fail-closed で除外する。

```sql
SELECT id, tenant_id, user_id, auth_time, amr, acr,
       authentication_pending, pending_purpose,
       enrollment_deadline, enrollment_bypass_id, step_up_at,
       started_at, last_seen_at, expires_at
FROM authentication_sessions
WHERE tenant_id = $1
  AND id = $2
  AND revoked_at IS NULL
  AND expires_at > $3;
```

`(tenant_id, id)` を primary/unique lookup とする。session id が tenant をまたいで
推測・再利用されても別 tenant の行を返さない。

### 2. ユーザー別一覧

一覧は Valkey `SCAN` ではなく、部分複合 index と keyset pagination で取得する。

```sql
CREATE INDEX authentication_sessions_active_user_idx
ON authentication_sessions
  (tenant_id, user_id, started_at DESC, id DESC)
WHERE revoked_at IS NULL;
```

```sql
SELECT ...
FROM authentication_sessions
WHERE tenant_id = $1
  AND user_id = $2
  AND revoked_at IS NULL
  AND expires_at > $3
  AND (started_at, id) < ($4, $5)
ORDER BY started_at DESC, id DESC
LIMIT $6;
```

最初の account UI は小さな既定上限で先頭ページだけを利用してよいが、repository
contract は continuation cursor を表現できる形にする。offset pagination は
セッション数増加時の走査量と同時更新による重複・欠落を避けるため採用しない。

### 3. 失効

通常の logout と個別/global revoke は物理削除せず `revoked_at` と
`revoke_reason` を冪等に設定する。同一操作の再実行は成功として扱う。

```sql
UPDATE authentication_sessions
SET revoked_at = COALESCE(revoked_at, $3),
    revoke_reason = COALESCE(revoke_reason, $4)
WHERE tenant_id = $1
  AND id = $2;
```

current 以外の全失効は `tenant_id + user_id` で一括更新し、current session id を
除外する。管理者/identity lifecycle からの全失効は除外なしで同じ repository
primitive を使う。物理削除は retention 経過後の housekeeping に限定する。

### 4. 書き込み量

`last_seen_at` は認証済み HTTP request ごとに更新しない。IdP が session を意味上
利用した `/authorize`、account/admin portal、step-up、token refresh 等でのみ touch
し、さらに一定間隔（初期値5分）より新しい行は更新しない。

```sql
UPDATE authentication_sessions
SET last_seen_at = $3
WHERE tenant_id = $1
  AND id = $2
  AND last_seen_at < $3 - interval '5 minutes';
```

idle timeout の判定精度はこの間隔分だけ粗くなることを SCL objective として明示する。
absolute expiry は touch で延長しない。大量 session の期限切れ物理削除は、全件 DELETE
ではなく primary key を選ぶ小 batch を反復する。

### 5. Valkey の責務

Valkey は login 完了後の active session を保存しない。次の短命・高頻度・消失時に
再試行可能な状態だけを担当する。

- WebAuthn challenge と enrollment ceremony。
- login/authorization の未完了 transaction。
- CSRF/step-up challenge。
- login attempt throttle と rate limit counter。
- DPoP、client assertion、logout token 等の replay guard。

Valkey 全損でログイン済み session は失われない。進行中 ceremony は再開が必要でもよい。

### 6. 性能ゲート

キャッシュ追加を設計上の前提にせず、次を測定可能にする。

- session-id lookup の p50/p95/p99 latency と DB query count。
- user session list の p95 latency（1、10、100、1,000 active sessions/user）。
- login/session create と revoke-all の throughput。
- connection pool wait time、lock wait、dead tuple、housekeeping batch duration。

初期目標は session-id lookup p99 20ms 以下、先頭50件のuser session list p99
100ms 以下とする。目標未達時は query/index/pool を先に改善し、その後に限定的な
cache WI を起票する。

### 7. 移行

現行 Valkey session は deploy 時に PostgreSQL へ一括移行しない。保存済み JSON に
完全な durable metadata と列挙保証がなく、二重正本期間を作る方が危険なためである。
切替 deploy では既存 browser session が再認証になることを明示する。rolling deploy
で新旧ノードを混在させず、migration 適用後に全 API node を同じ repository 設定で
起動する。

## Tasks

- [x] T001 [SCL] `LoginSession` の durable fields、失効 lifecycle、idle touch
      granularity、一覧 pagination と失効 scenario を先に更新し、`just yaml-check-scl`
      を通す。tenant_id / step_up_at / last_seen_at / revoked_at / revoke_reason を
      model に追加、RevokeMySession/RevokeMyOtherSessions を物理削除から tombstone
      失効へ改訂、idempotent 再失効と再起動後解決の extension を追加、
      SessionListLatency objective を追加。`just yaml-check-scl` — passed。
- [x] T002 [Decision] Valkey-only session を PostgreSQL single source of truth に
      置換し、切替時の既存 session 再認証を ADR に記録する。
      `decisions/ADR-126-postgresql-as-login-session-source-of-truth.md` を作成
      (tombstone 失効、Find/FindOwned 分離、Resolve 経由 touch、tenant_id 保持の
      明示的例外を記録)。
- [x] T003 [Domain] RED: `TestLoginSessionActive` / `TestLoginSessionRevokeIdempotent` /
      `TestLoginSessionTouch` を先に fail 確認 (未実装フィールド/メソッドでビルド失敗、
      scenario `ユーザーは自分の有効なセッションを一覧して失効できる`) → `LoginSession.Active`
      / `Revoke` (idempotent tombstone) / `Touch` (粗粒度 last_seen_at) と
      LastSeenAt/RevokedAt/RevokeReason フィールド、revoke ペア制約を実装 → GREEN
      (`go test ./backend/authentication/domain/...` — passed)。
- [x] T004 [Persistence] `infra/schema/postgres.sql` に `authentication_sessions`
      (tenant_id 保持は ADR-082 §4 の明示的例外として header に記録) と keyset/cleanup
      index を追加、sqlc query 8 本を実装。schema/sqlc 生成への依存上、
      `TestSessionRepositoryRoundTrip` (embedded-postgres, `pgtest.Require`) は
      repository 実装後に作成・実行して fail 確認 → 修正 → GREEN という順序になった
      (pending_purpose 既定値未設定、tenant 越境で invalid uuid、housekeeping cutoff が
      他 subtest の行を巻き込む、の 3 点で実際に fail し修正、self-attest)。
      port を Find (fail-closed active only) / FindOwned (owner 確認用) / Revoke
      (idempotent tombstone) / Touch (粗粒度) へ改訂し、memory adapter
      (`TestSessionStore`) も同じ contract に同期、Valkey の SessionStore
      (session:\* SCAN) を削除、production bootstrap
      (`postgres_valkey.go`) を `authnpostgres.SessionRepository` へ切替。
      `go build ./...` / `go test ./backend/authentication/...` — passed。
- [x] T005 [Infrastructure] `backend/cmd/internal/bootstrap/postgres_valkey.go` の
      `authentication.Module.SessionStore` を `&authnpostgres.SessionRepository{Pool:
      resilientDB}` へ切替。`backend/authentication/adapters/persistence/valkey/stores.go`
      から `SessionStore`（`session:*` SCAN 含む）を削除（WebAuthnSessionStore は Valkey に
      残す）。memory bootstrap (`bootstrap/memory.go`) は無変更 (PERSISTENCE=memory は
      引き続き memory adapter)。`go build ./...` / `go vet ./...` — passed。
- [x] T006 [Adapter] `account_sessions_handler.go` は `SessionManager.Store`
      (port 経由) への薄い委譲のみでハンドラ自身は状態を持たないため、"再起動後も
      解決できる" 保証の実体は repository 層にある。
      `TestSessionResolutionSurvivesProcessRestart`
      (`backend/authentication/adapters/persistence/postgres/sessions_test.go`) を追加:
      別インスタンスの SessionManager/SessionRepository (同一 Pool、"process A/B/C" を
      模擬) で作成→解決→失効→再解決を検証 (scenario 拡張「process 再起動を挟んで
      セッション一覧を取得する」)。実装は T004 で完成済みのため、この test は
      confirmatory (self-attest、T004 と同じ codegen 依存の理由)。副産物として
      `Active()` の厳密な `now.Before(ExpiresAt)` 判定 (SQL `expires_at > $3` と整合) が
      `step_up_test.go` の3箇所の境界一致 (`now.Add(-time.Hour)` = TTL ちょうど) を露呈し
      GREEN で fail したため `-30*time.Minute` に修正 (real RED→fix→GREEN)。
      `go test ./backend/...` — passed (全パッケージ)。
- [x] T007 [Performance] `TestSessionQueryPlansUseIndexes` を追加: tenant 内 5,000 行
      (対象 user 200 行 + 他 24 user 各200行) を `pgx.Batch` で投入・`ANALYZE` 後、
      session lookup と user list の `EXPLAIN (ANALYZE, BUFFERS)` を検証し、両方とも
      `Seq Scan` を含まず `authentication_sessions_pkey` /
      `authentication_sessions_active_user_idx` の Index Scan であることを確認 (GREEN、
      実行時間 lookup 0.008ms / list(50件) 0.018ms、embedded-postgres 上)。
      `BenchmarkSessionRepository_Find` / `BenchmarkSessionRepository_ListBySub`
      (1/10/100/1,000 sessions/user) を追加: Find ~51µs/op、List は 47µs(1件)→84µs
      (1,000件) と緩やかな増加で目標 (lookup p99 20ms、list p99 100ms) を大幅に下回る。
      **開示**: これは単一プロセス・embedded-postgres 上のローカル測定であり、
      Verification が要求する「100万 session、ネットワーク越しの本番相当負荷試験」は
      本セッションでは実施していない (環境制約)。クエリ計画と件数スケーリング特性の
      一次的な妥当性確認に留まる。本番投入前に本物の負荷試験を別途実施することを
      Completion で開示・推奨する。
- [x] T008 [Docs/Verify] ルート `README.md` の「High Availability & Shared State」節を
      更新: Valkey が保持する状態一覧から login session を除外し、PostgreSQL 側に
      「login sessions (since wi-253 / ADR-126)」を追記。deploy 切替時に Valkey 保存分の
      既存セッションは移行されず一度だけ再ログインが必要になること、切替後の新規
      セッションは Valkey 再起動や API replica 再起動を跨いで生存することを明記。
      `account_sessions_handler.go` 冒頭コメントの「物理削除」記述も tombstone に修正。
      全検証 (下記) を実行し green を確認。

## Verification

- `just yaml-check`
- `just sqlc-generate`
- `just verify-go`
- `just verify`
- PostgreSQL integration: create → process restart → resolve が成功する。
- PostgreSQL integration: revoke → process restart → resolve が未認証になる。
- `EXPLAIN (ANALYZE, BUFFERS)` 相当の証跡で、session lookup とuser listが
  sequential scanにならないことを確認する。
- 負荷確認: 100万session、1ユーザーあたり1/10/100/1,000 active sessionのfixtureで
  lookup/list/revoke-allを測定する。

## Risk Notes

最大のリスクは PostgreSQL への高頻度な `last_seen_at` 更新と、期限切れ行の一括削除
による write amplification / vacuum / lock 競合である。coarse touch、absolute expiry、
partial index、小batch cleanupで抑える。

PostgreSQL障害時は既存sessionをfail-closedにし、未認証として扱う。可用性を理由に
Valkeyの古い状態へfallbackしない。DB failover、pool sizing、statement timeoutは既存の
PostgreSQL運用基盤を利用し、本WIでは新たなmulti-store consistency protocolを作らない。

session id は固定長の不透明値で、複雑な未信頼文法を解釈しないためfuzz testは採用しない。
tenant isolation、期限境界、同時revoke/touchはproperty/競合testを採用する。

## Completion

- **Completed At**: 2026-07-18
- **Summary**:
  ログイン済み LoginSession の単一正本を Valkey から PostgreSQL `authentication_sessions`
  へ移した (ADR-126)。session id は tenant_id を含めた fail-closed な indexed lookup
  (`authentication_sessions_pkey`) で解決し、失効は物理削除ではなく `revoked_at` /
  `revoke_reason` の idempotent tombstone とした。`SessionStore` port を
  `Find`(active-only) / `FindOwned`(所有者確認) / `Revoke`(idempotent) / `Touch`(粗粒度
  last_seen_at) へ再設計し、`SessionManager.Resolve` から一括で touch することで
  oauth2/idmanagement/frontend への配線を増やさずに書き込み量を抑えた。memory /
  postgres 両 adapter を同じ contract に同期し、Valkey の login session 保存
  (`SCAN session:*` を含む) は撤去した。production bootstrap
  (`postgres_valkey.go`) を PostgreSQL 実装に切替済み。
- **Verification Results**:
  - `just yaml-check` - passed
  - `just sqlc-generate` - passed (差分は生成物のみ、コミット対象)
  - `just build-go` - passed
  - `just verify-go` (lint 0 issues, race test 全パッケージ) - passed
  - `just verify` (yaml-check / verify-go / verify-ui 含む全体) - passed
    (初回実行で `AccountDataPage.test.tsx` が 1 件 flaky fail したが、単体実行および
    UI unit test 一式の単独再実行 (397/397) で再現せず、本 WI が触れていない
    `frontend/` には変更が無いことを `git status` で確認済み。2 回目の `just verify`
    フル実行で完全 green を確認)
  - `TestSessionResolutionSurvivesProcessRestart` - passed (process 再起動シミュレーション:
    作成→別インスタンスで解決→失効→別インスタンスで未認証確認)
  - `TestSessionQueryPlansUseIndexes` - passed (`EXPLAIN (ANALYZE, BUFFERS)` で session
    lookup / user list の両方が Index Scan、Seq Scan なしを確認)
- **開示 (未達成・部分実装)**:
  - Verification が要求する「100万 session、1/10/100/1,000 active session/user の
    fixture での本番相当負荷試験」は本セッションでは実施していない。実施したのは
    embedded-postgres 単一プロセス上の 5,000 行 fixture でのクエリ計画検証と、
    1〜1,000 sessions/user の `go test -bench` 測定 (Find ~51µs/op、List 47〜84µs/op)
    のみで、これは目標 latency (lookup p99 20ms、list p99 100ms) を大きく下回る一次的な
    傾向確認に留まる。本番投入前に実ネットワーク・実スケールでの負荷試験を別途実施することを
    推奨する。
  - Plan §4 で例示した touch 契機 (`/authorize`、account/admin portal、token refresh) への
    個別配線はしていない。代わりに `SessionManager.Resolve` から一律に touch し、
    adapter 側の粗粒度ガードで書き込み量を抑える設計に変更した (ADR-126 に理由を記録)。
    将来的に他 context から明示的に touch を呼びたい場合も同じ port で追加できる。
  - housekeeping batch cleanup (`DeleteExpiredBatch`) は repository メソッドとテストまでで、
    定期実行する scheduler への配線はしていない。プロジェクト全体に既存の類似 scheduler
    基盤が無く (password_reset_tokens 等の他の expires_at 付きテーブルも同様に未配線)、
    本 WI 単独で新設するのは scope 外と判断した。
