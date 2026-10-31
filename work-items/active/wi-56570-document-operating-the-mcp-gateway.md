---
status: pending
authors: [tn]
risk: low
created_at: 2026-10-31
priority: p2
depends_on: [wi-11085-enforce-mcp-tool-calls-for-unmodified-servers]
change_kind: docs
spec_impact: { kind: none, reason: "既存の仕様どおりの配置と運用の手順を追加し、規範と API の振る舞いを変えない。" }
---

# MCP ゲートウェイの配置と運用の手順書を用意する

## 動機

`idmagic-mcp-gateway` の仕様は設定と振る舞いを定めるが、運用者が順に辿れる手順がない。
上流へゲートウェイ以外から到達できる配置では認可を迂回できるので、配置の条件と確認方法を手順として示す必要がある。

## 対象範囲

- `docs/runbooks/` の手順書：実行ファイルまたはコンテナの作り方、設定、IdMagic 側のクライアントと登録と規則。
- 上流を到達できなくする配置と、その確認の手順。
- 前段の TLS 終端、シークレットの置き方、指標の待ち受けの公開範囲。
- 監査ログと `ResourceAccessEvaluated` の突き合わせ、評価と上流の障害時の対応。

## 対象外

- 実在の MCP クライアントとの再現手順（[[wi-36722-publish-an-end-to-end-mcp-enforcement-walkthrough]]）。
- 成果物を公開するリリース用 CI。リポジトリ全体の課題である。

## 設計

[Directory Connector の手順書](../../docs/runbooks/directory-connector.md)と同じ構成にし、設定の項目は[仕様](../../docs/modules/mcp-gateway/features/runtime.md)へ参照して複製しない。
コンテナの作り方が現在の成果物にない場合は、一般的な操作として `mise` のタスクか `infra/docker` への追加を T001 で決める。

## 計画

1. 成果物の作り方を決める。
2. 手順を書き、手元の起動で確かめる。

## タスク

- [ ] T001 [Design] 実行ファイルまたはコンテナの作り方を確定する。
- [ ] T002 [Docs] 手順書を書く。
- [ ] T003 [Verify] 手順どおりに起動し、許可、拒否、上流へ直接届かないことを確かめる。

## 検証

- `mise run check-links`。
- 手順どおりの起動で確かめる。

## リスク

手順と仕様の設定がずれる。
設定の項目は仕様へ参照し、手順書には複製しない。
