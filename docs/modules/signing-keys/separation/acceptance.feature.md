# Feature: テナントと用途による鍵の分離の例

## Rule: REQ-SIGNINGKEYS-004 テナントごとの JWKS は互いに分離される

### Example: EX-SIGNINGKEYS-004-01 通常経路

- Given テナント "tenant-a" とテナント "tenant-b" がそれぞれ署名鍵を持つ
- When テナント "tenant-a" の管理者が署名鍵を回転する
- When クライアントがテナント "tenant-a" の JWKS を取得する
- Then レスポンスにはテナント "tenant-a" の `kid` だけが含まれ、テナント "tenant-b" の `kid` は含まれない

## Rule: REQ-SIGNINGKEYS-005 XML フェデレーション署名資格情報はテナントと用途で分離される

### Example: EX-SIGNINGKEYS-005-01 通常経路

- Given テナント "tenant-a" とテナント "tenant-b" が存在する
- And 両テナントが JWT Signing 鍵と XmlFederationSigning 鍵を持つ
- When テナント "tenant-a" が SAML Assertion を発行する
- Then Assertion はテナント "tenant-a" の有効な `XmlFederationSigning` 鍵で署名される
- Then テナント "tenant-b" の証明書でも、テナント "tenant-a" の JWT Signing 公開鍵でも署名を検証できない

## Rule: REQ-SIGNINGKEYS-006 XML フェデレーション鍵のローテーション中も既存の信頼関係を検証できる

### Example: EX-SIGNINGKEYS-006-01 通常経路

- Given `XmlFederationSigning` の現在の鍵 K1 がメタデータに掲載されている
- When 管理者が XmlFederationSigning 鍵を K2 へ回転する
- Then 新しい XML メッセージは K2 で署名される
- Then 猶予期間中の SAML / WS-Fed メタデータには K1 と K2 の証明書が掲載される
- Then 猶予期間終了後は K1 がメタデータから除去される

## Rule: REQ-SIGNINGKEYS-007 XML フェデレーション署名資格情報は再起動後も同一である

### Example: EX-SIGNINGKEYS-007-01 通常経路

- Given PostgreSQL または Vault プロバイダーでテナントの `XmlFederationSigning` 鍵が作成済みである
- When API プロセスを再起動する
- When クライアントが同じテナントのメタデータを取得する
- Then 有効な証明書のフィンガープリントは再起動前と一致する
