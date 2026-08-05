---
status: accepted
authors: [tn]
created_at: 2026-07-09
---

# ADR-087: WebAuthn / Passkey を phishing-resistant な第二要素として採用し、backup recovery code を併設する

## コンテキスト

現状の MFA は TOTP のみで、共有秘密ベースであるため実運用の最低ラインには届くが
**phishing-resistant ではない**（中間者に OTP を中継されうる）。Keycloak / Okta / Google
アカウント相当の IdP を目指すうえで、origin / RP に束縛され中継攻撃に耐える WebAuthn /
Passkey と、TOTP / passkey を紛失した際の復旧手段である backup recovery code は必須の
機能である。

WebAuthn ceremony は challenge store・RP ID / origin 検証・attestation・sign count 検証が
絡み、実装ミスの余地が大きい。加えて既存の MFA 抽象 `MfaFactor` は identity が
`(user_id, type)` で **1 種別 1 件**しか持てず、1 ユーザーが複数の authenticator を登録
できる WebAuthn の実態と構造的に合わない。ceremony の core は自前実装せず、Go の事実上の
標準ライブラリ `github.com/go-webauthn/webauthn` に委ねる方針が求められた。

本ステージのスコープは password + 第二要素 / step-up であり、passwordless-only tenant
policy と enterprise attestation の厳格 enforcement、device trust は Out of Scope とする。

## 決定

`WebAuthnCredential`（`webauthn_credentials`、identity=`credential_id`）を `MfaFactor` とは
別の独立エンティティとして新設する — 既存 `MfaFactor` の `(user_id, type)` identity は 1 種別
1 件しか持てず、1 ユーザーが複数 authenticator を登録できる WebAuthn の実態と構造的に合わない。
ceremony（CBOR/COSE 解析・署名検証）は自前実装せず `go-webauthn/webauthn` に委ねる。RP ID /
許可 origin は deployment config 由来とし起動時に検証、attestation は none（プライバシ優先）、
user verification は preferred、resident key は discouraged（本ステージは password + 第二要素 /
step-up に限り、passwordless は対象外）。assertion 検証で保存済み sign_count 以下の値が返った
場合（0-to-0 を除く）は credential clone の疑いとして拒否する。challenge は専用ストアを増やさず
既存の ephemeral `SessionStore` に束縛する。

`RecoveryCode`（`recovery_codes`、identity=`(user_id, code_hash)`）は hash-only・single-use・
再生成は全置換とし、TOTP / WebAuthn 喪失時の backup に限定して `User.mfa_enrolled` の真値には
数えない — backup を独立した第二要素として数えると、ユーザーが TOTP / passkey 無しで recovery
code のみに依存する運用を招くため。`mfa_enrolled` は TOTP factor または WebAuthn credential が
1 件以上存在することで導出する。第二要素成立で acr は `urn:idmagic:acr:mfa` へ昇格し、amr には
WebAuthn 成立時に `webauthn`（RFC 8176 登録値）、recovery code 消費時に `rc`（本アプリ固有の
非 IANA 値）を加える。

現在の設計は [`backend/authentication/ARCHITECTURE.md`](../backend/authentication/ARCHITECTURE.md)
の WebAuthn/passkey MFA and recovery codes セクションにある。

## 却下した代替案

- **既存 `mfa_factors` テーブルに WebAuthn を相乗りさせる**: identity が `(user_id, type)`
  で 1 種別 1 件のため、複数 authenticator を登録できない。secret 列に credential JSON を
  詰める案も、複数件・sign_count 更新・credential_id 検索に耐えず、専用テーブルに劣る。
- **WebAuthn ceremony を自前実装する**: CBOR / COSE / 各 attestation format の検証は誤りが
  致命的で、車輪の再発明。成熟した `go-webauthn/webauthn` に委ねる。
- **recovery code を平文または可逆で保存する**: 漏洩時に即座に第二要素を突破される。
  hash-only・single-use が backup secret の最小要件（NIST SP 800-63B §5.1.2）。
- **recovery code を独立した第二要素として `mfa_enrolled` に数える**: backup 手段を主要素
  扱いすると、ユーザーが TOTP / passkey 無しで recovery code のみに依存する運用を招く。
  backup は backup に留める。
- **passwordless（discoverable credential 必須）を初期から導入**: RP / UX / 移行の設計面積
  が大きく、本 WI では password + 第二要素 / step-up に絞る（Out of Scope）。

## 影響

- `spec/contexts/authentication.yaml`: `WebAuthnCredential` / `RecoveryCode` ほか models、
  4 events（+ 既存 `BackupCodeConsumed` 再利用）、8 interfaces、`WebAuthnPolicy` /
  `RecoveryCodePolicy`、`AuthenticationContextPolicy.mfa_amr_values` への `rc` 追加、
  `WebAuthnLevel3` 標準の adoption 昇格。derived artifacts を再生成する。
- `internal/shared/spec`: `WebAuthnCredential` / `RecoveryCode` twin、`WebAuthnTransport`
  enum、4 events。
- `internal/authentication`: use cases（webauthn / account_webauthn / verify_webauthn /
  recovery_codes）、ports（credential / recovery repository）、HTTP handlers、login 第二
  要素 endpoint。
- `internal/shared/adapters/persistence/{memory,postgres}` と `deploy/schema/postgres.sql`:
  `webauthn_credentials` / `recovery_codes` テーブル（user_id FK ON DELETE CASCADE、
  ADR-083 に従い tenant_id 列は持たない）。
- `internal/bootstrap`: 新 repo の wiring、WebAuthn RP config の env 読込と起動時検証。
- `go.mod`: `github.com/go-webauthn/webauthn` を追加。
- `ui/`: account portal の passkey / recovery code 管理、login の第二要素選択、step-up への
  passkey / recovery code 追加。
- `README` / `compose.yml`: RP ID / origin 設定、HTTPS 必須、localhost 開発時の注意。
