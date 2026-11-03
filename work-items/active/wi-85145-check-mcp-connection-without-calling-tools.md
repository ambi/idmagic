---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-04
priority: p3
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/admin-access.md, requirement: REQ-OAUTH2-088 }
---

# 業務ツールを実行せずに MCP の接続を確認し、不足する設定を案内する

## 動機

[wi-58793](../done/wi-58793-fill-mcp-connection-candidates-from-metadata-and-change-notices.md) から分けた。
接続先を登録した後、認証、一覧の取得、登録、割当、トークンの上限のどこで止まっているかを、業務ツールを実行せずに確かめる手段がない。

## 対象範囲

- 認証、一覧の取得、登録、割当、トークンの上限を確認する接続確認を提供する。
- 不足する設定を案内する。

## 対象外

- 業務ツールの実行。
- 報告や説明文から、権限や承認の要否を推測すること。

## 設計

着手時に、確認を行う主体を決める。候補は管理コンソール、Gateway、Agent 側である。
あわせて、閉域の MCP サーバーへ届かない IdMagic が、確認できる範囲も決める。

## 計画

今回は起票だけを行う。

## タスク

- [ ] T001 [Design] 確認の主体、項目、結果の表現を決める。
- [ ] T002 [Spec] 要件と契約を更新する。
- [ ] T003 [App] 接続確認と案内を実装する。
- [ ] T004 [Verify] 変更を検証する。

## 検証

着手時に、主要ユースケースと検証タスクを決める。

## リスク

確認のための呼び出しが業務ツールを実行すると、意図しない作用が起きる。
確認には一覧の取得とトークンの発行だけを使う。
