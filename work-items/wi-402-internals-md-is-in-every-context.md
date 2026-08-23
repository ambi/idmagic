---
depends_on: []
status: pending
authors: [tn]
risk: low
created_at: 2026-08-23
priority: p3
change_kind: docs
spec_impact: { kind: none, reason: "internals.md と decisions.md の棚卸しであり、規範的な要素には触れない。この 2 種は REQ-* の受け入れシナリオも規範 ID も TypeSpec symbol も持たないため、affected_spec が索引する対象が無い。判断や機構の説明が他の正規文書へ移る場合も、移る先は decisions.md であって規範側ではない。" }
---

# `internals.md` 21 件を棚卸しし、「rare」という規定と実態を一致させる

## Motivation

[SPECIFICATION_FORMAT.md](../SPECIFICATION_FORMAT.md) §3 は `internals.md` についてこう書いている。

> Write this only when the working of a mechanism cannot be recovered from the code. **Most contexts do not need it.**

[DOCUMENTATION_GUIDE.md](../DOCUMENTATION_GUIDE.md) §5.6 も「稀。機構の説明が要るときだけ」「ほとんどのコンテキストには不要である」と繰り返す。

**21 Context のうち 21 件が `internals.md` を持っている。** 例外は 1 件も無い。「most do not need it」と書かれた文書の種類が全数に存在する状態は、判定が働いていないことを意味する。ファイル集合が定型として埋められた疑いがある。

行数の分布（2026-08-23 実測）はその疑いを裏づける。

| 行数 | Context |
|---:|---|
| 8 | workloadidentity |
| 11 | audit, provisioning |
| 13 | api-tokens, identity-governance, sharedsignals |
| 15 | claim-mapping |
| 19 | data-keys |
| 21–35 | system, signing-keys, seeding, jobs, sourcing, saml, ws-federation |
| 41–93 | application, authorization, tenancy, identity-management, oauth2 |
| 128 | authentication |

下の 7 件は見出し 1 つと段落 1〜2 つしかない。`workloadidentity/internals.md` の 8 行は「アテステーションは検証するのであって信用しない」「複数の binding が一致したら拒否する」であり、**後者は理由つきの判断そのもの**で、§3 の言う「decisions と mechanism は寿命が違う」の decisions の側に属する。上の 5 件（authorization 以上）は、機構の説明として読める分量がある。

同じ棚卸しが `decisions.md` の側にも要る。**項目が 2 件しかない Context が 4 件ある**（api-tokens、claim-mapping、data-keys、workloadidentity、いずれも 4 行）。薄い `decisions.md` と薄い `internals.md` が同じ Context に並んでいるのは、置き場所の判定が曖昧なまま両方に少しずつ書かれた形に見える。

これは検査で守れない。§3 の判定基準——「この機構が壊れたとき、コードだけを読んで正しい直し方が分かるか」——は人しか判定できず、DOCUMENTATION_GUIDE §14 も「自動検査できない。レビューの目に頼る」と認めている。**だから定期的に人が通す以外に方法が無く、いま誰も通していない。**

## Scope

- 21 件の `internals.md` を 1 件ずつ §3 の基準で判定し、次のいずれかに解決する。
  - 機構の説明としてコードから復元できない → 残す。
  - 理由つきの判断である → `decisions.md` へ移す。
  - コードを読めば分かる、または他の正規文書（`states.md`、`scenarios.md`、`standards.md`、TypeSpec、スキーマ）が既に持っている → 落とす。
  - ファイルが空になったら削除し、その Context の `README.md` の索引から行を外す。
- 項目が 2 件以下の `decisions.md` 4 件について、判断が書き漏れているのか、本当にそれだけなのかを確かめる。
- 判定の結果を本 work item に記録する。次に棚卸しする人が、前回どう判定したかを読めるようにする。

## Out of Scope

- 内容の書き直し。残すと判定したものの推敲は行わない。移動と削除だけを扱う。
- 上位 5 件（authentication、oauth2、identity-management、tenancy、authorization）の分割。分量があること自体は問題ではない。§5.8 は「行数に上限は置かない。難しい設計は長い」と明記している。
- 新しい検査の追加。「rare」は機械判定できない。判定を機械へ渡そうとすると、行数の下限のような無意味な基準になる。
- `README.md` の索引と実ファイルの整合検査。**この整合は既に取れている**（21 Context × 6 種のファイルについてドリフト 0 件を確認済み）ので、守るものが無い。

## Design

未定。着手時に次の 3 点を確定して本節に記録する。

1. **判定の順序。** 薄い 7 件から始めれば「落とす/移す」の判断が多く、削除の効果が早く出る。一方、厚い 5 件から始めれば「何が残るべき internals なのか」の基準が先に固まり、薄い側の判定がぶれにくい。**厚い側を 1 件通して基準を言語化してから、薄い側へ降りるほうが合う見込みが高い。** ただし厚い側 1 件（authentication の 128 行）を読む手数は薄い 7 件の合計より重い。最初の 1 件でどちらが効くかを測る。

2. **`decisions.md` へ移す判断の書き方。** 移した項目には理由が要る（§3「An item with no reason is a restated rule, not a decision」）。`internals.md` は保証の側から書かれているので、**理由がそもそも書かれていない項目がある。** その場合、理由を推測して書くのではなく、書けないことを記録して残す。推測した理由は、後から読む人が事実として扱う。

3. **落とすときの確認。** 「コードを読めば分かる」の判定は、実際にコードを読んで確かめる。読まずに「分かるはずだ」で落とすと、コードから復元できない機構の説明を消すことになる。**この確認をやる余裕が無いなら、その件は落とさず残す。** 迷ったら残す側に倒す。

## Plan

- 最初の 1 件を通して、1 件あたりの手数と判定のぶれを測ってから、残りの順序を決める。
- Context 単位で完結させる。`internals.md` と `decisions.md` を同時に見ないと、移動先の判断ができない。
- 判定結果は Context ごとに本 work item へ追記する。まとめて書かない。中断しても、通した分の判定は残る。
- 「移すべきだが理由が書けない」が出たら、その項目は動かさず記録だけ残して先へ進む。

## Tasks

- [ ] T001 [Design] 判定の順序、`decisions.md` へ移すときの理由の扱い、落とすときの確認手順を確定し `## Design` に記録する。
- [ ] T002 [Spec] 最初の 1 Context を通し、1 件あたりの手数を測る。
- [ ] T003 [Spec] 薄い 7 件（workloadidentity、audit、provisioning、api-tokens、identity-governance、sharedsignals、claim-mapping）を判定して処理する。
- [ ] T004 [Spec] 残る 14 件を判定して処理する。
- [ ] T005 [Spec] 空になった `internals.md` を削除し、対応する `README.md` の索引から行を外す。
- [ ] T006 [Spec] 項目が 2 件以下の `decisions.md` 4 件を確認する。
- [ ] T007 [Verify] `mise run check-spec`、`mise run spec-render`、`mise run verify` を通す。

## Verification

- `mise run check-spec`
  - reason: ファイルを削除した Context が、正規文書の集合として妥当なままであることを確かめる。
- `mise run spec-render`
  - reason: 生成サイトが、削除した `internals.md` への導線を残していないことを確かめる。索引を外し忘れると、ここで壊れたリンクとして出る。
- `mise run spec-diff`
  - reason: 移動と削除で仕様がどう動いたかを、記憶ではなく差分から読む。
- `mise run verify`
- 手動: 残すと判定した `internals.md` から 1 件選び、その機構を知らない状態でコードだけを読んで直せるかを自問する。直せるなら、その判定は誤りである。

## Risk Notes

リスクは low。文書の移動と削除であり、製品にもテストにも触れない。

**ただし、この作業には静かに損をする形がある。** コードから復元できない機構の説明を「コードを読めば分かる」と誤判定して落とすと、失ったことに誰も気付かない。気付くのは、その機構が壊れて直せなくなったときである。検査は何も言わない。だから Design の 3 で「迷ったら残す」を先に決めておく。**削除件数を成果の指標にしない。** 21 件のうち 0 件しか落ちなくても、1 件ずつ判定を通したこと自体が成果である。

逆の失敗もある。判定を「全部残す」で流すと、この work item は何もしなかったのと同じになる。判定結果を Context ごとに記録することが、流していないことの唯一の証拠になる。

`decisions.md` へ移した項目に、推測した理由を書かないこと。理由の無い判断より、理由を偽装した判断のほうが悪い。
