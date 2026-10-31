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
  - { path: spec/modules/oauth2/main.tsp, symbol: EvaluateResourceAccess }
---

# 採否を判断したうえで tools/list を権限で絞り込む

## 動機

Agent に許可されないツールが見えると、Agent は呼べないツールを選び、拒否の往復が増える。
一方、[Ping の連携例](https://developer.pingidentity.com/identity-for-ai/integrations/idai-authorizing-gemini-on-behalf-of-users-with-ping.html)は `tools/list` を判断の対象外にしている。
ツール数だけ評価すると負荷が大きく、一覧の結果が実行の許可と誤解される懸念もあるので、効果を確かめてから採否を決める。

## 対象範囲

- 採否の判断を最初に行う。実クライアントで拒否の往復とツール選択への効果を測り、評価の負荷と比べる。
- 採用する場合、許可されたツールだけを一覧へ返す。複数の操作を一度に評価する入口を定める。

## 対象外

- 絞り込みを `tools/call` の評価の代わりにすること。一覧の後に権限が変わり得るので、実行時の評価は省かない。
- 条件ポリシーの属性を一覧で評価すること。呼び出しの引数は一覧の時点では存在しない。

## 設計

判断の材料は、実クライアントの挙動、ツール数と遅延、関係の判定の一括の可否である。
一括の入口は REQ-OAUTH2-062 と同じく提示トークンから主体を導き、登録した規則の範囲に閉じる。
DPoP の `jti` は一度しか使えないので、一覧の評価を一つの証明で行う形にする。

見送る場合は、判断の根拠を `## 要約` に書いて中止する。

## 計画

1. 効果と費用を測り、採否を決める。
2. 採用なら一括の評価の仕様を定め、実装する。

## タスク

- [ ] T001 [Decision] 実クライアントで効果と負荷を測り、採否を確定する。
- [ ] T002 [Spec] 一括の評価と一覧の絞り込みを仕様へ書く。
- [ ] T003 [Acceptance] 権限のないツールが返らず、絞り込み後も `tools/call` が評価される RED を作る。
- [ ] T004 [App] 実装する。
- [ ] T005 [Verify] 上限、ページング、変更通知を検証する。

## 検証

- 無改修の MCP サーバーの背後で一覧の内容を観測する受け入れテスト。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

一括の入口は権限の探索に使われ得る。
提示トークンの主体と登録した規則に閉じ、任意の主体を問えないようにする。
