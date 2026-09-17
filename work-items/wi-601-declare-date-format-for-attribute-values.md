---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-18
priority: p3
depends_on: [wi-585-rewrite-api-rules-as-complete-api-guidelines]
change_kind: bugfix
affected_spec:
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.AttributeValue }
---

# 属性値の日付を TypeSpec の日付型として宣言する

## Motivation

[API ガイドライン](../docs/design/application/api-guidelines.md)の「日時と日付」は、日付を RFC 3339 `full-date` 形式の文字列で表すと定める。
Go の domain は `AttributeValue.date` を `2006-01-02` 形式で検証しているが、TypeSpec はこのプロパティを形式のない `string` として宣言している。
生成したクライアントと API リファレンスからは、受け付ける形式を読み取れない。

## Scope

- `AttributeValue.date` を TypeSpec の `plainDate` 型で宣言する。
- 同じ扱いの日付が他のモデルにないかを確認する。
- API ガイドラインの適用状況を改める。

## Out of Scope

- 日付の検証ロジックの変更。

## Design

TypeSpec の `plainDate` は OpenAPI の `format: date` を出力する。
Go の検証は現行のままとし、宣言だけを実装に合わせる。

## Tasks

- [ ] T001 [Spec] TypeSpec の型を改める。
- [ ] T002 [Docs] API ガイドラインの適用状況を改める。
- [ ] T003 [Verify] 変更を検証する。

## Verification

- `mise run check-spec`
- `mise run check-api-compat`

## Risk Notes

生成したフロントエンドの型が `string` から変わる場合は、利用箇所の型検査で確かめる。
