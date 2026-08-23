# Authentication Standards

## OpenID Connect Core 1.0

1.0 incorporating errata set 2 — https://openid.net/specs/openid-connect-core-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-CORE-CODE-FLOW | required | MUST | 外部 OIDC 認証は authorization code flow を使い、ID Token の署名、issuer、audience、有効時間、nonce を検証する。 |
| OIDC-CORE-CSRF | required | SHOULD | callback は login attempt に束縛された単発 state を照合し、不一致または再利用を拒否する。 |

## OpenID Connect Discovery 1.0

1.0 incorporating errata set 2 — https://openid.net/specs/openid-connect-discovery-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-DISCOVERY-ISSUER | required | MUST | Discovery Metadata の `issuer` は設定した発行者と完全一致し、エンドポイントと JWKS URI は事前に許可された HTTPS オーソリティに限定する。 |

## TOTP Time-Based One-Time Password Algorithm

RFC 6238 — https://www.rfc-editor.org/rfc/rfc6238.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC6238-TOTP | optional | MUST | TOTP の認証要素を使うときは、共有シークレットと時間ステップから OTP を生成し検証する。 |

## Digital Identity Guidelines — Authentication and Authenticator Management

NIST SP 800-63B-4 — https://pages.nist.gov/800-63-4/sp800-63b.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| NIST63B4-PASSWORD-MINIMUM | excluded | MAY | 単一要素の認証に使うパスワードへ 15 文字以上の最小長は課さない。全体のデフォルトの下限を 12 文字とし、テナントはより長い下限へ上書きできる。 |
| NIST63B4-NO-COMPOSITION | required | MUST NOT | 文字種の混在のような、パスワードの構成規則を課さない。 |
| NIST63B4-PASSWORD-STORAGE | required | MUST | パスワードは salt とコストパラメーターを備え、オフライン攻撃に耐えるハッシュとして保存する。 |

## Web Authentication — An API for accessing Public Key Credentials Level 3

Candidate Recommendation Snapshot — https://www.w3.org/TR/webauthn-3/

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| WEBAUTHN3-AUTHENTICATION | required | MAY | WebAuthn の認証要素を使うときは、オリジンと Relying Party の範囲に限定された公開鍵クレデンシャルを検証する。 |
| WEBAUTHN3-REGISTRATION | required | MUST | WebAuthn の credential を登録するときは、attestation の challenge / RP ID / origin を検証し、COSE の公開鍵と sign count を保存する。 |

## Authentication Method Reference Values

RFC 8176 — https://www.rfc-editor.org/rfc/rfc8176.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8176-AMR-VOCABULARY | required | MUST | LoginSession.amr は RFC 8176 登録値 (pwd, otp, webauthn, hwk, swk) のサブセットに、本アプリ固有の非 IANA 拡張値 rc (recovery code) と tdev (信頼済みデバイス) を加えた語彙のみを許可する。 |
