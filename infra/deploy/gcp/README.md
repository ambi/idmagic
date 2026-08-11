# IdMagic デプロイ構成 — GCP 単一クラウド

compute / DB / イベント配信 を **単一 VPC・単一リージョン**に同居させる構成。レイテンシと egress で有利、CUD（確約利用割引）で大きく圧縮でき、運用は単一ベンダで完結する。前提ワークロードは中規模 SaaS・本番 HA・イベント配信(event-relay) 含む。



## 前提アーキテクチャ

- フロント: React+Vite の**純 SPA**（`frontend/` → `dist/`）。SSR 無し。
- ゲートウェイ: **Caddy**（`frontend/Caddyfile`）が静的配信＋同一オリジンで API/OIDC パスを backend へプロキシ。CSP は SPA HTML のみに付与。
- backend: 常駐 Go **2サービス** — `idmagic`(API `:8080`) / `idmagic-worker`。CGO 無し distroless（`infra/docker/Dockerfile`）。`PERSISTENCE=postgres` でステートレス・水平スケール可。
- データ: **PostgreSQL 17**（本体+blob＋セッション・OAuth 一時状態）。揮発性状態も同じ PostgreSQL が持つため、2 つ目のステートフル基盤は無い。
- 署名鍵: **DB-backed 永続鍵**を推奨（全レプリカ JWKS 一致）。Vault Transit も可。
- スキーマ: `psqldef` を**デプロイ工程で適用**（起動時適用は禁止、`--enable-drop` 禁止）。

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
        ┌─────────────────────────────────────┼─────────────────────────────┐
        ▼                                                                   ▼
  Cloud SQL for PostgreSQL                                            Secret Manager
   (REGIONAL = HA, 揮発性状態も同居)                                    / Cloud KMS

背景処理（HTTP を持たない常駐プロセス）:
  Cloud Run worker pools ─ idmagic-worker（ジョブ+保持スイープ）
```

## サービス対応

| プロセス | 実行形態 | 理由 |
|---|---|---|
| `idmagic`(API) | **Cloud Run Service** | HTTP(`:8080`) を提供。`minScale=2` で HA・オートスケール |
| `idmagic-worker` | **Cloud Run worker pools** | HTTP を持たない常駐ワーカ。`$PORT` リッスン不要の worker pools が適合 |

| `idmagic-seed` | Cloud Run Job（任意・一過性） | 初期シード |

> Cloud Run の通常 Service は `$PORT` への HTTP 応答が必須のため、HTTP を持たない worker/relay は **worker pools** を使う。

## デプロイ順（重要）

1. イメージビルド（既存 `infra/docker/Dockerfile`、2バイナリ入り distroless）→ Artifact Registry。
2. データ払い出し: Cloud SQL(REGIONAL)
3. Secret 登録（`DATABASE_URL` 等）
4. **スキーマ適用**: `psqldef --apply`（**起動時ではなくこの工程で**。`--enable-drop` 禁止）
5. サービス投入: `idmagic`(Service) → `idmagic-worker`(worker pools)
6. 前段: Cloud Load Balancing + Cloud CDN + Cloud Armor、DNS/TLS

雛形は [`provision.sh`](./provision.sh)（払い出し＋デプロイ）と [`cloudrun-idmagic.yaml`](./cloudrun-idmagic.yaml)（API Service）を参照。

## 主な環境変数

| 変数 | 値 | 備考 |
|---|---|---|
| `PERSISTENCE` | `postgres` | ステートレス・水平スケール前提 |
| `DATABASE_URL` | Secret | Cloud SQL（Private IP か Unix ソケット） |

| `KEY_PROVIDER` | `db` | DB-backed 署名鍵。全レプリカ JWKS 一致 |
| `ISSUER` | `https://id.example.com` | discovery の issuer と一致必須 |
| `OBSERVABILITY` / `OTEL_EXPORTER_OTLP_ENDPOINT` | `otel` / collector | OTLP 送出、`/metrics` は pull |

## HA / スケール

- API: `minScale=2`（最低2レプリカ）、`maxScale` は負荷に応じて。ステートレスなので水平スケール可。
- worker: リース制のため複数インスタンス可。`min-instances>=1`。
- DB: `REGIONAL`（同期スタンバイ）。揮発性状態も同一 DB に載るため 2 つ目のステートフル基盤は無い。
- 署名鍵は DB-backed で全レプリカ一致を担保（Vault を使う場合は別途）。

## コスト目安（中規模 SaaS・HA・リスト価格）

| 項目 | 構成 | 月額(USD) |
|---|---|---|
| Cloud Run | API×2 + worker pools | $180–250 |
| Cloud SQL PostgreSQL HA | 2–4 vCPU/8–16GB + 100GB SSD | $300–450 |
| LB + Cloud CDN + GCS(SPA) | | $30–60 |
| Secret Manager/KMS/ログ/egress | | $30–60 |
| **合計** | | **~$540–830（中心 ~$685）** |

ステートフル基盤は PostgreSQL 一つで、揮発性状態も同居する。2 つ目のキャッシュ基盤 (月 $150–200 規模) の固定費が発生しない。CUD（1年 20–25% / 3年 40–52%）で **~$520–700** まで低下しうる（compute/DB の compute 分に適用、storage は対象外）。
