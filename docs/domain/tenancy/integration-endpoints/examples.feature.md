# Feature: 連携エンドポイントの例

## Rule: REQ-TENANCY-001 管理者は正規ロケーションの連携情報を取得する

### Example: EX-TENANCY-001-01 通常経路

- Given admin が path または サブドメインの正規ロケーションから自身のテナントへアクセスしている
- When admin が連携エンドポイント画面を開く
- Then サーバーはリクエスト先テナントの正規の発行者から OAuth/OIDC、SAML、WS-Federation、SCIM、管理 API、本人用 API の URL を導出する
- Then 画面は OAuth/OIDC、SAML、WS-Federation、API のプロトコル単位で情報をまとめ、SAML 配下ではデフォルトを含むプロファイルごとにエンティティ ID、メタデータ、SSO、SLO、署名証明書を一組で表示する
- Then 画面は読み取り専用であり、Discovery Metadata と各プロトコルのメタデータを正本として案内し、個別値をコピーまたは証明書をダウンロードできる
- Then 正規の発行者と同じオリジンで配信するゲートウェイは、表示した公開プロトコル URL を対応するサーバーのエンドポイントへ転送する
- Then レスポンスにクライアントシークレット、API トークン、秘密鍵は含まれない

### Example: EX-TENANCY-001-02 admin が別テナントの realm を URL として指定しようとする

- Given admin が path または サブドメインの正規ロケーションから自身のテナントへアクセスしている
- When admin が連携エンドポイント画面を開く
- But admin が別テナントの realm を URL として指定しようとする
- Then 対象指定パラメータは存在せず、解決済みテナント以外の情報は返らない

## Rule: REQ-TENANCY-041 署名証明書のフィンガープリントは、SHA-256 をコロンで区切った大文字の 16 進で返す

### Example: EX-TENANCY-041-01 通常経路

- Given admin が自身のテナントへアクセスしている
- When admin が連携エンドポイントを取得する
- Then SAML の署名証明書のフィンガープリントは、コロンで区切った 32 組の大文字の 16 進であり、証明書の DER の SHA-256 と一致する

## Rule: REQ-TENANCY-042 署名用の資格情報を解決できないときは、連携情報を返さない

### Example: EX-TENANCY-042-01 資格情報を解決できない

- Given admin が自身のテナントへアクセスしている
- And 署名用の資格情報の解決が失敗する
- When admin が連携エンドポイントを取得する
- Then `federation_credentials_unavailable` の 503 が返り、どの URL も返らない
