---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-11-25
priority: p2
group: development-process
depends_on: []
change_kind: tooling
spec_impact: { kind: none, reason: "文書の差分を意味の単位で読むレビューの skill を足すだけで、製品の HTTP 応答、永続状態、イベント、規範要素は変えない。" }
---

# 文書の差分から変わった主張を取り出し、ほかの一次情報源との矛盾と重複をレビューする

## 動機

このリポジトリの変更の多くは文書である。
2026-10-25 から 2026-11-25 までの 305 件のコミットのうち 206 件が `docs/`、`.agents/`、`AGENTS.md` のいずれかを変えた。
その大半はエージェントが書いている。

文書の変更に効く機械の検査は、形と規範要素を確かめる。
`check-links` はリンク、`check-terminology` は採らない表記、`check-unspecified-vocabulary` は定義のない語、`check-spec-impact` と `spec-diff` は規範シナリオと TypeSpec の差分を扱う。
どれも、書き換えた文がほかの文書の主張と食い違うこと、別の文書が一次情報源として持つ規則を二度書いたこと、未完了の作業項目の対象範囲や対象外の意味が変わったことは検出しない。
[一次情報源の割り当て](../../docs/formats/documentation-guide.md#一次情報源の割り当て)は要件を一度だけ書くと定めるが、守られているかは書いた本人の注意に頼っている。

テキストの差分は、言い回しと整形の変更の中に意味の変更を埋もれさせる。
Thoughtworks Technology Radar Vol.35（2026-11）は、LLM が変わった事実、矛盾する記述、変わった受け入れ基準、範囲の変更を差分から取り出す **semantic review of AI-edited prose** を Assess に、検査を通らない大量の文書が矛盾したまま積もる AI-driven documentation sprawl を Caution に置いた。

## 対象範囲

- repo skill `review-document-change` を `.agents/skills/` に足す。入力は差分の起点で、手順は次のとおりにする。
  1. 差分から、変わった主張（規則、数値、条件、判断と理由、対象範囲と対象外、`affected_spec`）を一件ずつ取り出す。言い回しと整形だけの変更は除く。
  2. 主張ごとに主題の語で `rg` を実行し、ほかの文書の記述と食い違うもの、一次情報源の割り当てに反して同じ規則を二度書いたものを挙げる。
  3. 未完了の作業項目の変更では、対象範囲、対象外、設計の判断の意味がどう変わったかを挙げる。
  4. 結果は会話に報告し、記録には書かない。
- `implement-work-item` の手順 7 に、差分が `docs/`、`AGENTS.md`、`.agents/`、`work-items/active/` を含むとき、この skill で差分を読む手順を足す。差分を書いたエージェントとは別の新しいセッションで行う。
- 採否の試行を、採用の前に行う。直近の文書の変更を含むコミット 10 件に適用し、機械の検査が通したのに実際に直すべきだった矛盾か重複を 1 件以上見つけたときに採る。見つけなければ skill と手順を取り除き、中止として要約を書く。

## 対象外

- CI の検査としての実行。LLM の判定は同じ入力でも結果が揺れ、[リポジトリ検査の判定基準](../../tools/check/README.md#検査の判定基準)が求める機械での確実な判定にならない。
- 見つけた矛盾の自動の修正。直す側の文書を選ぶのは一次情報源の割り当ての判断である。
- コードのレビュー。利用者の環境の `code-review` skill が扱う。
- 既存の文書全体の矛盾の棚卸し。この項目は差分だけを読む。

## 設計

| 案 | 採否 | 理由 |
| --- | --- | --- |
| 差分を読む repo skill | 採る | 主張の取り出しと矛盾の判断は文脈を読む必要があり、文章の規則として書ける。どのエージェントでも使える |
| `check-terminology` の規則を増やす | 採らない | 部分文字列の一致なので、主張の食い違いを表せない |
| 利用者の環境の `code-review` skill に文書の軸を足す | 採らない | リポジトリの外にあり、ほかの作業者とエージェントに共有されない |
| 同じ文を探す機械の重複検出 | 採らない | 同じ規則を別の言い回しで書いた重複を見つけられず、引用の正当な重複を誤検出する |

別のセッションで読むのは、書いたエージェントが自分の意図で差分を補って読むためである。
wi-11469 でも、skill の変更前後を別のサブエージェントで比べて確かめた。

### 出典

| 出典 | 読んだ範囲 | 調査日 | 使う知見 |
| --- | --- | --- | --- |
| Thoughtworks Technology Radar Vol.35、Techniques（thoughtworks.com/radar/techniques） | 本文 | 2026-11-25 | semantic review of AI-edited prose（Assess）は、文書を編集するたびに skill として動かし、変わった事実、矛盾、受け入れ基準、範囲を取り出す。AI-driven documentation sprawl（Caution）は、検査を通らない文書が矛盾したまま積もる |

## 計画

1. 試行の対象として、直近の文書の変更を含むコミット 10 件を選ぶ。作業項目の起票、開発文書、設計文書をそれぞれ含める。
2. skill を書き、別のセッションで 10 件に適用して、見つけたものが実際に直すべきものかを判断する。
3. 採るなら `implement-work-item` に手順を足す。採らないなら skill を取り除いて中止する。

## タスク

- [ ] T001 [Skills] `review-document-change` を書く。
- [ ] T002 [Trial] 10 件のコミットに適用し、採否を決める。
- [ ] T003 [Skills] 採る場合、`implement-work-item` の手順 7 に手順を足す。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-agent-guidance`：新しい skill と変更した skill が呼ぶタスクの宣言と参照を確かめる。
- `mise run check-links` と `mise run check-terminology`：skill と開発文書のリンクと表記を確かめる。

## リスク

報告が誤検出ばかりだと、読まれない手順が毎回の作業に増える。
採否の試行で、直すべきものを見つけた件数と、誤検出の件数の両方を見て決める。
