# ApiTokens の検証

この文書は、ApiTokens の要件と統制のうち、例のテストのほかに確かめているものを扱う。
システム全体の検証の方式は[検証設計](../../../design/verification/README.md)に従う。

## パースの境界のファズテスト

`ParseScopes` は、Go のネイティブのファジングの対象である。
オラクルは、返すスコープが必ず宣言した集合の要素で、重複を持たず、宣言した入力を取りこぼさないことである。
宣言の外の文字列がそのまま `Scope` として通ると、API の認可の判定が知らない権限の名前を持ち回ることになる。

## 共有の拒否の宣言

管理 API の `InsufficientScopeError` は、各 Context の契約が 403 として約束するが、それを振る舞いとして宣言するのは REQ-APITOKENS-004 の例だけである。
セキュリティ統制の検査（`mise run check-repository`）は、[API アクセストークンの認証と認可の例](../authentication/examples.feature.md)からこの宣言を読み、すべての Context の拒否の宣言に加える。
