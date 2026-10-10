# Sourcing の検証

この文書は、Sourcing の要件と統制のうち、例のテストのほかに確かめているものを扱う。
システム全体の検証の方式は[検証設計](../../../design/verification/README.md)に従う。

## パースの境界のファズテスト

外部の IdP から届く SCIM の書き込みの本文は、Go のネイティブのファジングの対象である。
対象は、フィルターの式に加えて、`ParseUserWrite`、`ParseUserPatchOps`、`ParseGroupWrite`、`ParseGroupPatchOps` である。

オラクルは、拒否した本文から値を持ち出さないこと、受理した PATCH の操作が宣言した op と属性だけからなること、空の操作の列を成功として返さないことである。
未知の op がそのまま下位へ流れると、許可した属性の判定を経ずに適用される経路ができる。
