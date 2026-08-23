# ClaimMapping Scenarios

### REQ-CLAIMMAPPING-001: 対応付け規則のないカスタム属性はクレームとして発行されない
- ACTOR System
- GIVEN テナントに `visibility != Private` のカスタム属性が定義されている
- GIVEN RP の `ClaimMappingPolicy` にその属性をソースとする規則がない
- WHEN その RP 向けのクレームを解決する
- THEN 発行されるクレーム集合にその属性は含まれない

### REQ-CLAIMMAPPING-002: 公開できない属性をソースとする規則は発行を拒否する
- ACTOR System
- GIVEN `ClaimMappingPolicy` に、`attribute_defs` にないキーまたは `visibility=Private` のキーをソースとする規則がある
- WHEN その RP 向けのクレームを解決する
- THEN クレームを 1 つも発行せずに拒否する

### REQ-CLAIMMAPPING-003: 必須規則のソース属性が欠けていれば部分的な発行をしない
- ACTOR System
- GIVEN `ClaimMappingPolicy` に必須の規則がある
- GIVEN 解決済み属性にその規則のソース属性がない
- WHEN その RP 向けのクレームを解決する
- THEN 残りの規則だけを適用した部分的なクレーム集合を返さず、発行そのものを拒否する
