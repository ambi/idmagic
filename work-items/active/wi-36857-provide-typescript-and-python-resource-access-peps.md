---
status: pending
authors: [tn]
risk: medium
created_at: 2026-10-31
priority: p2
depends_on: []
change_kind: tooling
spec_impact: { kind: none, reason: "`EvaluateResourceAccess` の公開契約を使うクライアントを追加し、契約と評価の振る舞いを変えない。" }
---

# TypeScript と Python の参照 PEP を提供する

## 動機

参照 PEP は Go だけだが、MCP サーバーは TypeScript と Python で書かれることが多い。
無改修の MCP ゲートウェイを置けない環境でも、組み込みで評価を使えるようにする。

## 対象範囲

- TypeScript と Python の最小の PEP。通信の失敗、200 以外、`decision=false` で業務処理を呼ばない。
- 公開の MCP SDK へ組み込む例。
- 言語をまたぐ適合試験。

## 対象外

- パッケージレジストリへの公開と保守の約束。公開の時期が決まるまで `examples` に置く。
- Agent 側のトークン取得 SDK。

## 設計

Go の参照 PEP と同じ入出力の表を適合試験の共通データにし、言語間の差を検出する。
依存は各言語の標準的な HTTP に限り、IdMagic の内部型を写さない。

## 計画

1. 適合試験のデータを作る。
2. 二言語の PEP を実装する。

## タスク

- [ ] T001 [Design] 適合試験の形と実行環境を決める。必要なランタイムを `mise.toml` へ追加する。
- [ ] T002 [Test] 共通データで Go の PEP が通ることを確かめる。
- [ ] T003 [App] TypeScript と Python の PEP を実装する。
- [ ] T004 [Verify] 故障を注入し、業務処理を呼ぶ実装を適合試験が検出することを確かめる。

## 検証

- 追加する適合試験のタスクと `mise run test-go-package -- ./examples/resource-server-pep`。

## リスク

参照実装の誤りはそのまま利用者へ写される。
既定で不許可になる形にし、適合試験で固定する。
