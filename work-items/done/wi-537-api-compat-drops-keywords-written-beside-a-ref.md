---
depends_on: []
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-12
priority: p1
change_kind: tooling
spec_impact: { kind: none, reason: "検査器 tools/check の突き合わせ規則の欠陥であり、製品の振る舞いも規範文書も変えない。" }
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 変更は API 互換を検査する側にあり、製品にも規範文書にも届かない。リリースの読み手に見える差分が無い。
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - tools/check/src/api-compat.ts
    - tools/check/src/registry.ts
  tests:
    - tools/check/src/api-compat.test.ts
  stop_before_reading:
    - backend
    - frontend
    - docs
---

# `$ref` の傍らに書かれた keyword を API 互換検査が読み落とす

## Motivation

`check-api-compat` は `$ref` を解決するとき、**同じスキーマオブジェクトに並んで書かれた他の keyword を捨てる**。
`resolve()` は `$ref` を見つけると解決先をそのまま返し、`default` も `description` も持ち帰らない。

OpenAPI 3.1 は JSON Schema 2020-12 であり、**`$ref` の傍らの keyword は適用される**。無視されたのは 3.0 までの
規則である。生成器はこの形を実際に出力しており、生成された OpenAPI には `$ref` の傍らに `description` が 89 件、
`default` が 30 件ある。検査器はそのすべてを見ていない。

誤りは 2 方向に出る。実測した。

- **偽陽性。** inline な `type: string` + `default` を `$ref` + `default` へ変えると、default は同じ値のまま
  なのに `default value changed from "a" to undefined` を報告する。
- **偽陰性。** `$ref` の傍らの `default` が `"a"` から `"b"` へ変わっても、報告は**ゼロ件**である。baseline 側も
  現行側も解決先だけを見るので、どちらも `default` を持たないことになり、差が消える。

**重いのは偽陰性である。** この検査器はリリース baseline に対する破壊的変更を止めるための唯一のゲートであり、
`default` の変更は検査器自身が破壊的と宣言している変更の 1 つである。`$ref` を持つ項目については、その宣言が
効いていない。

[[wi-536-group-display-name-source-selects-nothing]] が `display_name_source` を列挙にしたときに、偽陽性の側が
5 件の報告として現れて見つかった。baseline を更新すれば報告は消えるが、それは偽陽性を凍結したうえで偽陰性を
温存する操作なので、検査器を直す。

## Scope

- `resolve()` が `$ref` の傍らの keyword を解決先へ重ねる。同じ keyword は傍らの側が勝つ。
- `required` は和を取る。2020-12 では解決先と傍らの両方が適用されるので、上書きにすると解決先が要求する
  項目が消える。
- `properties` は名前ごとに重ねる。同じ名前は傍らの側が勝つ。
- 循環を検出して展開を打ち切る場合も、傍らの keyword は返す。打ち切りは解決先を見ないという判断であって、
  傍らに書かれていることまで捨てる理由にはならない。

## Out of Scope

- `enum` の交差。2020-12 では解決先と傍らの `enum` は両方が適用されるので実効値は積集合だが、この検査器が
  読むのは「baseline の値が現行に残っているか」だけであり、傍らの値を採ればその問いには答えられる。
- `allOf` の合成。今の検査器は `allOf` を展開しない。`$ref` の傍らの keyword とは別の話である。
- `description` の差分を報告すること。文章の変更は破壊的変更ではない。重ねるが、比べない。
- `spec/idmagic.openapi.baseline.json` の更新。
- [[wi-536-group-display-name-source-selects-nothing]] の実装そのもの。

## Design

`resolve(schema, components, seen)` の戻り値を、**解決先に傍らの keyword を重ねたスキーマ**にする。
`$ref` 自身は重ねない。

```ts
resolve(schema) =
  schema に $ref が無ければ schema
  循環していれば 傍ら(schema)
  それ以外は merge(resolve(解決先), 傍ら(schema))
```

`merge` は keyword ごとに規則が違う。`required` は和、`properties` は名前ごとの重ね、それ以外は傍らが勝つ。
この 3 つで、検査器が読む keyword (`type`、`default`、`required`、`properties`、`enum`、`oneOf`、`items`) を
すべて覆う。

## Plan

1. 偽陰性と偽陽性を 1 件ずつ RED で固定する。
2. `resolve()` を重ねる実装に変える。
3. `required` と `properties` の重ね方を、それぞれの RED で固定する。
4. 生成された OpenAPI と baseline に対して `check-api-compat` を通す。

## Tasks

- [x] T001 [Unit RED] `$ref` の傍らの `default` が変わっても報告が出ないこと（偽陰性）と、inline から
  `$ref` へ移しても `default` が同じなら報告が出ないこと（偽陽性）を固定する。`type` の変更が傍らの
  `properties` ごと消えることも合わせて 4 件を RED にした。recipe: `mise run test-tools`（38 ファイル
  473 件で 1.4 秒なので、1 ファイルだけを走らせる recipe は足さない）。
- [x] T002 [Impl] `resolve()` が傍らの keyword を重ねる。
- [x] T003 [Unit RED] `required` の和と `properties` の重ねを固定する。この 2 件は T001 の 4 件では
  区別が付かない —— 素朴な `{...resolved, ...beside}` でも 4 件は通る。故障注入で RED を観測した
  （Change-Resistance Results の M2、M3、M4）。
- [x] T004 [Verify] `mise run check-api-compat` と `mise run verify`。

## Verification

- `$ref` の傍らの `default` の変更が報告される。
- inline から `$ref` へ移しただけの項目は報告されない。
- `$ref` の解決先が要求する項目が、傍らの `required` によって消えない。
- `mise run check-api-compat`
- `mise run verify`

## Risk Notes

- **偽陽性だけを直す。** 報告が消えたことを直った証拠にすると、偽陰性が残る。先に偽陰性を RED にする。
- **傍らを一律に上書きする。** `required` を上書きにすると、解決先が要求する項目を傍らが消せてしまい、
  新しい偽陰性を作る。和を取る。
- **baseline を更新して黙らせる。** 報告は消えるが、`$ref` を持つすべての項目について `default` の変更が
  以後も素通りする。

## Completion

- **Completed At**: 2026-09-12
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。変更は
  `tools/check/src/api-compat.ts` の `resolve()` だけで、製品にも規範文書にも届かない。
  `$ref` を解決するときに、同じスキーマオブジェクトに並んで書かれた keyword を捨てなくなった。
  **偽陰性が本体だった。** 見つかる切っ掛けは偽陽性（[[wi-536-group-display-name-source-selects-nothing]]
  が `display_name_source` を列挙にしたときの 5 件の誤報）だったが、同じ読み落としは逆向きにも効く。
  `$ref` の傍らの `default` が `"a"` から `"b"` へ変わっても報告は**ゼロ件**だった。baseline 側も現行側も
  解決先だけを見るので、どちらも `default` を持たないことになり、差が消える。`default` の変更は
  この検査器自身が破壊的と宣言している変更であり、生成された OpenAPI には `$ref` の傍らの `default` が
  30 件、`description` が 89 件ある。**`$ref` を持つ項目については、宣言した検査が効いていなかった。**
  **重ね方は keyword ごとに違う。** `required` は和を取る。上書きにすると、baseline が傍らで 1 つ足していた
  だけで解決先の要求が消え、現行の項目が「新しく必須になった」という誤報になる。`properties` は名前ごとに
  重ねる。上書きにすると、傍らで書き直していない解決先の項目が「消えた」という誤報になる。**この 2 つは
  素朴な `{...resolved, ...beside}` では区別が付かない**ので、それぞれを落とす検査を別に置いた。
  **誤報は 1 つの検査で止まらなかった。** `check-work-items` は破壊的変更の有無から
  `documentation_impact` の下限を導くので、偽陽性が出ている間は無関係な `in_progress` の記録まで
  「documentation_impact none is weaker than inferred upgrade_note」で落ちた。誤報を baseline の更新で
  黙らせていたら、この波及ごと残った。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-api-compat`（[[wi-536-group-display-name-source-selects-nothing]] の
    TypeSpec 変更を当てた状態で）
  - **Requirement**: N/A: 検査器の突き合わせ規則の欠陥であり、製品の規範要求ではない。
  - **Observed Failure**: exit 1。`fail API compatibility (5 breaking change(s) vs
    spec/idmagic.openapi.baseline.json)`。5 件はすべて
    `group_push.display_name_source: default value changed from "name" to undefined` で、生成された
    OpenAPI はその位置に `"default": "name"` を持っていた。
  - **Detection Reason**: 生成物を直接読めば `default` が変わっていないことが分かるので、報告そのものが
    誤りだと判定できる。修正後の同じコマンドは
    `ok API compatibility (no breaking changes vs spec/idmagic.openapi.baseline.json)` を返す。
- **Unit RED Evidence**:
  - **Test**: `tools/check/src/api-compat.test.ts` の `compareOpenApi — keywords written beside a $ref`
    6 件
  - **Requirement**: N/A: 同上。
  - **Observed Failure**: 実装前に 4 件が RED。`default` の変更が報告されない（偽陰性）、inline から
    `$ref` へ移すと報告が出る（偽陽性）、傍らの `required` が解決先の要求を消す、傍らの `properties` の
    `type` 変更が見えない。残る 2 件は素朴な重ねでも通るので、故障注入で RED を観測した。
  - **Detection Reason**: 検査器は純粋関数なので、baseline と現行の 2 文書を直接与えれば報告の有無が
    そのまま観測になる。実文書では `$ref` が 3104 箇所あり、どの誤報も誤検出も他の差分に紛れる。
- **Change-Resistance Results**:
  4 件の故障を注入し、どれを崩すとどの検査が落ちるかを観測した。生存はゼロ。注入は Edit で当て、
  結果が変わらない注入を「検出されなかった」と誤読しないよう、毎回 `mise run test-tools` の
  pass/fail 件数で着弾を確かめている。

  | 注入した故障 | 落ちたテストと観測 |
  |---|---|
  | M1 傍らの keyword を丸ごと捨てる（元の実装） | 4 件が落ちる。`default` の変更が報告されず、inline → `$ref` が誤報になり、傍らの `properties` の `type` 変更も消える |
  | M2 素朴な `{...resolved, ...beside}` にする | 2 件が落ちる。`required` の和と `properties` の重ねだけが崩れ、M1 の 4 件は通る —— 6 件が別々の判断を観測していることが読める |
  | M3 `required` の和をやめ、傍らで上書きする | `keeps a field the resolved schema already required…` が落ち、解決先が要求していた `id` が「新しく必須になった」と誤報される |
  | M4 `properties` の重ねをやめ、傍らで上書きする | `keeps a resolved property the reference does not restate` が落ち、書き直していない `note` が「消えた」と誤報される |

  **方法の限界。** 注入は `resolve` と `mergeBeside` の中だけを崩しており、`diffSchema` が読む keyword の
  集合（`type`、`default`、`required`、`properties`、`enum`、`oneOf`、`items`）そのものは動かしていない。
  `enum` を傍らに書いた場合の積集合は Out of Scope として扱ったので、検査も注入も無い。
- **Verification Results**:
  - `mise run verify` - passed
