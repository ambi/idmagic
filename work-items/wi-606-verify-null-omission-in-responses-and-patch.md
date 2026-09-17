---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-18
priority: p2
depends_on: [wi-585-rewrite-api-rules-as-complete-api-guidelines]
change_kind: tooling
spec_impact:
  kind: none
  reason: "既存のルールに実装が従っているかを検査するだけで、TypeSpec もプロダクトの振る舞いも変えない。検査が違反を見つけた場合の修正は、検出した箇所ごとに扱う。"
---

# レスポンスの null 出力と PATCH の省略の扱いを検査する

## Motivation

[API ガイドライン](../docs/design/application/api-guidelines.md)の「値の不在」は、レスポンスに `null` を出力せず、`PATCH` で省略したプロパティを変更しないと定める。
TypeSpec はこのルールに従っているが、Go の実装が従っているかは検証していない。
`mise run check-contract-drift` はプロパティ名だけを比較し、`omitempty` の有無やポインター型の扱いを見ない。

- レスポンスの構造体で、ポインター型、スライス、マップのフィールドが `omitempty` を持たないと、値がないときに `null` を出力する。
- `PATCH` のリクエストの構造体で、フィールドがポインター型でないと、省略したプロパティがゼロ値で上書きされる。

## Scope

- レスポンスの構造体のうち、`null` を出力しうるフィールドを拒否する検査を加える。
- `PATCH` のリクエストの構造体のうち、ポインター型でないフィールドを拒否する検査を加える。
- 検査が見つけた違反を列挙し、修正する。
- API ガイドラインの適用状況を改める。

## Out of Scope

- プロトコルエンドポイント。

## Design

検査は `tools/check/src/contract-drift.ts` が既に辿っている「operationId からルート、ハンドラー、構造体」の連鎖を再利用する。
構造体まで辿れなかった API 操作は、合格とせず部分解析として件数を出力する。
正規表現で Go の型を読む現行の方式では、埋め込みの構造体と別パッケージの型を追えない可能性がある。
その場合は `go/ast` を使う Go の検査として実装することを、実装前に比較する。

## Tasks

- [ ] T001 [Design] TypeScript の検査を拡張するか、Go で実装するかを決める。
- [ ] T002 [Tooling] 検査を追加し、現行の違反を RED として記録する。
- [ ] T003 [App] 違反を修正する。
- [ ] T004 [Docs] API ガイドラインの適用状況を改める。
- [ ] T005 [Verify] 変更を検証する。

## Verification

- `mise run check-contract-drift`
- `mise run test-tools`
- `mise run verify`

## Risk Notes

`PATCH` でフィールドをポインター型に変えると、ユースケースの入力も変わる。省略とゼロ値を区別する必要がないフィールド（真偽値の既定値など）がないかを、修正の前に確かめる。
