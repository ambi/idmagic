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

User の CSV がパスワードを設定する場合は、`PasswordHistoryWriterInTx` が行と同じ Tx へ履歴を追加する。
通常の履歴の Add と同じ処理を使い、IdManagement は履歴の保存表現と SQL を持たない。

Kerberos の principal の対応付けは運用管理者がデプロイする設定に置き、User のプロフィール属性としては保存しない。
認証応答の共有ストアは、SAML の応答に加えて `krb5:` の名前空間で Kerberos の再送を拒否する。
詳細は [Kerberos の実現方式](features/negotiate.md#実現方式)を参照する。

- **判断**：`users.id` を正式で全体で一意なユーザーの識別子とし、プロトコルの `sub` のクレームはここから導く。その逆ではない。
- **判断**：このモジュールの大半は認証の完了の前の要求を扱うので、ロールではなく、そのセッションで確かめた認証の要素と認証の時刻が境界になる。
