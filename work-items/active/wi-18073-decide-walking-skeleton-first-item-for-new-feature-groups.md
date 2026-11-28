---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-11-28
priority: p3
group: development-process
depends_on: []
change_kind: docs
spec_impact: { kind: none, reason: "新しい機能領域を複数の作業項目に分けるときの最初の項目の選び方を判断し、採るなら開発文書に書くだけで、製品の HTTP 応答、永続状態、イベント、規範要素は変えない。" }
---

# 採否を判断したうえで、新しい機能領域の最初の作業項目を端から端まで動く最小の縦切りにする

## 動機

[作業項目のグループ](../../docs/development/work-item-groups.md)には、`admin-governance`、`tenant-configuration`、`agent-governance` のように、新しい機能領域を複数の未完了の項目で扱うグループがある。
`implement-work-item` は、一つの記録が独立して受け入れられる一つの結果を扱うと定めるが、新しい領域のどの項目から着手するかの基準は定めていない。
ドメインモデル、永続化、API、画面を層ごとに別の項目で先に作ると、利用者が通しで操作できるまで設計の誤りが見つからない。

AWS の AI-DLC は、作業を Unit に分けるときに、最初の Unit を **walking skeleton**（端から端まで動く最小の統合された縦切り）にする選択を持つ。
最初の Unit は実際の端から端までの検証コマンドを通し、利用者が承認してから後の Unit に進む。

優先度は、グループの基準どおり採否の判断から始める項目として p3 にする。

## 対象範囲

- 新しい機能領域を複数の作業項目に分けた過去の例（`work-items/done/`）を読み、最初の項目の後に設計を変えた例があったかを調べる。
- 調べた結果から採否を決める。
- 採るなら、新しい機能領域の最初の項目を、主要ユースケースの一つを `acceptance` または `e2e` の境界で通す最小の縦切りにする規則を、`work-item-groups.md` または `specification-first-workflow.md` の担当する節に書く。

## 対象外

- 既存の機能を補う項目。既存の機能がすでに端から端まで動いているので、縦切りの規則は当たらない。
- 縦切りの承認を機械で強制するゲート。

## 設計

出典: AWS AI-DLC のドキュメント「Phases and Stages」の Construction と「Glossary」の Walking skeleton の項（https://awslabs.github.io/aidlc-workflows/guide/）。本文を読んだ。調査日は 2026-11-28。

AI-DLC から採る点は、最初の縦切りを端から端まで実際に通し、利用者が確かめてから残りに進むことである。
採らない点は、Unit の依存グラフと Bolt による配送計画である。
IdMagic では作業項目と `depends_on` がその役割を持つ。

縦切りの完了条件は、新しい仕組みを作らず、既存の[主要ユースケースの証拠](../../docs/development/specification-first-workflow.md#主要ユースケースの証拠)で表せる見込みである。

## 計画

1. 過去の例を調べ、結果を設計に書く。
2. 採否を利用者と決める。
3. 採るなら規則を書く。採らないなら理由を要約に書いて中止する。

## タスク

- [ ] T001 [Evaluate] 新しい機能領域を複数の項目に分けた過去の例を調べる。
- [ ] T002 [Decide] 採否を決める。
- [ ] T003 [Docs] 採るなら担当する文書に規則を書く。
- [ ] T004 [Verify] 検査を通す。

## 検証

- `mise run check-links`: 文書間のリンクを確かめる。
- `mise run check-work-items`: 記録の形式を確かめる。

## リスク

最小の縦切りを先に作ると、後の項目で層ごとの設計をやり直すことがある。
その費用と、設計の誤りを早く見つける効果を、過去の例で比べてから決める。
