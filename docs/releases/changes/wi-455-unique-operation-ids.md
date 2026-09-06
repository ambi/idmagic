# WI-455: Give every OpenAPI operation a unique operationId

作業項目は `wi-455-one-operation-id-naming-two-operations` である。

WI-455 は、SAML の既定/名前付きプロファイル経路、GET/POST binding、federation callback、RP-Initiated Logout の重複 `operationId` を経路ごとに分離する。

これまで生成契約が黙って捨てていた operation も個別の生成クライアントメソッドとして現れる。既存生成契約が参照していた代表経路の base ID は維持する。

規範上の条件は [OPENAPI31-OPERATION-ID](../../standards.md#openapi-specification-311) が定める。
