---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-04
priority: p2
depends_on: [wi-89346-detect-unspecified-behavior-mechanically, wi-83002-rewrite-remaining-identity-management-features-in-the-requirement-format]
change_kind: tooling
spec_impact:
  kind: none
  reason: "状態遷移表から予測するテストを加えるだけで、要件、TypeSpec の契約、永続状態、発行するイベント、外向きの呼び出しは変えない。表と実装の食い違いを見つけた場合の対応は、分類して別に扱う。"
---

# IdManagement の User 以外の状態機械に、状態遷移表から予測するモデルベースのテストを加える

## 動機

wi-89346 は、User のライフサイクルの状態遷移表を実行時に読み、管理 API の結果と比べるモデルベースのテストを加えた。
User 以外の状態機械は、wi-83002 で状態遷移表を持つが、表と実装の食い違いを検出するテストがない。

## 対象範囲

- AgentLifecycle、DataExportLifecycle、DynamicMembershipEvaluationLifecycle に、`backend/shared/testing_statematrix` で表を読むモデルベースのテストを加える。
  全セルを一度ずつ試す網羅と、種を固定した乱数の操作列の二つを置く。
- 時間や非同期の処理（エクスポートの生成、全件の再評価）を起こす操作の列を、テストから起こす方法を決める。
- 表と実装の食い違いを、仕様にない振る舞いの分類に従って扱う。

## 対象外

- IdManagement 以外の Context の状態機械。
  各 Context の書き直し（wi-71372、wi-35451）で状態遷移表を加えるときに扱う。

## 設計

User のテスト（`backend/idmanagement/handlers_http/user_lifecycle_model_test.go`）と同じく、操作の列の名前から要求への対応と、条件の作り方だけをテストに置く。
三つの状態機械で同じ組み立てが繰り返されるなら、予測と比較の部分を `testing_statematrix` へ移すかを検討する。

## タスク

- [ ] T001 [Test] AgentLifecycle のテストを加える。
- [ ] T002 [Test] DataExportLifecycle のテストを加える。
- [ ] T003 [Test] DynamicMembershipEvaluationLifecycle のテストを加える。
- [ ] T004 [Verify] 表のセルと実装に故障を入れて、テストが検出することを確かめる。

## 検証

- `mise run test-go-package -- ./backend/idmanagement/handlers_http`
- `mise run verify`

## リスク

- 非同期の処理を含む状態機械では、テストが時刻や実行順に依存しやすい。
  時刻とジョブの実行はテストの入力として渡し、待機で同期しない。
