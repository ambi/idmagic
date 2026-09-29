# Authentication

エンドユーザーの資格情報の検証、MFA、ログインセッション、ステップアップ認証、パスワードの変更とリセット、アカウントの復旧、ログイン時のフェデレーション、認証イベントを担う。

`User` / `Group` / `Agent` のライフサイクルそのものは `IdManagement` が担う。この Context が扱うのは、そのプリンシパルが本人であることをどう確かめ、確かめた結果をどうセッションとして保つかである。

| 文書 | 内容 |
|---|---|
| [Authentication の用語集](glossary.md) | この Context での語義 |
| [Authentication の標準仕様](standards.md) | 採用する外部標準仕様 |
| [Authentication の設計判断](decisions.md) | 複数の機能にまたがる設計判断 |
| [Authentication の内部設計](internals.md) | 複数の機能にまたがる機構の説明 |
| [Authentication のシナリオ](scenarios.feature.md) | ログイン、アカウントの API、無効なユーザーの拒否のように、複数の機能にまたがる受け入れシナリオ |

一つの機能だけの規則、状態遷移、設計判断、機構の説明は、次の機能ノードに置く。

| 機能ノード | 内容 |
|---|---|
| [外部 IdP との連携](federation/README.md) | 上流 IdP との接続、外部 subject の関連付け、JIT |
| [パスワード](password/README.md) | パスワードのポリシー、変更、有効期限、リセット |
| [TOTP](totp/README.md) | TOTP 認証要素の登録、照合、解除 |
| [多要素認証](mfa/README.md) | 第二要素を求める条件、MFA の強制と登録、認証器のリセット |
| [WebAuthn](webauthn/README.md) | WebAuthn による第二要素とステップアップ認証 |
| [復旧コード](recovery/README.md) | 復旧コードによる第二要素 |
| [ログインセッション](session/README.md) | セッションの保存、一覧、失効、サインアウト |
| [信頼済みデバイス](trusted-device/README.md) | 記憶したブラウザーによる第二要素の省略 |
| [セキュリティ通知](security-notification/README.md) | アカウントのセキュリティ上の変化の通知 |
| [サインイン履歴](sign-in-activity/README.md) | 本人によるサインイン履歴の参照 |
