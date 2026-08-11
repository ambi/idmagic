#!/usr/bin/env bash
# IdMagic — GCP 構成の払い出し（最小例 / シンプル雛形）
#
# 目的: Cloud SQL(Postgres HA) / Pub/Sub / Secret /
#       Artifact Registry を作成し、2サービス（API=Cloud Run Service,
#       worker=Cloud Run worker pools）をデプロイする流れを示す。
#
# 注意:
#  - これは雛形。値はプレースホルダなので実行前に環境へ合わせて置換すること。
#  - 一部は beta/preview 機能（worker pools）。gcloud components を更新のこと。
# - スキーマ適用(psqldef)は「起動時ではなくデプロイ工程」で行う。
set -euo pipefail

PROJECT="your-project"
REGION="asia-northeast1"
REPO="idmagic"                 # Artifact Registry リポジトリ
IMAGE="${REGION}-docker.pkg.dev/${PROJECT}/${REPO}/idmagic:latest"
DB_INSTANCE="idmagic-pg"
DB_NAME="idmagic"
DB_USER="idmagic"
# 0) コンテナイメージ（3バイナリ入り distroless / 既存 Dockerfile を再利用）
# ---------------------------------------------------------------------------
gcloud artifacts repositories create "$REPO" --repository-format=docker --location="$REGION" || true
gcloud builds submit --tag "$IMAGE" --project "$PROJECT" \
  --config /dev/stdin <<'YAML' .
steps:
  - name: gcr.io/cloud-builders/docker
    args: ["build","-f","infra/docker/Dockerfile","-t","${_IMAGE}","."]
images: ["${_IMAGE}"]
YAML

# ---------------------------------------------------------------------------
# 1) PostgreSQL（HA = REGIONAL）
# ---------------------------------------------------------------------------
gcloud sql instances create "$DB_INSTANCE" \
  --project "$PROJECT" --region "$REGION" \
  --database-version=POSTGRES_17 \
  --tier=db-custom-2-8192 \
  --availability-type=REGIONAL \
  --storage-size=100 --storage-type=SSD --storage-auto-increase
gcloud sql databases create "$DB_NAME" --instance "$DB_INSTANCE"
gcloud sql users create "$DB_USER" --instance "$DB_INSTANCE" --password "REPLACE_ME"


# ---------------------------------------------------------------------------
# 3) Secret（接続文字列は Secret Manager に格納し、サービスへ注入）
# ---------------------------------------------------------------------------
printf 'postgres://%s:REPLACE_ME@/%s?host=/cloudsql/%s:%s:%s' \
  "$DB_USER" "$DB_NAME" "$PROJECT" "$REGION" "$DB_INSTANCE" \
  | gcloud secrets create idmagic-database-url --data-file=- || \
  gcloud secrets versions add idmagic-database-url --data-file=-

# ---------------------------------------------------------------------------
# 4) スキーマ適用（psqldef / デプロイ工程・起動時禁止 --enable-drop 禁止）
#    CI から DATABASE_URL を PG* にマップして実行するのが基本。ここは手動例。
#    docker run --rm -v "$PWD/infra/schema:/schema:ro" \
#      -e PGHOST -e PGPORT -e PGUSER -e PGPASSWORD sqldef/psqldef:3.11 \
#      "$DB_NAME" --apply --file /schema/postgres.sql
# ---------------------------------------------------------------------------

# ---------------------------------------------------------------------------
# 5) デプロイ: API=Service, worker=worker pools（HTTP を持たないため）
# ---------------------------------------------------------------------------
gcloud run services replace infra/deploy/gcp/cloudrun-idmagic.yaml --region "$REGION"

gcloud beta run worker-pools deploy idmagic-worker \
  --image "$IMAGE" --region "$REGION" --command /app/idmagic-worker \
  --min-instances=1 --max-instances=3 \
  --set-env-vars=PERSISTENCE=postgres,OBSERVABILITY=otel \
  --set-secrets=DATABASE_URL=idmagic-database-url:latest



echo "done. 前段に Cloud Load Balancing + Cloud CDN + Cloud Armor、SPA は GCS+CDN を配置する。"
