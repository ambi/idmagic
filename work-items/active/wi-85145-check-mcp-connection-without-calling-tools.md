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

確認の項目は、IdMagic が手元の状態だけで判定できるものと、MCP サーバーへの到達を要するものに分かれる。

| 項目 | 判定に使えるもの | MCP サーバーへの到達 |
| --- | --- | --- |
| 登録 | 登録の状態（`Active`）、評価用のクライアントの紐付け（REQ-OAUTH2-088） | 不要 |
| 割当 | 操作割当（REQ-OAUTH2-083、REQ-IDMANAGEMENT-105）と、Agent の資格情報の束縛 | 不要 |
| トークンの上限 | クライアントのスコープ、ガードレール（REQ-OAUTH2-073） | 不要 |
| 一覧の取得（Gateway 経由） | Gateway が送ったカタログの `received_at`（REQ-OAUTH2-086、REQ-OAUTH2-087）。届いていれば Gateway から上流と IdMagic の双方へ届いている | 不要 |
| 一覧の取得（直接） | IdMagic からの `tools/list` | 必要。[[wi-38469-fetch-public-mcp-server-metadata-and-tools]] の取得の経路と制限を使い、別の外向きの経路を作らない |
| 認証 | PRM の `authorization_servers` と、トークンを提示した呼び出しの 401 以外の応答 | 必要 |

到達を要しない項目だけで先に提供できるかを、T001 で判断する。
到達を要する項目は wi-38469 の完了に依存するので、その場合は `depends_on` へ加える。

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
