# IdMagic デプロイ構成 — GCP 単一クラウド

compute / DB / Redis / イベント配信 を **単一 VPC・単一リージョン**に同居させる構成。レイテンシと egress で有利、CUD（確約利用割引）で大きく圧縮でき、運用は単一ベンダで完結する。前提ワークロードは中規模 SaaS・本番 HA・イベント配信(event-relay) 含む。

イベント配信は **Pub/Sub**（サーバレス・クラスタフロア無し）を採用する。relay は transport 中立（[ADR-120](../../../decisions/ADR-120-event-relay-transport-abstraction-and-pubsub.md)）で、GCP では `RELAY_SINK=pubsub`（`-tags pubsub` ビルド）を選ぶ。イベント層は ~$0–10/月に収まる。

## 前提アーキテクチャ

- フロント: React+Vite の**純 SPA**（`frontend/` → `dist/`）。SSR 無し。
- ゲートウェイ: **Caddy**（`frontend/Caddyfile`）が静的配信＋同一オリジンで API/OIDC パスを backend へプロキシ。CSP は SPA HTML のみに付与(ADR-076)。
- backend: 常駐 Go **3サービス** — `idmagic`(API `:8080`) / `idmagic-worker` / `idmagic-relay`。CGO 無し distroless（`infra/docker/Dockerfile`）。`PERSISTENCE=postgres_valkey` でステートレス・水平スケール可。
- データ: **PostgreSQL 17**（本体+blob, 44テーブル, FTS 無し）/ **Valkey/Redis**（セッション・OAuth 一時状態）/ **Pub/Sub**（outbox 中継、[ADR-120](../../../decisions/ADR-120-event-relay-transport-abstraction-and-pubsub.md)）。
- 署名鍵: **DB-backed 永続鍵(ADR-024)** を推奨（全レプリカ JWKS 一致）。Vault Transit(ADR-075) も可。
- スキーマ: `psqldef` を**デプロイ工程で適用**（起動時適用は ADR-071 で禁止、`--enable-drop` 禁止）。

## 構成図

```
ユーザ
  ▼
Cloud Load Balancing (HTTPS) + Cloud CDN + Cloud Armor(WAF)
  │
  ├─ 静的 SPA … GCS バケット + Cloud CDN
  │
  └─ /api・/authorize・/token・/.well-known 等 → Cloud Run (idmagic API, minScale=2, HA)
                                                     │
        ┌────────────────────────────────────────────┼─────────────────────────────┐
        ▼                        ▼                    ▼                              ▼
  Cloud SQL for PostgreSQL   Memorystore(Valkey)   Pub/Sub                       Secret Manager
   (REGIONAL = HA)            (STANDARD_HA)          (サーバレス, topic/event)      / Cloud KMS

背景処理（HTTP を持たない常駐プロセス）:
  Cloud Run worker pools ─ idmagic-worker（ジョブ+保持スイープ, ADR-099）
                          ─ idmagic-relay （outbox → Pub/Sub, RELAY_SINK=pubsub）
```

## サービス対応

| プロセス | 実行形態 | 理由 |
|---|---|---|
| `idmagic`(API) | **Cloud Run Service** | HTTP(`:8080`) を提供。`minScale=2` で HA・オートスケール |
| `idmagic-worker` | **Cloud Run worker pools** | HTTP を持たない常駐ワーカ。`$PORT` リッスン不要の worker pools が適合 |
| `idmagic-relay` | **Cloud Run worker pools** | 同上（outbox→Pub/Sub の常駐ポーラ、`-tags pubsub` ビルド） |
| `idmagic-seed` | Cloud Run Job（任意・一過性） | 初期シード |

> Cloud Run の通常 Service は `$PORT` への HTTP 応答が必須のため、HTTP を持たない worker/relay は **worker pools** を使う。

## デプロイ順（重要）

1. イメージビルド（既存 `infra/docker/Dockerfile`、3バイナリ入り distroless）→ Artifact Registry。relay の Pub/Sub を有効化するため `--build-arg RELAY_TAGS=pubsub` を付ける
2. データ払い出し: Cloud SQL(REGIONAL) / Memorystore(Valkey) / Pub/Sub topic ＋ relay 用サービスアカウント(`roles/pubsub.publisher`)
3. Secret 登録（`DATABASE_URL` / `VALKEY_URL` 等）
4. **スキーマ適用**: `psqldef --apply`（**起動時ではなくこの工程で**。ADR-071 / `--enable-drop` 禁止）
5. サービス投入: `idmagic`(Service) → `idmagic-worker`/`idmagic-relay`(worker pools, `RELAY_SINK=pubsub`)
6. 前段: Cloud Load Balancing + Cloud CDN + Cloud Armor、DNS/TLS

> 消費側（Pub/Sub サブスクライバ）がまだ無い間は **relay を起動しない**運用も可。イベントは outbox テーブルに永続するため取りこぼさず、サブスクライバができた時点で relay を有効化すればよい（イベント層 $0）。

雛形は [`provision.sh`](./provision.sh)（払い出し＋デプロイ）と [`cloudrun-idmagic.yaml`](./cloudrun-idmagic.yaml)（API Service）を参照。

## 主な環境変数

| 変数 | 値 | 備考 |
|---|---|---|
| `PERSISTENCE` | `postgres_valkey` | ステートレス・水平スケール前提 |
| `DATABASE_URL` | Secret | Cloud SQL（Private IP か Unix ソケット） |
| `VALKEY_URL` | Secret | Memorystore |
| `RELAY_SINK` | `pubsub` | relay の配信先（`kafka`\|`pubsub`\|`log`）。GCP は `pubsub` |
| `PUBSUB_PROJECT` | project-id | relay のみ必要（`RELAY_SINK=pubsub`） |
| `EVENT_SINK` | `outbox` | アプリ側で outbox 書き込みを有効化 |
| `KEY_PROVIDER` | `db` | DB-backed 署名鍵(ADR-024)。全レプリカ JWKS 一致 |
| `ISSUER` | `https://id.example.com` | discovery の issuer と一致必須 |
| `OBSERVABILITY` / `OTEL_EXPORTER_OTLP_ENDPOINT` | `otel` / collector | OTLP 送出、`/metrics` は pull |

## HA / スケール

- API: `minScale=2`（最低2レプリカ）、`maxScale` は負荷に応じて。ステートレスなので水平スケール可。
- worker/relay: リース制（ADR-099）のため複数インスタンス可。`min-instances>=1`。
- DB: `REGIONAL`（同期スタンバイ）。Valkey: `STANDARD_HA`（レプリカ）。
- 署名鍵は DB-backed で全レプリカ一致を担保（Vault を使う場合は別途）。

## コスト目安（中規模 SaaS・HA・リスト価格）

| 項目 | 構成 | 月額(USD) |
|---|---|---|
| Cloud Run | API×2 + worker/relay pools | $180–250 |
| Cloud SQL PostgreSQL HA | 2–4 vCPU/8–16GB + 100GB SSD | $300–450 |
| Memorystore for Valkey | Standard/HA ~5GB | $150–200 |
| Pub/Sub | outbox イベント（低〜中量, サーバレス） | $0–10 |
| LB + Cloud CDN + GCS(SPA) | | $30–60 |
| Secret Manager/KMS/ログ/egress | | $30–60 |
| **合計** | | **~$690–1,030（中心 ~$850）** |

Pub/Sub はサーバレスでクラスタの固定費が無く、イベント量（低〜中量）に対して小さいコストに収まる（[ADR-120](../../../decisions/ADR-120-event-relay-transport-abstraction-and-pubsub.md)）。CUD（1年 20–25% / 3年 40–52%）で **~$650–850** まで低下しうる（compute/DB/Redis の compute 分に適用、storage は対象外）。
