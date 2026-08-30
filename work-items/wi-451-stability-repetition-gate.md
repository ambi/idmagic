---
depends_on: []
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-08-30
priority: p1
change_kind: tooling
spec_impact: { kind: none, reason: "低頻度の競合や共有状態の汚染を反復実行で検出するテスト基盤であり、製品仕様は変更しない。検出した製品欠陥は個別の work item で修正する。" }
---

# 低頻度のテスト失敗を反復実行で検出し一度の失敗で落とす

## Motivation

通常のテストを一回通すだけでは、並行処理、時刻境界、共有 fixture の汚染による低頻度の失敗を安定して検出できない。通常 CI の再試行で GREEN にすると失敗を隠すため、選択した検査を複数回実行し、一度でも失敗したら不安定性として報告する専用 gate が必要である。

IdMagic は PostgreSQL と E2E の共有 fixture に所有者、寿命、清掃をすでに持つため、Keycloak 型のテスト用依存グラフは導入しない。本項目は既存 fixture を毎回の契約どおり初期化し、反復間の状態漏れと低頻度競合を検出する実行器に限定する。

## Scope

- 安定したテスト識別子、反復回数、並列度、seed、制限時間を受け取り、各反復結果を保存する `mise` タスクを作る。
- 一回でも失敗、timeout、panic、race を検出したら全体を失敗にし、後続の成功で上書きしない。
- 対象を、並行性、時刻、乱数、分散 lease、共有 PostgreSQL、ブラウザー E2E を変更したテストから選び、全テストを無差別に反復しない。
- 各反復の seed、順序、失敗回、ログ位置、fixture の世代を結果へ記録し、単独再現コマンドを提示する。
- `wi-131` が定める nightly/required matrix と整合し、初期導入は nightly と手動調査用にする。
- `wi-445` の主要ユースケース証拠とは別に扱い、反復回数で単体テストまたは E2E テストの欠落を補わない。

## Out of Scope

- 失敗した通常 CI を自動再試行して成功扱いにすること。
- 全テストの一律 50 回実行。
- 新しい汎用 DI コンテナー、fixture 依存グラフ、テストフレームワーク。
- 検出した flaky test の隔離や恒久的な許可リスト化。

## Design

反復対象はリポジトリに列挙した suite とし、任意のシェル文字列を CI 設定から実行しない。各 suite は所有者、根拠、実行する `mise` タスク、既定回数、最大時間を持つ。実行器は子タスクの終了コードと構造化結果だけを集約し、テストの意味を再解釈しない。

失敗結果は `first_failure_iteration`、`seed`、`command`、`duration`、成果物の場所を持つ。一度失敗した suite は全反復後も失敗であり、成功率が閾値以上でも GREEN にしない。実行時間を抑えるため、失敗後に診断用の最小回数だけ続けるか即時停止するかを suite ごとに固定するが、終了判定は変えない。

## Plan

1. 現在の race、PostgreSQL、ジョブ lease、時刻境界、E2E から反復する根拠がある最小 suite を選ぶ。
2. 常に一回だけ失敗する fixture で gate の RED を固定する。
3. 反復実行器、結果形式、再現コマンドを実装する。
4. 既存 fixture の初期化と清掃が反復ごとに成立することを検査する。
5. nightly と手動タスクへ接続し、実行時間と失敗率を測って required 化を別途判断する。

## Tasks

- [ ] T001 [Inventory] 反復対象 suite、失敗モード、既存 fixture の寿命、実行時間を記録する。
- [ ] T002 [Acceptance RED] 指定回だけ失敗する fixture が、後続成功にかかわらず gate を失敗させることを固定する。
- [ ] T003 [Tooling] suite、回数、seed、timeout を受け取る反復実行器と構造化結果を実装する。
- [ ] T004 [Fixtures] PostgreSQL と E2E fixture が反復ごとに必要な隔離と清掃を行うことを検査する。
- [ ] T005 [CI] `wi-131` の nightly matrix と整合する非 blocking 導入を行い、required 化の条件を記録する。
- [ ] T006 [Verify] 途中失敗、timeout、race、状態漏れ、全成功を与えて終了判定を確認する。

## Verification

- 新設する安定性反復用 `mise` タスク
- `mise run test-go-race`
- `mise run verify`
- 一回目、中間、最終回のいずれで失敗しても全体が失敗し、成功で上書きされない。
- 報告された seed と単独再現コマンドで同じ fixture の失敗を再実行できる。

## Risk Notes

リスクは medium。対象を広げすぎると遅い gate が無視され、狭すぎると同じ決定的テストを繰り返すだけになる。反復対象は具体的な低頻度失敗の仮説を持つ suite に限定し、通常 CI の再試行とは名前、結果、用途を分ける。
