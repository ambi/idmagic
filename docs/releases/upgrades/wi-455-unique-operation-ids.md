# WI-455: Regenerate clients for unique operationIds

作業項目は `wi-455-one-operation-id-naming-two-operations` である。

生成クライアントを OpenAPI から再生成する。SAML の既定プロファイル経路と POST binding、SAML federation callback、POST `/end_session` には新しいメソッド名が生成される。

既存の `SamlSingleSignOn`、`SamlSingleLogout`、`PublishSamlMetadata`、`DownloadSamlSigningCertificate`、`CompleteFederatedLogin`、`EndSession` は、従来の実行時契約が保持していた代表経路に残る。HTTP method、path、要求本文、応答本文は変更しない。

この変更にデータ移行や設定変更は不要である。互換性境界は [OPENAPI31-OPERATION-ID](../../standards.md#openapi-specification-311) が定める。
