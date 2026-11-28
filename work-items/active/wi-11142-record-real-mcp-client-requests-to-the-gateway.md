---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-11-29
priority: p2
group: mcp-gateway
depends_on: []
change_kind: tooling
spec_impact: { kind: none, reason: "実際の MCP のクライアントがゲートウェイと認可サーバーへ送る要求を記録し、受け入れの検査で再生する。規範シナリオ、標準の宣言、TypeSpec シンボルは変えない。再生が規範に反する実装を見つけたら、別の項目に起票する。" }
---

# 実際の MCP のクライアントがゲートウェイへ送る要求を記録し、受け入れの検査で再生する

## 動機

MCP ゲートウェイの受け入れの検査（`backend/shared/http/server_http/mcp_gateway_e2e_test.go` など）の要求は、`modelcontextprotocol/go-sdk`（v1.8.0）のクライアントが送る。
go-sdk は仕様の管理者が保守する独立のクライアントだが、利用者が実際に使う Claude Code や VS Code のクライアントではない。
実際のクライアントは、送るヘッダー（`MCP-Protocol-Version`、`Accept`、`Mcp-Session-Id`）、`initialize` の protocolVersion と capabilities、認可の流れ（保護されたリソースのメタデータの取得、Client ID Metadata Document か動的登録、PKCE、resource の指示）で go-sdk と違いうる。
その違いは、ゲートウェイの評価と上流への転送の結果を左右する。

2026-11-29 の [[wi-21740-decide-recordings-for-requests-idmagic-receives]] の判断では、ほかの相手と違い、この記録は手元のエージェントだけでは取れなかった。
実際のクライアントを手元のゲートウェイへつなぎ、利用者がブラウザーで認可する操作が要るためである。
Claude Code と VS Code の Client ID Metadata Document は、すでに記録がある（`backend/oauth2/internal/client/cimd_http/testdata`）。

## 対象範囲

- 手元のゲートウェイの前に要求を記録する受け口（ヘッダーと本文を保存し、資格情報を記録から除く）を置き、利用者がクライアントをつなぐ手順を `mise` のタスクと手順書にする。
- Claude Code と VS Code の、認可の流れと `initialize`、`tools/list`、`tools/call` の要求を記録する。
- 記録を受け入れの検査で再生し、期待値を MCP の仕様の文面から書く。

## 対象外

- 上流の MCP のサーバーの応答の記録。wi-65625 の系列が扱った。

## 設計

記録の受け口は、記録の道具（`record-negotiate`、`record-msal`）と同じく、ビルドタグ付きのテストか `mise` のタスクにし、CI は再生だけを実行する。
アクセストークン、認可コード、Cookie は記録から除き、取り除いた欄を取得方法に書く。

## 計画

1. 記録の受け口と手順を作る。
2. 利用者が手元でクライアントをつなぎ、記録を取る。
3. 記録を再生する検査を置く。

## タスク

- [ ] T001 [Tooling] 記録の受け口と手順を作る。
- [ ] T002 [Recording] Claude Code と VS Code の記録を取る（利用者の操作）。
- [ ] T003 [Acceptance] 記録を再生する検査を置く。
- [ ] T004 [Verify] 差分から選んだ検査を実行する。

## 検証

- `mise run test-go-package -- ./backend/shared/http/server_http`。

## リスク

記録に資格情報が残ると、手元のテナントのトークンが漏れる。
保存の前に資格情報を含まないことを検査する（`testing_transitrecord` の `CheckSecrets` にならう）。
