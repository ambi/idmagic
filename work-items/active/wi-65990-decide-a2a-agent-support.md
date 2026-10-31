---
status: pending
authors: [tn]
risk: medium
created_at: 2026-10-31
priority: p3
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-048 }
---

# A2A の Agent 連携に専用の対応が要るか判断して対応する

## 動機

[Okta](https://www.okta.com/newsroom/press-releases/ai-innovations-oktane-2026/) は A2A 接続を一般提供し、Entra も A2A の認可を説明している。
IdMagic の委譲は RFC 8693 の連鎖で Agent 間の呼び出しを表せるが、A2A のプロトコルに固有の接続例や Agent Card の検証はない。
既存の委譲で足りるのか、専用の機能が要るのかは確かめていない。

## 対象範囲

- 判断を最初に行う。A2A の現行仕様の認証要求と、既存の委譲で動く例を作って不足を確かめる。
- 不足がなければ接続例だけを提供する。
- 不足があれば、必要な最小の機能を仕様化する。

## 対象外

- Agent のマーケットプレイスと発見サービス。
- A2A の通信の中継。

## 設計

第一候補は既存の委譲と [[wi-11085-enforce-mcp-tool-calls-for-unmodified-servers]] の評価を使う接続例である。
Agent Card の署名検証は、受け取った主張と登録の Agent を結ぶ必要が示されたときだけ採る。
不足がなく例も不要と判断すれば、理由を `## 要約` に書いて中止する。

## 計画

1. 現行仕様を調べ、動く例で不足を確かめる。
2. 結果に応じて例か機能を追加する。

## タスク

- [ ] T001 [Decision] 仕様の調査と実験で不足を確定する。
- [ ] T002 [Spec] 機能が要る場合だけ規範を追加する。
- [ ] T003 [Acceptance] 二段の Agent 呼び出しの連鎖と、中間の停止後の拒否を観測する。
- [ ] T004 [Docs] 接続例を書く。

## 検証

- `mise run test-go-package -- ./backend/shared/http/server_http`：連鎖と停止を確かめる。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

Agent Card の自己申告を権限の根拠にすると権限の詐称につながる。
記述の主張から権限を導かない。
