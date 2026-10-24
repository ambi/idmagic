# Feature: HTTP API の共通の振る舞いの例

## Rule: REQ-SYSTEM-013 バックエンド API のエラーは英語で返る

### Example: EX-SYSTEM-013-01 通常経路

- When APIConsumer が不正な JSON を HTTP API に送信する
- Then System は既存のエラーコードと HTTP ステータスを返す
- Then System は英語の `message` を返す
- When OAuth / OIDC のリダイレクトエンドポイントがリクエストを拒否する
- Then System は既存の OAuth エラーコードと英語の `error_description` を返す
- When 未知の内部エラーが発生する
- Then System は既存のエラーコードと HTTP ステータスを維持し、英語のエラー本文を返す
