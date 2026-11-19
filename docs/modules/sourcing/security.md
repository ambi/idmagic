# Sourcing のセキュリティ

この文書は、Sourcing の信頼境界で、外部から届く入力の解析をどう確かめるかを扱う。
制御の種類ごとの検証手段は[脅威モデル](../../design/security/threat-model.md#制御の種類ごとの検証手段)が定める。

## パースの境界のファズテスト

外部の IdP から届く SCIM の書き込みの本文は、Go のネイティブのファジングの対象である。
対象は、フィルターの式に加えて、`ParseUserWrite`、`ParseUserPatchOps`、`ParseGroupWrite`、`ParseGroupPatchOps` である。

オラクルは、拒否した本文から値を持ち出さないこと、受理した PATCH の操作が宣言した op と属性だけからなること、空の操作の列を成功として返さないことである。
未知の op がそのまま下位へ流れると、許可した属性の判定を経ずに適用される経路ができる。
