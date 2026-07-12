---
status: accepted
authors: [tn]
created_at: 2026-07-12
---

# ADR-102: Runtime と database infrastructure 資材を infra に集約する

## コンテキスト

従来の `deploy/` には Dockerfile、Docker Compose、OTel Collector、PostgreSQL の宣言的 schema が
置かれている。`deploy` は成果物を本番へ届ける経路を連想させるが、これらはローカル起動・
コード生成・テスト・リリース前の schema 適用に横断して使う Runtime & Infrastructure 資材である。

特に PostgreSQL schema は sqlc、embedded PostgreSQL テスト、Compose の schema service、リリース
運用が同じ正本を参照する。特定の Go adapter 実装の内部でも、単なる deploy 補助でもない。
旧 ADR-068 の top-level `infra/` から `deploy/` への改名は、当時の
`internal/infrastructure/` との語義衝突を背景としていた。しかしその Go 実装構成は ADR-092 により
`backend/` へ移行しており、衝突の前提は現存しない。

## 決定

1. top-level の `deploy/` を `infra/` に改名する。
2. Dockerfile、Docker Compose、OTel Collector は `infra/docker/` に置く。
3. PostgreSQL の宣言的な現在形 schema とその運用手順は `infra/schema/` に置き、
   `infra/schema/postgres.sql` を唯一の正本とする。
4. sqlc、テスト、ローカル runtime、リリース前の schema 適用は、この正本を直接参照する。
   旧 `deploy/` パスのコピー、symlink、転送ファイルは残さない。
5. 既存の `deploy/` を参照する完了済み work item は、その時点の監査記録として変更しない。

SCL の規範要素、外部 API、PostgreSQL schema の内容、psqldef による適用方式は変更しない。

## 却下した代替案

- `deploy/` を維持し schema だけ別の root へ移す: Runtime 資材が二つのトップレベルへ分散し、
  Docker と database infrastructure の層境界が不明瞭になる。
- schema を `backend/shared/adapters/persistence/postgres/` に置く: schema は Go adapter の内部資材
  ではなく、sqlc・テスト・runtime・リリース運用が共有する正本である。
- `infra/postgres/schema/` に置く: 現在の資材は PostgreSQL schema のみであり、用途を先に示す
  `infra/schema/postgres.sql` の方が探索時に直接的である。
- 旧パスを symlink で残す: 正本の探索を曖昧にし、将来の内容乖離を検出しにくくする。

## 影響

- `ARCHITECTURE.md` と README の Runtime & Infrastructure の地図を `infra/` に同期する。
- ADR-071 の PostgreSQL schema 正本への参照を `infra/schema/postgres.sql` に更新する。
- sqlc、Go テスト、開発 CLI、Docker/Compose、CI、運用文書のパスを更新する。
- SCL 要素は変更しない。
