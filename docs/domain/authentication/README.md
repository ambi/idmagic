# Authentication

## 責務と境界

エンドユーザーの資格情報の検証、MFA、ログインセッション、ステップアップ認証、パスワードの変更とリセット、アカウントの復旧、ログインの時点のフェデレーション、認証のイベントを扱う。
このモジュールが扱うのは、プリンシパルが本人であることをどう確かめ、確かめた結果をどうセッションとして保つかである。

| 扱わないもの | 担当 |
| --- | --- |
| `User`、`Group`、`Agent` のライフサイクル | `IdManagement` |
| 下流向けの SAML の IdP と WS-Federation の発行 | `SAML`、`WS-Federation` |
| 外部からの継続的なアイデンティティの取り込み | `Sourcing` |
| OAuth の認可とトークンの発行 | `OAuth2` |
| 可逆なシークレットのエンベロープ暗号化 | `DataKeys` |

## モデル

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `LoginSession` | 認証時刻、`amr` と `acr`、ステップアップの時刻、保留の目的、有効期限、失効 | `User` を参照する |
| `MfaFactor` | TOTP の要素と、暗号化した seed | `User` を参照する |
| `WebAuthnCredential` | `credential_id`、COSE の公開鍵、`sign_count` | `User` を参照する |
| `RecoveryCode` | コードのハッシュと使用の時刻 | `User` を参照する |
| `TrustedDevice` | selector、verifier のハッシュ、ブラウザーと OS の系統のラベル、時刻、失効 | `User` を参照する |
| `MfaEnrollmentBypass` | 管理者が発行する、単回限りの登録の許可 | `User` を参照する |
| `PasswordHistory` | 直近のパスワードのハッシュ | `User` を参照する |
| `IdentityProviderConnection` | テナント単位の上流の OIDC と SAML の接続 | `Tenant` を `tenant_id` で参照する |
| `FederatedIdentity` | テナント、プロバイダー、外部の不変な subject の組 | `IdentityProviderConnection` と `User` を参照する |

いずれの Aggregate も `User` の識別子を参照するが、`User` の Aggregate には含まれない。
User の削除がこれらへどう連鎖するかは、[ユーザーの設計](../identity-management/user/design.md)で定める。

- **判断**：`users.id` を正式で全体で一意なユーザーの識別子とし、プロトコルの `sub` のクレームはここから導く。その逆ではない。
- **判断**：このモジュールの大半は認証の完了の前の要求を扱うので、ロールではなく、そのセッションで確かめた認証の要素と認証の時刻が境界になる。

## 公開する契約

HTTP の操作とモデルの形は TypeSpec の `Authentication` のタグが、採用する標準の規則は[Authentication の標準仕様](standards.md)が定める。
次の表は、それ以外にほかのモジュールと結ぶ契約である。

| 契約 | 相手 | 向き | 内容 |
| --- | --- | --- | --- |
| ログインセッションの解決と `amr`、`acr` | `OAuth2`、`SAML`、`WS-Federation`、管理とアカウントの API | このモジュールが提供する | セッションの Cookie から認証済みの主体と認証の強度を返す |
| ステップアップ認証の直近性 | `Application` のサインインポリシー、機微なセルフサービスの操作 | このモジュールが提供する | `max(auth_time, step_up_at)` から直近の認証を判定する |
| `EmailSender` | 共有の通知の部品 | このモジュールが使う | パスワードのリセット、メールアドレスの確認、セキュリティ通知を送る |
| 相関用のソルト | `Audit` の `TenantSaltStore` | このモジュールが使う | 利用者名と IP の相関のハッシュを、テナントごとに作る |
| ドメインイベント | 監査と下流が購読する | このモジュールが発行する | `UserAuthenticated`、`AuthenticationFailed`、`LoginThrottled`、`SessionEnded`、`PasswordChanged`、`MfaFactor…`、`WebAuthnCredential…`、`RecoveryCodes…`、`TrustedDevice…` ほか |

## 機能

| 機能群 | 機能 | 内容 |
| --- | --- | --- |
| ログインとセッション | [ログイン](sign-in/README.md) | パスワードによるログイン、ログインの流量の制限、無効なユーザーの拒否、ブラウザーの初期化 |
| ログインとセッション | [ログインセッション](session/README.md) | セッションの保存、一覧、失効、サインアウト |
| ログインとセッション | [外部 IdP との連携](federation/README.md) | 上流の IdP との接続、外部 subject の関連付け、JIT |
| 認証の要素 | [パスワード](password/README.md) | パスワードのポリシー、変更、有効期限、リセット |
| 認証の要素 | [多要素認証](mfa/README.md) | 第二要素を求める条件、MFA の強制と登録、認証器のリセット |
| 認証の要素 | [TOTP](totp/README.md) | TOTP の要素の登録、照合、解除 |
| 認証の要素 | [WebAuthn](webauthn/README.md) | WebAuthn による第二要素とステップアップ認証 |
| 認証の要素 | [復旧コード](recovery/README.md) | 復旧コードによる第二要素 |
| 認証の要素 | [信頼済みデバイス](trusted-device/README.md) | 記憶したブラウザーによる第二要素の省略 |
| 本人の操作 | [アカウントポータル](account-portal/README.md) | 本人の範囲に閉じたセルフサービス API と、機微な操作のステップアップ認証 |
| 本人の操作 | [セキュリティ通知](security-notification/README.md) | アカウントのセキュリティの上の変化の通知 |
| 本人の操作 | [サインイン履歴](sign-in-activity/README.md) | 本人によるサインイン履歴の参照 |

| 文書 | 内容 |
| --- | --- |
| [Authentication の用語集](glossary.md) | このモジュールでの語義 |
| [Authentication の標準仕様](standards.md) | 採用する外部標準仕様 |
| [Authentication の設計](design/README.md) | 話題ごとの設計と重要な判断 |
