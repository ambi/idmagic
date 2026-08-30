---
depends_on: [wi-448-feature-lifecycle-and-update-policy]
status: pending
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-08-30
priority: p1
change_kind: operations
affected_spec:
  - { path: docs/contexts/system/scenarios.md, requirement: REQ-SYSTEM-001 }
  - { path: docs/contexts/system/scenarios.md, requirement: REQ-SYSTEM-002 }
---

# 配備前に旧版と新版の更新互換性を判定する

## Motivation

現在のリリース手順は公開 API の破壊的差分を検査し、段階的に配備するが、旧配備と新成果物の機能版、設定、スキーマ移行を合わせて、ローリング更新が可能か再作成が必要かを事前に判定しない。誤った更新方式は、新旧プロセスが同じ永続状態を読めない、準備完了前にトラフィックを受ける、途中で更新を止められないといった障害を配備中に初めて表面化させる。

Keycloak の更新互換性検査から、旧配備の内部メタデータを利用者に解析させず、安定した判定と終了コードだけを自動化へ返す境界を採用する。独自 Operator と継続的な再調整は導入しない。

## Scope

- 旧配備の版付き `DeploymentMetadata` と新成果物の `BuildMetadata` を入力に、`EvaluateUpdateCompatibility(previous, next)` が `rolling_candidate`、`recreate`、`invalid` のいずれかと理由コードを返す。
- 判定入力に製品版、機能版と `UpdatePolicy`、設定の更新影響、データベーススキーマ互換区間を含める。
- 検査は読み取り専用とし、データベース移行、Kubernetes 資材の変更、プロセス停止を行わない。
- 自動化は版付き JSON の内部構造ではなく、文書化した終了コードと標準出力の安定した要約だけに依存する。
- `mise` タスクとしてローカルと CI から同じ判定器を実行し、リリース手順と配備 runbook から参照する。
- `rolling_candidate` は静的な互換性候補であり、配備許可を意味しない。後続の [[wi-450-mixed-version-release-acceptance]] が同じ版の組合せを実際の異版混在境界で検査し、リリースゲートが静的判定と動的証拠を合成して配備可否を決める。

## Out of Scope

- Kubernetes Operator、CRD、`observedGeneration`、conditions の実装。
- 配備や rollback の自動実行。
- 任意の過去版から任意の新版への互換性保証。対象はサポートする直前安定版から次版への更新である。
- 公開 HTTP API の互換性検査。既存の `check-api-compat` が所有する。

## Design

`DeploymentMetadata` は配備時に生成した不変の内部成果物とし、`schema_version`、`product_version`、`features`、`config_fingerprint`、`database_compatibility` を持つ。秘密値と設定値そのものは含めず、更新判断に必要な分類と digest だけを記録する。未知の metadata 版、欠落した必須入力、比較不能な版は `invalid` としてフェイルクローズする。

判定は決定表として実装する。すべての変更がローリング更新可能なら `rolling_candidate`、一つでも再作成を要求すれば `recreate`、入力矛盾または未対応の移行があれば `invalid` を返す。理由は機械可読な閉じたコードとして複数返し、人向け説明を別に生成する。

終了コードは `0=rolling_candidate`、`10=recreate`、`20=invalid`、`30=unexpected_failure` とし、内部 JSON の項目追加では自動化を壊さない。終了コードの割当は正準運用文書に記録し、後方互換な追加以外は版を上げる。外部自動化へ公開した終了コードは撤回できない契約として扱う。

## Plan

1. 配備前判定のシナリオ、入力、出力、失敗時の配備停止を仕様化する。
2. メタデータスキーマと決定表を Unit RED から実装する。
3. 現在の成果物と配備設定から秘密を含まないメタデータを生成する。
4. `mise` タスク、CI、リリース手順へ読み取り専用の事前検査として接続する。
5. `rolling_candidate` と版の組合せを後続の異版混在検査へ渡す契約を定め、配備許可の合成はリリースゲートへ残す。

## Tasks

- [ ] T001 [Spec] 更新互換性の判定、終了コード、フェイルクローズ、配備停止を正準シナリオと運用文書へ定める。
- [ ] T002 [Domain] メタデータ型と `EvaluateUpdateCompatibility` を代表的な決定表の Unit RED から実装する。
- [ ] T003 [Build] 新成果物と旧配備から秘密を含まない版付きメタデータを生成および取得する。
- [ ] T004 [CLI] 安定した終了コードと要約を返す読み取り専用コマンドを `mise` タスクで公開する。
- [ ] T005 [Release] CI とリリース手順で `invalid` と予期しない失敗を拒否し、`recreate` をローリング更新へ進めず、`rolling_candidate` には後続の動的証拠を要求する。
- [ ] T006 [Acceptance] 直前安定版、機能版変更、停止必須設定、未対応メタデータを実成果物境界で検査する。
- [ ] T007 [Verify] 主要ユースケースの Unit/E2E RED を記録し、`rolling_candidate`、`recreate`、`invalid`、未知入力、優先順位競合、終了コード変換を含む決定ロジックの全分岐へ体系的な変異検査または明示的な故障注入を行い、等価変異と検査限界を記録して標準検証を通す。

## Verification

- `mise run check-spec`
- 新設する更新互換性検査の `mise` タスク
- `mise run verify`
- `rolling_candidate`、`recreate`、`invalid`、予期しない失敗がそれぞれ安定した終了コードを返す。
- メタデータとログに秘密値が含まれず、未知の版や欠落入力が `rolling_candidate` へ倒れない。

## Risk Notes

リスクは high。誤った `rolling_candidate` 判定を配備許可として扱うと停止やデータ破損へ直結するため、未知の入力を再作成可能と推測せず `invalid` にする。判定器は配備を実行せず、実際の異版混在検査とロールバック手順をリリースゲートの別証拠として要求する。
