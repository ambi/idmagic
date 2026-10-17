# Authentication のデータ

この文書は、Authentication の Aggregate を扱う。

## 集約

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
User の削除がこれらへどう連鎖するかは、[ユーザーの設計](../identity-management/features/user.md#実現方式)で定める。

- **判断**：`users.id` を正式で全体で一意なユーザーの識別子とし、プロトコルの `sub` のクレームはここから導く。その逆ではない。
- **判断**：このモジュールの大半は認証の完了の前の要求を扱うので、ロールではなく、そのセッションで確かめた認証の要素と認証の時刻が境界になる。
