---
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-16
depends_on: [wi-153-identity-lifecycle-workflows, wi-217-lifecycle-workflow-durable-run-handoff, wi-218-lifecycle-workflow-action-execution-and-audit]
---

# 日時ベースの相対時刻トリガーを lifecycle workflow に追加する

## Motivation
Microsoft Entra ID Lifecycle Workflows と Okta Lifecycle Management の中心的な価値は、入社予定日や
退職日などの日付属性を起点にした「X 日前 / X 日後」の相対時刻トリガーである。これにより
入社前のアカウント準備 (pre-hire provisioning) や、退職後に猶予期間を置いたアクセス剥奪
(post-offboarding cleanup) を表現できる。

現在の IdMagic の LifecycleWorkflow は User mutation イベント (`user_created` /
`user_attributes_changed` / `user_status_changed`) のみを起点にしており、
`spec/contexts/identity-management.yaml` (L1531) も「日付・cron・待機 step は対象外」と明記して
日付が到来したことだけを理由にした trigger を持たない。この結果、「入社日の 3 営業日前にアカウントを
有効化する」「退職日の 30 日後にアカウントを削除する」といった典型的な joiner/mover/leaver シナリオを
IdMagic だけで完結できず、外部 cron や手動運用に頼らざるを得ない。

## Scope
- `spec/contexts/identity-management.yaml` の `WorkflowTriggerKind` に `date_attribute_offset`
  (仮称) を追加し、対象の日付型属性、offset 方向 (before/after)、offset 日数を持つ trigger 定義を
  追加する。
- 日次スキャン job (既存の [[wi-126-async-job-runner]] を利用) を追加し、対象日付属性を持つ全 User
  × enabled workflow を評価し、条件を満たす User に対して `WorkflowRun` を生成する。
- 同一 User × workflow × 対象日について 1 日 1 回のみ run を生成する重複排除
  (`source_occurrence_id` を評価日ベースで構成する等) を設計する。
- スキャン対象の日付属性は `TenantUserAttributeSchema` の日付型属性に限定する fail-closed な
  validation を追加する。
- ADR に、日次スキャン方式 (cron ではなく既存 job runner の recurring job) を採用する理由と、
  タイムゾーン・DST・スキャン遅延時の扱いを記録する。

## Out of Scope
- 汎用 cron trigger、任意時刻での待機 step、DAG 分岐。wi-153 の「任意 expression engine は採らない」
  方針を継続する。
- 分/時間単位の精度。日次粒度に限定する。
- [[wi-226-lifecycle-workflow-templates-and-on-demand-run]] の on-demand 実行 (本 WI は日付が
  到来したことによる自動発火のみを扱う)。

## Plan
- 既存の trigger evaluator / run planner の構造 (kind 別の trigger 定義 + filter) を再利用し、
  `date_attribute_offset` を 4 つ目の kind として追加する。
- 日次スキャンは DynamicGroupRule の全件再評価 job (ADR-111) のパターンを参考にする。
- 重複排除は「同一 User × workflow × revision × 評価日」を一意制約にする。

## Tasks
- [ ] T001 [SCL] `date_attribute_offset` trigger kind、日次スキャン scenario、objective を追加する。
- [ ] T002 [Decision] 日次スキャン方式、タイムゾーン方針、dedup 方針を ADR に記録する。
- [ ] T003 [App] trigger evaluator の拡張と日次スキャン job を実装する。
- [ ] T004 [Verify] 境界日 (offset 当日、前後日)、タイムゾーン、重複実行を検証する。

## Verification
- `just yaml-check`
- `just test-go`
- `just verify-go`
- 自動: 対象日付の N 日前に達した User に対して 1 回だけ run が生成される。
- 自動: スキャンが 1 日に複数回走っても同一 run が重複生成されない。
- 手動: 入社日 3 日前に account を enable する workflow を作成し、対象日到達時に run が生成される
  ことを確認する。

## Risk Notes
日次スキャンは大規模テナントで全 User 走査コストがかかるため、対象日付属性を持つ User への
インデックス等のパフォーマンス配慮が必要であり、[[wi-161-large-tenant-performance-foundation]] と
連携する可能性がある。タイムゾーンの扱いを誤ると意図しない日に trigger が発火するため、テナントの
タイムゾーン設定または固定 UTC 基準を ADR で明確に固定する。
