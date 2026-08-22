---
depends_on: []
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-05
priority: p2
change_kind: tooling
spec_impact: { kind: none, reason: "CI のカバレッジ計測と閾値強制、および開発ルールの明文化であり、製品の振る舞いと配線契約を変えない。" }
---

# テスト徹底のためのガバナンス構築と CI カバレッジ強制

## Motivation
起票時の前提のうち、フロントエンド側は解消している。[[wi-130-frontend-testing-environment-and-coverage]] で Vitest の環境が入り、
`mise run test-ui-unit` / `mise run test-ui-cover` が存在し、`frontend/src` には 100 件を超えるテストファイルがある。
`mise run verify` も `test-ui-unit` を依存に含む。

残っているのは**強制**である。カバレッジは計測できるが、下回っても CI は落ちない。
閾値が無い計測は、読まれないレポートを生むだけで網羅度を維持しない。

Go 側の実測 (2026-08-22、`mise run test-go-cover`) は 248 パッケージ中 **58 が 0%**、
パッケージ単純平均で約 52% である。強制が無いまま機能を足し続ければこの分布はさらに薄まる。
本 work item は、閾値と差分カバレッジを fail-closed の門にして、この状態を構造的に塞ぐ。

## Scope
- **CIへのカバレッジ計測ステップの統合**:
  - GitHub Actions ワークフロー (`.github/workflows/idmagic-ci.yaml`) において、Goバックエンドテストの際に `-coverprofile` を生成し、カバレッジを測定する。
  - フロントエンドの CI 実行フローに `mise run test-ui-cover` を追加し、カバレッジを測定する。
- **カバレッジしきい値（閾値）の強制**:
  - バックエンド全体およびフロントエンドの単体テストカバレッジについて、最低維持すべき閾値（例: 全体で 70%）を設定し、これを下回った場合に CI を失敗させる。
  - あるいは、PR において新規追加された行に対するカバレッジ（差分カバレッジ）をチェックし、新規機能に対するテストの追加を強制する。
- **開発ルールの明文化**:
  - `DEVELOPMENT.md` の Verification ladder に、カバレッジ閾値を段の 1 つとして書く。
    `AGENTS.md` は 1 行の表項目に留める (この repo は AGENTS.md を意図的に薄く保つ)。
- **ローカル検証コマンドの整備**:
  - `mise run verify` コマンドで、ローカルでもテストカバレッジの最低目標が満たされているか自動で簡易検証できるようにする。
  - 開発者がローカルでカバレッジを視覚的に確認しやすいよう、HTML形式でのカバレッジ出力レシピ (`mise run cover-go-html`) などを `mise.toml` に追加する。

## Out of Scope
- 個別のバックエンド/フロントエンドのテストコードの実装自体（これは `wi-129` や `wi-130` で段階的に実装する）。
- 静的コード解析（linter）のルール自体の厳格化。

## Plan
- 現在`mise run test-go-cover`とfrontend unit coverage、`.github/workflows/idmagic-ci.yaml`は存在するが、report生成と閾値gate/成果物管理が分離している。まずbaselineをpackage/layer別に計測し、generated code・command bootstrap等の除外根拠を固定する。
- repository全体のline coverage単一値ではなく、Go context domain/usecases/adapters、frontend presentation/hooks/api、変更差分coverage、critical packageの最低値を組み合わせる。高い既存coverageを低い全体thresholdで相殺しない。
- 初回gateは現在baseline以下に退行しない値から導入し、段階的target/dateをpolicyに置く。例外はpackage/path、根拠、owner、expiryを持ち、CI設定の手編集で恒久除外しない。
- unit coverageだけでなく`mise run check`、race、PostgreSQL contract、UI E2E/conformance等のrequired check matrixを変更種類に対応付ける。ただし重いnightly checkを全PRの無条件blockにはしない。
- ローカルと CI は同じ mise タスクと固定ツールバージョンを使い、カバレッジ成果物にはソースコミット、ツールバージョン、コマンドを記録する。生成された `frontend/coverage` や `.gocache` を作業ツリーに残さない。

## Tasks
- [ ] T001 [Baseline] clean worktreeでGo/UI coverageをpackage/path別に取得し、generated/fixture/bootstrap除外候補と現行CI check時間/flakinessを記録する。
- [ ] T002 [Policy] layer/critical package/diff thresholds、段階target、test pyramid、exception owner/expiry、required/nightly check matrixを文書化する。
- [ ] T003 [Go Tooling] coverprofile を正規化・統合し、パッケージおよび層の閾値と除外を検証する mise タスクを追加する。
- [ ] T004 [UI Tooling] Vitest coverage JSONからpath group/diff thresholdを検証し、coverage出力をtmp/artifactへ隔離するrecipeを追加する。
- [ ] T005 [Exceptions] machine-readable exception file/schemaとexpiry/unknown path validatorを実装し、減少理由をCI summaryへ出す。
- [ ] T006 [CI] PR required checks、nightly race/integration/E2E、coverage summary/artifact、base branch比較をworkflowへ追加する。
- [ ] T007 [Guardrail Tests] threshold未満、期限切れ例外、generated-only変更、new package 0%、base report欠落をfixtureで意図的に失敗させる。
- [ ] T008 [Verify/Rollout] current baselineでgreen、意図的退行でred、local/CI数値一致を確認し、段階引上げ日程と運用手順を記載する。

## Verification
- `mise run verify` がローカルで正常に実行され、カバレッジが検証されること。
- `AGENTS.md` や `GEMINI.md` にテストポリシーに関する記載が正しく追加されていること。
- カバレッジ不足のコードを意図的に作成した際、CI が適切に失敗すること。

## Risk Notes
カバレッジの強制により、意味のないアサーションを伴うダミーテストが記述される可能性がある。レビューポリシーにおいて、アサーションの質を確認することを推奨する。
CI上でのテスト実行時間の増加に配慮し、可能な限り並列実行やキャッシュ（GOCACHE / Bun cache）を活用する。
