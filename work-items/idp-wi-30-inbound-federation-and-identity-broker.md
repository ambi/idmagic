---
id: idp-wi-30-inbound-federation-and-identity-broker
title: "Inbound federation / Identity Broker を導入する"
created_at: 2026-06-20
authors: ["tn"]
status: pending
risk: high
---

# Motivation
実運用 IdP では、自身が認証するだけでなく、外部 OIDC / SAML IdP、
social login、組織別 IdP discovery、JIT provisioning、account linking を扱う。
Keycloak の Identity Brokering、Okta / Google の workforce federation 相当の機能である。

# Scope
- **decision**:
  - 新規 ADR: external identity provider、federated identity、account linking、 JIT provisioning の所有境界を定義する。自動 linking の条件と禁止条件を明記する。
- **scl**:
  - ExternalIdentityProvider / FederatedIdentity / AccountLinkingPolicy を追加する。
  - StartFederatedLogin / CompleteFederatedLogin / LinkExternalIdentity / UnlinkExternalIdentity を追加する。
  - IdP discovery interface を追加する。
- **go**:
  - OIDC RP adapter を実装し、外部 OIDC IdP の discovery / JWKS / nonce / state を検証する。
  - SAML SP adapter は SAML IdP WI の後続として追加できる形に port を切る。
  - social login は OIDC provider の preset として扱う。
  - JIT provisioning で User を作る場合、tenant policy と attribute mapping を必須にする。
  - account linking は step-up 済み session でのみ許可する。
- **ui**:
  - login 画面に tenant 設定済み external IdP の選択肢を表示する。
  - admin settings に IdP provider 登録 / mapping / discovery rule を追加する。
  - account portal に linked accounts を表示し、unlink を提供する。
- **documentation**:
  - README に OIDC external IdP 設定例、JIT provisioning の注意を書く。

# Out of Scope
- SCIM provisioning。別 WI。
- LDAP / AD / Kerberos。別 WI または将来判断。
- inbound SAML SP の完全実装は SAML library 選定後に段階化してよい。
- external IdP token の長期保管。

# Verification
- `just test-go`
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just lint-ui`
- `just build-ui`
- 手動: test OIDC provider から federated login し、JIT user 作成と subsequent login が同じ FederatedIdentity に紐づくことを確認する。
- 手動: email が一致する既存 user への自動 linking が policy 無しでは拒否されることを確認する。

# Risk Notes
federated login は account takeover の主要リスクになる。JIT provisioning と
account linking は便利だが危険なので、初期値は保守的にし、tenant admin が
明示設定した場合のみ自動化する。
