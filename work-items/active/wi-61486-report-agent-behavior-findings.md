---
status: pending
authors: [tn]
risk: medium
created_at: 2026-10-31
priority: p3
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/identity-management/features/agent.md, requirement: REQ-IDMANAGEMENT-094 }
---

# 採否を判断したうえで Agent の異常な行動を所見として示す

## 動機

[Entra ID Protection](https://learn.microsoft.com/en-us/entra/id-protection/concept-risky-agents) は、サインインの急増、不慣れな資源、失敗した試行を検出して Conditional Access へつなぐ。
IdMagic は評価の拒否や発行を監査へ残すが、管理者へ所見として示さない。
脅威情報や機械学習に依存する検出は IdMagic の強みにならないので、規則で足りる範囲と効果を先に見極める。

## 対象範囲

- 判断を最初に行う。監査の属性から計算できる所見の精度と運用の価値を、実データか再現データで確かめる。
- 拒否の急増、発行の急増、未使用の資源へのアクセスの所見。
- 管理者の無効化への導線。

## 対象外

- 脅威情報の収集、機械学習による検出。
- 所見からの自動停止。誤検知で業務を止めるので、精度を確認するまで採らない。
- 外部の検出器からの信号（[[wi-61710-decide-agent-detection-signals-and-web-bot-auth]]）。

## 設計

棚卸しと同じく状態を持たず監査の読み取りで計算する案を第一候補にし、保持期間と読み取りの費用で成り立つかを確かめる。
見送る場合は、判断の根拠を `## 要約` に書いて中止する。

## 計画

1. 規則ごとの効果を確かめ、採否を決める。
2. 採用した規則を棚卸しの所見へ追加する。

## タスク

- [ ] T001 [Decision] 規則の効果を確かめ、採否と閾値を確定する。
- [ ] T002 [Spec] 所見を REQ-IDMANAGEMENT-094 の周辺へ追加する。
- [ ] T003 [Acceptance] 所見の出現と非出現の RED を作る。
- [ ] T004 [App] 実装する。
- [ ] T005 [Verify] 件数の多いテナントの費用を確かめる。

## 検証

- `mise run test-go-package -- ./backend/idmanagement/...`。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

誤検知が多いと所見が読まれなくなる。
根拠の件数と期間を所見に添える。
