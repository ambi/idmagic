---
status: in_progress
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-17
priority: p2
depends_on: [wi-67249-render-ears-requirements-like-gherkin]
change_kind: tooling
spec_impact:
  kind: none
  reason: "生成サイトでの要件文の名札と主体の表示だけを変え、要件の文言、規範 ID、TypeSpec の契約、製品の振る舞い、永続状態、イベント、外向きの呼び出しは変更しない。"
---

# 生成サイトの要件文の表示を EARS の型に合わせる

## 動機

[wi-67249](../done/wi-67249-render-ears-requirements-like-gherkin.md) は、要件文の応答の節に Gherkin のキーワードの Then を付け、主体を表示から外した。
[EARS の型](https://alistairmavin.com/ears/)（Alistair Mavin）と比べると、次の三点が合わない。

- 応答は「the 〈system name〉 shall 〈system response〉」の Shall で示す。Then は Gherkin のキーワードである。
- EARS は要件に主体（system name）を一つ求める。主体を表示から外すと、EARS の型を満たさない。
- 望まない振る舞いは「If 〈trigger〉, then the 〈system name〉 shall …」であり、If には Then が対になる。

前置きの名札（Where、While、When、If）と節の順序は EARS と合っている。

## 対象範囲

- 応答の節の名札を Shall にする。色は受け入れ例の Then と同じにする。
- 応答の節は主体の「〈主体〉は、」から始め、主体を表示に残す。表示から外すのは前置きの区切りの語と読点だけにする。
- 前置きに If がある要件は、応答の節に Then と Shall の二つの名札を付ける。Then は If と同じ色にする。

## 対象外

- 原稿の書き方と、EARS の構文の検査は変えない。

## 設計

キーワードの種類を EARS の語（`where`、`while`、`when`、`if`、`then`、`shall`）にそろえ、一つの節が複数の名札を持てるようにする。
名札、CSS のクラス、テストを同じ語で書く。

## 計画

1. 節の読み取りと名札を改め、テストを直す。
2. 生成サイトで確かめる。

## タスク

- [x] T001 [Tooling] 応答の名札を Shall にし、主体を残し、If に Then を対にする。
- [x] T002 [Verify] 生成サイトとツールのテストで確かめる。

## 検証

- `mise run test-tools`
- `mise run check -- main`

## リスク

- 名札の文字と節の切れ目だけの変更であり、表示の崩れは生成サイトのテストで検出する。

## 要約

`mise run spec-diff` は main に対して規範仕様の変更なしと報告する。
生成サイトの要件文の表示を EARS の型に合わせた。応答の名札を Then から Shall にし、主体を表示に残し、望まない振る舞いの型では If と対になる Then を Shall の前に付ける。
前置きの名札と節の順序は EARS と合っていたので変えていない。
