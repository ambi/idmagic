# Feature: ホステッド UI とポータルの例

## Rule: REQ-SYSTEM-006 起動時設定により Vite 開発サーバー以外でも DemoLoginAffordance が表示される

### Example: EX-SYSTEM-006-01 通常経路

- Given Vite 開発サーバーではなくビルド済みのフロントエンドをデプロイしている
- When Operator が `VITE_DEMO_LOGIN_ENABLED` を `true` に設定してビルドする
- When EndUser が HomePage を表示する
- Then HomePage は DemoLoginAffordance を表示する
- When EndUser が DemoLoginAffordance を選択する
- Then `development` プロファイルが投入したデモユーザーの資格情報で `authorization_code` フローが完了する

### Example: EX-SYSTEM-006-02 `VITE_DEMO_LOGIN_ENABLED` が未設定または `true` 以外である

- Given Vite 開発サーバーではなくビルド済みのフロントエンドをデプロイしている
- When Operator が `VITE_DEMO_LOGIN_ENABLED` を `true` に設定してビルドする
- But `VITE_DEMO_LOGIN_ENABLED` が未設定または `true` 以外である
- Then `HomePage` は `DemoLoginAffordance` を表示しない

### Example: EX-SYSTEM-006-03 `development` プロファイルが適用されていない

- Given Vite 開発サーバーではなくビルド済みのフロントエンドをデプロイしている
- When Operator が `VITE_DEMO_LOGIN_ENABLED` を `true` に設定してビルドする
- When EndUser が HomePage を表示する
- Then HomePage は DemoLoginAffordance を表示する
- When EndUser が DemoLoginAffordance を選択する
- But `development` プロファイルが適用されていない
- Then 既知のデモ資格情報が存在しないため認可に失敗する

## Rule: REQ-SYSTEM-007 Vite 開発サーバーでの実行時は設定なしで DemoLoginAffordance が表示される

### Example: EX-SYSTEM-007-01 通常経路

- Given Vite 開発サーバーでフロントエンドを実行している
- When EndUser が HomePage を表示する
- Then `VITE_DEMO_LOGIN_ENABLED` の設定にかかわらず `HomePage` は `DemoLoginAffordance` を表示する

## Rule: REQ-SYSTEM-015 管理コンソールとアカウントポータルは失効セッションから同一画面に復帰する

### Example: EX-SYSTEM-015-01 通常経路

- Given Administrator がファーストパーティーの管理コンソールでアクセストークンを保持している
- And 保持しているアクセストークンが失効している
- When Administrator が AdminDashboard で管理 API を呼び出す
- Then API が 401 を返す
- Then 保持していたアクセストークン、リフレッシュトークン、OIDC コールバックの `state` を破棄する
- Then 直前の画面への同一オリジン相対の `return_to` を保ったまま再認可を 1 回だけ開始する
- Then 再ログイン完了後に元の AdminDashboard へ復帰する

### Example: EX-SYSTEM-015-02 再認可から復旧できない

- Given Administrator がファーストパーティーの管理コンソールでアクセストークンを保持している
- And 保持しているアクセストークンが失効している
- When Administrator が AdminDashboard で管理 API を呼び出す
- Then API が 401 を返す
- Then 保持していたアクセストークン、リフレッシュトークン、OIDC コールバックの `state` を破棄する
- Then 再認可から復旧できない
- Then 再ログイン導線を提示する
