---
status: pending
authors: [tn]
risk: medium
created_at: 2026-11-26
priority: p2
group: development-process
depends_on: [wi-27706-verify-upstream-oauth-token-endpoints-with-recordings]
change_kind: tooling
spec_impact: { kind: none, reason: "上流の MCP サーバーに対する前提を、記録の再生と状態を持つ模倣で確かめる検査を足す。規範シナリオ、標準の宣言、TypeSpec シンボルを追加も変更もしない。検査が規範に反する実装を見つけたら、その時点で affected_spec を宣言し直す。" }
---

# 上流の MCP サーバーへの前提を、記録の再生と模倣で確かめる

## 動機

MCP Gateway は上流の MCP サーバーの `tools/list` を集め、変更通知を待ってカタログを送り直す。
OAuth2 は、管理者の操作で公開された MCP サーバーの Protected Resource Metadata と `tools/list` を取得する。
上流の応答の形が想定と違うと、カタログが古くなるか空になり、Agent が使えるツールを誤る。

MCP の仕様は版の更新が速く、ほかの外部の相手より前提がずれやすい。
それぞれのアダプターのテストは `httptest` で応答を返す形であり、相手の実際の応答の形を、仕様の版と結び付けて固定していない（2026-11-26 に確かめた）。

## 対象範囲

- MCP の仕様の版ごとの要求と応答の例（初期化、`tools/list`、ページング、変更通知、エラー）と、RFC 9728 の Protected Resource Metadata の例を転記した記録と、MCP Gateway と OAuth2 の取得の再生の検査。記録の相手の版には MCP の仕様の版を書く。
- 状態を持つ MCP サーバーの模倣。ツールの追加と削除、変更通知、ページング、未対応の版、`401`、`5xx`、不正な本文を返せる。記録の再生と同じ応答を返すことを検査する。
- 記録と模倣の部品は [wi-27706](wi-27706-verify-upstream-oauth-token-endpoints-with-recordings.md) が決めた共有の置き場所のものを使う。
- 見つかった誤りの修正。製品の結果を変えるものは、着手後に利用者に確かめてから `affected_spec` を宣言する。

## 対象外

- 一覧の定期取得と認証の結果の表示。[wi-19974](wi-19974-refresh-public-mcp-tool-lists-and-check-their-authentication.md) が扱う。この項目の模倣はその検査にも使える。
- 特定の MCP サーバーの製品ごとの記録。

## 設計

記録の出どころは仕様の例の転記にし、取得方法に仕様の版と節を書く（[テスト方針](../../docs/development/testing.md#外部契約と設定の利用)）。
仕様の版を一つ上げるたびに、新しい版の例を別の記録として足し、古い版の記録は対応を続けるあいだ残す。
これにより、版の差で壊れたのか、記録が古いのかを区別できる。

## 計画

1. 対応する MCP の仕様の版を確かめ、例を転記して再生の検査を置く。
2. MCP サーバーの模倣を置き、記録との照合と、変更通知と障害の検査を置く。

## タスク

- [ ] T001 [Adapter] 記録と再生の検査を置く。
- [ ] T002 [Adapter] MCP サーバーの模倣と照合を置く。
- [ ] T003 [Verify] 実際の差分から選んだ検査を実行する。

## 検証

- 記録の応答を仕様に反する形へ変えると、再生の検査が落ちる。
- 模倣の応答を記録から外すと、照合が食い違いを報告する。
- `mise run verify-changed -- main`。

## リスク

仕様の例は実際のサーバーの癖（追加のフィールド、通知の順序）を含まない。
保証が仕様の例を解釈できることまでだと、記録の取得方法に書く。
