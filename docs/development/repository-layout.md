# リポジトリ構成

この文書は、リポジトリの最上位のディレクトリと、開発工程の関心事ごとの置き場所を示す。
`docs/` の配置は[文書ガイド](../formats/documentation-guide.md)、バックエンドのモジュール内部の層と依存規則は[バックエンド設計](../design/application/backend.md)、フロントエンドのコードの置き場所は[フロントエンド設計](../design/application/frontend.md)が定める。

## ディレクトリ

```text
.
├── backend/           # モジュール、共有ライブラリ、cmd/
├── frontend/          # React UI とゲートウェイ
├── docs/              # 現在有効な要件、設計、手順、記述規約
├── spec/              # TypeSpec、派生表示、OpenAPI のリリース基準
│   └── contexts/<context>/
├── infra/             # コンテナ、ローカル実行環境、データベーススキーマ
├── load/k6/           # テナント単位の OAuth SLO スモーク検査
├── mise.toml          # ツールのバージョンとリポジトリタスク
├── tools/             # 仕様、境界、互換性、描画、開発サイクルの道具
└── work-items/        # 一つの変更の計画、判断、完了の記録
```

依存は `spec` から実装と派生成果物へ向かって流れる。`backend` のドメイン層とユースケース層のパッケージが、アダプターやランタイムへ逆向きに依存することはない。

## 開発工程との対応

| 関心事 | 配置 | 内容 |
| --- | --- | --- |
| 要件 | `docs/requirements/**` | 目的、機能要件、品質要件、外部規範、システム横断シナリオ、用語集 |
| 全体設計と運用 | `docs/{design,operations}/**` | 要件を実現する構造と仕組み、受け入れ、運用 |
| モジュール設計 | `spec/contexts/**/*.tsp`、`docs/modules/**` | モデルと API は TypeSpec、要件と状態遷移は機能仕様、受け入れの例は任意の付録、判断と仕組みは設計領域ごとの設計に書く |
| 文書と記録の形式 | `docs/formats/*.md` | 内容の担当と配置、仕様と設計の記述規則、変更の計画と完了記録 |
| 開発の進め方と手順 | `docs/development/*.md` | 仕様先行のワークフロー、環境、生成、CI、テスト、リリース |
| リリース固有の利用者向け差分 | `docs/releases/{changes,upgrades}/wi-*.md` | 注目すべき変更の告知と、既存利用者が必要とする移行情報。現在状態は一次情報文書に書く |
| 手動の運用手順 | `docs/runbooks/*.md` | 障害時または手動作業の最中に読む手順 |
| 変更の記録 | `work-items/*.md` | 1 つの変更についての代替案、計画、作業、完了の記録 |
| ドメインモデル | `backend/<module>/(<feature>/)domain` | フレームワークに依存しないドメインモデル |
| アプリケーションロジック | `backend/<module>/(<feature>/)usecases` | フレームワークに依存しないユースケース |
| ポート | `backend/<module>/(<feature>/)ports` | HTTP、永続化、通知などのポート |
| アダプター | `backend/<module>/(<feature>/){handlers_http,db_postgres,...}` | HTTP、永続化、通知などのアダプター |
| ランタイム | `backend/cmd/`、`backend/cmd/internal/bootstrap` | 起動、依存注入 |
| インフラ基盤 | `infra` | インフラ基盤の設定コード |
| フロントエンド | `frontend` | フロントエンドコード |
