---
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-16
depends_on: [wi-153-identity-lifecycle-workflows, wi-219-lifecycle-workflow-admin-api, wi-220-lifecycle-workflow-admin-ui-and-operations]
---

# lifecycle workflow のテンプレートと既存ユーザーへの手動適用を追加する

## Motivation
Entra ID Lifecycle Workflows は事前定義された onboarding/offboarding テンプレート (pre-hire
employee onboarding、employee offboarding 等) を提供し、管理者がゼロから組み立てずに主要シナリオを
開始できる。また "on-demand" 実行により、workflow 有効化前から存在する既存ユーザーに対しても管理者が
明示的に対象を選んで適用できる。

IdMagic の wi-153 は意図的に「workflow enable 時の全 User への遡及適用」を Out of Scope にしており
(dry-run のみ可能)、既に workflow 導入前から在籍する退職者・異動者に workflow を遡って適用する手段が
ない。テンプレートもゼロからの手組み立てのみで、典型的な joiner/mover/leaver パターンを毎回
trigger/filter/action から構築する必要があり、導入コストが高い。

## Scope
- `spec/contexts/identity-management.yaml` に `LifecycleWorkflowTemplate` (読み取り専用のカタログ、
  trigger/action の雛形) を追加し、新規 workflow 作成時にテンプレートから初期値を埋められるように
  する。
- 初期テンプレート例として「入社時の標準グループ付与」「退職時のアクセス剥奪」「部署異動時のグループ
  入れ替え」を 3〜5 種、製品固定のカタログとして用意する。
- 管理者が特定の enabled workflow を、明示的に選択した既存 User 集合 (検索条件で絞った候補一覧からの
  チェック選択) に対して手動実行 (on-demand run) できる interface を追加する。on-demand run は通常の
  trigger 評価をバイパスするが、action 実行・checkpoint・audit・retry は通常の `WorkflowRun` と同じ
  経路を通す。
- on-demand run の対象選択は一括処理になり得るため、大量対象時は [[wi-126-async-job-runner]] 経由で
  run 生成自体を非同期化する。

## Out of Scope
- テンプレートの管理者によるカスタム作成・共有・エクスポート。初期は製品固定のカタログのみとする。
- workflow enable 時の自動遡及適用。「明示操作でのみ遡及適用する」という設計方針を維持し、暗黙の
  全件適用はしない。
- [[wi-225-lifecycle-workflow-date-based-triggers]] の日付ベース自動発火 (本 WI は管理者が明示的に
  トリガーする手動実行のみを扱う)。

## Plan
- テンプレートは「trigger/action の初期値を埋めるだけの読み取り専用データ」とし、保存後は通常の
  `LifecycleWorkflow` と同じ revision 管理に従う (テンプレートと workflow の間に永続的なリンクは
  作らない)。
- on-demand run は「`source_occurrence_id` を manual 実行の actor/timestamp ベースで生成する特別な
  trigger occurrence」として、既存の `WorkflowRun` 一意制約・重複排除の枠組みに載せる。

## Tasks
- [ ] T001 [SCL] `LifecycleWorkflowTemplate` モデルと on-demand run interface を追加する。
- [ ] T002 [Decision] on-demand run の対象選定方針と trigger 評価バイパスの整合性を ADR に記録する。
- [ ] T003 [App] テンプレートカタログと on-demand run usecase (大量対象の非同期化含む) を実装する。
- [ ] T004 [UI] workflow 作成画面へのテンプレート選択、run 対象選択 UI を追加する。
- [ ] T005 [Verify] SCL/Go/UI および大量対象時の非同期化を検証する。

## Verification
- `just yaml-check`
- `just test-go`
- `just verify-ui`
- 手動: テンプレートから作成した workflow が編集可能な状態で開始されることを確認する。
- 手動: 既存 User 50 名を選んで on-demand run を実行し、通常 run と同じ run 履歴・監査証跡が残る
  ことを確認する。

## Risk Notes
on-demand run は trigger 条件を無視して実行するため、意図しない User への誤適用リスクがある。
実行前に対象確認のプレビュー (dry-run 相当) を必須にし、`disable_user` 等の破壊的 action を含む
workflow では追加の確認ステップを設ける。
