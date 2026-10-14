---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-14
priority: p1
depends_on: []
change_kind: tooling
spec_impact:
  kind: none
  reason: "エージェントが着手時に読む開発文書の範囲と量を変える。製品の HTTP 応答、永続状態、イベント、TypeSpec と規範要件は維持する。"
---

# brief が読ませる開発文書の量を測り、変更に適用しない節を候補から外す

## 動機

`docs/development/` と `docs/formats/` は合わせて 313 KB ある。
2026-10-14 に、呼び出し元のない関数を一つ消すだけの [wi-66580](wi-66580-remove-the-unused-resolve-policy-for-tenant.md) へ `mise run brief -- wi-66580 --target backend/authentication` を実行すると、明示の候補だけで約 48 KB の節を読むよう示した。

| 候補 | バイト数 |
| --- | --- |
| `specification-first-workflow.md#3-開発サイクル` | 15,511 |
| `specification-first-workflow.md#4-証拠の要件` | 10,843 |
| `specification-first-workflow.md#仕様影響の宣言` | 4,334 |
| `testing.md#主要ユースケース` と `#テストダブル` | 8,476 |
| `work-item-format.md#着手時の文脈と設計` | 3,758 |
| そのほかの明示の候補 | 約 5,700 |

推定の候補には `spec/modules/authentication/main.tsp` と `models.tsp` の全体（150 KB）も含まれる。
`spec_impact: none` の refactor では、主要ユースケース、TypeSpec、証拠の大半の子節は適用しない。
候補の出力にサイズがないため、読み手は量を知らずに開き、作者は文書が増えたことに気付けない。

## 対象範囲

- `brief` が候補ごとのバイト数と合計を表示する。
- AGENTS.md の参照表の条件を見直し、`spec_impact: none` の変更や refactor に適用しない行を候補から外す。
- `specification-first-workflow.md` の 3 節と 4 節を、変更の種類ごとに読む子節へ分け、参照表から子節を指す。
- `spec_impact: none` の変更では、TypeSpec のファイル全体を推定の候補にしない。
- 分割の過程で見つけた重複を削り、`docs/development/` と `docs/formats/` の合計を純減させる。

## 対象外

- 規則そのものの追加と変更。読む範囲と書き方だけを変える。
- `format-rationale.md` と `documentation-guide.md` の全面的な書き直し。前者は既に通常の参照経路の外にある。
- 合計のバイト数を検査で強制すること。上限の数値は計測を重ねてから決める。

## 設計

量を減らす手段は、文書の短縮より先に、適用しない節を候補から外すことに置く。
本文を短くしても、適用しない節を読ませる限り費用は残るためである。
条件の書き方は[作業文脈の導出](../../docs/development/task-context.md#一次情報からの導出)の `<!-- context: -->` を使い、新しい条件のキーは足さない。
`spec_impact` による絞り込みが既存のキーで表せない場合に限り、`kinds` の値として扱う方法を先に検討する。

採らない案:

- `brief` が節の本文を要約して返す案。本文を連結しない現在の方針に反し、要約と一次情報の食い違いを生む。
- バイト数の上限を検査で強制する案。適切な上限がまだ分からず、数値のために規則を削る圧力になる。

## 計画

1. 代表的な 3 件（本番コードの refactor、feature、tools の変更）で、現在の候補と合計のバイト数を記録する。
2. `brief` にバイト数の表示を加える。
3. 参照表の条件と、3 節と 4 節の分割を変え、同じ 3 件で再計測する。
4. 推定の TypeSpec 候補の条件を変える。

## タスク

- [ ] T001 [Tooling] `brief` が候補ごとのバイト数と合計を表示する。RED を先に確かめる。
- [ ] T002 [Docs] 3 節と 4 節を変更の種類ごとの子節へ分け、重複を削る。
- [ ] T003 [Docs] AGENTS.md の参照表の条件を、分けた子節へ向ける。
- [ ] T004 [Tooling] `spec_impact: none` では TypeSpec 全体を推定の候補にしない。
- [ ] T005 [Verify] 3 件の前後の合計と、開発文書の合計バイト数の純減を記録する。

## 検証

- `mise run test-tools-file` で `brief` の表示と条件の判定を確かめる。
- 3 件の代表で、候補の合計が減り、各変更に適用する節が候補に残ることを確かめる。
- `mise run check-links` と `mise run check-work-items` を通す。

## リスク

適用する節まで候補から外し、エージェントが規則を読まずに作業する。
代表の 3 件で、外した節がその変更に適用しないことを一つずつ確かめ、判断を完了節に残す。
