# Structure

## Directories

```text
.
├── backend/           # Go Bounded Contexts, shared, cmd/
├── frontend/          # React UI and gateway
├── spec/              # TypeSpec, canonical specification/design Markdown, and release baseline
│   └── contexts/<context>/
├── infra/             # container, local runtime, and database schema assets
├── load/k6/           # tenant-local OAuth SLO smoke
├── tools/             # specification, boundary, compatibility, and rendering tools
└── work-items/        # units of work, decision history, and completion records
```

依存は `spec` から実装と派生成果物へ向かって流れる。`backend` のドメイン層とユースケース層のパッケージが、アダプターやランタイムへ逆向きに依存することはない。

## Development flow mapping

| 関心事 | 置き場所 | 詳細 |
| --- | --- | --- |
| 仕様と設計 | `spec/**/*.tsp`, `spec/` の Markdown | 規範的な振る舞い、契約、現在の根拠。変更はここから始まる。 |
| 変更の記録 | `work-items/*.md` | 1 つの変更についての代替案、計画、作業、完了の記録。 |
| ドメインモデル | `backend/<context>/(<feature>/)domain` | フレームワークに依存しないドメインモデル。 |
| アプリケーションロジック | `backend/<context>/(<feature>/)usecase` | フレームワークに依存しないユースケース。 |
| ポート | `backend/<context>/(<feature>/)ports` | HTTP、永続化、通知などのポート。 |
| アダプター | `backend/<context>/(<feature>/){handlers_http,db_postgres,...}` | HTTP、永続化、通知などのアダプター。 |
| ランタイム | `backend/cmd/`, `backend/cmd/internal/bootstrap` | 起動、依存注入。 |
| インフラ基盤 | `infra` | インフラ基盤の設定コード。 |
| フロントエンド | `frontend` | フロントエンドコード。 |

## Stack

- **バックエンド**：Go。
- **フロントエンド**：React/TypeScript、Bun。
- **データベース**：PostgreSQL。
- **インフラ基盤**：Docker Compose、Kubernetes、Prometheus、Grafana、Loki、Promtail、k6。

## Context internals

Bounded Context は通常、次の 4 層で構成する。

```text
backend/<context>/
  domain/            # エンティティ、値オブジェクト、状態遷移、純粋な検証
  usecase/           # 仕様で定めた操作を行うアプリケーションロジック
  ports/             # Repository、ストア、外部サービスの抽象
  handlers_http/     # 受信 HTTP アダプター
  db_memory/         # メモリ版 Repository アダプター
  db_postgres/       # PostgreSQL 版 Repository アダプター
```

アダプターはそれを所有する Context または機能の直下に置き、snake_case の `<role>_<technology>` で命名する。

`backend/shared/` は、複数の Context が実際に共有する技術的な能力のための場所である。

具象のドメインイベントの構造体は、それを所有する Context の `domain/events.go` に置く。`backend/shared/spec/events.go` はイベントのエンベロープとなるインターフェースと、そのワイヤ表現への変換だけを持つ。

2 つ以上の独立した機能を持つ Context は、4 層の構成に機能ごとの垂直分割を追加してよい：`backend/<context>/<feature>/{domain,ports,usecase,<role>_<technology>}/`。機能が 1 つしかない Context は分割しない。

```text
backend/idmanagement/
  module.go                 # Context ごとに 1 つ置く DI の組み立て
  domain/                   # 機能間で共有する型だけ（列挙、DomainEvent）
  usecase/                  # 機能をまたぐユースケース補助とエラー値だけ
  deps_http/                # Deps 型を定義する末端パッケージ
  handlers_http/            # ルート登録と機能をまたぐ統合テスト
  user/  group/  agent/     # 各機能の domain、ports、usecase、アダプター
```

## Frontend component structure

仕様上の機能とそろえた UI の境界は `frontend/src/features/<feature>/` に置く。その機能のビュー、ローカルコンポーネント、ヘルパー、テスト、ローカライズ辞書（`*.i18n.ts`）は必ずそのディレクトリに置く。特定の機能境界にひも付かない、横断的で再利用可能なコンポーネントは `frontend/src/components/` に置く。

## HTTP routing

HTTP ルーティングは `backend/shared/http/server_http/routes.go` で組み立てる。ここがテナント単位のルートを既定のテナントと `/realms/:tenant_id` の両方に登録し、制御面のテナント管理だけを `/realms/default/admin/tenants` に分離する。

各 Context のルーティングは `backend/<context>/handlers_http/routes.go` にある。正確なエンドポイントの一覧はそのファイルを参照する。新しい HTTP API は、それを所有する Context の `routes.go` に、同じ `handlers_http` 配下のハンドラーとともに登録する。Context 固有の Repository とルーティングの接続は `backend/<context>/module.go` に集約し、中央のルーターは Module を呼ぶだけにする。

## Architecture style

単一の Go モジュール内で Bounded Context の境界を保ちつつ、複数の実行単位が実装を共有する現在のアーキテクチャを **Modular Monolith** とする。Context 間は公開された言語とポートで接続する。

通常は複数の Context を 1 つの API プロセスに組み合わせ、リソースやレイテンシーの特性が異なるジョブと横断的なバッチ処理だけを別の実行単位にする。独立したデータ所有権、担当チーム、SLO が必要になるまではサービスを分割しない。この記述は現在の設計を示すものであり、将来も同じ構成を義務付けるものではない。

`backend/cmd/internal/bootstrap/deps.go` の `Dependencies` は HTTP 層へ渡す依存を集約し、メモリ、PostgreSQL、コンソール、OpenTelemetry など実行時の実装選択を吸収する。Context 固有の Repository は各 `Module` にまとめ、中央の `Dependencies` とサーバーの `Deps` はその Module を受け取る。ポートを追加した場合は、その Context の `ports/`、メモリと PostgreSQL の各アダプター、スキーマ変更の要否、`bootstrap.Dependencies`、`assembleMemory`、`assemblePostgres`、`support.Deps`、関連する HTTP ハンドラーまたはユースケースの構築処理を確認する。
