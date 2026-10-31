---
status: pending
authors: [tn]
risk: medium
created_at: 2026-10-31
priority: p2
depends_on: [wi-11085-enforce-mcp-tool-calls-for-unmodified-servers]
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/resource-access.md, requirement: REQ-OAUTH2-062 }
---

# MCP のMCP ゲートウェイを Envoy へ接続する

## 動機

既存のゲートウェイを運用する組織に、単体プロキシの追加を求めると導入の障壁になる。
[Ping の Google 連携](https://developer.pingidentity.com/identity-for-ai/integrations/idai-authorizing-gemini-on-behalf-of-users-with-ping.html)は Envoy の `ext_proc` を使っている。

## 対象範囲

- MCP ゲートウェイの中核を Envoy の外部処理から使う接続。
- 本文を読む必要に合う方式の選択と、失敗時に不許可となる設定例。

## 対象外

- 各クラウドのゲートウェイ製品への個別の連携。実際の導入先が決まってから扱う。
- 中核の判定の変更（[[wi-11085-enforce-mcp-tool-calls-for-unmodified-servers]]）。

## 設計

`tools/call` のツール名は本文にあるので、本文の転送を制御できる `ext_proc` を第一候補にする。
`ext_authz` は本文の上限と緩衝の条件を確かめて比較する。
既定の `failure_mode_allow` のような、失敗時に通す設定を手順に載せない。

## 計画

1. `ext_proc` と `ext_authz` を実測で比べる。
2. 接続を実装し、実在の Envoy で試験する。

## タスク

- [ ] T001 [Design] 方式を確定する。
- [ ] T002 [Acceptance] Envoy 経由の許可、拒否、外部処理の障害で上流の呼び出しを観測する RED を作る。
- [ ] T003 [App] 接続を実装する。
- [ ] T004 [Verify] ストリーミングと本文の上限を検証する。

## 検証

- 実在の Envoy を起動する試験を `mise` のタスクへ追加し、CI での実行可否を決める。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

Envoy の設定の誤りは執行の迂回になる。
手順に迂回の検査を含める。
