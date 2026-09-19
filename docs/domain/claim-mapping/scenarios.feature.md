# Feature: ClaimMapping のシナリオ

## Rule: REQ-CLAIMMAPPING-001 対応付け規則のないカスタム属性はクレームとして発行されない

Primary actor: `System`

### Example: EX-CLAIMMAPPING-001-01 通常経路

- Given テナントに `visibility != Private` のカスタム属性が定義されている
- And RP の `ClaimMappingPolicy` にその属性をソースとする規則がない
- When その RP 向けのクレームを解決する
- Then 発行されるクレーム集合にその属性は含まれない

## Rule: REQ-CLAIMMAPPING-002 公開できない属性をソースとする規則は発行を拒否する

Primary actor: `System`

### Example: EX-CLAIMMAPPING-002-01 通常経路

- Given `ClaimMappingPolicy` に、`attribute_defs` にないキーまたは `visibility=Private` のキーをソースとする規則がある
- When その RP 向けのクレームを解決する
- Then クレームを 1 つも発行せずに拒否する

## Rule: REQ-CLAIMMAPPING-003 必須規則のソース属性が欠けていれば部分的な発行をしない

Primary actor: `System`

### Example: EX-CLAIMMAPPING-003-01 通常経路

- Given `ClaimMappingPolicy` に必須の規則がある
- And 解決済み属性にその規則のソース属性がない
- When その RP 向けのクレームを解決する
- Then 残りの規則だけを適用した部分的なクレーム集合を返さず、発行そのものを拒否する
