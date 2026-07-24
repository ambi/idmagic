---
status: accepted
authors: [tn]
created_at: 2026-07-24
---

# ADR-139: 揮発性の認証 / OAuth2 一時状態を全て PostgreSQL に統合し Valkey を廃止する

## コンテキスト

idmagic は 2 つのステートフル基盤に依存してきた。durable state 全部を持つ
**PostgreSQL** (必須) と、揮発性の認証 / OAuth2 一時状態だけを持つ **Valkey**
(本番は GCP Memorystore STANDARD_HA) である。ADR-016 は「durable=Postgres /
volatile=Valkey」を定め、`PERSISTENCE=postgres_valkey` を本番構成名とした。
ADR-077 は login throttle を Valkey の共有カウンタに載せ、ephemeral state の HA 前提を
明文化した。ADR-126 は既に login session を Valkey → PostgreSQL に移している。

Valkey が現在持つのは 9 ストアだけである: 認可中間状態 (AuthorizationRequest)、
認可コード、PAR、device code、JTI リプレイ (DPoP / client assertion)、
アクセストークン失効リスト (denylist)、WebAuthn チャレンジ、login throttle、
SAML AuthnRequest リプレイ。**pub/sub もキューも永続データも Valkey には無い**
(ドメインイベントは Kafka / PubSub、ジョブキューは既に PostgreSQL、ADR-090)。
つまり Valkey は "durable でない一時状態の入れ物" にすぎず、PostgreSQL がどのみち必須で
ある以上、Valkey は運用上の 2 本目の critical path・追加コスト (月 $150-200 想定)・
セルフホスト導入障壁・HA 運用対象を増やすだけの "2 つ目の基盤" になっている。

全ストアは既に ports で抽象化され、memory / valkey / postgres が同一インターフェースを
満たす (ADR-016 の adapter replaceability、ADR-090 の context-local sqlc)。ADR-126 が
login session で先例を作った。本 ADR はその軌道の完遂として、残る揮発性ストアを全て
PostgreSQL に統合し Valkey を撤去する決定を固定する。

**スケーラビリティの検討**。9 ストアは 2 群に割れる。(A) 対話レート律速で短命の 8 ストア
(authorization request / code / PAR / device / replay / webauthn / saml replay /
login throttle) は、ユーザー操作やプロトコル往復の頻度でしか書かれず、TTL で消える。
PostgreSQL で容易に扱える。(B) `AccessTokenDenylist.IsRevoked` だけが introspection で
リクエスト毎に走り得る唯一のホット読取である。しかし PostgreSQL はスケールアップ
(ホットセットを RAM に常駐) と読取レプリカが得意で、本ワークロードは PostgreSQL が
苦手な水平書込シャーディングを踏まない。高 RPS の introspection は共有ストアの選択
(Valkey か Postgres か) とは独立に、「per-request lookup を避ける設計」で解くのが定石で
ある — 短命アクセストークンで denylist 参照自体を減らす、instance-local な短 TTL キャッシュ、
失効の push (revocation の out-of-band 伝播)。したがって「denylist のために Valkey が
要る」は偽の前提であり、本 WI の対象 (per-request の共有ストア参照を Postgres に置換する)
と denylist 高 RPS 最適化は別レイヤの課題である。

## 決定

1. **揮発性の認証 / OAuth2 一時状態 9 ストアを全て PostgreSQL に統合し、Valkey を廃止する。**
   各ストアの port 契約は不変のまま、`db_postgres` アダプタを新設する。ADR-126 の
   `authentication_sessions` を sqlc アダプタのテンプレートとし、既存の
   memory / valkey adapter と同じ契約テストでパリティを検証する。

2. **TTL の正しさは `expires_at > now()` 述語で担保し、GC から切り離す。** 全 read クエリに
   `AND expires_at > now()` (または相当のパラメータ) を付け、期限切れ行を絶対に返さない。
   物理削除 (GC) は空間回収のみを担い、cadence は正しさに無関係な best-effort とする。
   各ストアは `DeleteExpiredBatch(ctx, cutoff, limit)` (ADR-126 の
   `session_store` と同一シグネチャ) を実装し、常駐 `idmagic-worker` の周期 sweep から
   呼ぶ。

3. **原子操作は Valkey の Lua / SETNX / GETDEL を SQL に写し替える。**
   - Lua CAS → `UPDATE ... WHERE <cas 条件> RETURNING *` (行が返れば成功、
     `ErrNoRows` → nil ＝ Valkey の `goredis.Nil` / `false` と同義)。
   - `SET NX` → `INSERT ... ON CONFLICT DO NOTHING` ＋挿入行数 (rows==1 で新規)。
   - `GETDEL` → `DELETE ... RETURNING`。
   - `WATCH` 楽観ロック → tx 内 `SELECT ... FOR UPDATE`。

4. **storage 特性は揮発度で 2 分する (暫定、Phase 3 で staging 実測により確定する)。**
   消えても再開で回復する高 churn なストア (authorization request / code / PAR /
   device / replay / webauthn / saml replay) は **UNLOGGED** とし、WAL を書かず Valkey の
   揮発性に最も近づける。失効リスト (denylist) と login throttle は failover で状態が
   消えると防御が後退する (denylist は失効の取りこぼし、throttle は fail-closed 前提の
   崩れ) ため **LOGGED** とし、物理 standby へ複製する。throttle は `fillfactor 80` ＋
   HOT update で dead tuple を抑える。UNLOGGED / LOGGED の最終判断と GC 間隔は、
   dead tuple / autovacuum / p99 の staging 実測 (Phase 3) で確定する。

5. **fail-closed ポリシー (ADR-077) は維持し、依存を追加でなく削減する。** login throttle と
   denylist の共有性・fail-closed は変えない。共有ストアが PostgreSQL に変わるだけで、
   PostgreSQL は既に hard dependency なので、throttle / denylist を Postgres に載せても
   critical path は増えず、むしろ Valkey という 2 本目を消して 1 本に減る。到達不能時に
   throttle がエラーを返しログインを素通しさせない挙動は不変である。

6. **denylist の高 RPS 最適化は本 WI の対象外とし、共有ストア選択と独立の設計方針とする。**
   短命アクセストークン化・instance-local キャッシュ・push 失効・denylist の時間
   パーティション + `DROP PARTITION` GC は、必要になった時点で別 WI として起票する。
   本 ADR はこれらを "Valkey を残す理由にはならない" 独立レイヤの最適化として記録するに
   留める。

7. **移行はデータ移行不要・dual-write 不要とする (ADR-126 と同じ割り切り)。** ephemeral は
   切替時の in-flight フロー (進行中の /authorize・PAR・device・throttle counter) が
   放棄されても再開で回復するため、Valkey → Postgres のデータ移行も二重書きもしない。
   config の `PERSISTENCE` モード名は `postgres` に統一し、旧 `postgres_valkey` は alias を
   残さず削除する。切替は各環境の環境変数を `postgres` へ更新するデプロイと同時に行う
   (未更新の環境は起動時に "PERSISTENCE must be memory or postgres" で fail-fast する)。

8. **tenant scoping は opaque token key の高頻度 fail-closed lookup として `tenant_id` を
   保持する。** これら ephemeral は全て不透明トークン / コード / チャレンジをキーにした
   高頻度の fail-closed lookup であり、ADR-082 §4 / ADR-083 の「globally unique parent の
   child は tenant_id を省略する」既定に対する明示的な例外に該当する
   (`authentication_sessions` と同じ理由、ADR-126)。既存 `authentication_sessions` に倣い
   `tenant_id UUID NOT NULL` + `tenants(id) ON DELETE CASCADE` を持つ。

## 却下した代替案

- **Valkey を残す。** 運用負荷 (コスト・HA 運用・セルフホスト導入障壁・critical path が
  2 本) が実際に痛く、方向は ADR-126 で既に定まっている。Valkey が持つのは揮発状態だけで、
  pub/sub もキューも永続データも無いため、残す固有の価値がない。
- **別の Redis 系 (KeyDB / Dragonfly) に載せ替える。** プロトコル互換で差し替えは容易だが、
  "2 つ目の基盤" が消えないため本 WI の目的 (運用対象を 1 本に減らす) に反する。
- **主 DB を別物 (分散 KVS 等) に変える。** durable 層の巨大な再投資で本末転倒。ADR-016 が
  PostgreSQL を durable の正本と定めた判断は有効。
- **denylist のために Valkey を残す。** 上記コンテキストのとおり偽の前提。高 RPS
  introspection は共有ストア選択と独立に per-request lookup を避ける設計で解く。

## 影響

- **supersede**: [ADR-016](ADR-016-persistence-adapter-selection.md) の「volatile state は
  Valkey」節 (決定 2) と `PERSISTENCE=postgres_valkey` の構成名は本 ADR が置き換える。
  durable=Postgres / events=outbox→Kafka は有効のまま。
- **supersede (機構部)**: [ADR-077](ADR-077-shared-login-throttle-store-and-ephemeral-state-ha.md)
  の「throttle を Valkey 共有カウンタに載せる」機構は Postgres の `login_throttle_counters`
  へ置き換える。fail-closed ポリシー・共有カウンタ・SHA-256 識別子ハッシュ・tenant scoping は
  維持する (依存を追加でなく削減する)。
- **改訂**: [ADR-105](ADR-105-system-runtime-hardening-and-i18n-tooling.md) の Valkey 接続
  resilience / readiness Ping / ephemeral state 配置節、
  [ADR-106](ADR-106-identity-and-credential-policy-configuration.md) の Valkey 共有ストア
  参照は Postgres 単一依存に読み替える (これらが参照していた `ValkeyResilience` /
  `SharedEphemeralStateHA` SCL objective は SCL 3.0 移行 (ADR-103) で既に除去済みで、
  現行 SCL には Valkey 依存の objective は無い)。
- **SCL**: `spec/contexts/system.yaml` の readiness scenario extension
  (「PostgreSQL または Valkey へ到達できない」→「PostgreSQL へ到達できない」)、
  `spec/contexts/authentication.yaml` の `LoginSession` 記述 (「Valkey 全損を跨いで」→
  「プロセス再起動を跨いで」)、`spec/contexts/jobs.yaml` の dev infra 記述
  (「Valkey 互換 endpoint」除去) を Postgres 単一依存へ更新する。
- **schema**: `infra/schema/postgres.sql` に 9 テーブルを追加する
  (`oauth2_authorization_requests` / `oauth2_authorization_codes` / `oauth2_par_requests` /
  `oauth2_device_codes` / `oauth2_replay_jtis` / `oauth2_access_token_denylist` /
  `webauthn_sessions` / `login_throttle_counters` / `saml_authnrequest_replays`)。
  UNLOGGED / LOGGED を決定 4 に従い付し、`expires_at` index と tenant_id 保持理由を
  スキーマ冒頭コメントに追記する。
- **adapter**: `backend/oauth2/db_postgres/`・`backend/authentication/session/db_postgres/`
  (throttle)・`backend/authentication/webauthn/db_postgres/`・`backend/saml/db_postgres/` に
  postgres アダプタを新設し、契約テストで memory / valkey とパリティを取る。
- **bootstrap**: `backend/cmd/internal/bootstrap/postgres_valkey.go` を `postgres.go`
  (`assemblePostgres`) へ改称し、Valkey client / config / breaker / ValkeyPing を除去、
  9 バインドを `*postgres.*` へ置換する。`deps.go` は `case "postgres":` のみとし
  (旧 `postgres_valkey` は alias を残さず削除)、health 系から ValkeyPing を除く。
- **worker**: `EphemeralPurger` usecase を新設し `idmagic-worker` の周期 sweep へ配線する。
- **infra (Phase 4)**: `docker-compose.dev.yaml` の valkey service、k8s configmap /
  networkpolicy、gcp Memorystore / secret、`cloudrun-idmagic.yaml`、`dev.sh`、
  dev infra の miniredis、`redis/go-redis/v9` / `alicebob/miniredis/v2` 依存を撤去し、
  `PERSISTENCE=postgres` に統一する。

## 関連

- [ADR-126](ADR-126-postgresql-as-login-session-source-of-truth.md) — login session を
  先行して Postgres へ移した先例。本 ADR がその軌道を完遂する。
- [ADR-016](ADR-016-persistence-adapter-selection.md) — 本 ADR が volatile=Valkey を supersede。
- [ADR-077](ADR-077-shared-login-throttle-store-and-ephemeral-state-ha.md) — throttle 機構を supersede、fail-closed は維持。
- [ADR-090](ADR-090-context-local-persistence-and-sqlc.md) — context-local な sqlc アダプタ規約。
