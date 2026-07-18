---
status: pending
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

- [ ] T001 [SCL] `LoginSession` の durable fields、失効 lifecycle、idle touch
      granularity、一覧 pagination と失効 scenario を先に更新し、`just yaml-check-scl`
      を通す。
- [ ] T002 [Decision] Valkey-only session を PostgreSQL single source of truth に
      置換し、切替時の既存 session 再認証を ADR に記録する。
- [ ] T003 [Domain] RED: active/revoked/expired、tenant isolation、current/other/global
      revoke の property/contract test を先に fail 確認
      （`LoginSession` lifecycle、scenario `ユーザーは自身の有効なセッションを管理する`）
      → GREEN。
- [ ] T004 [Persistence] RED: PostgreSQL repository contract、user keyset pagination、
      idempotent revoke、coarse touch、batch cleanup の integration test を先に fail
      確認（interfaces `ListMySessions` / `RevokeMySession` /
      `RevokeMyOtherSessions`）→ GREEN。migration と sqlc query を追加する。
- [ ] T005 [Infrastructure] production bootstrap を PostgreSQL repository へ切り替え、
      Valkey `SessionStore` と `SCAN session:*` を除去する。
- [ ] T006 [Adapter] RED: restart 後も session が解決でき、revoke 後は再認証を要求する
      HTTP/E2E test を先に fail 確認（`LoginSession` lifecycle）→ GREEN。
- [ ] T007 [Performance] key lookup/list/revoke-all の負荷シナリオを追加し、Plan の
      latency、query plan、pool wait 指標を記録する。
- [ ] T008 [Docs/Verify] Valkey/PostgreSQL の責務とdeploy時の再認証を README に記載し、
      全検証を通す。

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
