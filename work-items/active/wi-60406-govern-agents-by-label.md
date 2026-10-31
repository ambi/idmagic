---
status: pending
authors: [tn]
risk: medium
created_at: 2026-10-31
priority: p2
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/identity-management/features/agent.md, requirement: REQ-IDMANAGEMENT-009 }
  - { path: docs/modules/identity-management/features/agent.md, requirement: REQ-IDMANAGEMENT-075 }
---

# 採否を判断したうえでラベルで選んだ Agent 群をまとめて統制する

## 動機

IdMagic の統制は Agent ごとであり、同じ種類の Agent が増えると設定と停止の手間が増える。
[Entra](https://learn.microsoft.com/en-us/entra/agent-id/security-for-ai-overview) は blueprint と属性へ方針を適用し、クラス全体を一度に無効化できる。
ただし階層の移植は必要性が確かめられておらず、ラベルで足りるかを先に判断する。

## 対象範囲

- 判断を最初に行う。ラベル、グループ、テンプレートのどれが必要かを利用場面から決める。
- 管理者が付ける Agent のラベル。
- ラベルを選択条件にした一括の無効化と、ガードレールや条件ポリシーの対象の指定。

## 対象外

- blueprint の親子の階層、資格情報の継承、agent user。
- Agent が自身のラベルを変更すること。統制の対象から逃れられるので許さない。

## 設計

| 案 | 判断 |
| --- | --- |
| 管理者が付けるラベル（第一候補） | 階層なしで選択でき、既存の Agent に後から適用できる |
| Agent をグループへ入れる | 既存の Group との意味の整理が必要 |
| 親子のテンプレート | 継承は強力だが、継承元の変更の波及を説明する必要がある |

一括の無効化は Agent ごとの `AgentDisabled` を発行し、失効の経路を変えない。
途中の失敗は同じ操作の再実行で回収する。

## 計画

1. 案を確定する。
2. ラベルと一括操作を実装し、対象の指定を関連項目へ渡す。

## タスク

- [ ] T001 [Decision] 統制の単位を確定する。
- [ ] T002 [Spec] ラベルと一括の無効化を仕様へ書く。
- [ ] T003 [Acceptance] 一括の無効化の後の発行と評価の拒否の RED を観測する。
- [ ] T004 [App] 実装する。
- [ ] T005 [UI] 管理コンソールへ加える。
- [ ] T006 [Verify] 部分失敗の再実行と権限の分離を検証する。

## 検証

- `mise run test-go-package -- ./backend/idmanagement/...`。
- `mise run test-go-package -- ./backend/shared/http/server_http`。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

一括の操作は影響が大きい。
対象の件数を実行前に示し、権限を分ける。
