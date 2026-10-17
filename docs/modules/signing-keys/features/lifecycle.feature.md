# Feature: 署名鍵のライフサイクルの例

## Rule: REQ-SIGNINGKEYS-001 署名鍵をローテーションしても以前の kid は JWKS に残る

### Example: EX-SIGNINGKEYS-001-01 通常経路

- Given `admin` ロールを持つ "operator" が認証済みである
- And 現在の署名鍵は `kid` "kid-old" を持つ
- When "operator" が管理画面で現在の署名鍵をローテーションする
- Then ローテーションによって `kid` "kid-new" が新しい有効鍵になる
- When クライアントが JWKS を取得する
- Then レスポンスに `kid` "kid-old" と "kid-new" の両方が含まれる

## Rule: REQ-SIGNINGKEYS-002 猶予期間終了後の署名鍵は JWKS から除去してアーカイブする

### Example: EX-SIGNINGKEYS-002-01 通常経路

- Given `kid` "kid-old" の `Verifying` 鍵は `expires_at` を経過している
- When スケジューラーがアーカイブ処理を実行する
- When クライアントが JWKS を取得する
- Then レスポンスに `kid` "kid-old" は含まれない
- Then SigningKeyArchived イベントに `kid`、`retiredAt`、`expiresAt`、`disposedAt` が記録される

## Rule: REQ-SIGNINGKEYS-003 ライフサイクル設定が不正なバッチは起動しない

### Example: EX-SIGNINGKEYS-003-01 通常経路

- Given `grace_days` が `cadence_days` 以上である
- When `system_admin` が `idmagic-batch signing-key-lifecycle` を起動する
- Then 設定エラーで終了し、鍵を回転しない

## Rule: REQ-SIGNINGKEYS-010 管理者は回転後の検証用鍵だけを即時無効化できる

### Example: EX-SIGNINGKEYS-010-01 通常経路

- Given 現在の署名鍵 K2 と、回転後に JWKS へ残る検証用鍵 K1 がある
- When 管理者が K1 を無効化する
- Then K1 は JWKS から除去される
- Then K2 は現在の署名鍵のまま残る

### Example: EX-SIGNINGKEYS-010-02 管理者が現在の署名鍵 K2 を無効化しようとする

- Given 現在の署名鍵 K2 と、回転後に JWKS へ残る検証用鍵 K1 がある
- When 管理者が K1 を無効化する
- But 管理者が現在の署名鍵 K2 を無効化しようとする
- Then エラー "InvalidRequestError"

## Rule: REQ-SIGNINGKEYS-011 署名鍵の回転と無効化は自テナントの管理者だけが要求できる

### Example: EX-SIGNINGKEYS-011-01 通常経路

- Given テナント "tenant-a" の現在の署名鍵は `kid` "kid-1" である
- When `admin` も `system_admin` も持たない "alice" が署名鍵の回転を要求する
- Then AccessDeniedError で拒否される
- Then 現在の署名鍵は "kid-1" のまま変わらず、SigningKeyRotated は発行されない

### Example: EX-SIGNINGKEYS-011-02 `signing-keys:read` だけを持つ API アクセストークンで回転または無効化を要求する

- Given テナント "tenant-a" の現在の署名鍵は `kid` "kid-1" である
- When `admin` も `system_admin` も持たない "alice" が署名鍵の回転を要求する
- But `signing-keys:read` だけを持つ API アクセストークンで回転または無効化を要求する
- Then 必要なスコープが `signing-keys:write` であることを示して拒否される
