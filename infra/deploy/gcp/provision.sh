#!/usr/bin/env bash
# IdMagic — GCP 構成の払い出し（最小例 / シンプル雛形）
#
# 目的: Cloud SQL(Postgres HA) / Memorystore(Valkey) / Pub/Sub / Secret /
#       Artifact Registry を作成し、3サービス（API=Cloud Run Service,
#       worker・relay=Cloud Run worker pools）をデプロイする流れを示す。
#       イベント配信は Pub/Sub（サーバレス）。relay は -tags pubsub ビルド（ADR-120）。
#
# 注意:
#  - これは雛形。値はプレースホルダなので実行前に環境へ合わせて置換すること。
#  - 一部は beta/preview 機能（worker pools）。gcloud components を更新のこと。
#  - スキーマ適用(psqldef)は「起動時ではなくデプロイ工程」で行う（ADR-071）。
set -euo pipefail

PROJECT="your-project"
REGION="asia-northeast1"
REPO="idmagic"                 # Artifact Registry リポジトリ
IMAGE="${REGION}-docker.pkg.dev/${PROJECT}/${REPO}/idmagic:latest"
DB_INSTANCE="idmagic-pg"
DB_NAME="idmagic"
DB_USER="idmagic"
RELAY_SA="idmagic-relay"       # Pub/Sub publish 用サービスアカウント名

# ---------------------------------------------------------------------------
# 0) コンテナイメージ（3バイナリ入り distroless / 既存 Dockerfile を再利用）
#    RELAY_TAGS=pubsub で idmagic-relay に Pub/Sub アダプタを組み込む（既定は無効, ADR-120）。
# ---------------------------------------------------------------------------
gcloud artifacts repositories create "$REPO" --repository-format=docker --location="$REGION" || true
gcloud builds submit --tag "$IMAGE" --project "$PROJECT" \
  --config /dev/stdin <<'YAML' .
steps:
  - name: gcr.io/cloud-builders/docker
    args: ["build","-f","infra/docker/Dockerfile","--build-arg","RELAY_TAGS=pubsub","-t","${_IMAGE}","."]
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
# 2) Valkey（Memorystore, Standard=HA）
# ---------------------------------------------------------------------------
gcloud memorystore instances create idmagic-valkey \
  --project "$PROJECT" --location "$REGION" \
  --node-type=shared-core-nano --replica-count=1 || \
gcloud redis instances create idmagic-valkey \
  --region "$REGION" --tier=STANDARD_HA --size=5 --redis-version=redis_7_0

# ---------------------------------------------------------------------------
# 3) Pub/Sub（イベント配信, ADR-120）。outbox.topic 列に対応する topic を作成。
#    一覧は backend/oauth2/db_postgres/outbox.go の eventTopics マップ。
#    per-aggregate ordering (partitionKey) を使うため message-ordering を有効化。
# ---------------------------------------------------------------------------
TOPICS=(
  oauth2.client.lifecycle.v1
  oauth2.authentication.v1
  oauth2.security-incident.v1
  oauth2.consent.v1
  oauth2.authorization-code.v1
  oauth2.token.v1
  oauth2.key-management.v1
  oauth2.par.v1
  oauth2.device-authorization.v1
  oauth2.administration.v1
  tenancy.lifecycle.v1
  iam.groups.v1
  iam.agents.v1
)
for topic in "${TOPICS[@]}"; do
  gcloud pubsub topics create "$topic" --project "$PROJECT" \
    --message-storage-policy-allowed-regions="$REGION" || true
done

gcloud iam service-accounts create "$RELAY_SA" --project "$PROJECT" || true
gcloud projects add-iam-policy-binding "$PROJECT" \
  --member="serviceAccount:${RELAY_SA}@${PROJECT}.iam.gserviceaccount.com" \
  --role="roles/pubsub.publisher"

# ---------------------------------------------------------------------------
# 4) Secret（接続文字列は Secret Manager に格納し、サービスへ注入）
# ---------------------------------------------------------------------------
printf 'postgres://%s:REPLACE_ME@/%s?host=/cloudsql/%s:%s:%s' \
  "$DB_USER" "$DB_NAME" "$PROJECT" "$REGION" "$DB_INSTANCE" \
  | gcloud secrets create idmagic-database-url --data-file=- || \
  gcloud secrets versions add idmagic-database-url --data-file=-
printf 'valkey://REPLACE_HOST:6379/0' | gcloud secrets create idmagic-valkey-url --data-file=- || true

# ---------------------------------------------------------------------------
# 5) スキーマ適用（psqldef / デプロイ工程・起動時禁止 ADR-071 / --enable-drop 禁止）
#    CI から DATABASE_URL を PG* にマップして実行するのが基本。ここは手動例。
#    docker run --rm -v "$PWD/infra/schema:/schema:ro" \
#      -e PGHOST -e PGPORT -e PGUSER -e PGPASSWORD sqldef/psqldef:3.11 \
#      "$DB_NAME" --apply --file /schema/postgres.sql
# ---------------------------------------------------------------------------

# ---------------------------------------------------------------------------
# 6) デプロイ: API=Service, worker/relay=worker pools（HTTP を持たないため）
# ---------------------------------------------------------------------------
gcloud run services replace infra/deploy/gcp/cloudrun-idmagic.yaml --region "$REGION"

gcloud beta run worker-pools deploy idmagic-worker \
  --image "$IMAGE" --region "$REGION" --command /app/idmagic-worker \
  --min-instances=1 --max-instances=3 \
  --set-env-vars=PERSISTENCE=postgres_valkey,EVENT_SINK=outbox,OBSERVABILITY=otel \
  --set-secrets=DATABASE_URL=idmagic-database-url:latest,VALKEY_URL=idmagic-valkey-url:latest

gcloud beta run worker-pools deploy idmagic-relay \
  --image "$IMAGE" --region "$REGION" --command /app/idmagic-relay \
  --service-account="${RELAY_SA}@${PROJECT}.iam.gserviceaccount.com" \
  --min-instances=1 --max-instances=2 \
  --set-env-vars=RELAY_SINK=pubsub,PUBSUB_PROJECT="$PROJECT" \
  --set-secrets=DATABASE_URL=idmagic-database-url:latest

echo "done. 前段に Cloud Load Balancing + Cloud CDN + Cloud Armor、SPA は GCS+CDN を配置する。"
