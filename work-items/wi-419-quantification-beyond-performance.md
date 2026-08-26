---
depends_on: []
status: pending
authors: [tn]
risk: medium
created_at: 2026-08-27
priority: p2
change_kind: docs
affected_spec:
  - { path: docs/capacity.md, requirement: SLO-PRIMARY-ERRORS }
  - { path: docs/capacity.md, requirement: SLO-TOKEN-AVAILABILITY }
---

# 量化の範囲を性能の外へ広げ、設計判断を定量的に比較できるようにする

## Motivation

`docs/capacity.md` は、名前を挙げていないだけで Competitive Engineering の核心を既に持っている。Specification target と Planning assumption と Measurement を区別し、「実測値が Planning assumption を下回っても Specification target を暗黙に引き下げない」と定め、目標値を書く前に測定境界（母集団、除外条件、時間窓）を定義し、`SLO-*` と `CAP-*` に安定 ID を与えて他の文書は数値を再掲せず ID を参照する。ここまで規律のある量化はめずらしい。

問題は範囲と使い道である。

第一に、量化されているのは遅延、非 5xx 比率、可用性、スループット、容量だけである。保守性、学習容易性、運用容易性、移植性、セキュリティには尺度が無い。`docs/standards.md` は WCAG を `required` かつ `MUST` と書くが、それは二値の宣言であって尺度ではなく、達成度を測る方法が無い。

第二に、目標が 1 個の数字であり、「これを割ったら出荷しない下限」と「狙う値」と「あれば望ましい値」が分かれていない。error budget は登場するが、それは下限と計画値の区別ではない。

第三に、`DOCUMENTATION_GUIDE.md` は自ら「budget を使い切ったときに何をするかを、目標値と同じ場所に書く。決めていないと、error budget は達成率の飾りになる」と定めているのに、`docs/capacity.md` にその記述が無い。SLO も burn rate のアラートも runbook 8 本も揃っていて、予算を使い切ったときの行動だけが欠けている。自分のガイドを自分が満たしていない。

第四に、設計判断を定量的に比較する場が無い。`WORK_ITEM_FORMAT.md` の `Design` は「選んだ設計、検討事項、却下した代替案」を散文で書かせるだけで、各案が各品質目標へ与える影響を見積もる表が無い。`SLO-*` と `CAP-*` という ID 体系が既にあるため、この表は追加の定義をほとんど必要とせずに作れる。

## Scope

- **error budget policy**：`docs/capacity.md` に、予算の消費が進んだときと使い切ったときに何をするかを書く。対象は目標ごと、行動は具体的に。
- **目標の段階**：`SLO-*` と `CAP-*` に、下限と計画値の区別を導入するかを判断し、導入するなら表の列として加える。
- **性能以外の品質属性**：保守性、運用容易性、学習容易性、移植性、セキュリティのうち、意味のある尺度と測定方法を定義できるものを選び、`capacity.md` と同じ Evidence classes の枠組みで書く。
- **Impact Estimation**：`WORK_ITEM_FORMAT.md` の `Design` に、代替案と品質目標 ID の交差表を書く形式を加える。medium 以上の変更で必須とし、low では任意とする。
- **既存 work item への遡及なし**：形式の追加は以後の work item にのみ適用する。

## Out of Scope

- 新しい SLO の数値そのものの設定。本 work item は枠組みを整えるところまでを担い、個々の目標値は対象の Context を持つ work item が決める。
- Measurement の取得。`docs/capacity.md` が述べるとおり本書にはまだ実測が無く、ステージングでの容量検証は別の作業である。
- 価値の増分を短い周期で届けて毎回測るという意味での evolutionary delivery。work item は意味上の変更の単位であって価値と測定の単位ではないため、その転換は本件の範囲を超える。
- DORA の 4 指標。開発プロセス自体の量化は wi-423 が扱う。

## Design

error budget policy の書き場所は `docs/capacity.md` の Service level objectives の直後とする。`DOCUMENTATION_GUIDE.md` が「目標値と同じ場所に書く」と指定しており、別ファイルへ切り出すと、目標を読んだ人が行動を読まずに済んでしまう。

目標の段階については、Planguage の Must / Plan / Wish をそのまま持ち込む案と、下限と計画値の 2 段階に留める案がある。採るのは後者である。この製品の目標は外部への約束（SLA）を持たない現状では、Wish は「あれば望ましい」を書く欄になり、書かれた値が何も左右しない。2 段階であれば、下限は縮退と出荷判断に直結し、計画値は容量算出の入力になる。段階を増やすのは、SLA が生じたときに再開する。

性能以外の品質属性については、量化できないものを無理に数値化しないことを原則とする。Competitive Engineering の主張は「すべてを量化せよ」だが、測定方法を定義できない数値は Planning assumption ですらなく、単なる願望になる。候補として実際に定義できそうなのは次のものである。運用容易性については、runbook を持つアラートの割合と、一次対応で解決できた事象の割合。移植性については、`EnvelopeCrypto` のような差し替え可能なポートについて、提供元を入れ替えたときに変更が必要なパッケージ数。学習容易性については、`initial_context` に挙がったファイル数の分布。いずれも `docs/capacity.md` の Evidence classes（Specification target、Planning assumption、Measurement）で区分する。保守性とセキュリティは、意味のある尺度を定義できるかを T002 で判断し、できなければ書かない。

Impact Estimation の表の形式は、行に代替案、列に品質目標 ID、セルに影響の見積もりと根拠を置く。セルの値を「+3」のような無次元の点数にする案は採らない。点数は根拠を失った状態で合計され、合計が判断を代行してしまう。セルには目標 ID の単位そのもの（ミリ秒、比率、レプリカ数）と、それが見積もりであることを記す。

`WORK_ITEM_FORMAT.md` への追加は、`Design` の中の任意の節ではなく、medium 以上で必須の記載事項として書く。同文書は既に「medium 以上の変更では `Design` と `Plan` を具体的にする」と述べており、その具体化の中身として位置づける。

## Plan

1. `docs/capacity.md` の既存の目標それぞれについて、予算の消費が進んだときの行動を書けるかを確認する。書けない目標があれば、それはアラートと runbook の欠落を示している。
2. error budget policy を書く。既存の runbook 8 本との対応を取り、行動が runbook を指す形にする。
3. 目標の段階を 2 段階にするかを判断し、するなら表へ列を足す。既存の値がどちらに当たるかを決める。
4. 性能以外の品質属性のうち、測定方法を定義できるものを選ぶ。定義できないものは書かない理由とともに落とす。
5. Impact Estimation の形式を `WORK_ITEM_FORMAT.md` へ加え、既存の work item のうち 1 件で試し書きして、形式が実用に耐えるかを確かめる。

## Tasks

- [ ] T001 [Spec] 既存の目標ごとに予算消費時の行動を洗い出し、runbook との対応を確認する。
- [ ] T002 [Spec] 性能以外の品質属性のうち、測定方法を定義できるものを選定する。
- [ ] T003 [Spec] `docs/capacity.md` に error budget policy を書く。
- [ ] T004 [Spec] 目標の段階の要否を判断し、必要なら表へ列を足す。
- [ ] T005 [Spec] 選定した品質属性を Evidence classes の枠組みで `docs/capacity.md` へ加える。
- [ ] T006 [Spec] `WORK_ITEM_FORMAT.md` へ Impact Estimation の形式を加え、既存 work item 1 件で試し書きする。
- [ ] T007 [Verify] `mise run check-slo-references` と `mise run check-spec` を通す。

## Verification

- `mise run check-slo-references` が、加えた項目と既存のアラート定義の対応を保ったまま通る。
- `docs/capacity.md` の全ての `SLO-*` について、予算を使い切ったときの行動が書かれている。
- 加えた品質属性の各行が、測定方法と Evidence class を持つ。
- `mise run check-work-items` が、Impact Estimation を追加した形式で通る。
- `mise run verify`

## Risk Notes

測定方法の無い品質目標を書くと、`docs/capacity.md` が持っている最大の美点、すなわち「数値の出どころが区分されている」ことが薄まる。定義できない属性は書かないという判断を、書かない理由とともに残す。

Impact Estimation を medium 以上で必須にすると、書く手間が増える。表が儀式になって全セルが「影響なし」で埋まる状態が最も悪い。試し書き（T006）で、実際の判断に効いた例を 1 つ作れるかを確認し、作れなければ必須化をやめて任意の形式として置く。

目標の段階を 2 段階にすると、既存の値がどちらなのかを決め直す必要がある。ここで既存の Specification target を計画値へ格下げすると、`docs/capacity.md` 自身が禁じている「暗黙の引き下げ」になる。既存の値はすべて下限として扱い、計画値は新たに置く。
