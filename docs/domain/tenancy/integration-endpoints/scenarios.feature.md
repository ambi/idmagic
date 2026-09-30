# Feature: 連携エンドポイントのシナリオ

## 結果

### Rule: REQ-TENANCY-001 管理者は正規ロケーションの連携情報を取得する

Primary actor: `TenantAdministrator`

#### Example: EX-TENANCY-001-01 通常経路

- Given admin が path または サブドメインの正規ロケーションから自身のテナントへアクセスしている
- When admin が連携エンドポイント画面を開く
- Then サーバーはリクエスト先テナントの正規の発行者から OAuth/OIDC、SAML、WS-Federation、SCIM、管理 API、本人用 API の URL を導出する
- Then 画面は OAuth/OIDC、SAML、WS-Federation、API のプロトコル単位で情報をまとめ、SAML 配下ではデフォルトを含むプロファイルごとにエンティティ ID、メタデータ、SSO、SLO、署名証明書を一組で表示する
- Then 画面は読み取り専用であり、Discovery Metadata と各プロトコルのメタデータを正本として案内し、個別値をコピーまたは証明書をダウンロードできる
- Then 正規の発行者と同じオリジンで配信するゲートウェイは、表示した公開プロトコル URL を対応するサーバーのエンドポイントへ転送する
- Then レスポンスにクライアントシークレット、API トークン、秘密鍵は含まれない

#### Example: EX-TENANCY-001-02 admin が別テナントの realm を URL として指定しようとする

- Given admin が path または サブドメインの正規ロケーションから自身のテナントへアクセスしている
- When admin が連携エンドポイント画面を開く
- But admin が別テナントの realm を URL として指定しようとする
- Then 対象指定パラメータは存在せず、解決済みテナント以外の情報は返らない
