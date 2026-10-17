# Feature: 鍵の提供元と健全性の例

## Rule: REQ-SIGNINGKEYS-008 KeyProvider の障害時は健全性を観測でき、JWKS は取得可能な範囲で返る

### Example: EX-SIGNINGKEYS-008-01 通常経路

- Given テナント "tenant-a" の KeyProvider が到達不能である
- And "sys-operator" は制御面テナントに所属し、有効ロールに `system_admin` を含む有効な User である
- When "sys-operator" が制御面テナントの経路で署名鍵の健全性一覧を取得する
- Then テナント "tenant-a" の `provider_healthy` は `false` として返る
- Then テナント "tenant-a" の JWKS は取得可能な範囲でキャッシュされた鍵を返す

## Rule: REQ-SIGNINGKEYS-009 制御面主体ではない管理者はシステムコンソールの署名鍵ヘルスにアクセスできない

### Example: EX-SIGNINGKEYS-009-01 通常経路

- Given "operator" は制御面テナントに所属するが、`admin` ロールだけを持ち `system_admin` ロールを持たない
- And "tenant-operator" は制御面テナント以外のテナントに所属し、直接または Group 経由で `system_admin` ロールを持つ
- When "operator" が署名鍵ヘルス一覧を呼び出す
- Then AccessDeniedError で拒否される
- When "tenant-operator" が自テナントの経路で署名鍵ヘルス一覧を呼び出す
- Then AccessDeniedError で拒否される
- Then レスポンスは他テナントの識別子、提供元、`active_kid`、鍵数、到達性のいずれも含まない
- Then テナント横断の健全性収集は実行されない

## Rule: REQ-SIGNINGKEYS-012 鍵素材を平文で永続化する構成は明示の選択を要する

### Example: EX-SIGNINGKEYS-012-01 通常経路

- Given デプロイ先の `PERSISTENCE` が `postgres` である
- And `KEY_PROVIDER` が指定されていない
- When `system_admin` が起動時設定を読むプロセスを起動する
- Then 起動は設定エラーで拒否され、`KEY_PROVIDER` を明示するよう示す
- Then 秘密鍵素材をデータベースへ平文で保存する KeyStore は構築されず、署名鍵も作成されない
