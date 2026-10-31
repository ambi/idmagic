---
status: pending
authors: [tn]
risk: high
created_at: 2026-10-31
priority: p2
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/mcp-gateway/features/runtime.md, requirement: REQ-MCPGATEWAY-003 }
---

# MCP ゲートウェイがツールの引数を条件の属性として渡す

## 動機

規則の条件は要求の `attributes` を読むが、MCP ゲートウェイは送らない。
属性を読む条件は評価不能で拒否されるので、改修していない MCP サーバーでは「100 ドルまでは自動」を表現できず、判定できるのはツール名までである。

## 対象範囲

- `tools/call` の `arguments` のうち、どの値を `attributes` として渡すかの設定。
- 引数の大きさが `attributes` の上限を超える場合の扱い。

## 対象外

- 承認した操作の `operation_parameters` の対応（[[wi-83492-decide-approved-operations-through-the-mcp-gateway]]）。

## 設計

全引数を既定で渡すと、秘密を含む引数が評価へ流れる。渡す引数を明示する形を第一候補にする。

## 計画

1. 設定の形を決め、REQ-MCPGATEWAY-003 を更新する。
2. 上流の実行回数を観測する受け入れテストを先に書く。

## タスク

- [ ] T001 [Design] 渡す引数の宣言と上限超過の扱いを確定する。
- [ ] T002 [Acceptance] 金額の閾値で上流へ転送するかを観測する RED を作る。
- [ ] T003 [App] ゲートウェイを実装する。

## 検証

- `mise run test-go-package -- ./backend/shared/http/server_http`
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

ゲートウェイの申告は MCP クライアントが送った引数であり、業務の事実ではない。
