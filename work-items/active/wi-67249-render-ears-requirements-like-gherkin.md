---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-17
priority: p2
depends_on: [wi-38847-make-module-readme-an-overview]
change_kind: tooling
spec_impact:
  kind: none
  reason: "生成サイトでの要件文の表示だけを変え、要件の文言、規範 ID、TypeSpec の契約、製品の振る舞い、永続状態、イベント、外向きの呼び出しは変更しない。"
---

# 生成サイトで EARS の要件文を、受け入れ例の Gherkin と同じように構文で色分けして表示する

## 動機

機能仕様の要件は、日本語の EARS の一文を箇条書きで並べる。
どの文も「〈前提〉場合、〈主体〉は、〈応答〉。」の形で主体を繰り返すので、条件と応答の対応が文に埋もれ、生成サイトでも原稿と同じ箇条書きとして読みにくい。
受け入れ例の Gherkin は、生成サイトが `Given`、`When`、`Then` のキーワードを意味に応じて色分けし、読み手は段の役割を一目で区別できる。

## 対象範囲

- 生成サイトの機能仕様のページで、要件の箇条を EARS の型（常時、事象、状態、任意の機能、望ましくない事象）として読み、前提の部分、主体、応答を、Gherkin のキーワードと同じ見た目の規則で区別して表示する。
- 原稿の書き方は変えない。

## 対象外

- 要件の原稿の書式と、EARS の構文の検査は変えない。

## 設計

実装の前に、[仕様フォーマットの EARS 形式](../../docs/formats/specification-format.md#ears-形式)の型ごとの日本語の目印と、生成サイトの Gherkin の色分けの仕組みを読み、表示の形を決める。

## 計画

1. EARS の型の目印を読み取る関数と、その表示を生成サイトに足す。
2. 生成サイトのテストで型ごとの表示を確かめる。

## タスク

- [ ] T001 [Tooling] 要件の箇条を EARS の型として表示する。
- [ ] T002 [Verify] 生成サイトで確かめる。

## 検証

- `mise run test-tools-file -- render-docs/src/render.test.ts`
- `mise run check-rendered-docs`

## リスク

- 目印の読み取りを誤ると、一部の要件だけ色分けが崩れる。読み取れない文は色分けせず原文のまま表示する。
