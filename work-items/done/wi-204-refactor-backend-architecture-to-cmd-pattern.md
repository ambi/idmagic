---
status: completed
authors: [Agent]
risk: medium
created_at: 2026-07-12
depends_on: []
---

# リポジトリ構造を Screaming Architecture と Go の標準レイアウトに適合させる

## Motivation
現状の `backend/` 直下には、`authentication` や `oauth2` といった Bounded Contexts（業務ドメイン）と並んで、`bootstrap`、`relay`、`devinfra` といったシステム起動・インフラDI層が混在している。これにより、プロジェクトのルートを見た際に、このシステムが何の業務を解決するシステムなのかがひと目で分かりにくい（Screaming Architecture に反する）。
また、`bootstrap` ディレクトリの中に API サーバー (`server.go`) とワーカープロセス (`worker.go`) の起動ロジックが同居しており、関心事の分離が中途半端である。
これらを解決し、ドメインとインフラを明確に分離するため、実行可能プログラムのエントリーポイントとその DI ロジックを `backend/cmd/` 以下に集約する。

## Scope
- `backend/bootstrap`
- `backend/relay`
- `backend/devinfra`
- `backend/cmd/idmagic`
- `backend/cmd/idmagic-worker`
- `backend/cmd/idmagic-relay`
- `backend/cmd/idmagic-dev-infra`
- `backend/ARCHITECTURE.md`

## Out of Scope
- 各 Bounded Context の内部ロジックやドメインモデルの変更
- DB スキーマの変更
- 外部向けの API 仕様（エンドポイントやレスポンス形式）の変更

## Plan
- **エントリーポイントの移動**:
  各アプリケーションの起動ロジック（`server.go`, `worker.go`, `relay`, `devinfra`）を、それぞれ固有の `backend/cmd/<app_name>` ディレクトリ内に移動する。
- **共通 DI 層の隔離**:
  すべてのプロセスから参照される共通の DI とインフラ初期化のロジック（`deps.go`, `memory.go`, `postgres_valkey.go` 等）を `backend/cmd/internal/bootstrap` に移動し、非公開関数を Export（大文字化）して API として提供する。これによりドメイン層からの逆依存をコンパイラレベルで遮断し、DI層として純粋な役割を持たせる。
- **ドキュメント更新**:
  `backend/ARCHITECTURE.md` に記載されているディレクトリレイアウト規則を新構造に沿って更新する。

## Tasks
- [x] T001 [App] `idmagic` API サーバーのエントリーポイント起動ロジック (`bootstrap/server.go` など) を `backend/cmd/idmagic` に移動・統合する。
- [x] T002 [App] `idmagic-worker` の起動ロジック (`bootstrap/worker.go`) を `backend/cmd/idmagic-worker` に移動・統合する。
- [x] T003 [App] `backend/relay` を `backend/cmd/idmagic-relay` の内部パッケージに移動し、ディレクトリを整理する。
- [x] T004 [App] `backend/devinfra` を `backend/cmd/idmagic-dev-infra` の内部パッケージに移動し、ディレクトリを整理する。
- [x] T005 [App] 共通の DI および Bootstrap ロジック (`bootstrap` の残り) を `backend/cmd/internal/bootstrap` に移動し、全 import を更新する。
- [x] T006 [App] `backend/ARCHITECTURE.md` の記述を更新する。
- [x] T007 [Verify] コンパイル、単体テスト (`just verify-go`) を実行して通過させる。

## Verification
- `just verify-go` コマンドで Go のフォーマット、lint、ビルド、テストがすべて成功すること。
- （可能であれば）ローカル環境で `just dev` を立ち上げ、UI や API が正常に応答すること。

## Risk Notes
- 大規模なファイルの移動と import path の書き換えが発生するため、一時的にビルドが壊れるリスクがあるが、IDE やサブエージェントによる一括置換で軽減する。
- 起動に必要な DI (deps.go) の依存関係の不整合が起きないよう、慎重にパッケージを分割する。

## Completion
- **Completed At**: 2026-07-12
- **Summary**:
  Screaming Architecture に従い、`backend/` 直下に配置されていたインフラ層・エントリーポイント起動ロジック (`bootstrap`, `relay`, `devinfra`) をすべて `backend/cmd/` 以下へ移動し整理しました。APIサーバー用とWorker用の起動ロジック (`server.go`, `worker.go`) はそれぞれ完全に独立した `main` パッケージとして `backend/cmd/idmagic/` と `backend/cmd/idmagic-worker/` に配置しました。また、共通のDIコンテナ構築ロジック (`deps.go` 等) は `backend/cmd/internal/bootstrap` 内に集約隔離し、内部の関数群（`assemble`等）をExport化（`Assemble`等）して呼び出すインターフェースへと整備しました。これにより、ドメイン層からの逆依存をコンパイラレベルで遮断しています。また、`ARCHITECTURE.md` の関連記述も新ディレクトリ構造に合わせて更新しました。
- **Verification Results**:
  - `just verify-go` - passed
  - All tests passed.
