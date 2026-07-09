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

`spec/contexts/authentication.yaml` に SCL-first で反映し、`internal/authentication`
配下と `ui/` に実装する。wi-26 で導入。既存の self-service TOTP MFA と step-up
再認証（[ADR-043](file:///Users/tn/src/idmagic/decisions/ADR-043-account-portal-csrf-and-step-up.md)）
を土台に、第二要素の選択肢として WebAuthn と recovery code を足す。全テーブルの
tenant key 方針は [ADR-083](file:///Users/tn/src/idmagic/decisions/ADR-083-globally-unique-client-id.md)
に従う。

### 1. WebAuthn は独立エンティティ・独立 ceremony として追加する

`MfaFactor` に押し込まず、`WebAuthnCredential`（identity=`credential_id`、新テーブル
`webauthn_credentials`）を新設し、1 ユーザーが複数 credential を登録できるようにする。
COSE 公開鍵・sign_count・transports・aaguid・backup_eligible / backup_state・label を保持
する。登録（attestation）と認証（assertion）は TOTP と別の use case・別 HTTP endpoint と
して独立に定義する（`Start/FinishWebAuthnRegistration`・`StartBrowserWebAuthn`・
`SubmitBrowserWebAuthn`）。ceremony の実装は `go-webauthn/webauthn` を用い、自前では
署名検証・CBOR 解析を書かない。

### 2. RP ID / origin 検証と attestation 方針

RP ID と許可 origin は deployment config（環境変数 `WEBAUTHN_RP_ID` /
`WEBAUTHN_RP_ORIGINS` / `WEBAUTHN_RP_DISPLAY_NAME`）由来とし、起動時に検証する。
ceremony ごとに challenge / origin / RP ID / user handle をサーバー側で検証する。
attestation は **none**（プライバシ優先）とし、機種を強制する enterprise attestation の
厳格 enforcement は行わない。user verification は **preferred**、resident key
（discoverable credential）は **discouraged**（password + 第二要素 / step-up 用途に限り、
passwordless は対象外）。

### 3. sign count 逆行は clone として拒否する

assertion 検証時、保存済み sign_count 以下の値が返った場合（0 と 0 を除く逆行）は
credential clone の疑いとして当該認証を拒否する。成功時は sign_count と last_used_at を
更新する。

### 4. challenge は既存 SessionStore に束縛し、新ストアを増やさない

登録・認証の challenge（go-webauthn の SessionData）は、専用ストアを増やさず既存の
ephemeral な `SessionStore`（memory / Valkey）へ、登録は sub、ログインは pending login
session id をキーに短命保存する。

### 5. recovery code は hash-only・single-use・set 全置換

`RecoveryCode`（identity=`(user_id, code_hash)`、新テーブル `recovery_codes`）を新設する。
平文は生成時に一度だけ表示し、DB には SHA-256 hex のみ保存する。1 コードは single-use で
`consumed_at` を立てて再利用不可にする。再生成は既存 set を全置換する。recovery code は
TOTP / WebAuthn 喪失時の backup であり、**単独では第二要素にしない**（`User.mfa_enrolled`
の真値には数えない）。生成・再生成・失効は step-up を必須とする。

### 6. mfa_enrolled は TOTP または WebAuthn の存在で導出する

`User.mfa_enrolled` は「TOTP factor **または** WebAuthn credential が 1 件以上存在する」で
導出する。TOTP / WebAuthn の解除時は残存要素に応じて再計算する。ログインの第二要素は
選択式とし、enrolled 状態に応じて TOTP / passkey / recovery code から選べる。

### 7. acr / amr への反映

第二要素成立で acr は `urn:idmagic:acr:mfa` へ昇格する。amr には WebAuthn 成立時に
`webauthn`（RFC 8176 登録値）を、recovery code 消費時に `rc` を加える。`rc` は RFC 8176 に
登録が無いため、本アプリ固有（非 IANA）の amr 値であることを SCL / 本 ADR で明記する。

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
