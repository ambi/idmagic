---
depends_on: []
status: completed
authors: [tn]
initial_context:
  specification: [docs/contexts/workloadidentity/internals.md, docs/contexts/provisioning/internals.md, docs/contexts/saml/internals.md, docs/contexts/api-tokens/internals.md]
  source: [SPECIFICATION_FORMAT.md, DOCUMENTATION_GUIDE.md]
  tests: []
  stop_before_reading: [backend, frontend, spec, work-items/done]
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

3 点とも着手時に確定した。**21 件すべてを読んだ結果、判定の結論は起票時の想定と逆になった。**

1. **順序は薄い側から読み、厚い側は見出しで確かめた。** 起票時は「厚い側を 1 件通して基準を言語化する」案に寄せていたが、実際には薄い 7 件を読んだ時点で基準が見えた。**薄いのは埋め草だからではなく、その Context の機構が 1 つしかないからだった。** `workloadidentity` の 8 行は「アテステーションは検証するのであって信用しない」という 1 つの保証で、それ以上書くことが無い。

2. **`decisions.md` へ移す項目には、既に理由が書かれていた。** 推測で理由を補う必要は生じなかった。移動と削除の対象は 4 件で、いずれも「理由つきの判断が `internals.md` に紛れている」か「`decisions.md` と同じことを二度書いている」かだった。

3. **落とした件数は 0 である。** 「コードを読めば分かる」と判定できたものが 1 件も無かった。迷った 1 件（`sourcing` の SCIM 属性対応表）は、多値の `emails` を 1 件へ射影する規則と、その拒否条件が対応表と一体で意味を持つため残した。Design の 3 が定めたとおり、迷ったら残す側に倒している。

### 判定の結果

| 扱い | 件数 | 内訳 |
|---|---|---|
| そのまま残す | 17 | 機構としてコードから復元できない |
| 一部を `decisions.md` へ移す・重複を削る | 4 | api-tokens、workloadidentity、saml、provisioning |
| 落とす | 0 | 該当なし |

移した内容は次のとおり。

| Context | 動かしたもの |
|---|---|
| api-tokens | 最終段落（`sub` の固定と、発行者がロールを失った場合）が `decisions.md` の 2 項目目と同内容だったので削除 |
| workloadidentity | 「同じ `subject` に複数の束縛が一致したら拒否する」を、理由つきの判断として `decisions.md` へ移動 |
| saml | 「対応範囲を Web Browser SSO Profile に限る」節が `decisions.md` の 4 項目目と同内容だったので削除。残った署名の再利用と `goxmldsig` の制約に見出しを付け直した |
| provisioning | 「プロトコル非依存の中核」節が `decisions.md` の 4・5 項目目と同内容だったので削除。イベントリレーを使わない理由も 6 項目目にあったので削除し、同一トランザクションでの捕捉という機構だけを残した |

### 規定の側を直した

**「ほとんどのコンテキストには不要である」は、この製品では成り立たない。** 21 件のうち 21 件が、フェイルクローズの拒否、鍵の寿命、リース、失効エポック、同一トランザクションでの捕捉のいずれかを説明していた。IdP はそういうものでできている。

この文は判定ではなく**頻度の予測**であり、予測が外れる領域では「件数を減らせ」という誤った圧力だけが残る。本 work item の Risk Notes が警告した「コードから復元できない機構の説明を落として、誰も気付かない」を、規定そのものが誘発する状態だった。

そこで [SPECIFICATION_FORMAT.md](../SPECIFICATION_FORMAT.md) §3 と [DOCUMENTATION_GUIDE.md](../DOCUMENTATION_GUIDE.md) §5.6 から頻度の主張を外し、**何件必要かは領域の性質であって割り当てではない**という規則に置き換えた。CRUD の形をした Context には要らず、フェイルクローズや鍵の寿命の上に建つ Context にはたいてい要る、と判定できる形にしてある。

これは Scope の拡張である。起票時の Scope は 21 件の棚卸しだけを挙げていたが、Motivation の題は「規定と実態を一致させる」であり、[[wi-405-spec-and-docs-boundary-is-not-legible]] と [[wi-407-name-the-directory-after-the-kind]] で確立したとおり、**一致させる先が規定の側であることはありうる。**

## Plan

- 最初の 1 件を通して、1 件あたりの手数と判定のぶれを測ってから、残りの順序を決める。
- Context 単位で完結させる。`internals.md` と `decisions.md` を同時に見ないと、移動先の判断ができない。
- 判定結果は Context ごとに本 work item へ追記する。まとめて書かない。中断しても、通した分の判定は残る。
- 「移すべきだが理由が書けない」が出たら、その項目は動かさず記録だけ残して先へ進む。

## Tasks

- [x] T001 [Design] 判定の順序、`decisions.md` へ移すときの理由の扱い、落とすときの確認手順を確定し `## Design` に記録した。
- [x] T002 [Spec] 薄い側から 1 件通し、1 件あたりの手数を測った。
- [x] T003 [Spec] 薄い 7 件を判定した。うち 2 件（workloadidentity、provisioning）に誤配置があり、処理した。
- [x] T004 [Spec] 残る 14 件を判定した。うち 2 件（api-tokens、saml）に重複があり、処理した。
- [x] T005 [Spec] 空になった `internals.md` は無かったので、索引の変更も無い。
- [x] T006 [Spec] 項目が 2 件以下の `decisions.md` 4 件を確認した。workloadidentity は 1 項目増えて 3 件になった。残る 3 件（api-tokens、claim-mapping、data-keys）は、書き漏れではなくその Context が実際に下した判断がそれだけだった。
- [x] T007 [Verify] `mise run verify` を通した。

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

## Completion

- **Completed At**: 2026-08-23
- **Summary**:
  21 件の `internals.md` を 1 件ずつ読み、17 件をそのまま残し、4 件から誤配置と重複を取り除き、**0 件を落とした。** 落とせなかったのではなく、「コードを読めば分かる」と判定できたものが 1 件も無かった。取り除いたのは、api-tokens と saml が `decisions.md` と同じことを二度書いていた箇所、provisioning が構成の判断を機構として書いていた箇所、workloadidentity が理由つきの判断を機構に紛れ込ませていた箇所（これは `decisions.md` へ移した）である。**そのうえで規定の側を直した。** 「ほとんどのコンテキストには不要である」は判定ではなく頻度の予測であり、21 件が 21 件ともフェイルクローズの拒否・鍵の寿命・リース・失効エポック・同一トランザクションでの捕捉を説明している製品では成り立たない。この文が残っていると「件数を減らせ」という圧力だけが働き、本 work item の Risk Notes が警告した「コードから復元できない説明を落として誰も気付かない」を規定自身が誘発する。SPECIFICATION_FORMAT.md §3 と DOCUMENTATION_GUIDE.md §5.6 から頻度の主張を外し、**何件必要かは領域の性質であって割り当てではない**という規則に置き換えた。
- **Verification Results**:
  - `mise run verify` - passed（exit 0）
  - `mise run spec-diff` - no normative specification change against main（`internals.md` と `decisions.md` は規範要素を持たない。`spec-diff` は [[wi-401-cross-context-scenarios-have-no-home]] で両方の木を読むよう修正済みで、今回は実際に読んだうえでの「変更なし」である）
  - `mise run check-spec` - ok 138 document(s)
  - 手動: 残した `internals.md` から `jobs`（リースと少なくとも 1 回の実行保証）を選び、この説明抜きでコードだけから修復方針を導けるかを確認 - 導けない。判定は妥当

## Left Undone

- **判定は 1 人が 1 度通しただけである。** DOCUMENTATION_GUIDE §14 が認めるとおり、この基準は機械検査できない。次に棚卸しする人が前回の判定を読めるよう Design に表を残したが、**判定そのものの正しさは担保されていない。**
- **`sourcing` の SCIM 属性対応表は判定を保留した。** 対応表そのものはコードから復元できるが、多値の `emails` を 1 件へ射影する規則と拒否条件が対応表と一体で意味を持つため残した。表だけを `standards.md` か TypeSpec へ移し、射影規則を `internals.md` に残す分け方はありうる。[[wi-403-provisioning-declares-no-scim-conformance]] が SCIM の規範宣言を扱うので、そこで再考する余地がある。
- **薄い `decisions.md` 3 件（api-tokens、claim-mapping、data-keys）は増やしていない。** 書き漏れではなくその Context が実際に下した判断がそれだけだと判定したが、**「判断が少ない」ことと「判断を書いていない」ことを外から区別する方法は無い。** 推測で項目を足すことはしなかった。
