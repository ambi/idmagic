---
depends_on: []
status: pending
authors: ["tn"]
risk: high
created_at: 2026-06-21
---

# 検証可能クレデンシャル (OID4VCI 発行 / OID4VP 検証) に対応する

## Motivation
eIDAS 2.0 / EUDI Wallet、ISO mDL (モバイル運転免許証)、Microsoft Entra
Verified ID など、wallet ベースの検証可能クレデンシャル (Verifiable
Credentials, VC) が標準化・実運用フェーズに入っている。現代の IdP は従来の
OIDC token 発行に加え、(1) credential issuer として user に検証可能な
クレデンシャルを wallet へ発行 (OpenID for Verifiable Credential Issuance,
OID4VCI)、(2) verifier / relying party として wallet からの提示を検証
(OpenID for Verifiable Presentations, OID4VP) する役割を担い始めている。

idmagic は現状 OIDC の id_token / access_token 発行に閉じており、wallet
連携・選択的開示 (selective disclosure)・holder binding を伴う分散型
クレデンシャルに未対応。本 WI は issuer (OID4VCI) を主・verifier (OID4VP) を
従として導入し、SD-JWT VC を既定フォーマットに、status list による失効を
備える。これにより「IdP が発行した属性を、user が自分の wallet に保持し、
必要な相手へ必要な claim だけ提示する」という分散型 ID の中核に対応する。

## Scope
- **decision**:
  - 新規 ADR: credential フォーマット (SD-JWT VC を既定、mdoc は将来)、issuer metadata、holder binding (cnf)、選択的開示、status list による失効、署名鍵 (既存 KeyStore / per-tenant 鍵の流用) を確定する。
  - フロー確定: OID4VCI は authorization code flow + pre-authorized code flow、 credential offer の受け渡し (QR / deep link)。OID4VP は presentation request (DCQL / presentation_definition) と vp_token 検証 (direct_post) を採用する。
- **scl**:
  - 新規 model: VerifiableCredential / CredentialConfiguration / CredentialOffer / CredentialRequest / CredentialResponse / PresentationRequest / VpVerificationResult / CredentialStatus。
  - 新規イベント: CredentialOffered / CredentialIssued / CredentialRevoked / PresentationVerified / PresentationRejected。
  - 新規 interface: GetCredentialIssuerMetadata / CreateCredentialOffer / IssueCredential / GetCredentialStatusList / CreatePresentationRequest / VerifyPresentation。permission `AdminCredentialConfigWrite`。
- **go**:
  - OID4VCI issuer: credential offer 生成、token endpoint の pre-authorized_code grant、/credential endpoint で SD-JWT VC を署名発行 (holder binding cnf)。
  - 失効: Token Status List による失効公開と admin からの失効操作。
  - OID4VP verifier: presentation request 生成、vp_token / SD-JWT VC の署名・ holder binding・status・有効期限・claims を fail-closed で検証する。
  - Postgres adapter: credential_configurations / issued_credentials / credential_status テーブルと index。
- **http**:
  - /.well-known/openid-credential-issuer、/credential_offer、/credential、 status list endpoint、OID4VP の authorization request (request_uri) と response_uri (direct_post)。
- **ui**:
  - end-user: wallet へのクレデンシャル取得 (credential offer の QR / deep link)。
  - admin: credential configuration の定義・発行履歴・失効。

## Out of Scope
- mdoc / ISO 18013-5 (mDL) の完全準拠 (SD-JWT VC を先行、mdoc は将来 WI)。
- 特定 wallet 実装 (EUDI Reference Wallet 等) との相互運用認証の取得。
- デバイスバインド鍵 / Secure Enclave 連携。
- VP ベースで IdP 自身へログインするフロー (まず VerifyPresentation API のみ)。
- トラストフレームワーク / trust list (発行者の信頼連鎖) の構築。

## Verification
- `just test-go`
  - reason: SD-JWT VC の署名・選択的開示・holder binding 検証、pre-authorized code フロー、status list 失効の反映、vp_token 検証の成否境界 (失効後は拒否)。
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just lint-ui`
- `just build-ui`
- 手動: credential offer を発行 → mock wallet で取得 → OID4VP で提示 → verify が 成功する。admin で失効 → 同じ提示の verify が拒否される流れを確認する。

## Risk Notes
VC は暗号 (選択的開示 / holder binding) と新興仕様 (草案の版差) を伴い、誤実装は
検証バイパス (失効・束縛・署名の見落とし) に直結する。フォーマットは SD-JWT VC に
絞り、VerifyPresentation は「署名・holder binding・status・有効期限・claims」を
すべて満たす場合のみ成功とする (fail-closed)。issuer 鍵は既存 KeyStore に集約し、
鍵管理を二重化しない。
