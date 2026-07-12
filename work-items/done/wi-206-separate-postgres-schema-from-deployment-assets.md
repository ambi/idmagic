---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-12
depends_on: []
---

# deploy を infra に改名し PostgreSQL schema の infrastructure 所有を明確にする

## Motivation

`deploy/` には Dockerfile、Compose、OTel Collector、PostgreSQL schema が混在している。
このうち `schema/postgres.sql` は PostgreSQL database infrastructure の宣言的な現在形であり、
sqlc の入力、ローカル Compose の schema service、リリース前の明示的な schema 適用が同じ
ファイルを正本として参照する。これは Go の adapter 実装に閉じた資材でも、単なる deploy 補助
ファイルでもない。

Docker/Compose と PostgreSQL schema はいずれも Runtime & Infrastructure 層の資材であるため、
トップレベルの `deploy/` を `infra/` に改名する。schema は目的を先に表す
`infra/schema/postgres.sql` に置く。旧パスを残すと正本やディレクトリ境界が曖昧になるため、
すべての参照を同一変更で更新し、`deploy/` は削除する。

## Scope

- **architecture / decision**:
  - `deploy/` を `infra/` に改名し、Runtime & Infrastructure 層の資材を一つのトップレベル
    境界に集約する判断を ADR に記録し、`ARCHITECTURE.md` に反映する。
  - `infra/schema/postgres.sql` を、sqlc・開発環境・リリース運用が横断して利用する PostgreSQL
    database infrastructure の正本とするよう ADR-071 を更新する。
- **database infrastructure**:
  - `deploy/schema/postgres.sql` と同ディレクトリの運用 README を `infra/schema/` へ移す。
  - `pgtest`、各 PostgreSQL adapter のテスト、開発用 infrastructure CLI、sqlc 設定などが新しい
    schema 正本を参照するよう更新する。
- **runtime infrastructure / documentation**:
  - `deploy/docker/` を `infra/docker/` へ移し、Dockerfile、Compose、`justfile`、README、CI と
    ADR・work item 内の現行パス参照を更新する。
  - `deploy/` を削除し、互換コピー・symlink・転送 README を残さない。

## Out of Scope

- psqldef による宣言的 schema 管理方式、デプロイ前 apply の責務、または SQL の内容を変更すること。
- schema を bounded context ごとの複数ファイルへ分割すること、または Go package の配下に置くこと。
- PostgreSQL 以外の persistence adapter や本番デプロイパイプラインの新設。

## Plan

- Docker/Compose と schema はともに Runtime & Infrastructure 層の資材であるため、`deploy/` を
  `infra/` に一括改名する。`infra/schema/postgres.sql` を、コード生成・開発環境・リリース運用が
  共有する database infrastructure の正本にする。
- Compose の schema service は `infra/schema/` の正本をコンテナへ渡して適用する。
- パス移動は内部参照・運用文書・派生設定を同一変更で更新する。旧パスを残す案は、正本の曖昧化と
  将来の内容乖離を招くため採用しない。
- アプリケーションの外部契約や SCL の意味は変わらないため、`spec/scl.yaml` は変更対象に含めない。

## Tasks

- [x] T001 [Decision] `infra/` への改名と database infrastructure の所有を ADR に記録し、ADR-071 と
  `ARCHITECTURE.md` を更新する。
- [x] T002 [Infrastructure] Docker/Compose を `infra/docker/`、PostgreSQL schema と運用 README を
  `infra/schema/` へ移し、`deploy/` を削除する。
- [x] T003 [References] Go 実装・テスト、sqlc、Docker/Compose、justfile、README、CI の参照パスを更新する。
- [x] T004 [Verify] `deploy/` への現行参照が残らないことを確認し、リポジトリ検証を実行する。

## Verification

- `just yaml-check-work-items`
- `just check-ids`
- `just sqlc-generate`
- `just verify-go`
- `just yaml-check-architecture`
- `just check-compose`
- `test ! -e deploy`

## Risk Notes

パス変更漏れにより、Docker build、開発用 schema 適用、embedded PostgreSQL テスト、sqlc 生成の
いずれかが古い場所を参照して失敗するおそれがある。旧パスを残さず、静的検索、生成、Go 検証、
Compose 起動確認を組み合わせて検出する。SQL 内容と適用方式を変えないため、データ schema 自体への
リスクは増やさない。

## Completion

- **Completed At**: 2026-07-12
- **Summary**:
  - `deploy/` を削除し、Docker/Compose/OTel 資材を `infra/docker/`、PostgreSQL の宣言的 schema と
    運用手順を `infra/schema/` へ移した。
  - sqlc、embedded PostgreSQL テスト、開発用 infrastructure CLI、Docker/Compose、CI、README、
    現行 work item の参照を新しい正本へ更新した。
  - ADR-102 に Runtime と database infrastructure 資材の配置判断を記録し、ADR-071 と
    `ARCHITECTURE.md` を現状へ同期した。
- **Verification Results**:
  - `just sqlc-generate` — passed
  - `just check-compose` — passed
  - `just verify` — passed
  - `just yaml-check` — passed
  - `just check-ids` — passed
  - `test ! -e deploy` — passed
- **Affected Guarantees State**:
  - guarantee: Runtime と database infrastructure の資材が単一の `infra/` 境界から探索できる。
  - state: passed
  - guarantee: PostgreSQL schema の正本を sqlc、テスト、runtime、運用手順が同じパスで参照する。
  - state: passed
  - guarantee: schema の内容と psqldef 適用方式はディレクトリ改名によって変化しない。
  - state: passed
- **Evidence**:
  - procedure: ローカル作業ツリーで移動後の schema を sqlc 生成、embedded PostgreSQL を含む
    全体検証、Compose 構成検証に通した。
  - commands: `just sqlc-generate`, `just check-compose`, `just verify`, `just yaml-check`,
    `just check-ids`, `test ! -e deploy`
  - source: pre-commit working tree
  - result: passed
