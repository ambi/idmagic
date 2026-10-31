---
status: pending
authors: [tn]
risk: medium
created_at: 2026-10-31
priority: p2
depends_on: [wi-10407-measure-and-rate-limit-resource-access-evaluation]
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/resource-access.md, requirement: REQ-OAUTH2-062 }
  - { path: spec/modules/oauth2/models.tsp, symbol: ResourceAccessRule }
---

# Agent とツールの単位で実行回数を制限する

## 動機

短命なトークンを持つ Agent でも、暴走すれば許可された操作を大量に実行できる。
[PingGateway](https://developer.pingidentity.com/blog/securing-mcp-servers-with-pings-mcp-gateway/) はツール、クライアント、主体ごとの流量制限を説明している。
[[wi-30707-agent-guardrail-policies-at-issuance-and-exchange]] の評価点は発行と交換に限られ、実行の回数には届かない。

## 対象範囲

- 規則に Agent、利用者、ツールの単位の流量と回数の上限を宣言する。
- 超過を理由付きの不許可にし、監査へ残す。
- 計数の障害時に許可しない。

## 対象外

- 予約と確定を要する予算の保証。消費量の報告の認証が別に必要である。
- リソースサーバー単位の入口の保護（[[wi-10407-measure-and-rate-limit-resource-access-evaluation]]）。

## 設計

超過の判定は評価の論理積の最後に置き、他の理由で不許可の要求を計数しない。
計数の保存先は既存の `shared/ratelimit` を第一候補にし、複数プロセスで上限が守られるかを確かめる。
回数は「評価で許可した数」であり、業務が実行した数ではないことを仕様に書く。

## 計画

1. 単位、窓、計数の保存先を決める。
2. 仕様を更新し、並行の試験を先に書く。

## タスク

- [ ] T001 [Design] 単位、窓、計数の保存先、超過の理由を確定する。
- [ ] T002 [Spec] 規則と評価の仕様を更新する。
- [ ] T003 [Acceptance] 上限内の許可、超過の不許可、計数の障害の RED を観測する。
- [ ] T004 [App] 実装する。
- [ ] T005 [Verify] 並行の要求と複数プロセスの上限を検証する。

## 検証

- `mise run test-go-package -- ./backend/oauth2/internal/token/usecases`。
- `mise run test-go-package -- ./backend/shared/http/server_http`。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

利用者を単位にすると計数の鍵に識別子が入る。
監査とログへの露出を既存の秘匿の規則に合わせる。
