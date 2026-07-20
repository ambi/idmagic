---
context: repo
updated_at: 2026-07-20
contexts:
  System:
    spec: spec/contexts/system.yaml
    summary: "横断ユーザー体験、共有語彙、runtime composition。"
  Tenancy:
    spec: spec/contexts/tenancy.yaml
    summary: "テナント境界と設定。"
  IdManagement:
    spec: spec/contexts/identity-management.yaml
    summary: "User、Group、Agent のライフサイクル。"
  IdGovernance:
    spec: spec/contexts/identity-governance.yaml
    summary: "LifecycleWorkflow (JML 自動化) と IGA policy/orchestration。"
  Authentication:
    spec: spec/contexts/authentication.yaml
    summary: "資格情報、MFA、ログインセッション。"
  OAuth2:
    spec: spec/contexts/oauth2.yaml
    summary: "OAuth 2.0 / OIDC プロトコル。"
  Audit:
    spec: spec/contexts/audit.yaml
    summary: "横断監査 read model。"
  Application:
    spec: spec/contexts/application.yaml
    summary: "Application catalog と割当。"
  ClaimMapping:
    spec: spec/contexts/claim-mapping.yaml
    summary: "protocol-neutral claim release。"
  SigningKeys:
    spec: spec/contexts/signing-keys.yaml
    summary: "tenant-scoped signing key lifecycle。"
  WsFederation:
    spec: spec/contexts/ws-federation.yaml
    summary: "WS-Federation / WS-Trust。"
  Saml:
    spec: spec/contexts/saml.yaml
    summary: "SAML 2.0 IdP。"
  Scim:
    spec: spec/contexts/scim.yaml
    summary: "SCIM 2.0 inbound provisioning。"
  Provisioning:
    spec: spec/contexts/provisioning.yaml
    summary: "SCIM 2.0 outbound provisioning (下流 SaaS への user/group push lifecycle management)。"
  Jobs:
    spec: spec/contexts/jobs.yaml
    summary: "durable asynchronous jobs。"
  Seeding:
    spec: spec/contexts/seeding.yaml
    summary: "環境別 seed profile、計画、安全な適用 orchestration。"
  ra:
    spec: tools/ra/spec/scl.yaml
    summary: "RA workspace orchestration。"
  yaml-check:
    spec: tools/yaml-check/spec/scl.yaml
    summary: "SCL と Architecture schema/semantic validation。"
  scl-to-html:
    spec: tools/scl-to-html/spec/scl.yaml
    summary: "SCL HTML renderer。"
  scl-to-jsonschema:
    spec: tools/scl-to-jsonschema/spec/scl.yaml
    summary: "SCL JSON Schema generator。"
  scl-to-openapi:
    spec: tools/scl-to-openapi/spec/scl.yaml
    summary: "SCL OpenAPI generator。"
modules:
  claimmapping-domain:
    path: backend/claimmapping/domain
    responsibility: "protocol-neutral な claim release policy と issued claim の公開語彙。"
    context: ClaimMapping
    layer: domain
    role: published_interface
    realizes:
      - { context: ClaimMapping, kind: model, element: ClaimMappingSource }
      - { context: ClaimMapping, kind: model, element: ClaimMappingRule }
      - { context: ClaimMapping, kind: model, element: NameIdConfiguration }
      - { context: ClaimMapping, kind: model, element: ClaimMappingPolicy }
      - { context: ClaimMapping, kind: model, element: IssuedClaim }
  claimmapping-usecases:
    path: backend/claimmapping/usecases
    responsibility: "identity 属性から protocol-neutral claim を fail-closed で射影する公開 application service。"
    context: ClaimMapping
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: claimmapping-domain, via: published_interface }
      - { module: idmanagement-domain, via: published_interface }
      - { module: idmanagement-user-domain, via: published_interface }
  signingkeys-domain:
    path: backend/signingkeys/domain
    responsibility: "tenant-scoped signing key metadata、状態語彙、domain event。"
    context: SigningKeys
    layer: domain
    role: published_interface
    realizes:
      - { context: SigningKeys, kind: model, element: SignatureAlgorithm }
      - { context: SigningKeys, kind: model, element: KeyProvider }
      - { context: SigningKeys, kind: model, element: KeyUsage }
      - { context: SigningKeys, kind: model, element: SigningKey }
  signingkeys-ports:
    path: backend/signingkeys/ports
    responsibility: "SigningKeys の鍵 repository/provider port。"
    context: SigningKeys
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: signingkeys-domain, via: published_interface }
  signingkeys-usecases:
    path: backend/signingkeys/usecases
    responsibility: "鍵 rotation と tenant key health の application service。"
    context: SigningKeys
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: shared-spec, via: technical_shared }
      - { module: signingkeys-domain, via: published_interface }
      - { module: signingkeys-ports, via: published_interface }
      - { module: tenancy-ports, via: published_interface }
      - { module: tenancy-public, via: published_interface }
  signingkeys-adapters:
    path: backend/signingkeys/adapters
    responsibility: "SigningKeys の HTTP、memory/PostgreSQL/Vault/crypto adapter。"
    context: SigningKeys
    layer: adapters
    role: binding
    depends_on:
      - { module: http-support, via: binding }
      - { module: shared-adapters, via: technical_shared }
      - { module: signingkeys-domain, via: published_interface }
      - { module: signingkeys-ports, via: published_interface }
      - { module: signingkeys-usecases, via: published_interface }
      - { module: tenancy-domain, via: published_interface }
      - { module: tenancy-ports, via: published_interface }
      - { module: tenancy-public, via: published_interface }
  signingkeys-public:
    path: backend/signingkeys/
    responsibility: "SigningKeys root package の公開 facade。"
    context: SigningKeys
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: signingkeys-ports, via: published_interface }
  signingkeys-composition:
    path: backend/signingkeys/module.go
    responsibility: "SigningKeys の adapter と port を束ねる composition module。"
    context: SigningKeys
    layer: infrastructure
    role: composition_root
    depends_on:
      - { module: signingkeys-ports, via: published_interface }
  application-domain:
    path: backend/application/domain
    responsibility: "Application のドメインモデルと純粋な規則。"
    context: Application
    layer: domain
    role: published_interface
  application-ports:
    path: backend/application/ports
    responsibility: "Application の公開 port と外界への抽象。"
    context: Application
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: application-domain, via: published_interface }
  application-usecases:
    path: backend/application/usecases

    responsibility: "Application のユースケース。"
    context: Application
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: application-domain, via: published_interface }
      - { module: application-ports, via: published_interface }
      - { module: authentication-domain, via: published_interface }
      - { module: authentication-usecases, via: published_interface }
      - { module: shared-services, via: technical_shared }
      - { module: shared-spec, via: technical_shared }
      - { module: tenancy-public, via: published_interface }
  application-adapters:
    path: backend/application/adapters

    responsibility: "Application の HTTP・永続化 adapter。"
    context: Application
    layer: adapters
    role: binding
    depends_on:
      - { module: application-domain, via: published_interface }
      - { module: application-ports, via: published_interface }
      - { module: application-usecases, via: published_interface }
      - { module: claimmapping-domain, via: published_interface }
      - { module: http-support, via: binding }
      - { module: idmanagement-group-ports, via: binding }
      - { module: idmanagement-user-domain, via: binding }
      - { module: idmanagement-user-ports, via: binding }
      - { module: oauth2-domain, via: binding }
      - { module: oauth2-ports, via: binding }
      - { module: oauth2-usecases, via: binding }
      - { module: saml-domain, via: binding }
      - { module: saml-ports, via: binding }
      - { module: shared-adapters, via: binding }
      - { module: shared-spec, via: binding }
      - { module: wsfederation-domain, via: binding }
      - { module: wsfederation-ports, via: binding }
  audit-ports:
    path: backend/audit/ports
    responsibility: "Audit の公開 port と外界への抽象。"
    context: Audit
    layer: use_cases
    role: published_interface
  audit-usecases:
    path: backend/audit/usecases

    responsibility: "Audit のユースケース。"
    context: Audit
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: audit-ports, via: published_interface }
      - { module: shared-spec, via: technical_shared }
  audit-adapters:
    path: backend/audit/adapters

    responsibility: "Audit の HTTP・永続化 adapter。"
    context: Audit
    layer: adapters
    role: binding
    depends_on:
      - { module: audit-ports, via: published_interface }
      - { module: audit-usecases, via: published_interface }
      - { module: http-support, via: binding }
      - { module: idmanagement-user-domain, via: binding }
      - { module: idmanagement-user-ports, via: binding }
      - { module: oauth2-domain, via: binding }
      - { module: shared-adapters, via: binding }
      - { module: shared-spec, via: binding }
      - { module: tenancy-domain, via: binding }
  authentication-domain:
    path: backend/authentication/domain

    responsibility: "Authentication feature 横断の認証コンテキスト・イベント型と純粋規則。"
    context: Authentication
    layer: domain
    role: published_interface
    depends_on:
      - { module: shared-spec, via: technical_shared }
      - { module: tenancy-domain, via: published_interface }
  authentication-password-domain:
    path: backend/authentication/password/domain
    responsibility: "Password policy のドメイン規則。"
    context: Authentication
    layer: domain
    role: published_interface
    depends_on:
      - { module: tenancy-domain, via: published_interface }
  authentication-password-ports:
    path: backend/authentication/password/ports
    responsibility: "Password hash・履歴・reset token・breach 検査の公開 port。"
    context: Authentication
    layer: use_cases
    role: published_interface
  authentication-password-usecases:
    path: backend/authentication/password/usecases
    responsibility: "Password policy、変更、reset のユースケース。"
    context: Authentication
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: authentication-domain, via: published_interface }
      - { module: authentication-password-ports, via: published_interface }
      - { module: authentication-ports, via: published_interface }
      - { module: authentication-usecases, via: published_interface }
      - { module: idmanagement-domain, via: published_interface }
      - { module: idmanagement-user-domain, via: published_interface }
      - { module: idmanagement-user-ports, via: published_interface }
      - { module: shared-spec, via: technical_shared }
      - { module: tenancy-public, via: published_interface }
  authentication-password-adapters:
    path: backend/authentication/password/adapters
    responsibility: "Password の HTTP・memory・PostgreSQL adapter。"
    context: Authentication
    layer: adapters
    role: binding
    depends_on:
      - { module: authentication-httpdeps, via: binding }
      - { module: authentication-mfa-usecases, via: published_interface }
      - { module: authentication-password-domain, via: published_interface }
      - { module: authentication-password-ports, via: published_interface }
      - { module: authentication-password-usecases, via: published_interface }
      - { module: http-support, via: binding }
      - { module: shared-adapters, via: technical_shared }
      - { module: shared-kernel, via: technical_shared }
      - { module: tenancy-domain, via: binding }
      - { module: tenancy-public, via: binding }
  authentication-totp-domain:
    path: backend/authentication/totp/domain
    responsibility: "TOTP factor のドメイン型。"
    context: Authentication
    layer: domain
    role: published_interface
    depends_on:
      - { module: shared-spec, via: technical_shared }
  authentication-totp-ports:
    path: backend/authentication/totp/ports
    responsibility: "TOTP factor repository の公開 port。"
    context: Authentication
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: authentication-totp-domain, via: published_interface }
      - { module: shared-spec, via: technical_shared }
  authentication-totp-usecases:
    path: backend/authentication/totp/usecases
    responsibility: "TOTP 生成・検証のユースケース。"
    context: Authentication
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: authentication-totp-ports, via: published_interface }
      - { module: shared-spec, via: technical_shared }
  authentication-totp-adapters:
    path: backend/authentication/totp/adapters
    responsibility: "TOTP factor の memory・PostgreSQL adapter。"
    context: Authentication
    layer: adapters
    role: binding
    depends_on:
      - { module: authentication-totp-domain, via: published_interface }
      - { module: shared-adapters, via: technical_shared }
      - { module: shared-spec, via: technical_shared }
  authentication-webauthn-domain:
    path: backend/authentication/webauthn/domain
    responsibility: "WebAuthn credential のドメイン型。"
    context: Authentication
    layer: domain
    role: published_interface
    depends_on:
      - { module: shared-spec, via: technical_shared }
  authentication-webauthn-ports:
    path: backend/authentication/webauthn/ports
    responsibility: "WebAuthn credential・ceremony session の公開 port。"
    context: Authentication
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: authentication-webauthn-domain, via: published_interface }
  authentication-webauthn-usecases:
    path: backend/authentication/webauthn/usecases
    responsibility: "WebAuthn enrollment・authentication・factor 検証のユースケース。"
    context: Authentication
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: authentication-domain, via: published_interface }
      - { module: authentication-totp-ports, via: published_interface }
      - { module: authentication-usecases, via: published_interface }
      - { module: authentication-webauthn-domain, via: published_interface }
      - { module: authentication-webauthn-ports, via: published_interface }
      - { module: idmanagement-user-domain, via: published_interface }
      - { module: idmanagement-user-ports, via: published_interface }
      - { module: shared-spec, via: technical_shared }
  authentication-webauthn-adapters:
    path: backend/authentication/webauthn/adapters
    responsibility: "WebAuthn の HTTP・memory・PostgreSQL・Valkey adapter。"
    context: Authentication
    layer: adapters
    role: binding
    depends_on:
      - { module: authentication-httpdeps, via: binding }
      - { module: authentication-webauthn-domain, via: published_interface }
      - { module: authentication-webauthn-usecases, via: published_interface }
      - { module: http-support, via: binding }
      - { module: shared-adapters, via: technical_shared }
  authentication-mfa-domain:
    path: backend/authentication/mfa/domain
    responsibility: "MFA enrollment decision・bypass のドメイン型。"
    context: Authentication
    layer: domain
    role: published_interface
  authentication-mfa-ports:
    path: backend/authentication/mfa/ports
    responsibility: "MFA enrollment bypass repository の公開 port。"
    context: Authentication
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: authentication-mfa-domain, via: published_interface }
  authentication-mfa-usecases:
    path: backend/authentication/mfa/usecases
    responsibility: "TOTP・WebAuthn 横断の enrollment・second factor・step-up orchestration。"
    context: Authentication
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: authentication-domain, via: published_interface }
      - { module: authentication-mfa-domain, via: published_interface }
      - { module: authentication-mfa-ports, via: published_interface }
      - { module: authentication-password-ports, via: published_interface }
      - { module: authentication-recovery-ports, via: published_interface }
      - { module: authentication-recovery-usecases, via: published_interface }
      - { module: authentication-session-usecases, via: published_interface }
      - { module: authentication-totp-domain, via: published_interface }
      - { module: authentication-totp-ports, via: published_interface }
      - { module: authentication-totp-usecases, via: published_interface }
      - { module: authentication-usecases, via: published_interface }
      - { module: authentication-webauthn-ports, via: published_interface }
      - { module: authentication-webauthn-usecases, via: published_interface }
      - { module: idmanagement-user-domain, via: published_interface }
      - { module: idmanagement-user-ports, via: published_interface }
      - { module: shared-spec, via: technical_shared }
      - { module: tenancy-public, via: published_interface }
  authentication-mfa-adapters:
    path: backend/authentication/mfa/adapters
    responsibility: "MFA orchestration の HTTP・memory・PostgreSQL adapter。"
    context: Authentication
    layer: adapters
    role: binding
    depends_on:
      - { module: authentication-httpdeps, via: binding }
      - { module: authentication-mfa-domain, via: published_interface }
      - { module: authentication-mfa-usecases, via: published_interface }
      - { module: authentication-webauthn-adapters, via: binding }
      - { module: authentication-webauthn-usecases, via: published_interface }
      - { module: http-support, via: binding }
      - { module: shared-adapters, via: technical_shared }
  authentication-session-domain:
    path: backend/authentication/session/domain
    responsibility: "Login session・pending request のドメイン型。"
    context: Authentication
    layer: domain
    role: published_interface
    depends_on:
      - { module: authentication-domain, via: published_interface }
      - { module: shared-spec, via: technical_shared }
  authentication-session-ports:
    path: backend/authentication/session/ports
    responsibility: "Login session store・login throttle の公開 port。"
    context: Authentication
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: authentication-session-domain, via: published_interface }
      - { module: shared-spec, via: technical_shared }
  authentication-session-usecases:
    path: backend/authentication/session/usecases
    responsibility: "Login session lifecycle と cookie 管理のユースケース。"
    context: Authentication
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: authentication-domain, via: published_interface }
      - { module: authentication-mfa-domain, via: published_interface }
      - { module: authentication-session-domain, via: published_interface }
      - { module: authentication-session-ports, via: published_interface }
      - { module: authentication-usecases, via: published_interface }
      - { module: shared-spec, via: technical_shared }
      - { module: tenancy-public, via: published_interface }
  authentication-session-adapters:
    path: backend/authentication/session/adapters
    responsibility: "Login session の HTTP・memory・PostgreSQL・Valkey adapter。"
    context: Authentication
    layer: adapters
    role: binding
    depends_on:
      - { module: authentication-httpdeps, via: binding }
      - { module: authentication-session-domain, via: published_interface }
      - { module: authentication-session-ports, via: published_interface }
      - { module: authentication-session-usecases, via: published_interface }
      - { module: http-support, via: binding }
      - { module: oauth2-usecases, via: binding }
      - { module: shared-adapters, via: technical_shared }
      - { module: shared-spec, via: technical_shared }
      - { module: tenancy-public, via: binding }
  authentication-recovery-domain:
    path: backend/authentication/recovery/domain
    responsibility: "Recovery code のドメイン型。"
    context: Authentication
    layer: domain
    role: published_interface
    depends_on:
      - { module: shared-spec, via: technical_shared }
  authentication-recovery-ports:
    path: backend/authentication/recovery/ports
    responsibility: "Recovery code repository の公開 port。"
    context: Authentication
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: authentication-recovery-domain, via: published_interface }
  authentication-recovery-usecases:
    path: backend/authentication/recovery/usecases
    responsibility: "Recovery code の生成・再生成・消費ユースケース。"
    context: Authentication
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: authentication-domain, via: published_interface }
      - { module: authentication-recovery-domain, via: published_interface }
      - { module: authentication-recovery-ports, via: published_interface }
      - { module: authentication-usecases, via: published_interface }
      - { module: idmanagement-user-ports, via: published_interface }
      - { module: shared-spec, via: technical_shared }
  authentication-recovery-adapters:
    path: backend/authentication/recovery/adapters
    responsibility: "Recovery code の HTTP・memory・PostgreSQL adapter。"
    context: Authentication
    layer: adapters
    role: binding
    depends_on:
      - { module: authentication-httpdeps, via: binding }
      - { module: authentication-recovery-domain, via: published_interface }
      - { module: authentication-recovery-usecases, via: published_interface }
      - { module: http-support, via: binding }
      - { module: shared-adapters, via: technical_shared }
  authentication-ports:
    path: backend/authentication/ports

    responsibility: "Authentication feature 横断の event・email・continuation 公開 port。"
    context: Authentication
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: authentication-domain, via: published_interface }
      - { module: shared-spec, via: technical_shared }
  authentication-usecases:
    path: backend/authentication/usecases

    responsibility: "Authentication feature 横断の ACR・event・retention・lifecycle helper。"
    context: Authentication
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: audit-ports, via: published_interface }
      - { module: authentication-domain, via: published_interface }
      - { module: authentication-ports, via: published_interface }
      - { module: authentication-totp-ports, via: published_interface }
      - { module: authentication-webauthn-ports, via: published_interface }
      - { module: idmanagement-domain, via: published_interface }
      - { module: idmanagement-user-domain, via: published_interface }
      - { module: idmanagement-user-ports, via: published_interface }
      - { module: shared-spec, via: technical_shared }
      - { module: tenancy-public, via: published_interface }
  authentication-adapters:
    path: backend/authentication/adapters

    responsibility: "Authentication feature 横断 route と event/email persistence adapter。"
    context: Authentication
    layer: adapters
    role: binding
    depends_on:
      - { module: audit-ports, via: binding }
      - { module: authentication-domain, via: published_interface }
      - { module: authentication-httpdeps, via: binding }
      - { module: authentication-mfa-adapters, via: binding }
      - { module: authentication-mfa-usecases, via: published_interface }
      - { module: authentication-password-adapters, via: binding }
      - { module: authentication-password-ports, via: published_interface }
      - { module: authentication-ports, via: published_interface }
      - { module: authentication-recovery-adapters, via: binding }
      - { module: authentication-recovery-ports, via: published_interface }
      - { module: authentication-recovery-usecases, via: published_interface }
      - { module: authentication-session-adapters, via: binding }
      - { module: authentication-session-usecases, via: published_interface }
      - { module: authentication-totp-ports, via: published_interface }
      - { module: authentication-usecases, via: published_interface }
      - { module: authentication-webauthn-adapters, via: binding }
      - { module: authentication-webauthn-ports, via: published_interface }
      - { module: authentication-webauthn-usecases, via: published_interface }
      - { module: http-support, via: binding }
      - { module: idmanagement-usecases, via: binding }
      - { module: idmanagement-user-ports, via: binding }
      - { module: idmanagement-user-usecases, via: binding }
      - { module: oauth2-domain, via: binding }
      - { module: oauth2-ports, via: binding }
      - { module: oauth2-usecases, via: binding }
      - { module: shared-adapters, via: binding }
      - { module: shared-kernel, via: binding }
      - { module: shared-spec, via: binding }
      - { module: tenancy-domain, via: binding }
      - { module: tenancy-ports, via: binding }
      - { module: tenancy-public, via: binding }
  authentication-httpdeps:
    path: backend/authentication/adapters/http/httpdeps
    responsibility: "Authentication HTTP handler 群が共有する Deps と account 認証 helper の leaf package。"
    context: Authentication
    layer: adapters
    role: binding
    depends_on:
      - { module: audit-ports, via: binding }
      - { module: authentication-domain, via: published_interface }
      - { module: authentication-mfa-ports, via: published_interface }
      - { module: authentication-mfa-usecases, via: published_interface }
      - { module: authentication-password-ports, via: published_interface }
      - { module: authentication-ports, via: published_interface }
      - { module: authentication-recovery-ports, via: published_interface }
      - { module: authentication-recovery-usecases, via: published_interface }
      - { module: authentication-session-usecases, via: published_interface }
      - { module: authentication-totp-ports, via: published_interface }
      - { module: authentication-webauthn-ports, via: published_interface }
      - { module: authentication-webauthn-usecases, via: published_interface }
      - { module: http-support, via: binding }
      - { module: idmanagement-usecases, via: binding }
      - { module: idmanagement-user-ports, via: binding }
      - { module: idmanagement-user-usecases, via: binding }
      - { module: oauth2-ports, via: binding }
      - { module: oauth2-usecases, via: binding }
      - { module: shared-spec, via: binding }
      - { module: tenancy-ports, via: binding }
  idmanagement-domain:
    path: backend/idmanagement/domain
    responsibility: "IdManagement の feature 横断ドメイン型（enum・DomainEvent）。feature 固有の集約モデルは idmanagement-{user,group,agent}-domain が持つ (ADR-130)。"
    context: IdManagement
    layer: domain
    role: published_interface
    depends_on: []
  idmanagement-user-domain:
    path: backend/idmanagement/user/domain
    responsibility: "User feature 垂直スライスの集約モデル（User・属性）と純粋な規則 (ADR-130)。"
    context: IdManagement
    layer: domain
    role: published_interface
    depends_on:
      - { module: idmanagement-domain, via: published_interface }
      - { module: shared-spec, via: technical_shared }
  idmanagement-group-domain:
    path: backend/idmanagement/group/domain
    responsibility: "Group feature 垂直スライスの集約モデル（Group・動的メンバーシップ規則）と純粋な規則 (ADR-130)。"
    context: IdManagement
    layer: domain
    role: published_interface
    depends_on:
      - { module: idmanagement-domain, via: published_interface }
      - { module: idmanagement-user-domain, via: published_interface }
      - { module: shared-spec, via: technical_shared }
  idmanagement-agent-domain:
    path: backend/idmanagement/agent/domain
    responsibility: "Agent feature 垂直スライスの集約モデル（Agent・資格情報束縛）と純粋な規則 (ADR-130)。"
    context: IdManagement
    layer: domain
    role: published_interface
    depends_on:
      - { module: idmanagement-domain, via: published_interface }
      - { module: shared-spec, via: technical_shared }
  idmanagement-user-ports:
    path: backend/idmanagement/user/ports
    responsibility: "User feature の公開 port と外界への抽象 (ADR-130)。"
    context: IdManagement
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: idmanagement-user-domain, via: published_interface }
  idmanagement-group-ports:
    path: backend/idmanagement/group/ports
    responsibility: "Group feature の公開 port と外界への抽象 (ADR-130)。"
    context: IdManagement
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: idmanagement-group-domain, via: published_interface }
  idmanagement-agent-ports:
    path: backend/idmanagement/agent/ports
    responsibility: "Agent feature の公開 port と外界への抽象 (ADR-130)。"
    context: IdManagement
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: idmanagement-agent-domain, via: published_interface }
  idmanagement-usecases:
    path: backend/idmanagement/usecases
    responsibility: "IdManagement の feature 横断 usecase ヘルパー（role 正規化・emit 等）と feature 横断エラー変数。feature 固有のユースケースは idmanagement-{user,group,agent}-usecases が持つ (ADR-130)。"
    context: IdManagement
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: shared-spec, via: technical_shared }
  idmanagement-user-usecases:
    path: backend/idmanagement/user/usecases
    responsibility: "User feature のユースケース。"
    context: IdManagement
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: authentication-password-ports, via: published_interface }
      - { module: authentication-password-usecases, via: published_interface }
      - { module: authentication-ports, via: published_interface }
      - { module: authentication-session-ports, via: published_interface }
      - { module: authentication-totp-ports, via: published_interface }
      - { module: authentication-usecases, via: published_interface }
      - { module: idmanagement-domain, via: published_interface }
      - { module: idmanagement-group-ports, via: published_interface }
      - { module: idmanagement-group-usecases, via: published_interface }
      - { module: idmanagement-usecases, via: published_interface }
      - { module: idmanagement-user-domain, via: published_interface }
      - { module: idmanagement-user-ports, via: published_interface }
      - { module: jobs-domain, via: published_interface }
      - { module: oauth2-ports, via: published_interface }
      - { module: shared-services, via: technical_shared }
      - { module: shared-spec, via: technical_shared }
      - { module: tenancy-domain, via: published_interface }
      - { module: tenancy-ports, via: published_interface }
      - { module: tenancy-public, via: published_interface }
  idmanagement-group-usecases:
    path: backend/idmanagement/group/usecases
    responsibility: "Group feature のユースケース（動的グループ規則の評価・reconcile を含む）。"
    context: IdManagement
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: idmanagement-domain, via: published_interface }
      - { module: idmanagement-group-domain, via: published_interface }
      - { module: idmanagement-group-ports, via: published_interface }
      - { module: idmanagement-usecases, via: published_interface }
      - { module: idmanagement-user-domain, via: published_interface }
      - { module: idmanagement-user-ports, via: published_interface }
      - { module: jobs-domain, via: published_interface }
      - { module: jobs-ports, via: published_interface }
      - { module: jobs-usecases, via: published_interface }
      - { module: shared-spec, via: technical_shared }
      - { module: tenancy-domain, via: published_interface }
      - { module: tenancy-ports, via: published_interface }
      - { module: tenancy-public, via: published_interface }
  idmanagement-agent-usecases:
    path: backend/idmanagement/agent/usecases
    responsibility: "Agent feature のユースケース（OAuth2Client 資格情報束縛を含む）。"
    context: IdManagement
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: idmanagement-agent-domain, via: published_interface }
      - { module: idmanagement-agent-ports, via: published_interface }
      - { module: idmanagement-domain, via: published_interface }
      - { module: idmanagement-usecases, via: published_interface }
      - { module: idmanagement-user-ports, via: published_interface }
      - { module: oauth2-ports, via: published_interface }
      - { module: shared-spec, via: technical_shared }
      - { module: tenancy-public, via: published_interface }
  idmanagement-user-adapters:
    path: backend/idmanagement/user/adapters
    responsibility: "User feature の HTTP・in-memory・PostgreSQL 永続化 adapter (ADR-130 Phase 2)。ハンドラは Deps のフリー関数として実装され、Deps 型自体は idmanagement-httpdeps (leaf package) が所有する。"
    context: IdManagement
    layer: adapters
    role: binding
    depends_on:
      - { module: authentication-mfa-usecases, via: binding }
      - { module: authentication-password-usecases, via: binding }
      - { module: authentication-session-usecases, via: binding }
      - { module: authentication-usecases, via: binding }
      - { module: http-support, via: binding }
      - { module: idmanagement-httpdeps, via: published_interface }
      - { module: idmanagement-domain, via: published_interface }
      - { module: idmanagement-usecases, via: published_interface }
      - { module: idmanagement-user-domain, via: published_interface }
      - { module: idmanagement-user-usecases, via: published_interface }
      - { module: jobs-domain, via: binding }
      - { module: jobs-ports, via: binding }
      - { module: jobs-usecases, via: binding }
      - { module: oauth2-domain, via: binding }
      - { module: oauth2-usecases, via: binding }
      - { module: shared-adapters, via: technical_shared }
      - { module: shared-kernel, via: technical_shared }
      - { module: tenancy-public, via: binding }
  idmanagement-group-adapters:
    path: backend/idmanagement/group/adapters
    responsibility: "Group feature の HTTP・in-memory・PostgreSQL 永続化 adapter (ADR-130 Phase 2)。ハンドラは Deps のフリー関数として実装され、Deps 型自体は idmanagement-httpdeps (leaf package) が所有する。"
    context: IdManagement
    layer: adapters
    role: binding
    depends_on:
      - { module: http-support, via: binding }
      - { module: idmanagement-httpdeps, via: published_interface }
      - { module: idmanagement-group-domain, via: published_interface }
      - { module: idmanagement-group-usecases, via: published_interface }
      - { module: idmanagement-usecases, via: published_interface }
      - { module: idmanagement-user-domain, via: published_interface }
      - { module: shared-adapters, via: technical_shared }
  idmanagement-agent-adapters:
    path: backend/idmanagement/agent/adapters
    responsibility: "Agent feature の HTTP・in-memory・PostgreSQL 永続化 adapter (ADR-130 Phase 2)。ハンドラは Deps のフリー関数として実装され、Deps 型自体は idmanagement-httpdeps (leaf package) が所有する。"
    context: IdManagement
    layer: adapters
    role: binding
    depends_on:
      - { module: http-support, via: binding }
      - { module: idmanagement-httpdeps, via: published_interface }
      - { module: idmanagement-agent-domain, via: published_interface }
      - { module: idmanagement-agent-usecases, via: published_interface }
      - { module: idmanagement-domain, via: published_interface }
      - { module: idmanagement-usecases, via: published_interface }
      - { module: shared-adapters, via: technical_shared }
  idmanagement-httpdeps:
    path: backend/idmanagement/adapters/http/httpdeps
    responsibility: "IdManagement HTTP 層の Deps 型（leaf package）。user/group/agent の adapters/http と context ルートの routes.go 双方が依存するため、module 依存グラフの循環を避けて独立した module にしている (ADR-130 Phase 2)。"
    context: IdManagement
    layer: adapters
    role: binding
    depends_on:
      - { module: authentication-password-ports, via: binding }
      - { module: authentication-ports, via: binding }
      - { module: authentication-totp-ports, via: binding }
      - { module: http-support, via: binding }
      - { module: idmanagement-agent-ports, via: published_interface }
      - { module: idmanagement-group-ports, via: published_interface }
      - { module: idmanagement-user-ports, via: published_interface }
      - { module: jobs-ports, via: binding }
      - { module: oauth2-ports, via: binding }
      - { module: oauth2-usecases, via: binding }
      - { module: scim-ports, via: binding }
      - { module: shared-spec, via: binding }
      - { module: tenancy-ports, via: binding }
  idmanagement-adapters:
    path: backend/idmanagement/adapters

    responsibility: "IdManagement の route 登録集約点 (routes.go)・feature 横断統合テスト。feature 分割の対象外 (ADR-130): Deps 型自体は idmanagement-httpdeps (leaf package) が持ち、ハンドラ実装はフリー関数として user/group/agent へ分割した (Phase 2)。postgres 永続化は全て user/group/agent へ分割済み。lifecycle_workflows の postgres query/sqlcgen は IdGovernance 所有として backend/idgovernance/adapters/persistence/postgres へ物理移設した（ADR-117 が後続 WI へ後回しにしていた context-local sqlc 分割、ADR-090）。"
    context: IdManagement
    layer: adapters
    role: binding
    depends_on:
      - { module: idmanagement-agent-adapters, via: published_interface }
      - { module: idmanagement-group-adapters, via: published_interface }
      - { module: idmanagement-httpdeps, via: published_interface }
      - { module: idmanagement-user-adapters, via: published_interface }
  idgovernance-domain:
    path: backend/idgovernance/domain
    responsibility: "IdGovernance の LifecycleWorkflow モデル、状態、純粋な評価規則。"
    context: IdGovernance
    layer: domain
    role: published_interface
    depends_on:
      - { module: idmanagement-domain, via: published_interface }
      - { module: idmanagement-user-domain, via: published_interface }
  idgovernance-ports:
    path: backend/idgovernance/ports
    responsibility: "IdGovernance の workflow 定義・run・transactional capture の公開 port。"
    context: IdGovernance
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: idgovernance-domain, via: published_interface }
      - { module: idmanagement-domain, via: published_interface }
      - { module: idmanagement-group-domain, via: published_interface }
      - { module: idmanagement-user-domain, via: published_interface }
  idgovernance-usecases:
    path: backend/idgovernance/usecases
    responsibility: "LifecycleWorkflow の管理、run planning、実行を担う application service。"
    context: IdGovernance
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: application-domain, via: published_interface }
      - { module: application-ports, via: published_interface }
      - { module: authentication-ports, via: published_interface }
      - { module: idgovernance-domain, via: published_interface }
      - { module: idgovernance-ports, via: published_interface }
      - { module: idmanagement-domain, via: published_interface }
      - { module: idmanagement-group-domain, via: published_interface }
      - { module: idmanagement-group-ports, via: published_interface }
      - { module: idmanagement-user-domain, via: published_interface }
      - { module: idmanagement-user-ports, via: published_interface }
      - { module: jobs-domain, via: published_interface }
      - { module: jobs-ports, via: published_interface }
      - { module: jobs-usecases, via: published_interface }
      - { module: shared-spec, via: technical_shared }
      - { module: tenancy-public, via: published_interface }
  idgovernance-adapters:
    path: backend/idgovernance/adapters
    responsibility: "IdGovernance の HTTP と memory/PostgreSQL persistence adapter。lifecycle_workflows の postgres query/sqlcgen は本 context が所有する（ADR-117 が後続 WI へ後回しにしていた context-local sqlc 分割、ADR-090。旧 backend/idmanagement/adapters/persistence/postgres から物理移設）。"
    context: IdGovernance
    layer: adapters
    role: binding
    depends_on:
      - { module: application-ports, via: binding }
      - { module: authentication-ports, via: binding }
      - { module: http-support, via: binding }
      - { module: idgovernance-domain, via: published_interface }
      - { module: idgovernance-ports, via: published_interface }
      - { module: idgovernance-usecases, via: published_interface }
      - { module: idmanagement-group-ports, via: binding }
      - { module: idmanagement-user-adapters, via: binding }
      - { module: idmanagement-user-domain, via: binding }
      - { module: idmanagement-user-ports, via: binding }
      - { module: jobs-ports, via: binding }
      - { module: shared-adapters, via: binding }
      - { module: shared-spec, via: binding }
      - { module: tenancy-public, via: binding }
  jobs-domain:
    path: backend/jobs/domain
    responsibility: "Jobs のドメインモデルと純粋な規則。"
    context: Jobs
    layer: domain
    role: published_interface
  jobs-ports:
    path: backend/jobs/ports
    responsibility: "Jobs の公開 port と外界への抽象。"
    context: Jobs
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: jobs-domain, via: published_interface }
  jobs-usecases:
    path: backend/jobs/usecases

    responsibility: "Jobs のユースケース。"
    context: Jobs
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: jobs-domain, via: published_interface }
      - { module: jobs-ports, via: published_interface }
      - { module: shared-services, via: technical_shared }
      - { module: shared-spec, via: technical_shared }
  jobs-adapters:
    path: backend/jobs/adapters

    responsibility: "Jobs の HTTP・永続化 adapter。"
    context: Jobs
    layer: adapters
    role: binding
    depends_on:
      - { module: jobs-domain, via: published_interface }
      - { module: jobs-ports, via: published_interface }
      - { module: jobs-usecases, via: published_interface }
      - { module: shared-adapters, via: binding }
      - { module: shared-spec, via: binding }
  seeding-domain:
    path: backend/seeding/domain
    responsibility: "Seeding の環境 policy、profile、plan と純粋な検証規則。"
    context: Seeding
    layer: domain
    role: published_interface
    realizes:
      - { context: Seeding, kind: model, element: SeedRequest }
      - { context: Seeding, kind: model, element: SeedPlan }
  seeding-usecases:
    path: backend/seeding/usecases
    responsibility: "Seed request の安全な検証と、将来の context contributor orchestration を担う application service。"
    context: Seeding
    layer: use_cases
    role: published_interface
    realizes:
      - { context: Seeding, kind: interface, element: SeedData }
    depends_on:
      - { module: seeding-domain, via: published_interface }
  oauth2-domain:
    path: backend/oauth2/domain

    responsibility: "OAuth2 のドメインモデルと純粋な規則。"
    context: OAuth2
    layer: domain
    role: published_interface
    depends_on:
      - { module: shared-spec, via: technical_shared }
      - { module: signingkeys-domain, via: published_interface }
      - { module: tenancy-domain, via: published_interface }
  oauth2-ports:
    path: backend/oauth2/ports

    responsibility: "OAuth2 の公開 port と外界への抽象。"
    context: OAuth2
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: idmanagement-user-domain, via: published_interface }
      - { module: oauth2-domain, via: published_interface }
      - { module: shared-spec, via: technical_shared }
  oauth2-usecases:
    path: backend/oauth2/usecases

    responsibility: "OAuth2 のユースケース。"
    context: OAuth2
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: idmanagement-user-domain, via: published_interface }
      - { module: idmanagement-user-ports, via: published_interface }
      - { module: oauth2-domain, via: published_interface }
      - { module: oauth2-ports, via: published_interface }
      - { module: shared-kernel, via: technical_shared }
      - { module: shared-spec, via: technical_shared }
      - { module: signingkeys-domain, via: published_interface }
      - { module: tenancy-domain, via: published_interface }
      - { module: tenancy-ports, via: published_interface }
      - { module: tenancy-public, via: published_interface }
  oauth2-adapters:
    path: backend/oauth2/adapters

    responsibility: "OAuth2 の HTTP・永続化 adapter。"
    context: OAuth2
    layer: adapters
    role: binding
    depends_on:
      - { module: application-domain, via: binding }
      - { module: application-usecases, via: binding }
      - { module: audit-ports, via: binding }
      - { module: authentication-domain, via: binding }
      - { module: authentication-mfa-domain, via: binding }
      - { module: authentication-mfa-ports, via: binding }
      - { module: authentication-mfa-usecases, via: binding }
      - { module: authentication-password-ports, via: binding }
      - { module: authentication-password-usecases, via: binding }
      - { module: authentication-ports, via: binding }
      - { module: authentication-recovery-ports, via: binding }
      - { module: authentication-recovery-usecases, via: binding }
      - { module: authentication-session-ports, via: binding }
      - { module: authentication-session-usecases, via: binding }
      - { module: authentication-totp-ports, via: binding }
      - { module: authentication-totp-usecases, via: binding }
      - { module: authentication-usecases, via: binding }
      - { module: authentication-webauthn-ports, via: binding }
      - { module: authentication-webauthn-usecases, via: binding }
      - { module: http-support, via: binding }
      - { module: idmanagement-agent-ports, via: binding }
      - { module: idmanagement-domain, via: binding }
      - { module: idmanagement-user-domain, via: binding }
      - { module: idmanagement-user-ports, via: binding }
      - { module: oauth2-domain, via: published_interface }
      - { module: oauth2-ports, via: published_interface }
      - { module: oauth2-usecases, via: published_interface }
      - { module: shared-adapters, via: binding }
      - { module: shared-spec, via: binding }
      - { module: signingkeys-domain, via: published_interface }
      - { module: signingkeys-ports, via: published_interface }
      - { module: tenancy-domain, via: binding }
      - { module: tenancy-ports, via: binding }
      - { module: tenancy-public, via: binding }
  saml-domain:
    path: backend/saml/domain

    responsibility: "Saml のドメインモデルと純粋な規則。"
    context: Saml
    layer: domain
    role: published_interface
    depends_on:
      - { module: claimmapping-domain, via: published_interface }
      - { module: shared-spec, via: technical_shared }
  saml-ports:
    path: backend/saml/ports
    responsibility: "Saml の公開 port と外界への抽象。"
    context: Saml
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: saml-domain, via: published_interface }
  saml-usecases:
    path: backend/saml/usecases

    responsibility: "Saml のユースケース。"
    context: Saml
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: claimmapping-usecases, via: published_interface }
      - { module: application-domain, via: published_interface }
      - { module: authentication-domain, via: published_interface }
      - { module: idmanagement-user-ports, via: published_interface }
      - { module: saml-domain, via: published_interface }
      - { module: saml-ports, via: published_interface }
      - { module: shared-spec, via: technical_shared }
      - { module: wsfederation-domain, via: published_interface }
  saml-adapters:
    path: backend/saml/adapters

    responsibility: "Saml の HTTP・永続化 adapter。"
    context: Saml
    layer: adapters
    role: binding
    depends_on:
      - { module: application-domain, via: binding }
      - { module: authentication-domain, via: binding }
      - { module: authentication-session-usecases, via: binding }
      - { module: authentication-usecases, via: binding }
      - { module: claimmapping-domain, via: published_interface }
      - { module: claimmapping-usecases, via: published_interface }
      - { module: http-support, via: binding }
      - { module: idmanagement-user-ports, via: binding }
      - { module: saml-domain, via: published_interface }
      - { module: saml-ports, via: published_interface }
      - { module: saml-usecases, via: published_interface }
      - { module: shared-adapters, via: binding }
      - { module: shared-kernel, via: binding }
      - { module: shared-spec, via: binding }
      - { module: wsfederation-adapters, via: binding }
      - { module: wsfederation-domain, via: binding }
  scim-domain:
    path: backend/scim/domain
    responsibility: "Scim のドメインモデルと純粋な規則。"
    context: Scim
    layer: domain
    role: published_interface
  scim-ports:
    path: backend/scim/ports
    responsibility: "Scim の公開 port と外界への抽象。"
    context: Scim
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: scim-domain, via: published_interface }
  scim-usecases:
    path: backend/scim/usecases

    responsibility: "Scim のユースケース。"
    context: Scim
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: idmanagement-domain, via: published_interface }
      - { module: idmanagement-group-domain, via: published_interface }
      - { module: idmanagement-group-ports, via: published_interface }
      - { module: idmanagement-user-domain, via: published_interface }
      - { module: idmanagement-user-ports, via: published_interface }
      - { module: scim-domain, via: published_interface }
      - { module: scim-ports, via: published_interface }
      - { module: shared-spec, via: technical_shared }
  scim-adapters:
    path: backend/scim/adapters

    responsibility: "Scim の HTTP・永続化 adapter。"
    context: Scim
    layer: adapters
    role: binding
    depends_on:
      - { module: http-support, via: binding }
      - { module: scim-domain, via: published_interface }
      - { module: scim-ports, via: published_interface }
      - { module: scim-usecases, via: published_interface }
      - { module: shared-adapters, via: binding }
      - { module: shared-kernel, via: binding }
  provisioning-domain:
    path: backend/provisioning/domain
    responsibility: "Provisioning のドメインモデルと純粋な規則。"
    context: Provisioning
    layer: domain
    role: published_interface
  provisioning-ports:
    path: backend/provisioning/ports
    responsibility: "Provisioning の公開 port と外界への抽象。"
    context: Provisioning
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: provisioning-domain, via: published_interface }
  provisioning-scim:
    path: backend/provisioning/scim
    responsibility: "SCIM プロトコル別 outbound feature slice (ADR-128 決定2)。"
    context: Provisioning
    layer: adapters
    role: binding
    depends_on:
      - { module: provisioning-domain, via: published_interface }
      - { module: provisioning-ports, via: published_interface }
  provisioning-usecases:
    path: backend/provisioning/usecases
    responsibility: "Provisioning のユースケース (capture/dispatch/deliver/quarantine と admin 操作)。"
    context: Provisioning
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: application-domain, via: published_interface }
      - { module: application-ports, via: published_interface }
      - { module: idmanagement-user-ports, via: published_interface }
      - { module: jobs-domain, via: published_interface }
      - { module: jobs-usecases, via: published_interface }
      - { module: provisioning-domain, via: published_interface }
      - { module: provisioning-ports, via: published_interface }
      - { module: shared-spec, via: technical_shared }
  provisioning-adapters:
    path: backend/provisioning/adapters
    responsibility: "Provisioning の HTTP・永続化・IdManagement 属性取得 adapter。"
    context: Provisioning
    layer: adapters
    role: binding
    depends_on:
      - { module: application-ports, via: binding }
      - { module: http-support, via: binding }
      - { module: idmanagement-domain, via: binding }
      - { module: idmanagement-user-ports, via: published_interface }
      - { module: provisioning-domain, via: published_interface }
      - { module: provisioning-ports, via: published_interface }
      - { module: provisioning-usecases, via: published_interface }
      - { module: shared-adapters, via: binding }
  provisioning-composition:
    path: backend/provisioning/
    responsibility: "Provisioning の adapter と port を束ねる composition module (Module 型、cross-context notifier/job handler の配線)。"
    context: Provisioning
    layer: infrastructure
    role: composition_root
    depends_on:
      - { module: application-ports, via: published_interface }
      - { module: http-support, via: composition_root }
      - { module: idmanagement-user-ports, via: published_interface }
      - { module: jobs-ports, via: published_interface }
      - { module: jobs-usecases, via: published_interface }
      - { module: provisioning-adapters, via: composition_root }
      - { module: provisioning-domain, via: published_interface }
      - { module: provisioning-ports, via: published_interface }
      - { module: provisioning-scim, via: published_interface }
      - { module: provisioning-usecases, via: published_interface }
  tenancy-domain:
    path: backend/tenancy/domain

    responsibility: "Tenancy のドメインモデルと純粋な規則。"
    context: Tenancy
    layer: domain
    role: published_interface
    depends_on:
      - { module: shared-kernel, via: technical_shared }
      - { module: shared-spec, via: technical_shared }
  tenancy-ports:
    path: backend/tenancy/ports

    responsibility: "Tenancy の公開 port と外界への抽象。"
    context: Tenancy
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: idmanagement-user-domain, via: published_interface }
      - { module: tenancy-domain, via: published_interface }
  tenancy-usecases:
    path: backend/tenancy/usecases

    responsibility: "Tenancy のユースケース。"
    context: Tenancy
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: idmanagement-user-domain, via: published_interface }
      - { module: shared-services, via: technical_shared }
      - { module: shared-spec, via: technical_shared }
      - { module: tenancy-domain, via: published_interface }
      - { module: tenancy-ports, via: published_interface }
  tenancy-adapters:
    path: backend/tenancy/adapters

    responsibility: "Tenancy の HTTP・永続化 adapter。"
    context: Tenancy
    layer: adapters
    role: binding
    depends_on:
      - { module: authentication-password-usecases, via: binding }
      - { module: authentication-usecases, via: binding }
      - { module: http-support, via: binding }
      - { module: idmanagement-domain, via: binding }
      - { module: idmanagement-group-ports, via: binding }
      - { module: idmanagement-user-domain, via: binding }
      - { module: idmanagement-user-ports, via: binding }
      - { module: shared-adapters, via: binding }
      - { module: shared-spec, via: binding }
      - { module: tenancy-domain, via: published_interface }
      - { module: tenancy-ports, via: published_interface }
      - { module: tenancy-usecases, via: published_interface }
  wsfederation-domain:
    path: backend/wsfederation/domain

    responsibility: "WsFederation のドメインモデルと純粋な規則。"
    context: WsFederation
    layer: domain
    role: published_interface
    depends_on:
      - { module: claimmapping-domain, via: published_interface }
      - { module: idmanagement-domain, via: published_interface }
      - { module: shared-spec, via: technical_shared }
  wsfederation-ports:
    path: backend/wsfederation/ports
    responsibility: "WsFederation の公開 port と外界への抽象。"
    context: WsFederation
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: wsfederation-domain, via: published_interface }
  wsfederation-usecases:
    path: backend/wsfederation/usecases

    responsibility: "WsFederation のユースケース。"
    context: WsFederation
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: claimmapping-usecases, via: published_interface }
      - { module: application-domain, via: published_interface }
      - { module: authentication-domain, via: published_interface }
      - { module: idmanagement-user-domain, via: published_interface }
      - { module: idmanagement-user-ports, via: published_interface }
      - { module: shared-spec, via: technical_shared }
      - { module: wsfederation-domain, via: published_interface }
      - { module: wsfederation-ports, via: published_interface }
  wsfederation-adapters:
    path: backend/wsfederation/adapters

    responsibility: "WsFederation の HTTP・永続化 adapter。"
    context: WsFederation
    layer: adapters
    role: binding
    depends_on:
      - { module: claimmapping-usecases, via: published_interface }
      - { module: claimmapping-domain, via: published_interface }
      - { module: application-domain, via: binding }
      - { module: authentication-domain, via: binding }
      - { module: authentication-password-ports, via: binding }
      - { module: authentication-ports, via: binding }
      - { module: authentication-session-ports, via: binding }
      - { module: authentication-session-usecases, via: binding }
      - { module: authentication-usecases, via: binding }
      - { module: http-support, via: binding }
      - { module: idmanagement-user-domain, via: binding }
      - { module: idmanagement-user-ports, via: binding }
      - { module: oauth2-ports, via: binding }
      - { module: shared-adapters, via: binding }
      - { module: shared-kernel, via: binding }
      - { module: shared-spec, via: binding }
      - { module: wsfederation-domain, via: published_interface }
      - { module: wsfederation-ports, via: published_interface }
      - { module: wsfederation-usecases, via: published_interface }
  application-public:
    path: backend/application/
    responsibility: "Application root package の公開 facade。"
    context: Application
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: application-domain, via: published_interface }
  application-composition:
    path: backend/application/module.go

    responsibility: "Application の adapter と port を束ねる composition module。"
    context: Application
    layer: infrastructure
    role: composition_root
    depends_on:
      - { module: application-adapters, via: published_interface }
      - { module: application-domain, via: published_interface }
      - { module: application-ports, via: published_interface }
      - { module: application-usecases, via: published_interface }
      - { module: http-support, via: composition_root }
      - { module: idmanagement-group-ports, via: composition_root }
      - { module: idmanagement-user-ports, via: composition_root }
      - { module: oauth2-ports, via: composition_root }
      - { module: saml-ports, via: composition_root }
      - { module: wsfederation-ports, via: composition_root }
  audit-public:
    path: backend/audit/
    responsibility: "Audit root package の公開 facade。"
    context: Audit
    layer: use_cases
    role: published_interface
  audit-composition:
    path: backend/audit/module.go
    responsibility: "Audit の adapter と port を束ねる composition module。"
    context: Audit
    layer: infrastructure
    role: composition_root
    depends_on:
      - { module: audit-adapters, via: published_interface }
      - { module: audit-ports, via: published_interface }
      - { module: audit-usecases, via: published_interface }
  authentication-public:
    path: backend/authentication/
    responsibility: "Authentication root package の公開 facade。"
    context: Authentication
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: authentication-domain, via: published_interface }
  authentication-composition:
    path: backend/authentication/module.go
    responsibility: "Authentication の adapter と port を束ねる composition module。"
    context: Authentication
    layer: infrastructure
    role: composition_root
    depends_on:
      - { module: authentication-adapters, via: published_interface }
      - { module: authentication-domain, via: published_interface }
      - { module: authentication-mfa-ports, via: published_interface }
      - { module: authentication-password-ports, via: published_interface }
      - { module: authentication-ports, via: published_interface }
      - { module: authentication-recovery-ports, via: published_interface }
      - { module: authentication-session-ports, via: published_interface }
      - { module: authentication-session-usecases, via: published_interface }
      - { module: authentication-totp-ports, via: published_interface }
      - { module: authentication-usecases, via: published_interface }
      - { module: authentication-webauthn-ports, via: published_interface }
  idmanagement-public:
    path: backend/idmanagement/
    responsibility: "IdManagement root package の公開 facade。"
    context: IdManagement
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: idmanagement-domain, via: published_interface }
  idmanagement-composition:
    path: backend/idmanagement/module.go
    responsibility: "IdManagement の adapter と port を束ねる composition module。feature に分割せず 1 つの Module に束ねる (ADR-091, ADR-130)。"
    context: IdManagement
    layer: infrastructure
    role: composition_root
    depends_on:
      - { module: idmanagement-agent-ports, via: published_interface }
      - { module: idmanagement-group-ports, via: published_interface }
      - { module: idmanagement-user-ports, via: published_interface }
  idgovernance-public:
    path: backend/idgovernance/
    responsibility: "IdGovernance root package の公開 facade。"
    context: IdGovernance
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: idgovernance-ports, via: published_interface }
  idgovernance-composition:
    path: backend/idgovernance/module.go
    responsibility: "IdGovernance の adapter と port を束ねる composition module。"
    context: IdGovernance
    layer: infrastructure
    role: composition_root
    depends_on:
      - { module: idgovernance-adapters, via: published_interface }
      - { module: idgovernance-ports, via: published_interface }
      - { module: idmanagement-user-ports, via: published_interface }
  jobs-public:
    path: backend/jobs/
    responsibility: "Jobs root package の公開 facade。"
    context: Jobs
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: jobs-domain, via: published_interface }
  jobs-composition:
    path: backend/jobs/module.go
    responsibility: "Jobs の adapter と port を束ねる composition module。"
    context: Jobs
    layer: infrastructure
    role: composition_root
    depends_on:
      - { module: jobs-adapters, via: published_interface }
      - { module: jobs-domain, via: published_interface }
      - { module: jobs-ports, via: published_interface }
      - { module: jobs-usecases, via: published_interface }
  oauth2-public:
    path: backend/oauth2/
    responsibility: "OAuth2 root package の公開 facade。"
    context: OAuth2
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: oauth2-domain, via: published_interface }
  oauth2-composition:
    path: backend/oauth2/module.go
    responsibility: "OAuth2 の adapter と port を束ねる composition module。"
    context: OAuth2
    layer: infrastructure
    role: composition_root
    depends_on:
      - { module: oauth2-adapters, via: published_interface }
      - { module: oauth2-domain, via: published_interface }
      - { module: oauth2-ports, via: published_interface }
      - { module: oauth2-usecases, via: published_interface }
  saml-public:
    path: backend/saml/
    responsibility: "Saml root package の公開 facade。"
    context: Saml
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: saml-domain, via: published_interface }
  saml-composition:
    path: backend/saml/module.go

    responsibility: "Saml の adapter と port を束ねる composition module。"
    context: Saml
    layer: infrastructure
    role: composition_root
    depends_on:
      - { module: http-support, via: composition_root }
      - { module: idmanagement-user-ports, via: composition_root }
      - { module: saml-adapters, via: published_interface }
      - { module: saml-domain, via: published_interface }
      - { module: saml-ports, via: published_interface }
      - { module: saml-usecases, via: published_interface }
      - { module: wsfederation-adapters, via: composition_root }
  scim-public:
    path: backend/scim/
    responsibility: "Scim root package の公開 facade。"
    context: Scim
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: scim-domain, via: published_interface }
  scim-composition:
    path: backend/scim/module.go

    responsibility: "Scim の adapter と port を束ねる composition module。"
    context: Scim
    layer: infrastructure
    role: composition_root
    depends_on:
      - { module: http-support, via: composition_root }
      - { module: idmanagement-group-ports, via: composition_root }
      - { module: idmanagement-user-ports, via: composition_root }
      - { module: scim-adapters, via: published_interface }
      - { module: scim-domain, via: published_interface }
      - { module: scim-ports, via: published_interface }
      - { module: scim-usecases, via: published_interface }
      - { module: shared-spec, via: composition_root }
  tenancy-public:
    path: backend/tenancy/
    responsibility: "Tenancy root package の公開 facade。"
    context: Tenancy
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: tenancy-domain, via: published_interface }
  tenancy-composition:
    path: backend/tenancy/module.go
    responsibility: "Tenancy の adapter と port を束ねる composition module。"
    context: Tenancy
    layer: infrastructure
    role: composition_root
    depends_on:
      - { module: tenancy-adapters, via: published_interface }
      - { module: tenancy-domain, via: published_interface }
      - { module: tenancy-ports, via: published_interface }
      - { module: tenancy-usecases, via: published_interface }
  wsfederation-public:
    path: backend/wsfederation/
    responsibility: "WsFederation root package の公開 facade。"
    context: WsFederation
    layer: use_cases
    role: published_interface
    depends_on:
      - { module: wsfederation-domain, via: published_interface }
  wsfederation-composition:
    path: backend/wsfederation/module.go

    responsibility: "WsFederation の adapter と port を束ねる composition module。"
    context: WsFederation
    layer: infrastructure
    role: composition_root
    depends_on:
      - { module: authentication-password-ports, via: composition_root }
      - { module: authentication-ports, via: composition_root }
      - { module: authentication-session-ports, via: composition_root }
      - { module: http-support, via: composition_root }
      - { module: idmanagement-user-ports, via: composition_root }
      - { module: oauth2-ports, via: composition_root }
      - { module: wsfederation-adapters, via: published_interface }
      - { module: wsfederation-domain, via: published_interface }
      - { module: wsfederation-ports, via: published_interface }
      - { module: wsfederation-usecases, via: published_interface }
  shared-kernel:
    path: backend/shared/kernel
    responsibility: "context 横断の最小共有ドメイン語彙。"
    context: System
    layer: domain
    role: technical_shared
  shared-spec:
    path: backend/shared/spec

    responsibility: "SCL の Go binding と仕様整合検査。"
    context: System
    layer: domain
    role: technical_shared
    depends_on:
      - { module: shared-kernel, via: technical_shared }
  shared-adapters:
    path: backend/shared/adapters

    responsibility: "HTTP、crypto、persistence、observability の共有 adapter。"
    context: System
    layer: adapters
    role: technical_shared
    depends_on:
      - { module: authentication-ports, via: published_interface }
      - { module: http-support, via: technical_shared }
      - { module: idmanagement-group-domain, via: published_interface }
      - { module: idmanagement-user-domain, via: published_interface }
      - { module: jobs-domain, via: published_interface }
      - { module: oauth2-domain, via: published_interface }
      - { module: oauth2-ports, via: published_interface }
      - { module: shared-kernel, via: technical_shared }
      - { module: shared-services, via: technical_shared }
      - { module: shared-spec, via: technical_shared }
      - { module: signingkeys-domain, via: published_interface }
      - { module: signingkeys-ports, via: published_interface }
      - { module: tenancy-domain, via: published_interface }
      - { module: tenancy-public, via: published_interface }
  http-support:
    path: backend/shared/adapters/http/support

    responsibility: "context 横断 HTTP handler binding と request support。"
    context: System
    layer: adapters
    role: technical_shared
    depends_on:
      - { module: application-domain, via: published_interface }
      - { module: application-ports, via: published_interface }
      - { module: application-usecases, via: published_interface }
      - { module: authentication-domain, via: published_interface }
      - { module: authentication-session-usecases, via: published_interface }
      - { module: authentication-usecases, via: published_interface }
      - { module: idmanagement-group-domain, via: published_interface }
      - { module: idmanagement-group-ports, via: published_interface }
      - { module: idmanagement-user-domain, via: published_interface }
      - { module: idmanagement-user-ports, via: published_interface }
      - { module: oauth2-ports, via: published_interface }
      - { module: oauth2-usecases, via: published_interface }
      - { module: shared-kernel, via: technical_shared }
      - { module: shared-services, via: technical_shared }
      - { module: shared-spec, via: technical_shared }
      - { module: tenancy-domain, via: published_interface }
      - { module: tenancy-ports, via: published_interface }
      - { module: tenancy-public, via: published_interface }
  http-server:
    path: backend/shared/adapters/http/server

    responsibility: "context route を API runtime へ束ねる composition module。"
    context: System
    layer: infrastructure
    role: composition_root
    depends_on:
      - { module: application-public, via: composition_root }
      - { module: audit-adapters, via: composition_root }
      - { module: audit-public, via: composition_root }
      - { module: authentication-adapters, via: composition_root }
      - { module: authentication-domain, via: composition_root }
      - { module: authentication-password-ports, via: composition_root }
      - { module: authentication-ports, via: composition_root }
      - { module: authentication-public, via: composition_root }
      - { module: authentication-recovery-ports, via: composition_root }
      - { module: authentication-session-usecases, via: composition_root }
      - { module: authentication-totp-ports, via: composition_root }
      - { module: authentication-usecases, via: composition_root }
      - { module: authentication-webauthn-ports, via: composition_root }
      - { module: http-support, via: technical_shared }
      - { module: idgovernance-adapters, via: composition_root }
      - { module: idgovernance-public, via: composition_root }
      - { module: idmanagement-adapters, via: composition_root }
      - { module: idmanagement-agent-ports, via: composition_root }
      - { module: idmanagement-group-ports, via: composition_root }
      - { module: idmanagement-public, via: composition_root }
      - { module: idmanagement-user-ports, via: composition_root }
      - { module: jobs-public, via: composition_root }
      - { module: oauth2-adapters, via: composition_root }
      - { module: oauth2-ports, via: composition_root }
      - { module: oauth2-public, via: composition_root }
      - { module: provisioning-composition, via: composition_root }
      - { module: saml-public, via: composition_root }
      - { module: scim-public, via: composition_root }
      - { module: shared-adapters, via: technical_shared }
      - { module: signingkeys-adapters, via: composition_root }
      - { module: signingkeys-public, via: composition_root }
      - { module: signingkeys-ports, via: composition_root }
      - { module: tenancy-adapters, via: composition_root }
      - { module: tenancy-domain, via: composition_root }
      - { module: tenancy-ports, via: composition_root }
      - { module: tenancy-public, via: composition_root }
      - { module: wsfederation-adapters, via: composition_root }
      - { module: wsfederation-public, via: composition_root }
  shared-services:
    path: backend/shared
    responsibility: "共有 logging、resilience、media validation、version capability。"
    context: System
    layer: use_cases
    role: technical_shared
    depends_on:
      - { module: shared-kernel, via: technical_shared }
      - { module: shared-spec, via: technical_shared }
  backend:
    path: backend

    responsibility: "Go bounded contexts を組み立てる composition root。"
    context: System
    layer: infrastructure
    role: composition_root
    depends_on:
      - { module: authentication-ports, via: composition_root }
      - { module: authentication-session-ports, via: composition_root }
      - { module: authentication-session-usecases, via: composition_root }
      - { module: authentication-usecases, via: composition_root }
      - { module: bootstrap, via: published_interface }
      - { module: http-server, via: published_interface }
      - { module: http-support, via: technical_shared }
      - { module: idmanagement-usecases, via: composition_root }
      - { module: idgovernance-public, via: composition_root }
      - { module: idgovernance-usecases, via: composition_root }
      - { module: jobs-domain, via: composition_root }
      - { module: jobs-public, via: composition_root }
      - { module: jobs-usecases, via: composition_root }
      - { module: shared-adapters, via: technical_shared }
      - { module: shared-services, via: technical_shared }
      - { module: shared-spec, via: technical_shared }
      - { module: seeding-domain, via: composition_root }
      - { module: tenancy-usecases, via: composition_root }
  batch:
    path: backend/cmd/idmagic-batch
    responsibility: "外部 scheduler から one-shot で起動する横断 batch composition root。"
    context: System
    layer: infrastructure
    role: composition_root
    depends_on:
      - { module: bootstrap, via: composition_root }
      - { module: shared-services, via: technical_shared }
      - { module: signingkeys-usecases, via: composition_root }
      - { module: tenancy-public, via: composition_root }
  worker:
    path: backend/cmd/idmagic-worker
    responsibility: "durable job queue の claim・handler 実行を担う API 分離 worker composition root (ADR-099)。"
    context: System
    layer: infrastructure
    role: composition_root
    depends_on:
      - { module: bootstrap, via: composition_root }
      - { module: idgovernance-usecases, via: composition_root }
      - { module: idmanagement-group-usecases, via: composition_root }
      - { module: idmanagement-user-usecases, via: composition_root }
      - { module: jobs-domain, via: composition_root }
      - { module: jobs-ports, via: composition_root }
      - { module: jobs-public, via: composition_root }
      - { module: jobs-usecases, via: composition_root }
      - { module: provisioning-adapters, via: composition_root }
      - { module: provisioning-composition, via: composition_root }
      - { module: provisioning-usecases, via: composition_root }
      - { module: shared-adapters, via: technical_shared }
      - { module: shared-services, via: technical_shared }
      - { module: shared-spec, via: technical_shared }
  bootstrap:
    path: backend/cmd/internal/bootstrap

    responsibility: "API runtime の依存注入と runtime adapter 選択。"
    context: System
    layer: infrastructure
    role: composition_root
    depends_on:
      - { module: application-adapters, via: composition_root }
      - { module: application-domain, via: composition_root }
      - { module: application-ports, via: composition_root }
      - { module: application-public, via: composition_root }
      - { module: audit-adapters, via: composition_root }
      - { module: audit-ports, via: composition_root }
      - { module: audit-public, via: composition_root }
      - { module: audit-usecases, via: composition_root }
      - { module: authentication-adapters, via: composition_root }
      - { module: authentication-domain, via: composition_root }
      - { module: authentication-mfa-adapters, via: composition_root }
      - { module: authentication-password-adapters, via: composition_root }
      - { module: authentication-password-ports, via: composition_root }
      - { module: authentication-password-usecases, via: composition_root }
      - { module: authentication-ports, via: composition_root }
      - { module: authentication-public, via: composition_root }
      - { module: authentication-recovery-adapters, via: composition_root }
      - { module: authentication-session-adapters, via: composition_root }
      - { module: authentication-session-ports, via: composition_root }
      - { module: authentication-totp-adapters, via: composition_root }
      - { module: authentication-totp-domain, via: composition_root }
      - { module: authentication-totp-ports, via: composition_root }
      - { module: authentication-usecases, via: composition_root }
      - { module: authentication-webauthn-adapters, via: composition_root }
      - { module: authentication-webauthn-usecases, via: composition_root }
      - { module: claimmapping-domain, via: composition_root }
      - { module: http-support, via: technical_shared }
      - { module: idgovernance-adapters, via: composition_root }
      - { module: idgovernance-composition, via: composition_root }
      - { module: idgovernance-domain, via: composition_root }
      - { module: idgovernance-ports, via: composition_root }
      - { module: idgovernance-public, via: composition_root }
      - { module: idgovernance-usecases, via: composition_root }
      - { module: idmanagement-adapters, via: composition_root }
      - { module: idmanagement-agent-adapters, via: composition_root }
      - { module: idmanagement-domain, via: composition_root }
      - { module: idmanagement-group-adapters, via: composition_root }
      - { module: idmanagement-group-domain, via: composition_root }
      - { module: idmanagement-group-ports, via: composition_root }
      - { module: idmanagement-public, via: composition_root }
      - { module: idmanagement-user-adapters, via: composition_root }
      - { module: idmanagement-user-domain, via: composition_root }
      - { module: idmanagement-user-ports, via: composition_root }
      - { module: jobs-adapters, via: composition_root }
      - { module: jobs-public, via: composition_root }
      - { module: oauth2-adapters, via: composition_root }
      - { module: oauth2-domain, via: composition_root }
      - { module: oauth2-ports, via: composition_root }
      - { module: oauth2-public, via: composition_root }
      - { module: provisioning-adapters, via: composition_root }
      - { module: provisioning-composition, via: composition_root }
      - { module: provisioning-usecases, via: composition_root }
      - { module: saml-adapters, via: composition_root }
      - { module: saml-domain, via: composition_root }
      - { module: saml-ports, via: composition_root }
      - { module: saml-public, via: composition_root }
      - { module: scim-adapters, via: composition_root }
      - { module: scim-public, via: composition_root }
      - { module: seeding-domain, via: composition_root }
      - { module: seeding-usecases, via: composition_root }
      - { module: shared-adapters, via: technical_shared }
      - { module: shared-services, via: technical_shared }
      - { module: shared-spec, via: technical_shared }
      - { module: signingkeys-adapters, via: composition_root }
      - { module: signingkeys-domain, via: composition_root }
      - { module: signingkeys-ports, via: composition_root }
      - { module: signingkeys-public, via: composition_root }
      - { module: tenancy-adapters, via: composition_root }
      - { module: tenancy-domain, via: composition_root }
      - { module: tenancy-public, via: composition_root }
      - { module: tenancy-usecases, via: composition_root }
      - { module: wsfederation-adapters, via: composition_root }
      - { module: wsfederation-domain, via: composition_root }
      - { module: wsfederation-ports, via: composition_root }
      - { module: wsfederation-public, via: composition_root }
  frontend-lib:
    path: frontend/src/lib

    responsibility: "i18n、validation、navigation 等の browser 共有 capability。"
    context: System
    layer: adapters
    role: technical_shared
  frontend-i18n:
    path: frontend/src/lib/i18n
    responsibility: "browser UI の locale と翻訳辞書基盤。"
    context: System
    layer: adapters
    role: technical_shared
  frontend-utils:
    path: frontend/src/lib/utils.ts
    responsibility: "UI 表示値の純粋な変換 helper。"
    context: System
    layer: adapters
    role: technical_shared
    depends_on:
      - { module: frontend-i18n, via: technical_shared }
      - { module: frontend-types, via: technical_shared }
  frontend-types:
    path: frontend/src/types.ts

    responsibility: "browser UI の共有 wire/value 型。"
    context: System
    layer: adapters
    role: technical_shared
    depends_on:
      - { module: frontend-lib, via: technical_shared }
      - { module: frontend-i18n, via: technical_shared }
  frontend-admin-nav:
    path: frontend/src/lib/adminNav.ts

    responsibility: "管理 UI navigation の組み立て。"
    context: System
    layer: adapters
    role: binding
    depends_on:
      - { module: frontend-api, via: published_interface }
      - { module: frontend-lib, via: technical_shared }
      - { module: frontend-shell-i18n, via: technical_shared }
      - { module: frontend-i18n, via: technical_shared }
  frontend-system-nav:
    path: frontend/src/lib/systemNav.ts

    responsibility: "system UI navigation の組み立て。"
    context: System
    layer: adapters
    role: binding
    depends_on:
      - { module: frontend-lib, via: technical_shared }
      - { module: frontend-shell-i18n, via: technical_shared }
      - { module: frontend-i18n, via: technical_shared }
  frontend-branding-hook:
    path: frontend/src/lib/useTenantBranding.ts
    responsibility: "tenant branding API を React state へ結ぶ hook。"
    context: System
    layer: adapters
    role: binding
    depends_on:
      - { module: frontend-api, via: published_interface }
      - { module: frontend-types, via: technical_shared }
  frontend-shell-i18n:
    path: frontend/src/components/shell.i18n.ts

    responsibility: "UI shell の翻訳辞書。"
    context: System
    layer: adapters
    role: technical_shared
    depends_on:
      - { module: frontend-lib, via: technical_shared }
      - { module: frontend-i18n, via: technical_shared }
  frontend-router:
    path: frontend/src/router.tsx

    responsibility: "TanStack Router と browser error boundary。"
    context: System
    layer: adapters
    role: technical_shared
    depends_on:
      - { module: frontend-api, via: published_interface }
      - { module: frontend-lib, via: technical_shared }
      - { module: frontend-routes, via: published_interface }
      - { module: frontend-i18n, via: technical_shared }
  frontend-api:
    path: frontend/src/api

    responsibility: "browser UI と Go API の wire binding。"
    context: System
    layer: adapters
    role: binding
    depends_on:
      - { module: frontend-lib, via: technical_shared }
      - { module: frontend-types, via: technical_shared }
      - { module: frontend-i18n, via: technical_shared }
  frontend-components:
    path: frontend/src/components

    responsibility: "UI shell と共有 presentation component。"
    context: System
    layer: adapters
    role: technical_shared
    depends_on:
      - { module: frontend-admin-nav, via: published_interface }
      - { module: frontend-api, via: published_interface }
      - { module: frontend-lib, via: technical_shared }
      - { module: frontend-branding-hook, via: published_interface }
      - { module: frontend-shell-i18n, via: technical_shared }
      - { module: frontend-system-nav, via: published_interface }
      - { module: frontend-i18n, via: technical_shared }
      - { module: frontend-utils, via: technical_shared }
  frontend-features:
    path: frontend/src/features

    responsibility: "管理・self-service・認証 UI feature。"
    context: System
    layer: adapters
    role: binding
    depends_on:
      - { module: frontend-api, via: published_interface }
      - { module: frontend-components, via: technical_shared }
      - { module: frontend-lib, via: technical_shared }
      - { module: frontend-types, via: technical_shared }
      - { module: frontend-i18n, via: technical_shared }
      - { module: frontend-utils, via: technical_shared }
  frontend-routes:
    path: frontend/src/routes

    responsibility: "browser route と feature composition。"
    context: System
    layer: adapters
    role: binding
    depends_on:
      - { module: frontend-api, via: published_interface }
      - { module: frontend-components, via: technical_shared }
      - { module: frontend-features, via: published_interface }
      - { module: frontend-lib, via: technical_shared }
      - { module: frontend-types, via: technical_shared }
      - { module: frontend-i18n, via: technical_shared }
  frontend:
    path: frontend

    responsibility: "React browser runtime と配信設定。"
    context: System
    layer: infrastructure
    role: composition_root
    depends_on:
      - { module: frontend-api, via: published_interface }
      - { module: frontend-components, via: technical_shared }
      - { module: frontend-features, via: published_interface }
      - { module: frontend-lib, via: technical_shared }
      - { module: frontend-router, via: technical_shared }
      - { module: frontend-routes, via: published_interface }
      - { module: frontend-i18n, via: technical_shared }
  specification:
    path: spec
    responsibility: "IdMagic SCL の規範仕様と派生契約。"
    context: System
    layer: specification_core
    role: published_interface
  yaml-check-tool:
    path: tools/yaml-check
    responsibility: "SCL、Work Item、Architecture の schema と semantic 検査。"
    context: yaml-check
    layer: adapters
    role: technical_shared
    realizes:
      - { context: yaml-check, kind: interface, element: CheckYaml }
  scl-to-html-tool:
    path: tools/scl-to-html

    responsibility: "SCL と変更記録の HTML 描画。"
    context: scl-to-html
    layer: adapters
    role: technical_shared
    depends_on:
      - { module: yaml-check-tool, via: technical_shared }
  scl-to-jsonschema-tool:
    path: tools/scl-to-jsonschema

    responsibility: "SCL model の JSON Schema 生成。"
    context: scl-to-jsonschema
    layer: adapters
    role: technical_shared
    depends_on:
      - { module: scl-to-html-tool, via: technical_shared }
  scl-to-openapi-tool:
    path: tools/scl-to-openapi

    responsibility: "SCL interface の OpenAPI 生成。"
    context: scl-to-openapi
    layer: adapters
    role: technical_shared
    depends_on:
      - { module: scl-to-html-tool, via: technical_shared }
      - { module: scl-to-jsonschema-tool, via: technical_shared }
  ra-tools:
    path: tools/ra
    responsibility: "workspace 発見、検証、派生物生成の composition root。"
    context: ra
    layer: infrastructure
    role: composition_root
    realizes:
      - { context: ra, kind: interface, element: CheckTraceability }
      - { context: ra, kind: interface, element: CheckArchitecture }
    depends_on:
      - { module: scl-to-html-tool, via: composition_root }
      - { module: scl-to-jsonschema-tool, via: composition_root }
      - { module: scl-to-openapi-tool, via: composition_root }
      - { module: yaml-check-tool, via: composition_root }
  verification:
    path: verification
    responsibility: "SCL realization、check、revision 付き evidence の外部 binding。"
    context: ra
    layer: adapters
    role: binding
  deploy-infra:
    path: infra
    responsibility: "container、database schema、ローカル runtime 構成。"
    context: System
    layer: deploy_pipeline
    role: implementation
  load-testing:
    path: load/k6
    responsibility: "tenant-local OAuth flows の SLO smoke を実行する k6 運用資産。"
    context: System
    layer: deploy_pipeline
    role: implementation
    realizes:
      - { context: System, kind: scenario, element: "Operatorは分離された運用資産でSLOを検証する" }
runtime_units:
  idmagic-api:
    kind: api
    entrypoint: backend/cmd/idmagic/main.go
    modules: [backend, bootstrap]
  idmagic-worker:
    kind: worker
    entrypoint: backend/cmd/idmagic-worker/main.go
    modules: [worker, bootstrap, jobs-usecases, jobs-adapters, provisioning-usecases, provisioning-adapters]
  idmagic-batch:
    kind: batch
    entrypoint: backend/cmd/idmagic-batch/main.go
    modules: [batch, bootstrap, signingkeys-usecases]
  idmagic-relay:
    kind: relay
    entrypoint: backend/cmd/idmagic-relay/main.go
    modules: [backend, shared-adapters]
  idmagic-ui:
    kind: ui
    entrypoint: frontend/src/main.tsx
    modules: [frontend, frontend-routes, frontend-features, frontend-api, frontend-components]
complexity:
  budgets:
    - id: ui-page-lines
      include: ["frontend/src/**/*Page.tsx"]
      exclude: ["**/*.test.tsx", "**/*.spec.tsx", "**/routeTree.gen.ts"]
      metric: source_lines
      limit: 400
    - id: ui-page-local-state
      include: ["frontend/src/**/*Page.tsx"]
      exclude: ["**/*.test.tsx", "**/*.spec.tsx", "**/routeTree.gen.ts"]
      metric: react_local_state_hooks
      limit: 10
    - id: go-source-lines
      include: ["backend/**/*.go"]
      exclude: ["**/*_test.go", "**/generated/**", "**/sqlcgen/**"]
      metric: source_lines
      limit: 800
  debts:
    - id: wi45-ui-page-lines-admin-application-detail-page
      budget: ui-page-lines
      path: frontend/src/features/admin-applications/AdminApplicationDetailPage.tsx
      ceiling: 410
      owner: maintainers
      reason: "wi-45 T007b でプロビジョニング導線ボタンを追加し、既に上限ちょうどだった 400 行を 2 行超過。ProvisioningNavButton は既に AdminApplicationProvisioningShared.tsx へ抽出済みで追加分はこれ以上圧縮できない。wi-234 の list/detail 分割と合わせて解消する。"
      work_item: wi-234-complexity-ratchet
      expires_at: 2026-10-01
    - id: wi234-ui-page-lines-account-profile-page
      budget: ui-page-lines
      path: frontend/src/features/account/AccountProfilePage.tsx
      ceiling: 499
      owner: maintainers
      reason: "wi-234 で責務境界に沿って分割する既存超過。"
      work_item: wi-234-complexity-ratchet
      expires_at: 2026-10-01
    - id: wi234-ui-page-lines-account-security-page
      budget: ui-page-lines
      path: frontend/src/features/account/AccountSecurityPage.tsx
      ceiling: 699
      owner: maintainers
      reason: "wi-234 で責務境界に沿って分割する既存超過。"
      work_item: wi-234-complexity-ratchet
      expires_at: 2026-10-01
    - id: wi234-ui-page-lines-admin-application-edit-page
      budget: ui-page-lines
      path: frontend/src/features/admin-applications/AdminApplicationEditPage.tsx
      ceiling: 860
      owner: maintainers
      reason: "wi-234 T002 で list/detail/edit へ分割済み。edit は OIDC/WS-Fed/SAML/サインインポリシーの4プロトコル分岐フォームが同居し、残存超過はプロトコル単位のフォームセクション分割で解消する。"
      work_item: wi-234-complexity-ratchet
      expires_at: 2026-10-01
    - id: wi234-ui-page-lines-admin-audit-events-page
      budget: ui-page-lines
      path: frontend/src/features/admin-audit-events/AdminAuditEventsPage.tsx
      ceiling: 538
      owner: maintainers
      reason: "wi-234 で責務境界に沿って分割する既存超過。"
      work_item: wi-234-complexity-ratchet
      expires_at: 2026-10-01
    - id: wi234-ui-page-lines-admin-dashboard-page
      budget: ui-page-lines
      path: frontend/src/features/admin-dashboard/AdminDashboardPage.tsx
      ceiling: 483
      owner: maintainers
      reason: "wi-234 で責務境界に沿って分割する既存超過。"
      work_item: wi-234-complexity-ratchet
      expires_at: 2026-10-01
    - id: wi234-ui-page-lines-admin-sign-in-policy-page
      budget: ui-page-lines
      path: frontend/src/features/admin-sign-in-policy/AdminSignInPolicyPage.tsx
      ceiling: 457
      owner: maintainers
      reason: "wi-234 で責務境界に沿って分割する既存超過。"
      work_item: wi-234-complexity-ratchet
      expires_at: 2026-10-01
    - id: wi234-ui-page-lines-admin-tenant-attributes-page
      budget: ui-page-lines
      path: frontend/src/features/admin-tenants/AdminTenantAttributesPage.tsx
      ceiling: 432
      owner: maintainers
      reason: "wi-234 で責務境界に沿って分割する既存超過。"
      work_item: wi-234-complexity-ratchet
      expires_at: 2026-10-01
    - id: wi234-ui-page-lines-admin-users-list-page
      budget: ui-page-lines
      path: frontend/src/features/admin-users/AdminUsersListPage.tsx
      ceiling: 560
      owner: maintainers
      reason: "wi-234 T002 で list/detail/edit/create/import へ分割済み。list は一覧テーブルと右ペイン詳細 (UserDetails) を同居しており残存超過。"
      work_item: wi-234-complexity-ratchet
      expires_at: 2026-10-01
    - id: wi234-ui-page-lines-admin-user-detail-page
      budget: ui-page-lines
      path: frontend/src/features/admin-users/AdminUserDetailPage.tsx
      ceiling: 493
      owner: maintainers
      reason: "wi-234 T002 で分割済み。プロフィール/属性/ライフサイクル/ロールとグループの全網羅ビューで残存超過。wi-28 T007 で admin セッション管理 (UserSessionsSection) を追加しセッション/ロールとグループをそれぞれ独立したカードへ分離 (レイアウトフィードバック対応)、5行分増加 (ロジック本体は AdminUsersShared.tsx 側に分割済み)。"
      work_item: wi-234-complexity-ratchet
      expires_at: 2026-10-01
    - id: wi234-ui-page-lines-admin-user-edit-page
      budget: ui-page-lines
      path: frontend/src/features/admin-users/AdminUserEditPage.tsx
      ceiling: 489
      owner: maintainers
      reason: "wi-234 T002 で分割済み。属性エディタとロール変更確認ステップが同居し残存超過。"
      work_item: wi-234-complexity-ratchet
      expires_at: 2026-10-01
    - id: wi234-ui-page-lines-admin-user-import-page
      budget: ui-page-lines
      path: frontend/src/features/admin-users/AdminUserImportPage.tsx
      ceiling: 436
      owner: maintainers
      reason: "wi-234 T002 で分割済み。CSV import の dry-run/apply ウィザードとエラー表示 helper が同居し残存超過。"
      work_item: wi-234-complexity-ratchet
      expires_at: 2026-10-01
    - id: wi234-ui-page-lines-system-tenants-page
      budget: ui-page-lines
      path: frontend/src/features/system-tenants/SystemTenantsPage.tsx
      ceiling: 412
      owner: maintainers
      reason: "wi-234 で責務境界に沿って分割する既存超過。"
      work_item: wi-234-complexity-ratchet
      expires_at: 2026-10-01
    - id: wi234-ui-page-local-state-account-security-page
      budget: ui-page-local-state
      path: frontend/src/features/account/AccountSecurityPage.tsx
      ceiling: 11
      owner: maintainers
      reason: "wi-234 で責務境界に沿って分割する既存超過。"
      work_item: wi-234-complexity-ratchet
      expires_at: 2026-10-01
    - id: wi234-ui-page-local-state-admin-application-edit-page
      budget: ui-page-local-state
      path: frontend/src/features/admin-applications/AdminApplicationEditPage.tsx
      ceiling: 34
      owner: maintainers
      reason: "wi-234 T002 で list/detail/edit へ分割済み。edit の残存超過はプロトコル単位のフォームセクション分割で解消する。"
      work_item: wi-234-complexity-ratchet
      expires_at: 2026-10-01
    - id: wi234-ui-page-local-state-admin-group-edit-page
      budget: ui-page-local-state
      path: frontend/src/features/admin-groups/AdminGroupEditPage.tsx
      ceiling: 12
      owner: maintainers
      reason: "wi-234 T003 で list/detail/create/edit へ分割済み。edit は基本情報フォームと動的ルール編集/プレビューの状態が同居し残存超過。"
      work_item: wi-234-complexity-ratchet
      expires_at: 2026-10-01
    - id: wi234-ui-page-local-state-admin-sign-in-policy-page
      budget: ui-page-local-state
      path: frontend/src/features/admin-sign-in-policy/AdminSignInPolicyPage.tsx
      ceiling: 11
      owner: maintainers
      reason: "wi-234 で責務境界に沿って分割する既存超過。"
      work_item: wi-234-complexity-ratchet
      expires_at: 2026-10-01
    - id: wi234-ui-page-local-state-admin-user-edit-page
      budget: ui-page-local-state
      path: frontend/src/features/admin-users/AdminUserEditPage.tsx
      ceiling: 12
      owner: maintainers
      reason: "wi-234 T002 で分割済み。残存超過はプロフィール/属性/ロール変更確認の状態が同一 component に残るため。"
      work_item: wi-234-complexity-ratchet
      expires_at: 2026-10-01
    - id: wi234-ui-page-local-state-system-tenants-page
      budget: ui-page-local-state
      path: frontend/src/features/system-tenants/SystemTenantsPage.tsx
      ceiling: 12
      owner: maintainers
      reason: "wi-234 で責務境界に沿って分割する既存超過。"
      work_item: wi-234-complexity-ratchet
      expires_at: 2026-10-01
---

# Architecture: repo

## Overview

この文書は、AI エージェントが `idmagic` の変更に必要な文脈を小さく取得するための索引である。人間向けの包括的な設計説明ではない。詳細な仕様は SCL、判断理由は ADR、完了済みの変更履歴は work item を読む。

更新コストを抑えるため、ここには頻繁に増減するエンドポイント一覧・フィールド一覧・画面一覧を置かない。それらはコード、`spec/contexts/*.yaml`、`README.md`、UI 側の文書を正とする。

## Structure

```text
.
├── backend/       # Go bounded contexts（claimmapping / signingkeys を含む）、shared、cmd/
├── frontend/      # React UI と gateway
├── spec/          # SCL と派生契約
├── infra/         # コンテナ・ローカル実行・database schema 資材
├── load/k6/       # tenant-local OAuth SLO smoke
├── tools/         # RA/SCL CLI、renderer、schema validator
├── verification/  # traceability manifest と revision 付き evidence
├── decisions/     # Architecture Decision Records
└── work-items/    # 作業単位と完了記録
```

依存は `spec` から各実装・派生物へ向かい、`backend` の domain/usecases は adapter と runtime へ逆依存しない。

## Stack

- Go、React/TypeScript、Bun、PostgreSQL、Valkey、Docker Compose、Kubernetes、Prometheus、Grafana、k6。
- User 属性による動的 Group membership の式評価には、制限付き CEL (`cel-go`) を使う。

## Structural Decisions

- `backend/` と `frontend/` の成果物境界および Go entry point の配置は [ADR-092](decisions/ADR-092-backend-and-frontend-top-level-directories.md) に従う。
- Runtime と database infrastructure 資材の配置は [ADR-102](decisions/ADR-102-infrastructure-root-for-runtime-and-database-assets.md) に従う。
- technical shared context と context-owned adapter の分離、および context 固有の永続化 adapter の同居は [ADR-070](decisions/ADR-070-technical-shared-context-for-cross-context-adapters.md) と [ADR-090](decisions/ADR-090-context-local-persistence-and-sqlc.md) に従う。
- durable job queue (PostgreSQL `FOR UPDATE SKIP LOCKED` リース) と `idmagic-worker` プロセス分離・耐障害性は [ADR-098](decisions/ADR-098-durable-job-queue-skip-locked-lease.md) と [ADR-099](decisions/ADR-099-job-worker-execution-model-and-fault-tolerance.md) に従う。
- 動的 Group membership の CEL 環境、排他的 membership type、rule version による fail-closed は [ADR-111](decisions/ADR-111-cel-dynamic-group-membership-rules.md) に従う。
- SCL 規範要素、Architecture module、宣言済み check、revision 付き evidence の直接追跡は [ADR-115](decisions/ADR-115-direct-workspace-traceability-graph.md) に従う。
- context、RA layer、module dependency、runtime、実 import、complexity ceiling を検査可能な地図として保つ方針は [ADR-116](decisions/ADR-116-executable-architecture-map.md) に従う。
- LifecycleWorkflow を IdManagement の record-of-truth から IdGovernance の policy/orchestration へ分離する境界は [ADR-117](decisions/ADR-117-extract-identity-governance-context.md) に従う。
- 環境別 seed の policy と execution orchestration は record context から分離し、各 context の公開 command surface を介して適用する ([ADR-118](decisions/ADR-118-extract-environment-aware-seeding-context.md))。
- Outbound provisioning (SCIM client) は inbound の `Scim` (server) とは独立の `Provisioning` context とし、protocol 非依存コア + protocol 別 feature slice で構成する。配送は既存 outbox を観測せず、呼び出し元の Postgres トランザクション内で `ProvisioningDelivery` を書く same-Tx capture で確定する ([ADR-128](decisions/ADR-128-extract-provisioning-context-and-transactional-delivery-capture.md))。

## 読む順序

機能変更では次の順に読む。

1. `spec/scl.yaml` の `context_map` で対象 bounded context と依存先を特定する。
2. 対象 context の `spec/contexts/<context>.yaml` を読む。機能追加・挙動変更は SCL-first で行う。
3. 該当 ADR を読む。迷ったら `decisions/` をファイル名検索し、古い work item の要約だけで判断しない。
4. Go 実装は対象 context の `domain/`、`usecases/`、`ports/`、`adapters/` の順に読む。
5. HTTP や永続化の横断挙動を触る場合だけ `backend/shared/` と `backend/cmd/internal/bootstrap/` を読む。
6. UI を触る場合は `frontend/ARCHITECTURE.md` と `frontend/src/features/README.md` を先に読む。

実装から仕様へ逆引きする場合は、パッケージ名と SCL context 名がほぼ対応する。例外的な共有物は `backend/shared/` に集約される。

## RA レイヤ対応

`idmagic` は Regenerative Architecture の同心円を Go の package 境界で表す。

| RA レイヤ | 保存・実装場所 | 読み方 |
| --- | --- | --- |
| Specification Core | `spec/scl.yaml`, `spec/contexts/*.yaml` | 規範仕様。変更は原則ここから始める。 |
| Decision Record | `decisions/*.md` | SCL だけでは分からない採用理由・除外理由。 |
| Application Logic | `backend/<context>/domain`, `backend/<context>/usecases`, `backend/shared/spec` | フレームワーク非依存のドメイン・ユースケース・SCL binding。 |
| Adapter Layer | `backend/<context>/adapters`, `backend/shared/adapters` | HTTP、persistence、crypto、policy、notification など外界との接続。 |
| Runtime & Infrastructure | `backend/cmd/`, `backend/cmd/internal/bootstrap`, `infra/`, `frontend/`, `docker compose` | 起動、DI、配信、プロセス境界。 |

`backend/shared/spec` は SCL の Go binding と派生検証であり、仕様核そのものではない。SCL の内容を変える代わりに Go binding だけを調整しない。

## Context Map

SCL context と Go package の主な対応は次の通り。

| SCL context | Go package | 主な責務 |
| --- | --- | --- |
| `System` | `backend/cmd/internal/bootstrap`, `backend/shared/adapters/http/server`, `frontend/` | 横断 UX、起動、ルーティング集約、health。 |
| `Tenancy` | `backend/tenancy` | tenant / realm、tenant-scoped settings、user attribute schema、control-plane tenant 管理。 |
| `IdManagement` | `backend/idmanagement` | User、Group、Agent、自己プロフィール、identity lifecycle、CEL 動的 membership rule と再評価。 |
| `Authentication` | `backend/authentication` | 資格情報検証、MFA、ログインセッション、step-up、パスワード変更・リセット、認証イベント。 |
| `OAuth2` | `backend/oauth2` | OAuth 2.0 / OIDC protocol endpoint、client、consent、token、role policy。 |
| `Application` | `backend/application` | Application catalog、protocol binding、assignment、portal ordering/category。 |
| `Audit` | `backend/audit` | authentication / identity-management / oauth2 / tenancy / signing-keys / application / saml / wsfederation を横断する監査イベントの read model。検索属性 registry、PII 変換、管理 API、保持期間を所有する。 |
| `ClaimMapping` | `backend/claimmapping` | protocol-neutral な claim release policy、identity 属性 projection、fail-closed validation。 |
| `Scim` | `backend/scim` | SCIM 2.0 Inbound Provisioning サーバー、外部プロバイダからのユーザー・グループ同期、Bearer Token 認証、soft-delete 統合。 |
| `Provisioning` | `backend/provisioning`（未実装、後続タスクで新設） | SCIM 2.0 outbound provisioning。下流 SaaS への user/group push lifecycle management。真実源は idmagic の User/Group、下流は mirror。protocol 非依存コア + protocol 別 feature slice (`provisioning/scim` など) で構成する ([ADR-128](decisions/ADR-128-extract-provisioning-context-and-transactional-delivery-capture.md))。 |
| `Jobs` | `backend/jobs` | テナント境界を保つ汎用非同期ジョブ基盤。durable job queue (PostgreSQL SKIP LOCKED リース)、worker runtime、handler registry を所有する。業務ロジックは呼び出し元 context の usecase に残る。管理 UI/API は `wi-157`。 |
| `Seeding` | `backend/seeding` | 環境別 profile、dry-run、redacted plan、適用 policy を所有する。業務データとその永続化は各 record context に残す。 |
| `SigningKeys` | `backend/signingkeys` | tenant-scoped 鍵 metadata、rotation、repository port、管理/JWKS HTTP、memory/PostgreSQL/Vault adapter。JWT/XML wire signer は protocol/technical adapter に残す。 |
| `WsFederation` | `backend/wsfederation` | WS-Fed passive、WS-Trust active STS、federation metadata、MEX、RP trust。 |
| `Saml` | `backend/saml` | SAML 2.0 IdP、SP trust、metadata、SSO/SLO。 |

context 間の公開語彙と依存は `spec/scl.yaml` の `context_map` が正である。新しい依存を追加する場合は、直接 import を増やす前に context map の `depends_on` を見直す。

## Go Package Conventions

各 bounded context は原則として次の形を取る。

```text
backend/<context>/
  domain/      # エンティティ、値オブジェクト、状態機械、純粋な検証
  usecases/    # 仕様上の操作を実行するアプリケーション論理
  ports/       # repository、store、外部 service への抽象
  adapters/    # HTTP、wire format、外部 protocol 固有処理
```

`domain/` は Echo、PostgreSQL、Valkey、HTTP request/response を知らない。`usecases/` は `ports/` に依存し、具体 adapter には依存しない。`adapters/http` は入力の wire 変換、HTTP status、cookie/header、CSRF/Origin など境界処理を持つ。`usecases/` が adapter を import しない依存方向は全 context 共通で、外界の能力（署名・割当ゲート・認証解決など）は `ports/` の抽象か usecase パッケージ内の interface で受け、adapter が具体実装を注入する（例: `oauth2` の `ports.TokenIssuer`、`saml` / `wsfederation` の `ApplicationGate` interface）。

`domain/` と `usecases/` の有無は「その context 固有ロジックの有無」で決まり、4 層すべてを機械的に置くわけではない。共有される SCL Go binding は `backend/shared/spec` に残し（ADR-070）、context 固有の業務型は各 context の `domain/` が所有する（ADR-089）。`tenancy` のように binding を超える固有ドメインロジックを持たない context は per-context `domain/` を持たない。逆に `idmanagement`（User/Group/Agent 集約、属性スキーマ、field validation）や `saml` / `wsfederation`（プロトコル固有の解析・claim mapping）のように固有ロジックを持つ context は `domain/` を、SSO/sign-in のオーケストレーション（SP/RP 解決・署名検証・割当ゲート・claim 発行）を持つ context は `usecases/` を持つ。ブラウザ federation の発行判断はすべて `usecases/` にあり、`adapters/http` は wire と HTTP 境界に閉じる。

`backend/shared/` は「複数 context が本当に共有する technical capability」だけに使う。context 固有の概念を便利だからという理由で `shared` に置くと、次の変更で読む範囲が広がる。domain event の具象 struct は owning context の `domain/events.go` に置き、`backend/shared/spec/events.go` は event envelope interface と wire marshal だけを持つ。Audit の分類は具象型 registry ではなく安定した event type discriminator を読む。

### Feature 垂直スライス

2 つ以上の独立した sub-domain（feature）を持つ context では、上記 4 層の格子に
`backend/<context>/<feature>/{domain,ports,usecases,adapters/...}/` という
feature 垂直スライス層を追加できる（[ADR-130](decisions/ADR-130-idmanagement-feature-vertical-slice.md)）。
単一 feature の context には導入しない（stutter を作らない）。パイロットは
`idmanagement` で、`user`/`group`/`agent` の 3 feature に分割した:

```text
backend/idmanagement/
  module.go                 # context ルートに1つ（DI 束は feature に分割しない）
  domain/                   # feature 横断の共有型のみ（enum・DomainEvent）
  usecases/                 # feature 横断の共有 usecase ヘルパー・エラー変数のみ
  adapters/
    http/
      routes.go              # Deps 型定義の再エクスポート・route 登録の集約点
      httpdeps/               # Deps 型定義そのもの（leaf package、後述）
      extra_identity_test.go  # feature 横断の統合テスト
  user/
    domain/  ports/  usecases/
    adapters/http/  adapters/persistence/{memory,postgres}/
  group/
    domain/  ports/  usecases/
    adapters/http/  adapters/persistence/{memory,postgres}/
  agent/
    domain/  ports/  usecases/
    adapters/http/  adapters/persistence/{memory,postgres}/
```

`adapters/http` と `adapters/persistence/postgres` は Go の言語制約・コード生成単位により
素朴には分割できなかったが（domain/ports/usecases とは異なる設計判断を要した）、
別の設計で分割できると判明し実施した（ADR-130）。

- **adapters/http**: ハンドラは元々 `Deps` 構造体のメソッド（`func (d Deps) handleX`）
  として実装されていた。Go はメソッドを receiver 型と同一パッケージにしか定義できないため、
  素朴に `Deps` を feature ごとの embedded 部分構造体へ分割すると、feature 横断の port 参照
  （例: group ハンドラが `UserRepo` を、agent ハンドラが `UserRepo`/`ClientRepo` を参照）
  により各部分構造体へ同じフィールドを重複定義する必要が生じる。代わりに `Deps` 型定義を
  `httpdeps` という独立した leaf package へ切り出し、ハンドラを
  `func handleX(d Deps, c *echo.Context) error` という**フリー関数**へ変換して
  feature パッケージへ移した。フリー関数は receiver 型と同一パッケージである必要が
  ないため、`Deps` 型を分割せずに実装コードだけを feature ごとに分離できる。
  `routes.go` は `type Deps = httpdeps.Deps`（型 alias）で再エクスポートするため、
  外部の `idmhttp.Deps{...}` 構築コード（bootstrap・テスト）は無変更のまま。
- **adapters/persistence/postgres**: `sqlc.yaml` の idmanagement 用エントリを feature
  単位の複数エントリへ分割し、`queries/*.sql` と生成される `sqlcgen/` を feature
  ディレクトリへ移した。feature 横断のテスト fixture ヘルパー（`seedTenant`/`seedUser` 等）
  は Go の `_test.go` がパッケージをまたげない制約により、各 feature パッケージへ複製した。
  `lifecycle_workflows` テーブルの query/sqlcgen は IdGovernance context 所有
  （wi-237/ADR-117 が後続 WI へ後回しにしていた context-local sqlc 分割、ADR-090）のため
  `backend/idgovernance/adapters/persistence/postgres/` へ物理移設し、
  `backend/idmanagement/adapters/persistence/` は完全に消滅した。

package 名は各層のまま（`domain`/`ports`/`usecases`/`http`/`memory`/`postgres`）とし、
同一 context の複数 feature を同時 import する箇所は named import
（`userdomain`, `groupdomain` 等）で区別する。feature の adapters/http パッケージが
`Deps` 型を参照する箇所は、各パッケージ内の `deps.go` に置いた
`type Deps = httpdeps.Deps` alias を使う。

## HTTP Routing

HTTP route の集約点は `backend/shared/adapters/http/server/routes.go` である。ここで default tenant と `/realms/:tenant_id` の両方に tenant-scoped routes を登録し、control-plane tenant 管理だけを `/realms/default/admin/tenants` に分ける。

各 context の route は `backend/<context>/adapters/http/routes.go` に置く。エンドポイントの正確な一覧はそのファイルを読む。新しい HTTP API は、所有 context の `routes.go` に登録し、handler は同じ `adapters/http` 配下に置く。context 固有の repository とルート配線は `backend/<context>/module.go` に集約し、中央 router は Module を呼び出すだけにする（ADR-091）。

## Bootstrap And Adapters

`backend/cmd/idmagic/` の main パッケージは起動処理を担い、起動時 DI は `backend/cmd/internal/bootstrap` が所有する。また、`backend/cmd/idmagic-relay/main.go` は outbox → Kafka リレープロセスを起動するもので、`backend/cmd/idmagic-relay/internal/relay` の `Run()` を呼ぶ。`backend/cmd/idmagic-worker/` は durable job の claim と handler 実行だけを担当し、API から独立して水平スケールする（[ADR-099](decisions/ADR-099-job-worker-execution-model-and-fault-tolerance.md)）。`backend/cmd/idmagic-batch/` は外部 scheduler から one-shot で起動され、retention sweep または signing-key lifecycle を一度実行して終了する（[ADR-124](decisions/ADR-124-scheduled-batch-execution-boundary.md)）。各 runtime unit は同一 Go module と bounded context 実装を再利用する。

`backend/cmd/internal/bootstrap/deps.go` の `Dependencies` は HTTP 層へ渡す境界の集約で、memory / postgres_valkey / outbox / otel などの runtime 選択を吸収する。context 固有の repository は各 `Module` に束ね、中央 `Dependencies` と server `Deps` には Module を渡す。新しい port を追加したら、少なくとも次を確認する。

- 対象 context の `ports/`
- memory adapter
- postgres adapter と migration が必要か
- `bootstrap.Dependencies`
- `assembleMemory` / `assemblePostgresValkey`
- `support.Deps`
- 対象 HTTP handler または usecase の constructor

## Durable Job Worker

`Jobs` context は業務処理そのものではなく、tenant-owned な非同期処理に共通する
enqueue、永続化、claim、リース、heartbeat、retry、dead-letter、cancel の実行基盤を
所有する。各 JobKind の params 解釈と副作用は consumer context の usecase に残り、
`backend/cmd/idmagic-worker/worker.go` が起動時にそれらの handler を registry へ
composition する。API process は Job を enqueue するが実行せず、worker process は
HTTP request を処理せず Job の実行だけを担う。

`JobKind` は登録時に `ExecutionLane`（`latency_sensitive` / `default` / `bulk`、
ADR-129）を1つだけ持つ。lane は `domain.RegisterKind(kind, lane)` が決め、
enqueue 呼び出し元は指定できない。`Job.Lane` は enqueue 時に kind の登録情報から
自動的に決まり、claim は対象 lane 内の Job だけを対象にする。

```text
API / consumer usecase
  └─ EnqueueJob (lane は kind から自動決定)
       └─ JobRepository ──> PostgreSQL jobs (lane 列、lane-prefixed index)
                                  │
idmagic-worker                    │ poll (lane ごとに独立)
  ├─ lifecycle workflow dispatcher│ （未 enqueue run の回収）
  ├─ Runner (lane=latency_sensitive) ── ClaimBatch(lane) <───┘
  ├─ Runner (lane=default)          ── ClaimBatch(lane) <───┘
  ├─ Runner (lane=bulk)             ── ClaimBatch(lane) <───┘
  │    ├─ HandlerRegistry ──> consumer context usecase (共有 registry)
  │    ├─ Heartbeat ────────> lease_expires_at 延長
  │    └─ Complete / Fail ──> succeeded / queued(retry) / failed
  └─ jobsQueueDepthSamplingLoop ──> lane 別 queue depth/active gauge (10s 間隔)
```

`ClaimBatch` は対象 lane 内でだけ、due になった `queued` Job とリース失効済みの
`running` Job を取得する（他 lane の Job は取得しない、ADR-129 lane isolation）。
PostgreSQL では `WHERE lane = $lane AND (...) ORDER BY run_at FOR UPDATE SKIP LOCKED`
と同一 statement 内の `running` 更新により、複数 worker が同じ Job の有効リースを
同時に取得しない。lane ごとに独立した `Runner`（独立した concurrency semaphore）が
空いている実行枠数だけ batch claim し、既定4枠で handler を並行実行する。1
process は複数 lane の `Runner` を同時に起動できる（`JOB_WORKER_LANES` 未設定時の
compat mode。development・docker-compose の既定）し、1 lane だけの `Runner` を
持つ dedicated deployment にもできる（production の既定、`infra/k8s/base/worker.yaml`
の `idmagic-worker-{latency-sensitive,default,bulk}` 3 Deployment、lane 別
concurrency は `JOB_WORKER_CONCURRENCY_<LANE>`）。lane 内の claim 候補は概ね
`run_at` の古い順だが、並行実行・複数 process・同一時刻の Job があるので lane 内
でも厳密な開始順・完了順は保証しない。lane を跨いだ順序保証は最初から目指さない
（bulk backlog がどれだけ滞留しても latency_sensitive の実行枠を奪わない、という
容量隔離が目的であり、lane 内の数値 priority は採用しない）。

実行保証は at-least-once である。claim ごとに attempts を増やし、
`lease_owner` と `lease_expires_at` を設定する。実行中は lease 期間の1/3ごとに
heartbeat し、成功時は lease 所有者だけが complete できる。失敗時は指数 backoff
後の `run_at` で `queued` に戻し、`max_attempts` 到達時は `failed` に確定する。
process crash や強制終了で heartbeat が止まった Job は、リース失効後に別 worker が
再 claim する可能性があるため、handler は dedup key と consumer 側の整合性境界を
使って冪等にする。

`idmagic-worker` は `/metrics`（MetricsExposition、system.yaml）を独立した
management-only HTTP listener として公開する（idmagic-api の `/metrics` とは別
プロセス・別 instance）。lane 別の `jobs_claim_latency_seconds` /
`jobs_outcome_total` / `jobs_retry_total` / `jobs_queue_depth` を持ち、
`tenant_id`/`job_id` を label に含めない。

SIGTERM/SIGINT では新規 claim を停止し、in-flight handler の完了を drain 猶予まで
待つ。猶予超過後は process を終了し、明示的に再 enqueue せずリース自然失効で回復する。
poll interval、process 内 concurrency、lease、retry backoff は
`JOB_POLL_INTERVAL`、`JOB_WORKER_CONCURRENCY`、`JOB_LEASE_DURATION`、
`JOB_BACKOFF_BASE`、`JOB_BACKOFF_CAP` で process 全体に設定する。これらは現在、
JobKind ごとの QoS や consumer 固有の順序保証・rate limit を提供しない。

定期的な全 tenant retention と signing-key lifecycle は durable Jobs に混在させず、
外部 scheduler から `idmagic-batch` を one-shot 起動する。worker 内の
lifecycle workflow dispatcher は、業務実行を直接行う定期 batch ではなく、
同一 transaction で確定済みだが Job と未関連付けの WorkflowRun を再走査して
durable queue へ安全に handoff する回復経路である。永続 queue の判断は
[ADR-098](decisions/ADR-098-durable-job-queue-skip-locked-lease.md)、process・
配信・drain の判断は
[ADR-099](decisions/ADR-099-job-worker-execution-model-and-fault-tolerance.md)、
scheduled batch との境界は
[ADR-124](decisions/ADR-124-scheduled-batch-execution-boundary.md)を正とする。

## Persistence

永続化 port と repository 実装は所有 context 側に置く。context 固有の memory / postgres adapter は `backend/<context>/adapters/persistence/{memory,postgres}` に同居し、`backend/shared/adapters/persistence/` は DB pool、row scanner、transaction helper、Valkey client などの技術的共通部品だけを持つ（ADR-090）。

PostgreSQL の構造を増やすときは、まず `infra/schema/postgres.sql` の現在形 schema を更新する。構造差分は `psqldef` の dry-run で確認し、デプロイ前ジョブで適用する。既存データの backfill、値変換、削除前の退避など、構造差分だけでは表せない変更は、対象 WI の runbook または専用 SQL script として明示する。アプリ起動時の migration runner は持たない。memory adapter はテスト・ローカル demo の基準にもなるため、postgres だけを更新しない。

### データベース設計ポリシー (ADR-082 / ADR-084)

データベースのスキーマやテーブル構造を設計する際は、以下の方針を遵守する。

#### 1. 列型選定ルール
- **自由文字列 (上限なし)**: `TEXT` 型を使用する。`varchar` (制約なし) は使用しない。
- **上限のある文字列**: `TEXT` 型に `CHECK (char_length(col) <= N)` 制約を付与するか、`varchar(N)` に統一する。使い分けと具体的な最大文字数は `wi-128-string-length-limits-policy` に従う。
- **内部生成 ID**: `idmagic` が `spec.NewUUIDv4()` で内部生成する ID 列（`users.id`, `clients.client_id`, `groups.id`, `agents.id`, `audit_events.id`, `scim_tokens.id` 等）は、すべて `UUID` 型とする。Go 側では `string` 型のまま扱い、pgx 接続時の text codec 登録 (`RegisterUUIDAsText`) によって自動変換する。
- **外部決定 ID**: 外部（SP/RP メタデータ等）が値を決定する ID（`entity_id`, `wtrealm`, `scim_id`, `kid` 等）は `TEXT` 型を維持する。
- **時刻**: 一貫して `TIMESTAMPTZ` 型を使用する（マイクロ秒精度を真値とし、schema で丸めない）。
- **有限集合 (ステータス等)**: `TEXT` + `CHECK (col IN (...))` で値集合を表現し、PostgreSQL enum は原則使用しない。

#### 2. tenant_id 保持の 4 分類ルール
外部から parent 経由で辿れるという理由だけで機械的に `tenant_id` を全テーブルに追加しない。以下の分類に従って判断する。
- **tenant-owned aggregate**: `tenant_id` を PK または UNIQUE キーに含める（例: `users`, `groups`, `clients`）。
- **tenant-scoped natural key を参照する child**: 参照先が `(tenant_id, local_id)` の複合キーで識別される場合、child にも `tenant_id` を持たせ、composite FK (複合外部キー) でテナント不一致を DB 制約で防ぐ（例: `consents`, `refresh_tokens`）。
- **globally unique parent に従属する child**: 親のキーが UUID などでグローバル一意である場合は `tenant_id` を重複保持しない（例: `mfa_factors`, `password_history`）。
- **append-only / audit**: クエリ境界や監査隔離単位として必要な場合にのみ保持する（例: `audit_events`, `outbox`）。

## UI Boundary

React UI は Go API とは別成果物・別プロセスで、gateway によって同一オリジンへ統合される。詳細は `frontend/ARCHITECTURE.md` を読む。

UI の画面実装は `frontend/src/features/`、route は `frontend/src/routes/` が中心である。API の wire contract を変える場合は、Go handler/usecase と UI API client (`frontend/src/api*.ts`) の両方を確認する。

## Verification Entry Points

通常の Go 変更では `justfile` の正規入口を使う。

```bash
just verify-go
```

UI 変更では `frontend/README.md` と `frontend/tests/e2e/README.md` の検証手順を読む。SCL や work item を変更した場合は、ルートの `tools/yaml-check` 系の検証も対象に含める。

## Documentation Policy

新しい説明を追加する前に、次を確認する。

- SCL に書くべき規範要件ではないか。
- ADR に書くべき再導出不能な判断理由ではないか。
- work item に書くべき一回限りの実施記録ではないか。
- コードや schema から機械的に読める一覧を手書き複製していないか。

この文書に追加してよいのは、AI が読む入口を狭める安定した地図だけである。機能ごとの詳細、最新のエンドポイント網羅表、全テスト一覧、全環境変数一覧は置かない。
