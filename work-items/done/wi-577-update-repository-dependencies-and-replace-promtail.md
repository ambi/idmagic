---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-13
priority: p1
depends_on: []
change_kind: maintenance
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 開発環境と配備資産の依存関係を更新する保守変更であり、利用者向けの機能または公開契約は変わらない。
  references: []
initial_context:
  specification:
    - docs/design/observability/README.md
    - docs/design/observability/logging.md
    - docs/domain/structure.md
  typespec: []
  source:
    - mise.toml
    - go.mod
    - frontend/package.json
    - tools/package.json
    - infra/docker
    - infra/k8s/monitoring
  tests:
    - tools/check/src/mise-config.test.ts
    - tools/render-spec-docs/src/render.test.ts
  stop_before_reading: []
spec_impact: { kind: none, reason: 利用者に見える振る舞いと公開 API を変えず、依存関係とログ収集基盤の実装を更新するため。 }
---

# リポジトリ全体の依存関係を更新し Promtail を Alloy へ置き換える

## Motivation

リポジトリのランタイム、ライブラリ、開発ツール、CI Action、コンテナイメージには利用可能な安定版との差がある。

Promtail は 2026 年 3 月 2 日に EOL を迎えているため、単純なバージョン更新では保守されるログ収集基盤にならない。

## Scope

- Go、Bun、フロントエンド、組み込みツール、mise 管理ツール、CI Action を最新安定版へ更新する。
- コンテナイメージとコンテナで実行する独立 CLI を最新安定版へ更新する。
- Docker Compose と Kubernetes の Promtail を Grafana Alloy へ置き換える。
- Loki へ送るラベル、structured metadata、タイムスタンプの意味を移行前後で維持する。
- 現行設計文書を Alloy を使う構成へ更新する。
- バージョン検査は宣言の重複ではなく、更新可能性と固定方法を検査する。
- 反復利用しない一時コマンドは mise タスクへ追加しないことをエージェント指示に明記する。

## Out of Scope

- 利用者に見える機能、公開 API、TypeSpec の変更。
- 本番環境への配備または外部システムの更新。
- 依存ライブラリの未採用機能を使うための機能開発。

## Design

依存関係の版は、それぞれのパッケージマネージャーまたは `mise.toml` の宣言を正とする。

検査コードには現在の版を複製せず、明示的な安定版固定であることを検査する。

Alloy は Docker では Docker Engine API からコンテナログを読み、Kubernetes では DaemonSet として同一ノードの Pod ログを読む。

アプリケーションの JSON Lines に対する `service` と `level` のラベル化、`trace_id`、`span_id`、`request_id` の structured metadata 化、アプリケーション時刻の採用を維持する。

## Plan

1. 全更新面を棚卸しし、公式リリースとパッケージマネージャーが示す最新安定版へ更新する。
2. 互換性に影響するメジャー更新を確認し、必要なコードと設定を適応する。
3. Promtail 設定を Alloy 設定へ変換して見直し、Compose と Kubernetes の実行資産を置き換える。
4. 対象別検証、脆弱性監査、全体検証、更新漏れ検査を実行する。

## Tasks

- [x] T001 [Inventory] 更新対象を棚卸しする。
- [x] T002 [Dependencies] 言語、ライブラリ、開発ツール、CI Action を更新する。
- [x] T003 [Operations] コンテナイメージを更新し、Promtail を Alloy へ置き換える。
- [x] T004 [Docs] 現行設計文書を Alloy の構成へ更新する。
- [x] T005 [Verify] 対象別検証、監査、全体検証、更新漏れ検査を実行する。

## Verification

- `mise run check-work-items`
- `mise run check-compose`
- `mise run check-k8s -- dev`
- `mise run check-k8s -- prod`
- `mise run check-monitoring`
- `mise run check-k6`
- `mise run check-spec`
- `mise run check-boundaries`
- `mise run audit-dependencies`
- `mise run audit-go-reachability`
- `mise run verify-full`

## Risk Notes

複数のメジャー更新とログ収集エージェントの置換を同時に行うため、設定が構文上は有効でもログのラベルまたは読み取り位置が変わる可能性がある。

公式の変換器と設定検証を使い、Compose と Kubernetes のレンダリング、対象別テスト、全体検証を組み合わせて緩和する。

## Completion

- **Completed At**: 2026-09-14
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返した。
  リポジトリ全体の依存関係を最新安定版へ更新し、EOL 済みの Promtail を Grafana Alloy 1.19.2 へ置き換えた。
  ログのラベル、structured metadata、タイムスタンプの意味は維持した。
  `mise-config.test.ts` はスキャナーの現在版を複製せず、安定版へ明示固定されていることを検査するようになった。
  一時的な変換コマンドは mise タスクに残さず、反復利用しないコマンドの例外を `AGENTS.md` に明記した。
- **Acceptance RED Evidence**:
  - **Test**: `mise run audit-dependencies`
  - **Requirement**: N/A: 公開仕様を変えない依存関係の保守作業である。
  - **Observed Failure**: Mermaid 12 が推移依存として解決した `lodash-es 4.17.23` に High 1 件と Medium 1 件の修正可能な脆弱性が検出された。
  - **Detection Reason**: 直接依存の更新検査では見えないロックファイル内の脆弱な推移依存を識別できる。
- **Unit RED Evidence**:
  - **Test**: `mise run test-go-test -- ./backend/shared/storage/db_postgres TestResilientDBQueryKeepsTimeoutContextUntilRowsClose`
  - **Requirement**: N/A: 公開仕様を変えない依存関係の互換修正である。
  - **Observed Failure**: pgx 5.11.0 が `Rows.TypeMap() *pgtype.Map` を追加したため、テスト用 Rows がインターフェースを満たさずコンパイルに失敗した。
  - **Detection Reason**: 更新後の pgx インターフェースをテストダブルが完全に実装しているかをコンパイラーが検査する。
- **Change-Resistance Results**:
  Alloy 設定の `loki.write.default.receiver` を存在しない `loki.write.missing.receiver` へ変えた一時設定に対し、Alloy 1.19.2 の `validate` は receiver が存在しないとして失敗した。
  実構成では Loki 3.7.7 と Alloy 1.19.2 を専用 Compose プロジェクトで起動し、Docker discovery から Loki write までの全コンポーネントが正常に初期化された。
  E2E のセッション失効検査は総数比較だと並行するセッション作成を誤検出したため、失効対象 ID が再読込後も存在しないことを検査する形へ変更し、`verify-full` の並行実行で成功した。
- **Verification Results**:
  - `mise run verify-full` - passed
  - `mise run audit-dependencies` - passed
  - `mise run audit-go-reachability` - passed
  - `mise run check-compose` - passed
  - `mise run check-k8s -- dev` - passed
  - `mise run check-k8s -- prod` - passed
  - `mise run check-monitoring` - passed
  - `mise run check-k6` - passed
  - `mise run check-schema` - passed
  - `mise run check-spec` - passed
  - `mise run check-boundaries` - passed
  - `mise run check-container-images -- <全固定済み第三者イメージ>` - passed
  - 更新済みの全イメージを使う開発用 Compose のビルドと起動 - passed
