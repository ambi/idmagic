---
status: accepted
authors: [tn]
created_at: 2026-07-18
---

# ADR-126: ログイン済みセッションの単一正本を Valkey から PostgreSQL へ変更する

## コンテキスト

production の `SessionStore` は Valkey の `session:*` にログイン済み `LoginSession` を
保存してきた。ユーザー別一覧・全失効は `SCAN` 後にアプリケーション側で `user_id` を
絞り込む実装で、セッション総数に比例して探索量が増える。加えて Valkey の再起動・
eviction・全損時には有効なログイン状態そのものが失われ、意図しない大量再認証が
起きる。

PostgreSQL と Valkey の双方に同じ active session を保持する案は、失効時の更新順序、
stale cache、cache invalidation、再試行 outbox まで必要になり、セキュリティ上重要な
logout 経路の複雑さが増す。Keycloak 26 は persistent user sessions を既定にし、
database を session source of truth として cache-on-demand を後段に置く
(https://www.keycloak.org/2024/12/storing-sessions-in-kc26,
https://www.keycloak.org/server/caching)。idmagic でもまず PostgreSQL の indexed
query に性能を委ね、測定で必要性が示されるまで active session cache を導入しない
単一ストア構成から始める (wi-253)。

## 決定

1. **PostgreSQL `authentication_sessions` を LoginSession の単一正本とする。** Valkey は
   login 完了後の active session を保存しない。認証解決・一覧・失効はすべて
   PostgreSQL の indexed query で行う。

2. **失効は物理削除ではなく tombstone。** `revoked_at` / `revoke_reason` を
   `COALESCE` で冪等に設定し、同一セッションへの再失効要求は成功として扱う
   (2 回目以降は no-op)。物理削除は retention 経過後の housekeeping batch に限定する。
   これにより self-service revoke API の「対象が既に存在しない」と「対象が既に
   失効済み」を区別しつつ、後者を利用者にエラーとして見せない設計にできる。

3. **認証解決とオーナー確認で異なる可視性の lookup を分ける。** `SessionStore.Find`
   は `revoked_at IS NULL AND expires_at > now` の行だけを返す fail-closed な
   解決専用メソッドとし、`FindOwned` は revoked/expired を含めて対象行の所有者確認に
   使う。1 つの `Find` に両方の意味を持たせると、認証解決側が revoked セッションを
   誤って有効と扱うリグレッションを作りやすいため分離する。

4. **`last_seen_at` の touch は `SessionManager.Resolve` から常時呼び、書き込み抑制は
   永続化層の粗粒度ガードに任せる。** Plan で例示した touch 契機
   (`/authorize`、account/admin portal、step-up、token refresh) は `backend/oauth2` /
   `backend/idmanagement` / `frontend` にまたがり、本 WI の scope
   (`backend/authentication` に閉じる) を越える。代わりに、既に全認証済みリクエストが
   経由する `Resolve` から無条件に touch を呼び、adapter 側の
   `last_seen_at < now - LoginSessionTouchInterval` 条件更新に実際の書き込み判断を
   委ねる。呼び出し側の分岐を増やさずに書き込み量を抑える目的を達成できる。他
   context からの明示的な touch 呼び出し (token refresh 等) は、必要になった時点で
   同じ port を使って追加できる。

5. **`tenant_id` 列を持つ ([ADR-082](ADR-082-user-domain-id-and-tenant-key-policy.md) /
   [ADR-083](ADR-083-globally-unique-client-id.md) の「globally unique parent の
   child は tenant_id を省略する」という既定に対する明示的な例外)。**
   `authentication_sessions.id` は globally unique な UUID で `user_id`
   (globally unique) からも tenant を辿れるが、session id は browser cookie
   由来の不透明値であり、認証解決のたびに fail-closed な境界検証を経る
   セキュリティクリティカルな lookup である。`tenant_id` を検索条件に含めることで、
   session id が推測または別 tenant で再利用されても DB 層で別 tenant の行を
   返さないことを保証する。ADR-082 §4 が認める「per-tenant 高頻度検索・監査隔離が
   要件なら例外として保持」に該当し、`refresh_tokens` の tenant index と同じ理由に
   よる意図的な保持である。

## 却下した代替案

- **PostgreSQL 読み取りに Valkey cache を前段に置く。** Keycloak の cache-on-demand
  に相当するが、初期実装から二重正本の複雑さ (invalidation、再試行 outbox)
  を持ち込む。計測で indexed query の性能が目標未達と分かってから限定的な cache WI
  を起票する。
- **`Find` を 1 メソッドのまま `includeRevoked bool` 引数で分岐させる。** 呼び出し側が
  引数を誤ると認証解決が revoked セッションを有効と誤認するリグレッションに直結する
  ため、型で強制できる 2 メソッド分離を採る。
- **touch 契機を Plan の例示どおり `/authorize`・account/admin portal・token refresh に
  個別配線する。** 正確だが `backend/oauth2` / `backend/idmanagement` /
  `frontend` への変更が必要になり、本 WI の `stop_before_reading` 境界を越える。
  `Resolve` 経由 + adapter 側ガードで同じ書き込み抑制効果を context 横断なしに得る。
- **deploy 時に Valkey の既存 session を PostgreSQL へ一括移行する。** 保存済み JSON に
  完全な durable metadata と列挙保証がなく、二重正本期間を作る方が危険。切替 deploy
  では既存 browser session の再認証を許容する。

## 影響

- `spec/contexts/authentication.yaml`: `models.LoginSession` に `tenant_id` /
  `step_up_at` / `last_seen_at` / `revoked_at` / `revoke_reason` を追加し、
  `interfaces.RevokeMySession` / `RevokeMyOtherSessions` の説明を物理削除から
  tombstone 失効へ改訂、`objectives.SessionListLatency` を追加 (wi-253 T001)。
- `infra/schema/postgres.sql`: `authentication_sessions` テーブルと index を新設
  (wi-253 T004)。tenant_id 保持列挙 (schema 冒頭コメント) に追記する。
- `backend/authentication/ports/session_store.go`: `Find` / `FindOwned` / `Revoke` /
  `Touch` / `ListBySub` / `DeleteAllForSub` へ改訂。
- `backend/authentication/adapters/persistence/{memory,postgres}`: 新しい port
  契約の実装。Valkey の `SessionStore` (`adapters/persistence/valkey/stores.go`)
  および `SCAN session:*` は production bootstrap から除去する。
- `backend/cmd/internal/bootstrap/postgres_valkey.go`: production の
  `SessionStore` を PostgreSQL 実装へ切り替える。
