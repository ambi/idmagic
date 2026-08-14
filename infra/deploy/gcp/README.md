# GCP への IdMagic の配備

コンピューティングリソース、データベース、イベントリレーを **単一 VPC・単一リージョン** に配置する。レイテンシーと egress コストを抑え、CUD（確約利用割引）で費用を圧縮し、運用を単一ベンダー内で完結させる。中規模 SaaS、本番の高可用性、イベントリレーを前提とする。



## アーキテクチャ

- フロントエンド: React + Vite の**純粋な SPA**（`frontend/` → `dist/`）。SSR は使わない。
- ゲートウェイ: **Caddy**（`frontend/Caddyfile`）が静的アセットを配信し、同一オリジンで API / OIDC のパスをバックエンドへプロキシする。CSP は SPA の HTML だけに付与する。
- バックエンド: 常駐する 2 個の Go プロセス、`idmagic`（API `:8080`）と `idmagic-worker`。CGO を使わない distroless イメージ（`infra/docker/Dockerfile`）で実行し、`PERSISTENCE=postgres` によりステートレスに水平スケーリングできる。
- データ: **PostgreSQL 17** が業務データ、BLOB、セッション、OAuth の一時状態を保持する。揮発性の状態も同じ PostgreSQL に置くため、2 個目のステートフル基盤はない。
- 署名鍵: 全レプリカの JWKS を一致させるため、**データベース保存の永続鍵**を推奨する。Vault Transit も利用できる。
- スキーマ: `psqldef` を**配備工程で適用**する。起動時の適用と `--enable-drop` の使用は禁止する。

## トポロジー

```
利用者
  ▼
Cloud Load Balancing（HTTPS）+ Cloud CDN + Cloud Armor（WAF）
  │
  ├─ 静的 SPA … GCS バケット + Cloud CDN
  │
  └─ /api・/authorize・/token・/.well-known など → Cloud Run（idmagic API、minScale=2、高可用性）
                                                     │
        ┌─────────────────────────────────────┼─────────────────────────────┐
        ▼                                                                   ▼
  Cloud SQL for PostgreSQL                                            Secret Manager
   （REGIONAL = 高可用性、揮発性状態も同居）                              / Cloud KMS

バックグラウンド処理（HTTP を持たない常駐プロセス）：
  Cloud Run worker pools ─ idmagic-worker（ジョブ実行と保持期間スイープ）
```

## サービスの対応

| プロセス | 実行基盤 | 選定理由 |
|---|---|---|
| `idmagic`（API） | **Cloud Run Service** | HTTP（`:8080`）を提供する。`minScale=2` で高可用性と自動スケーリングを実現する。 |
| `idmagic-worker` | **Cloud Run worker pools** | HTTP を持たない常駐 `worker` プロセスであり、`$PORT` のリッスンが不要な worker pools が適する。 |
| `idmagic-seed` | Cloud Run Job（任意・一過性） | 初期 seed |

> Cloud Run の通常の Service は `$PORT` への HTTP レスポンスが必須なので、HTTP を持たない `worker` プロセスとイベントリレーには **worker pools** を使う。

## 配備順序

1. 既存の `infra/docker/Dockerfile` から、2 個のバイナリを含む distroless イメージをビルドし、Artifact Registry へ登録する。
2. Cloud SQL（REGIONAL）を用意する。
3. `DATABASE_URL` などのシークレットを Secret Manager に登録する。
4. `psqldef --apply` で**スキーマを適用する**。起動時ではなくこの工程で実行し、`--enable-drop` は使わない。
5. `idmagic`（Service）、`idmagic-worker`（worker pools）の順に配備する。
6. Cloud Load Balancing、Cloud CDN、Cloud Armor、DNS、TLS を配備する。

ひな型は [`provision.sh`](./provision.sh)（準備と配備）と [`cloudrun-idmagic.yaml`](./cloudrun-idmagic.yaml)（API Service）を参照する。

## 環境変数

| 変数 | 設定値または参照 | 説明 |
|---|---|---|
| `PERSISTENCE` | `postgres` | ステートレス・水平スケール前提 |
| `DATABASE_URL` | Secret Manager の `idmagic-database-url` シークレットの `latest` 版 | Cloud Run が環境変数へ注入する Cloud SQL 接続文字列（プライベート IP または Unix ソケット） |
| `KEY_PROVIDER` | `db` | データベース保存の署名鍵。全レプリカで JWKS が一致 |
| `ISSUER` | `https://id.example.com` | Discovery Metadata の `issuer` と一致必須 |
| `OBSERVABILITY` / `OTEL_EXPORTER_OTLP_ENDPOINT` | `otel` / コレクター | OTLP 送信、`/metrics` はプル方式 |

## 高可用性とスケーリング

- API: `minScale=2`（最低 2 レプリカ）とし、`maxScale` は負荷に応じて決める。ステートレスなので水平スケーリングできる。
- `worker` プロセス: リース方式なので複数インスタンスを実行できる。`min-instances>=1` とする。
- データベース: `REGIONAL`（同期スタンバイ）とする。揮発性の状態も同じデータベースに置くため、2 個目のステートフル基盤はない。
- 署名鍵はデータベースに保存し、全レプリカで一致させる。Vault を使う場合も共通の鍵を参照させる。

## 費用の目安

| 項目 | 構成 | 月額（USD） |
|---|---|---|
| Cloud Run | API×2 + worker pools | $180–250 |
| Cloud SQL PostgreSQL HA | 2–4 vCPU/8–16GB + 100GB SSD | $300–450 |
| LB + Cloud CDN + GCS(SPA) | | $30–60 |
| Secret Manager/KMS/ログ/egress | | $30–60 |
| **合計** | | **~$540–830（中心 ~$685）** |

ステートフル基盤は PostgreSQL 1 個であり、揮発性の状態も同居する。2 個目のキャッシュ基盤（月 $150–200 規模）の固定費は発生しない。CUD（1 年 20–25% / 3 年 40–52%）により **~$520–700** まで低下しうる。割引はコンピューティングとデータベースの演算リソースに適用され、ストレージは対象外である。
