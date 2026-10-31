---
status: pending
authors: [tn]
risk: high
created_at: 2026-10-31
priority: p2
depends_on: [wi-11085-enforce-mcp-tool-calls-for-unmodified-servers]
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/resource-access.md, requirement: REQ-OAUTH2-065 }
  - { path: docs/modules/mcp-gateway/features/runtime.md, requirement: REQ-MCPGATEWAY-003 }
---

# MCP ゲートウェイで承認した操作をどう扱うか判断して対応する

## 動機

MCP ゲートウェイは `operation_key` と `operation_parameters` を評価へ渡さないので、`requires_approved_operation` の規則を持つツールは評価が 400 を返し、503 で止まる。
安全側で止まるが、人が承認した高影響の操作を改修していない MCP サーバーで実行できない。

## 対象範囲

- 改修していないサーバーで承認した操作を許可するか、許可するならどの保証で許可するかの判断。
- 採用した判断の仕様と実装。`tools/call` の `arguments` と `operation_parameters` の対応を含む。
- 使えない場合に、運用者と MCP クライアントが原因を区別できる応答と記録。

## 対象外

- 評価の不許可から承認を始める流れ（[[wi-34766-return-structured-approval-requests-from-evaluation]]）。
- 資源側の業務結果の記録を、ゲートウェイが代わりに持つこと。

## 設計

REQ-OAUTH2-065 の一回性は、資源が `operation_key` ごとに業務作用を一回に保つことを前提にする。
改修していないサーバーはその記録を持たない。
MCP には再試行をまたいで同じ値を保つ鍵がなく、再試行では JSON-RPC の `id` も変わる。
このため、実装の前に T001 で次の候補を比べ、選択は利用者が確定する。

| 候補 | 利点 | 懸念 |
| --- | --- | --- |
| 改修していないサーバーでは使えないと定める | 一回性の保証を弱めない | 高影響の操作に参照 PEP の組み込みが要る |
| 冪等と宣言したツールに限って許可する | 再実行の害がない範囲で使える | 宣言の誤りが二重実行になる |
| 明示した設定で一回性の保証を弱めて許可する | 多くのサーバーで使える | 結果の不明な要求の再実行を防げない |

鍵を本文の内容から導く案は、正当な同じ内容の二回目の操作を再送と区別できないので、候補を採るときに併せて確かめる。

## 計画

1. 候補を比べ、利用者が保証の範囲を決める。
2. 仕様を更新し、決めた範囲の受け入れテストを先に書く。

## タスク

- [ ] T001 [Decision] 改修していないサーバーでの一回性の保証と採否を確定する。
- [ ] T002 [Spec] REQ-OAUTH2-065 と REQ-MCPGATEWAY-003 を更新する。
- [ ] T003 [Acceptance] 承認、実行、再送、別の引数で上流の実行回数を観測する RED を作る。
- [ ] T004 [App] 決めた範囲を実装する。

## 検証

- 実在の MCP SDK のサーバーの実行回数を観測する受け入れテスト。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

一回性を誤って約束すると、人が一度だけ承認した操作が二度実行される。
保証できない範囲は許可せず、弱める場合は仕様と設定に明示する。
