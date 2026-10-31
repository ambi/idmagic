---
status: pending
authors: [tn]
risk: medium
created_at: 2026-10-31
priority: p1
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/resource-access.md, requirement: REQ-OAUTH2-062 }
  - { path: spec/modules/oauth2/main.tsp, symbol: EvaluateResourceAccess }
---

# リソースアクセスの評価に性能目標と流量制限を設ける

## 動機

`POST /resource-access/evaluate` は業務操作ごとに呼ばれる同期の判断点である。
経路上の執行部品を置くと、すべての `tools/call` の遅延と可用性がこの入口に依存する。
しかし [[wi-80420-enforce-agent-access-at-external-resource-servers]] は評価を容量の仮定に入れず、負荷を実測せず、流量制限も掛けていない。

## 対象範囲

- 評価の応答時間と処理量の目標を定め、関係の深さ、DPoP の有無、失効判定を含む現実的な条件で測定する。
- 登録したリソースサーバーのクライアント単位の流量制限と、超過時の応答を定める。
- 超過時に業務操作を許可しないことを保証する。

## 対象外

- Agent やツールの単位の回数制限（[[wi-25649-limit-agent-tool-calls-at-runtime]]）。
- 評価の結果の再利用。停止の保証を「次の評価から不許可」とする REQ-OAUTH2-062 の判断を変えない。

## 設計

既存の `shared/ratelimit` をクライアント ID の鍵で使い、429 と `Retry-After` を第一候補にする。
参照 PEP は 429 を通信の失敗と同じく不許可として扱う。
目標は `docs/requirements/quality.md` のサービス目標へ、測定方法は `docs/design/performance/` へ置く。

## 計画

1. 測定の条件と目標を決める。
2. 現状を測定し、ボトルネックがあれば局所的に改善する。
3. 流量制限の仕様と TypeSpec を更新し、実装する。

## タスク

- [ ] T001 [Design] 目標、測定条件、制限の鍵と既定値を決める。
- [ ] T002 [Spec] REQ-OAUTH2-062 と `EvaluateResourceAccess` に 429 を加える。
- [ ] T003 [Acceptance] 超過時に 429 を返し、参照 PEP が業務処理を呼ばないことの RED を観測する。
- [ ] T004 [App] 流量制限を実装する。
- [ ] T005 [Verify] 負荷を測定し、結果を性能設計へ記録する。

## 検証

- `mise run test-go-package -- ./backend/shared/http/server_http`：正式な組み立てで超過の応答を観測する。
- `mise run test-go-package -- ./examples/resource-server-pep`：429 で業務処理を呼ばないことを確かめる。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

制限が厳しいと正当な操作が止まる。
既定値は測定結果から決め、テナントの運用で調整できるかを T001 で判断する。
