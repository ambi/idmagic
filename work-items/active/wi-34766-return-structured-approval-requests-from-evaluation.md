---
status: pending
authors: [tn]
risk: medium
created_at: 2026-10-31
priority: p2
depends_on: [wi-83492-decide-approved-operations-through-the-mcp-gateway]
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/resource-access.md, requirement: REQ-OAUTH2-065 }
  - { path: docs/modules/oauth2/features/approval.md, requirement: REQ-OAUTH2-041 }
---

# 評価の不許可から承認の開始まで Agent を導く

## 動機

承認と実行の照合は REQ-OAUTH2-065 が定めるが、承認を始めるまでの流れが弱い。
評価が `approval_required` を返しても、Agent が次に送るべき `authorization_details` は応答から分からない。
MCP クライアントへ承認待ちを伝える手順もないので、高影響の操作を実行の途中で人の判断へつなげない。

## 対象範囲

- `approval_required` の応答に、Agent が CIBA を始めるのに必要な明細を返す。
- MCP ゲートウェイが承認待ちを MCP クライアントへ伝える形を定める。
- 承認者への通知は既存のメールを使う。

## 対象外

- CIBA の `push`。通知基盤の決定が要る。
- 条件から承認要求を導く判定（[[wi-11371-evaluate-runtime-conditions-on-resource-access]]）。

## 設計

返す明細は要求の操作から組み立て、トークンのクレームを漏らさない。
不許可の応答にクレームを返さないとする REQ-OAUTH2-062 の判断と両立するよう、返す項目を要求の写しに限る。
MCP クライアントへの伝え方は、ツールのエラー結果か elicitation かを 2026-07-28 版と実クライアントの対応で T001 に決める。

## 計画

1. 応答の形と MCP での伝え方を決める。
2. 評価の不許可、CIBA、承認、再評価、実行までの一連の試験を先に書く。

## タスク

- [ ] T001 [Design] 応答の形と MCP での伝え方を確定する。
- [ ] T002 [Spec] REQ-OAUTH2-065 と TypeSpec を更新する。
- [ ] T003 [Acceptance] 不許可から実行までの RED を観測する。
- [ ] T004 [App] 評価と MCP ゲートウェイを実装する。
- [ ] T005 [Verify] 引数の変更、期限切れ、並行の要求を検証する。

## 検証

- `mise run test-go-package -- ./backend/shared/http/server_http`：承認の E2E を拡張する。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

Agent が自分の要求で承認画面を埋められるので、承認者への通知の濫用を既存の流量制限で抑える。
`fields` は画面に表示するので、秘密を入れない前提を維持する。
