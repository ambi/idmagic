---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p3
depends_on: [wi-26063-move-remaining-contexts-to-the-feature-and-design-layout, wi-71372-rewrite-tenancy-specifications-in-the-requirement-format, wi-73023-fix-the-japanese-ears-syntax-and-check-requirement-statements]
change_kind: docs
evidence_policy: risk-based-v4
documentation_impact:
  level: release_note
  reason: 製品の振る舞いは変えないが、実装だけが守っていた振る舞い（WS-Federation のパッシブサインアウトなど）を要件として約束するので、依存してよい境界が増えたことを利用者へ知らせる。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-35451-rewrite-remaining-context-specifications-in-the-requirement-format.md }
initial_context:
  specification:
    - SPECIFICATION_FORMAT.md
    - docs/domain/tenancy/lifecycle/README.md
    - work-items/done/wi-71372-rewrite-tenancy-specifications-in-the-requirement-format.md
    - work-items/done/wi-83002-rewrite-remaining-identity-management-features-in-the-requirement-format.md
  source:
    - tools/check/src/check-unspecified-vocabulary.ts
    - tools/check/src/unspecified-vocabulary.ts
    - tools/check/unspecified-vocabulary-debt.json
  stop_before_reading: [frontend]
affected_spec:
  - { path: docs/domain/api-tokens/token-management/README.md, requirement: REQ-APITOKENS-001 }
  - { path: docs/domain/api-tokens/authentication/README.md, requirement: REQ-APITOKENS-002 }
  - { path: docs/domain/api-tokens/token-management/README.md, requirement: REQ-APITOKENS-003 }
  - { path: docs/domain/api-tokens/authentication/README.md, requirement: REQ-APITOKENS-004 }
  - { path: docs/domain/application/catalog/README.md, requirement: REQ-APPLICATION-001 }
  - { path: docs/domain/application/catalog/README.md, requirement: REQ-APPLICATION-002 }
  - { path: docs/domain/application/portal/README.md, requirement: REQ-APPLICATION-003 }
  - { path: docs/domain/application/catalog/README.md, requirement: REQ-APPLICATION-004 }
  - { path: docs/domain/application/catalog/README.md, requirement: REQ-APPLICATION-005 }
  - { path: docs/domain/application/catalog/README.md, requirement: REQ-APPLICATION-006 }
  - { path: docs/domain/application/catalog/README.md, requirement: REQ-APPLICATION-007 }
  - { path: docs/domain/application/catalog/README.md, requirement: REQ-APPLICATION-008 }
  - { path: docs/domain/application/sign-in-policy/README.md, requirement: REQ-APPLICATION-009 }
  - { path: docs/domain/application/sign-in-policy/README.md, requirement: REQ-APPLICATION-010 }
  - { path: docs/domain/application/assignment/README.md, requirement: REQ-APPLICATION-011 }
  - { path: docs/domain/application/assignment/README.md, requirement: REQ-APPLICATION-012 }
  - { path: docs/domain/application/catalog/README.md, requirement: REQ-APPLICATION-013 }
  - { path: docs/domain/application/assignment/README.md, requirement: REQ-APPLICATION-014 }
  - { path: docs/domain/application/catalog/README.md, requirement: REQ-APPLICATION-015 }
  - { path: docs/domain/audit/event-search/README.md, requirement: REQ-AUDIT-001 }
  - { path: docs/domain/audit/event-search/README.md, requirement: REQ-AUDIT-002 }
  - { path: docs/domain/audit/event-search/README.md, requirement: REQ-AUDIT-003 }
  - { path: docs/domain/audit/event-search/README.md, requirement: REQ-AUDIT-004 }
  - { path: docs/domain/audit/event-search/README.md, requirement: REQ-AUDIT-005 }
  - { path: docs/domain/audit/event-search/README.md, requirement: REQ-AUDIT-006 }
  - { path: docs/domain/audit/event-search/README.md, requirement: REQ-AUDIT-007 }
  - { path: docs/domain/authentication/federation/README.md, requirement: REQ-AUTHENTICATION-001 }
  - { path: docs/domain/authentication/federation/README.md, requirement: REQ-AUTHENTICATION-002 }
  - { path: docs/domain/authentication/federation/README.md, requirement: REQ-AUTHENTICATION-003 }
  - { path: docs/domain/authentication/federation/README.md, requirement: REQ-AUTHENTICATION-025 }
  - { path: docs/domain/authentication/federation/README.md, requirement: REQ-AUTHENTICATION-037 }
  - { path: docs/domain/authentication/webauthn/README.md, requirement: REQ-AUTHENTICATION-038 }
  - { path: docs/domain/authentication/account-portal/README.md, requirement: REQ-AUTHENTICATION-039 }
  - { path: docs/domain/authentication/mfa/README.md, requirement: REQ-AUTHENTICATION-015 }
  - { path: docs/domain/authentication/mfa/README.md, requirement: REQ-AUTHENTICATION-018 }
  - { path: docs/domain/authentication/mfa/README.md, requirement: REQ-AUTHENTICATION-019 }
  - { path: docs/domain/authentication/mfa/README.md, requirement: REQ-AUTHENTICATION-020 }
  - { path: docs/domain/authentication/mfa/README.md, requirement: REQ-AUTHENTICATION-022 }
  - { path: docs/domain/authentication/mfa/README.md, requirement: REQ-AUTHENTICATION-023 }
  - { path: docs/domain/authentication/password/README.md, requirement: REQ-AUTHENTICATION-008 }
  - { path: docs/domain/authentication/password/README.md, requirement: REQ-AUTHENTICATION-010 }
  - { path: docs/domain/authentication/password/README.md, requirement: REQ-AUTHENTICATION-016 }
  - { path: docs/domain/authentication/password/README.md, requirement: REQ-AUTHENTICATION-024 }
  - { path: docs/domain/authentication/recovery/README.md, requirement: REQ-AUTHENTICATION-036 }
  - { path: docs/domain/authentication/account-portal/README.md, requirement: REQ-AUTHENTICATION-004 }
  - { path: docs/domain/authentication/sign-in/README.md, requirement: REQ-AUTHENTICATION-005 }
  - { path: docs/domain/authentication/sign-in/README.md, requirement: REQ-AUTHENTICATION-007 }
  - { path: docs/domain/authentication/sign-in/README.md, requirement: REQ-AUTHENTICATION-009 }
  - { path: docs/domain/authentication/security-notification/README.md, requirement: REQ-AUTHENTICATION-030 }
  - { path: docs/domain/authentication/security-notification/README.md, requirement: REQ-AUTHENTICATION-031 }
  - { path: docs/domain/authentication/security-notification/README.md, requirement: REQ-AUTHENTICATION-032 }
  - { path: docs/domain/authentication/security-notification/README.md, requirement: REQ-AUTHENTICATION-033 }
  - { path: docs/domain/authentication/security-notification/README.md, requirement: REQ-AUTHENTICATION-034 }
  - { path: docs/domain/authentication/session/README.md, requirement: REQ-AUTHENTICATION-013 }
  - { path: docs/domain/authentication/session/README.md, requirement: REQ-AUTHENTICATION-021 }
  - { path: docs/domain/authentication/session/README.md, requirement: REQ-AUTHENTICATION-035 }
  - { path: docs/domain/authentication/sign-in-activity/README.md, requirement: REQ-AUTHENTICATION-014 }
  - { path: docs/domain/authentication/totp/README.md, requirement: REQ-AUTHENTICATION-011 }
  - { path: docs/domain/authentication/totp/README.md, requirement: REQ-AUTHENTICATION-012 }
  - { path: docs/domain/authentication/totp/README.md, requirement: REQ-AUTHENTICATION-017 }
  - { path: docs/domain/authentication/trusted-device/README.md, requirement: REQ-AUTHENTICATION-026 }
  - { path: docs/domain/authentication/trusted-device/README.md, requirement: REQ-AUTHENTICATION-027 }
  - { path: docs/domain/authentication/trusted-device/README.md, requirement: REQ-AUTHENTICATION-028 }
  - { path: docs/domain/authentication/trusted-device/README.md, requirement: REQ-AUTHENTICATION-029 }
  - { path: docs/domain/authentication/webauthn/README.md, requirement: REQ-AUTHENTICATION-006 }
  - { path: docs/domain/authorization/model/README.md, requirement: REQ-AUTHORIZATION-001 }
  - { path: docs/domain/authorization/relation-tuple/README.md, requirement: REQ-AUTHORIZATION-002 }
  - { path: docs/domain/authorization/check/README.md, requirement: REQ-AUTHORIZATION-003 }
  - { path: docs/domain/authorization/check/README.md, requirement: REQ-AUTHORIZATION-004 }
  - { path: docs/domain/authorization/check/README.md, requirement: REQ-AUTHORIZATION-005 }
  - { path: docs/domain/authorization/check/README.md, requirement: REQ-AUTHORIZATION-006 }
  - { path: docs/domain/authorization/check/README.md, requirement: REQ-AUTHORIZATION-007 }
  - { path: docs/domain/authorization/relation-tuple/README.md, requirement: REQ-AUTHORIZATION-008 }
  - { path: docs/domain/authorization/check/README.md, requirement: REQ-AUTHORIZATION-009 }
  - { path: docs/domain/authorization/model/README.md, requirement: REQ-AUTHORIZATION-010 }
  - { path: docs/domain/claim-mapping/issuance/README.md, requirement: REQ-CLAIMMAPPING-001 }
  - { path: docs/domain/claim-mapping/issuance/README.md, requirement: REQ-CLAIMMAPPING-002 }
  - { path: docs/domain/claim-mapping/issuance/README.md, requirement: REQ-CLAIMMAPPING-003 }
  - { path: docs/domain/data-keys/lifecycle/README.md, requirement: REQ-DATAKEYS-001 }
  - { path: docs/domain/data-keys/lifecycle/README.md, requirement: REQ-DATAKEYS-002 }
  - { path: docs/domain/data-keys/lifecycle/README.md, requirement: REQ-DATAKEYS-003 }
  - { path: docs/domain/data-keys/lifecycle/README.md, requirement: REQ-DATAKEYS-004 }
  - { path: docs/domain/data-keys/lifecycle/README.md, requirement: REQ-DATAKEYS-005 }
  - { path: docs/domain/data-keys/health/README.md, requirement: REQ-DATAKEYS-006 }
  - { path: docs/domain/identity-governance/lifecycle-workflow/README.md, requirement: REQ-IDGOVERNANCE-001 }
  - { path: docs/domain/identity-governance/lifecycle-workflow/README.md, requirement: REQ-IDGOVERNANCE-002 }
  - { path: docs/domain/identity-governance/workflow-run/README.md, requirement: REQ-IDGOVERNANCE-003 }
  - { path: docs/domain/identity-governance/workflow-run/README.md, requirement: REQ-IDGOVERNANCE-004 }
  - { path: docs/domain/identity-governance/workflow-run/README.md, requirement: REQ-IDGOVERNANCE-005 }
  - { path: docs/domain/identity-governance/workflow-run/README.md, requirement: REQ-IDGOVERNANCE-006 }
  - { path: docs/domain/identity-governance/lifecycle-workflow/README.md, requirement: REQ-IDGOVERNANCE-007 }
  - { path: docs/domain/identity-governance/workflow-run/README.md, requirement: REQ-IDGOVERNANCE-008 }
  - { path: docs/domain/identity-governance/workflow-run/README.md, requirement: REQ-IDGOVERNANCE-009 }
  - { path: docs/domain/identity-governance/workflow-run/README.md, requirement: REQ-IDGOVERNANCE-010 }
  - { path: docs/domain/identity-governance/lifecycle-workflow/README.md, requirement: REQ-IDGOVERNANCE-011 }
  - { path: docs/domain/identity-governance/lifecycle-workflow/README.md, requirement: REQ-IDGOVERNANCE-012 }
  - { path: docs/domain/identity-governance/lifecycle-workflow/README.md, requirement: REQ-IDGOVERNANCE-013 }
  - { path: docs/domain/identity-governance/lifecycle-workflow/README.md, requirement: REQ-IDGOVERNANCE-014 }
  - { path: docs/domain/jobs/local-development/README.md, requirement: REQ-JOBS-001 }
  - { path: docs/domain/jobs/queue/README.md, requirement: REQ-JOBS-002 }
  - { path: docs/domain/jobs/queue/README.md, requirement: REQ-JOBS-003 }
  - { path: docs/domain/jobs/queue/README.md, requirement: REQ-JOBS-004 }
  - { path: docs/domain/jobs/queue/README.md, requirement: REQ-JOBS-005 }
  - { path: docs/domain/jobs/queue/README.md, requirement: REQ-JOBS-006 }
  - { path: docs/domain/jobs/queue/README.md, requirement: REQ-JOBS-007 }
  - { path: docs/domain/jobs/queue/README.md, requirement: REQ-JOBS-008 }
  - { path: docs/domain/jobs/queue/README.md, requirement: REQ-JOBS-009 }
  - { path: docs/domain/jobs/queue/README.md, requirement: REQ-JOBS-010 }
  - { path: docs/domain/jobs/queue/README.md, requirement: REQ-JOBS-011 }
  - { path: docs/domain/jobs/admin/README.md, requirement: REQ-JOBS-012 }
  - { path: docs/domain/jobs/admin/README.md, requirement: REQ-JOBS-013 }
  - { path: docs/domain/jobs/admin/README.md, requirement: REQ-JOBS-014 }
  - { path: docs/domain/jobs/admin/README.md, requirement: REQ-JOBS-015 }
  - { path: docs/domain/oauth2/token/README.md, requirement: REQ-OAUTH2-001 }
  - { path: docs/domain/oauth2/consent/README.md, requirement: REQ-OAUTH2-002 }
  - { path: docs/domain/oauth2/admin-access/README.md, requirement: REQ-OAUTH2-003 }
  - { path: docs/domain/oauth2/admin-access/README.md, requirement: REQ-OAUTH2-004 }
  - { path: docs/domain/oauth2/authorization/README.md, requirement: REQ-OAUTH2-005 }
  - { path: docs/domain/oauth2/token/README.md, requirement: REQ-OAUTH2-006 }
  - { path: docs/domain/oauth2/client/README.md, requirement: REQ-OAUTH2-007 }
  - { path: docs/domain/oauth2/consent/README.md, requirement: REQ-OAUTH2-008 }
  - { path: docs/domain/oauth2/authorization/README.md, requirement: REQ-OAUTH2-009 }
  - { path: docs/domain/oauth2/token/README.md, requirement: REQ-OAUTH2-010 }
  - { path: docs/domain/oauth2/token/README.md, requirement: REQ-OAUTH2-011 }
  - { path: docs/domain/oauth2/token/README.md, requirement: REQ-OAUTH2-012 }
  - { path: docs/domain/oauth2/token/README.md, requirement: REQ-OAUTH2-013 }
  - { path: docs/domain/oauth2/protocol-endpoints/README.md, requirement: REQ-OAUTH2-014 }
  - { path: docs/domain/oauth2/authorization/README.md, requirement: REQ-OAUTH2-015 }
  - { path: docs/domain/oauth2/client/README.md, requirement: REQ-OAUTH2-016 }
  - { path: docs/domain/oauth2/client/README.md, requirement: REQ-OAUTH2-017 }
  - { path: docs/domain/oauth2/token/README.md, requirement: REQ-OAUTH2-018 }
  - { path: docs/domain/oauth2/token/README.md, requirement: REQ-OAUTH2-019 }
  - { path: docs/domain/oauth2/token/README.md, requirement: REQ-OAUTH2-020 }
  - { path: docs/domain/oauth2/token/README.md, requirement: REQ-OAUTH2-021 }
  - { path: docs/domain/oauth2/authorization/README.md, requirement: REQ-OAUTH2-022 }
  - { path: docs/domain/oauth2/logout/README.md, requirement: REQ-OAUTH2-023 }
  - { path: docs/domain/oauth2/logout/README.md, requirement: REQ-OAUTH2-024 }
  - { path: docs/domain/oauth2/logout/README.md, requirement: REQ-OAUTH2-025 }
  - { path: docs/domain/oauth2/token/README.md, requirement: REQ-OAUTH2-026 }
  - { path: docs/domain/oauth2/device/README.md, requirement: REQ-OAUTH2-027 }
  - { path: docs/domain/oauth2/client/README.md, requirement: REQ-OAUTH2-028 }
  - { path: docs/domain/oauth2/token/README.md, requirement: REQ-OAUTH2-029 }
  - { path: docs/domain/oauth2/protocol-endpoints/README.md, requirement: REQ-OAUTH2-030 }
  - { path: docs/domain/oauth2/consent/README.md, requirement: REQ-OAUTH2-031 }
  - { path: docs/domain/oauth2/consent/README.md, requirement: REQ-OAUTH2-032 }
  - { path: docs/domain/oauth2/protocol-endpoints/README.md, requirement: REQ-OAUTH2-033 }
  - { path: docs/domain/oauth2/token/README.md, requirement: REQ-OAUTH2-034 }
  - { path: docs/domain/oauth2/client/README.md, requirement: REQ-OAUTH2-035 }
  - { path: docs/domain/oauth2/client/README.md, requirement: REQ-OAUTH2-036 }
  - { path: docs/domain/oauth2/client/README.md, requirement: REQ-OAUTH2-037 }
  - { path: docs/domain/oauth2/consent/README.md, requirement: REQ-OAUTH2-038 }
  - { path: docs/domain/oauth2/token/README.md, requirement: REQ-OAUTH2-039 }
  - { path: docs/domain/oauth2/protocol-endpoints/README.md, requirement: REQ-OAUTH2-040 }
  - { path: docs/domain/oauth2/approval/README.md, requirement: REQ-OAUTH2-041 }
  - { path: docs/domain/oauth2/approval/README.md, requirement: REQ-OAUTH2-042 }
  - { path: docs/domain/oauth2/approval/README.md, requirement: REQ-OAUTH2-043 }
  - { path: docs/domain/oauth2/token/README.md, requirement: REQ-OAUTH2-044 }
  - { path: docs/domain/oauth2/token/README.md, requirement: REQ-OAUTH2-045 }
  - { path: docs/domain/oauth2/token/README.md, requirement: REQ-OAUTH2-046 }
  - { path: docs/domain/oauth2/token/README.md, requirement: REQ-OAUTH2-047 }
  - { path: docs/domain/oauth2/token/README.md, requirement: REQ-OAUTH2-048 }
  - { path: docs/domain/oauth2/token/README.md, requirement: REQ-OAUTH2-049 }
  - { path: docs/domain/oauth2/approval/README.md, requirement: REQ-OAUTH2-050 }
  - { path: docs/domain/oauth2/admin-access/README.md, requirement: REQ-OAUTH2-051 }
  - { path: docs/domain/oauth2/admin-access/README.md, requirement: REQ-OAUTH2-052 }
  - { path: docs/domain/oauth2/authorization/README.md, requirement: REQ-OAUTH2-053 }
  - { path: docs/domain/oauth2/authorization/README.md, requirement: REQ-OAUTH2-054 }
  - { path: docs/domain/provisioning/connection/README.md, requirement: REQ-PROVISIONING-001 }
  - { path: docs/domain/provisioning/connection/README.md, requirement: REQ-PROVISIONING-002 }
  - { path: docs/domain/provisioning/synchronization/README.md, requirement: REQ-PROVISIONING-003 }
  - { path: docs/domain/provisioning/synchronization/README.md, requirement: REQ-PROVISIONING-004 }
  - { path: docs/domain/provisioning/synchronization/README.md, requirement: REQ-PROVISIONING-005 }
  - { path: docs/domain/provisioning/synchronization/README.md, requirement: REQ-PROVISIONING-006 }
  - { path: docs/domain/provisioning/task/README.md, requirement: REQ-PROVISIONING-007 }
  - { path: docs/domain/provisioning/task/README.md, requirement: REQ-PROVISIONING-008 }
  - { path: docs/domain/provisioning/task/README.md, requirement: REQ-PROVISIONING-009 }
  - { path: docs/domain/provisioning/task/README.md, requirement: REQ-PROVISIONING-010 }
  - { path: docs/domain/provisioning/synchronization/README.md, requirement: REQ-PROVISIONING-011 }
  - { path: docs/domain/provisioning/connection/README.md, requirement: REQ-PROVISIONING-012 }
  - { path: docs/domain/provisioning/connection/README.md, requirement: REQ-PROVISIONING-013 }
  - { path: docs/domain/provisioning/connection/README.md, requirement: REQ-PROVISIONING-014 }
  - { path: docs/domain/provisioning/connection/README.md, requirement: REQ-PROVISIONING-015 }
  - { path: docs/domain/provisioning/synchronization/README.md, requirement: REQ-PROVISIONING-016 }
  - { path: docs/domain/provisioning/synchronization/README.md, requirement: REQ-PROVISIONING-017 }
  - { path: docs/domain/provisioning/task/README.md, requirement: REQ-PROVISIONING-018 }
  - { path: docs/domain/provisioning/synchronization/README.md, requirement: REQ-PROVISIONING-019 }
  - { path: docs/domain/saml/idp-profile/README.md, requirement: REQ-SAML-001 }
  - { path: docs/domain/saml/sso/README.md, requirement: REQ-SAML-002 }
  - { path: docs/domain/saml/idp-profile/README.md, requirement: REQ-SAML-003 }
  - { path: docs/domain/saml/idp-profile/README.md, requirement: REQ-SAML-004 }
  - { path: docs/domain/saml/service-provider/README.md, requirement: REQ-SAML-005 }
  - { path: docs/domain/saml/sso/README.md, requirement: REQ-SAML-006 }
  - { path: docs/domain/saml/sso/README.md, requirement: REQ-SAML-007 }
  - { path: docs/domain/saml/sso/README.md, requirement: REQ-SAML-008 }
  - { path: docs/domain/saml/sso/README.md, requirement: REQ-SAML-009 }
  - { path: docs/domain/seeding/seed-run/README.md, requirement: REQ-SEEDING-001 }
  - { path: docs/domain/seeding/seed-run/README.md, requirement: REQ-SEEDING-002 }
  - { path: docs/domain/seeding/seed-run/README.md, requirement: REQ-SEEDING-003 }
  - { path: docs/domain/seeding/seed-run/README.md, requirement: REQ-SEEDING-004 }
  - { path: docs/domain/seeding/seed-run/README.md, requirement: REQ-SEEDING-005 }
  - { path: docs/domain/seeding/seed-run/README.md, requirement: REQ-SEEDING-006 }
  - { path: docs/domain/seeding/seed-run/README.md, requirement: REQ-SEEDING-007 }
  - { path: docs/domain/seeding/seed-run/README.md, requirement: REQ-SEEDING-008 }
  - { path: docs/domain/seeding/seed-run/README.md, requirement: REQ-SEEDING-009 }
  - { path: docs/domain/seeding/seed-run/README.md, requirement: REQ-SEEDING-010 }
  - { path: docs/domain/sharedsignals/revocation/README.md, requirement: REQ-SHAREDSIGNALS-001 }
  - { path: docs/domain/sharedsignals/receiver/README.md, requirement: REQ-SHAREDSIGNALS-003 }
  - { path: docs/domain/sharedsignals/receiver/README.md, requirement: REQ-SHAREDSIGNALS-004 }
  - { path: docs/domain/sharedsignals/receiver/README.md, requirement: REQ-SHAREDSIGNALS-005 }
  - { path: docs/domain/sharedsignals/transmitter/README.md, requirement: REQ-SHAREDSIGNALS-006 }
  - { path: docs/domain/sharedsignals/revocation/README.md, requirement: REQ-SHAREDSIGNALS-007 }
  - { path: docs/domain/sharedsignals/stream/README.md, requirement: REQ-SHAREDSIGNALS-008 }
  - { path: docs/domain/sharedsignals/stream/README.md, requirement: REQ-SHAREDSIGNALS-009 }
  - { path: docs/domain/sharedsignals/receiver/README.md, requirement: REQ-SHAREDSIGNALS-010 }
  - { path: docs/domain/sharedsignals/stream/README.md, requirement: REQ-SHAREDSIGNALS-011 }
  - { path: docs/domain/signing-keys/lifecycle/README.md, requirement: REQ-SIGNINGKEYS-001 }
  - { path: docs/domain/signing-keys/lifecycle/README.md, requirement: REQ-SIGNINGKEYS-002 }
  - { path: docs/domain/signing-keys/lifecycle/README.md, requirement: REQ-SIGNINGKEYS-003 }
  - { path: docs/domain/signing-keys/separation/README.md, requirement: REQ-SIGNINGKEYS-004 }
  - { path: docs/domain/signing-keys/separation/README.md, requirement: REQ-SIGNINGKEYS-005 }
  - { path: docs/domain/signing-keys/separation/README.md, requirement: REQ-SIGNINGKEYS-006 }
  - { path: docs/domain/signing-keys/separation/README.md, requirement: REQ-SIGNINGKEYS-007 }
  - { path: docs/domain/signing-keys/provider/README.md, requirement: REQ-SIGNINGKEYS-008 }
  - { path: docs/domain/signing-keys/provider/README.md, requirement: REQ-SIGNINGKEYS-009 }
  - { path: docs/domain/signing-keys/lifecycle/README.md, requirement: REQ-SIGNINGKEYS-010 }
  - { path: docs/domain/signing-keys/lifecycle/README.md, requirement: REQ-SIGNINGKEYS-011 }
  - { path: docs/domain/signing-keys/provider/README.md, requirement: REQ-SIGNINGKEYS-012 }
  - { path: docs/domain/sourcing/scim/README.md, requirement: REQ-SOURCING-001 }
  - { path: docs/domain/sourcing/scim/README.md, requirement: REQ-SOURCING-002 }
  - { path: docs/domain/sourcing/scim/README.md, requirement: REQ-SOURCING-003 }
  - { path: docs/domain/sourcing/scim/README.md, requirement: REQ-SOURCING-004 }
  - { path: docs/domain/sourcing/scim/README.md, requirement: REQ-SOURCING-005 }
  - { path: docs/domain/sourcing/scim/README.md, requirement: REQ-SOURCING-006 }
  - { path: docs/domain/sourcing/scim/README.md, requirement: REQ-SOURCING-007 }
  - { path: docs/domain/system/operations/README.md, requirement: REQ-SYSTEM-001 }
  - { path: docs/domain/system/operations/README.md, requirement: REQ-SYSTEM-002 }
  - { path: docs/domain/system/localization/README.md, requirement: REQ-SYSTEM-003 }
  - { path: docs/domain/system/localization/README.md, requirement: REQ-SYSTEM-004 }
  - { path: docs/domain/system/localization/README.md, requirement: REQ-SYSTEM-005 }
  - { path: docs/domain/system/hosted-ui/README.md, requirement: REQ-SYSTEM-006 }
  - { path: docs/domain/system/hosted-ui/README.md, requirement: REQ-SYSTEM-007 }
  - { path: docs/domain/system/localization/README.md, requirement: REQ-SYSTEM-008 }
  - { path: docs/domain/system/localization/README.md, requirement: REQ-SYSTEM-009 }
  - { path: docs/domain/system/localization/README.md, requirement: REQ-SYSTEM-010 }
  - { path: docs/domain/system/localization/README.md, requirement: REQ-SYSTEM-011 }
  - { path: docs/domain/system/operations/README.md, requirement: REQ-SYSTEM-012 }
  - { path: docs/domain/system/localization/README.md, requirement: REQ-SYSTEM-013 }
  - { path: docs/domain/system/api-boundary/README.md, requirement: REQ-SYSTEM-014 }
  - { path: docs/domain/system/hosted-ui/README.md, requirement: REQ-SYSTEM-015 }
  - { path: docs/domain/system/startup-configuration/README.md, requirement: REQ-SYSTEM-016 }
  - { path: docs/domain/system/startup-configuration/README.md, requirement: REQ-SYSTEM-017 }
  - { path: docs/domain/system/admission-control/README.md, requirement: REQ-SYSTEM-018 }
  - { path: docs/domain/system/admission-control/README.md, requirement: REQ-SYSTEM-019 }
  - { path: docs/domain/system/api-boundary/README.md, requirement: REQ-SYSTEM-020 }
  - { path: docs/domain/system/api-boundary/README.md, requirement: REQ-SYSTEM-021 }
  - { path: docs/domain/workloadidentity/attestation-exchange/README.md, requirement: REQ-WORKLOADIDENTITY-001 }
  - { path: docs/domain/workloadidentity/attestation-exchange/README.md, requirement: REQ-WORKLOADIDENTITY-002 }
  - { path: docs/domain/workloadidentity/attestation-exchange/README.md, requirement: REQ-WORKLOADIDENTITY-003 }
  - { path: docs/domain/workloadidentity/attestation-exchange/README.md, requirement: REQ-WORKLOADIDENTITY-004 }
  - { path: docs/domain/workloadidentity/attestation-exchange/README.md, requirement: REQ-WORKLOADIDENTITY-005 }
  - { path: docs/domain/workloadidentity/attestation-exchange/README.md, requirement: REQ-WORKLOADIDENTITY-006 }
  - { path: docs/domain/workloadidentity/attestation-exchange/README.md, requirement: REQ-WORKLOADIDENTITY-007 }
  - { path: docs/domain/workloadidentity/trust-configuration/README.md, requirement: REQ-WORKLOADIDENTITY-008 }
  - { path: docs/domain/workloadidentity/trust-configuration/README.md, requirement: REQ-WORKLOADIDENTITY-009 }
  - { path: docs/domain/workloadidentity/trust-configuration/README.md, requirement: REQ-WORKLOADIDENTITY-010 }
  - { path: docs/domain/ws-federation/relying-party/README.md, requirement: REQ-WSFEDERATION-001 }
  - { path: docs/domain/ws-federation/passive-sign-in/README.md, requirement: REQ-WSFEDERATION-002 }
  - { path: docs/domain/ws-federation/passive-sign-in/README.md, requirement: REQ-WSFEDERATION-003 }
  - { path: docs/domain/ws-federation/active-sts/README.md, requirement: REQ-WSFEDERATION-004 }
  - { path: docs/domain/ws-federation/active-sts/README.md, requirement: REQ-WSFEDERATION-005 }
  - { path: docs/domain/ws-federation/passive-sign-in/README.md, requirement: REQ-WSFEDERATION-006 }
---

# 残りの Context の機能仕様を、EARS 形式の要件だけで書く方式へ書き直す

## 動機

wi-26063 は、IdManagement と Tenancy 以外の 19 の Context を機能仕様と内部設計の構造へ移すが、規則の内容と書き方は変えない。
wi-17076 で定めた書き方は、構成の移行の後に別に適用する必要がある。
二つの書き方が並ぶ間は、読み手は Context ごとに読み方を変えなければならず、`check-unspecified-vocabulary` も IdManagement の外では働かない。

## 対象範囲

- wi-26063 で移した各 Context の機能仕様を、`user` と同じ方式で書き直す。
  要件の ID は変えず、本文を EARS 形式に改める。
- 状態機械を持つ Context（Authentication、DataKeys、IdentityGovernance、Jobs、OAuth2、Provisioning、SharedSignals、SigningKeys、WorkloadIdentity）に状態遷移表（マトリクス形式）を加える。
- 書き直した Context を `check-unspecified-vocabulary` の対象に加え、許容リストを減らす。
- すべての Context を書き直した後、状態遷移表（マトリクス形式）の存在を `check-spec` で求め、`SPECIFICATION_FORMAT.md` の記述に検査済みの印を付ける。
- 書き直しで見つけた未記載の振る舞いを、仕様にない振る舞いの分類に従って扱い、(c) は起票する。

## 対象外

- 構成の移行。
  wi-26063 が扱う。
- (c) に分類した振る舞いの実装の変更。
  分類の表の (c) ごとに起票した記録が扱う。
- 状態機械のモデルベースのテスト。
  wi-51017 が、wi-86874 の方式で扱う。

## 設計

一つの作業で書き直す Context の数と順序は、着手時に、wi-83002 と wi-71372 で測った一件あたりの費用から決める。

着手時に決めたこと：

- 19 Context は、機能仕様の README が約 3,600 行、要件が約 260 件で、Tenancy（535 行、43 件）の約 7 倍ある。それでも分割せず、この記録で行う。小さい Context から順に書き直し、Context ごとにチェックポイントを作る。
- 非同期の状態機械のモデルベースのテストは、時刻とジョブの扱いと共通化を wi-86874 が決めてから加える。wi-86874 を依存先に加えた。
- 書き直しを始めると、既存の日本語の EARS の型が主体も前置きの標識も定めず、どの文でも書けることが分かった。型と検査を wi-73023 で直してから再開する。wi-73023 を依存先に加え、`pending` に戻した。
- 中断時点で、作業ブランチには次の変更がある。再開時に引き継ぐ。
  - `check-unspecified-vocabulary` の対象を 19 Context に広げ、導入時点の 199 件を許容リストに載せた。System の語彙は `backend/shared/spec` から読む。
  - ApiTokens と ClaimMapping の空だった要件に要件文を書いた。wi-73023 の構文で書き直す。
  - EX-APITOKENS-002-02、03 の `Then` を、テストが固定している「主体を返さずに拒否する」に直した。

再開時に決めたこと：

- wi-73023 の完了後に再開した。要件文は wi-73023 の日本語の EARS の構文で書き、Context を書き直すたびに `check-spec` の構文の検査の一覧（`EARS_CONTEXTS`）と `check-unspecified-vocabulary` の対象に残す。
- 再開時に測ると、ApiTokens と ClaimMapping の 7 件を除く約 250 件の要件は、本文が空だった（見出しと例の付録だけがある）。Tenancy と同じく、例の付録、TypeSpec、実装、テストが固定している応答から要件文を書く。
- 状態機械は 21 個あり、どれにも状態遷移表（マトリクス形式）がない。
- 順序は、小さい Context から ClaimMapping、ApiTokens、WsFederation、DataKeys、Audit、Sourcing、Saml、Seeding、Authorization、WorkloadIdentity、SharedSignals、SigningKeys、Application、IdGovernance、Jobs、Provisioning、System、Authentication、OAuth2 とする。
- wi-86874 は、仕様の書き直しを進めた後、モデルベースのテストを加える前に実装する。
- すべての Context の書き直しと T005 を終えた時点で、残りの作業の大半が約 30 個の状態機械のモデルベースのテストだと分かった。テストは仕様の変更と独立に受け入れられるので、利用者の判断でこの記録から外して wi-51017 へ移し、wi-86874 を依存先から外した。

### Context ごとの進み具合

| Context | 要件文 | 状態遷移表 | 語彙の許容リスト |
| --- | --- | --- | --- |
| ClaimMapping | 済 | 該当なし | 済（対象なし） |
| ApiTokens | 済 | 該当なし | 済（対象なし） |
| WsFederation | 済 | 該当なし | 済（6 件を消した） |
| DataKeys | 済 | 済（DataEncryptionKeyLifecycle） | 済（1 件を消した） |
| Audit | 済 | 該当なし | 済（1 件を消した） |
| Sourcing | 済 | 該当なし | 済（対象なし） |
| Saml | 済 | 該当なし | 済（6 件を消した） |
| Seeding | 済 | 該当なし | 済（対象なし） |
| Authorization | 済 | 該当なし | 済（9 件を消した） |
| WorkloadIdentity | 済 | 済（WorkloadTrustBundleLifecycle、AgentWorkloadBindingLifecycle） | 済（17 件を消した） |
| SharedSignals | 済 | 済（SsfStreamLifecycle、SecurityEventDeliveryLifecycle） | 一部（17 件を消した。共通のエラーコード 6 件が残る） |
| SigningKeys | 済 | 済（SigningKeyLifecycle） | 済（2 件を消した） |
| Application | 済 | 該当なし | 済（20 件を消した） |
| IdGovernance | 済 | 済（WorkflowDefinitionLifecycle、WorkflowRunLifecycle） | 済（5 件を消した） |
| Jobs | 済 | 済（JobLifecycle） | 一部（3 件を消した。IdManagement が返す `jobs_unavailable` が残る） |
| Provisioning | 済 | 済（ProvisioningTaskLifecycle） | 一部（7 件を消した。発行されない 3 件が残る） |
| System | 済 | 該当なし | 一部（共有の `EmailSent` が残る） |
| Authentication | 済 | 済（TrustedDeviceLifecycle、IdentityProviderConnectionLifecycle） | 一部（43 件を消した。発行されない 9 件と OAuth2 が返す `webauthn_not_enrolled` が残る） |
| OAuth2 | 済 | 済（DeviceCodeFlow、LogoutNotificationLifecycle、ConsentLifecycle、ClientSecretCredentialLifecycle、ApprovalRequestLifecycle、AuthorizationCodeFlow、AuthorizationCodeRecordLifecycle、PARRecordLifecycle、RefreshTokenLifecycle） | 一部（40 件を消した。発行されない `AuthorizationDetailsRejected` が残る） |

### 未記載の振る舞いの分類

| # | Context | 見つけた振る舞い | 分類 | 扱い |
| --- | --- | --- | --- | --- |
| 1 | WsFederation | `wsignout1.0` と `wsignoutcleanup1.0` によるサインアウトが、どの要件にもない | (a) | REQ-WSFEDERATION-006 を加えた |
| 2 | WsFederation | EX-WSFEDERATION-001-02 は `wsfed:read` だけの変更を AccessDeniedError と書いていたが、実装とテストは 403 `insufficient_scope` を返す | (a) | 要件と例の付録をテストが固定している応答に直した |
| 4 | Saml | SAML の Single Logout（LogoutRequest の検証、ローカルセッションの破棄、LogoutResponse）が、どの要件にもない | (a) | REQ-SAML-009 を加えた |
| 5 | Saml | EX-SAML-005-02 は `saml:read` だけの変更を AccessDeniedError と書いていたが、実装とテストは 403 `insufficient_scope` を返す | (a) | 要件と例の付録をテストが固定している応答に直した |
| 6 | WorkloadIdentity | 実装が返す `workload_trust_bundle_not_found`、`agent_workload_binding_not_found`、`agent_workload_binding_pattern_conflict` を TypeSpec が宣言していない | (c) | 宣言の漏れとして起票する。wi-12645 で起票した |
| 7 | WorkloadIdentity | EX-WORKLOADIDENTITY-009-01 は別テナントの Agent への関連付けを InvalidRequestError と書いていたが、実装とテストは 422 `agent_workload_binding_agent_not_found` を返す | (a) | 要件と例の付録をテストが固定している応答に直した |
| 8 | WorkloadIdentity | 信頼設定と関連付けの削除の後の操作が 404 になること | (a) | 状態遷移表に終端の状態 `deleted` を加えた |
| 9 | SharedSignals | 遷移の表は上限に達した配送が `failed` から `dead_letter` へ移ると書いていたが、実装は上限に達した試行で `pending` から `dead_letter` へ移し、`failed` は再試行の時刻に `pending` へ戻る | (a) | 遷移の表を実装の遷移に直し、状態遷移表を加えた |
| 10 | SharedSignals | ストリームの削除の後の操作が 404 になること | (a) | 状態遷移表に終端の状態 `deleted` を加えた |
| 11 | SigningKeys | 検証用の鍵の無効化は、PostgreSQL の構成では鍵をアーカイブし、メモリの構成では鍵を削除し、どちらもイベントを発行しない | (c) | 構成による食い違いと監査の欠落として起票する。要件には両方の構成で共通する「JWKS とメタデータから除く」だけを書いた。wi-83307 で起票した |
| 12 | SigningKeys | 遷移の表の `SigningKeyRetired` を発行する経路がなく、`Retired` は重複期間を過ぎたことを表すだけである。無効化は `Verifying` と `Retired` から `Archived` へ直接移す | (a) | wi-83002 と同じく、遷移の表の Event を遷移の契機の名前として扱い、無効化の遷移を加えた |
| 13 | Application | カテゴリの作成、更新、削除、アプリケーションへの設定が、どの要件にもない | (a) | REQ-APPLICATION-015 を加えた |
| 14 | Application | 実装が返す `invalid_icon` と `invalid_sign_in_policy` を TypeSpec が宣言していない | (c) | 宣言の漏れとして起票する。wi-12645 で起票した |
| 15 | Application | EX-APPLICATION-004-02、003-03 は AccessDeniedError、008-02 は InvalidRequestError と書いていたが、実装は 403 `insufficient_scope` と 400 `invalid_icon` を返す | (a) | 要件と例の付録を実装の応答に直した |
| 16 | IdGovernance | 状態の表は `partially_failed` と `failed` を終端の状態としていたが、管理者の再試行で `queued` に戻る。`enabled` のワークフローの再有効化と、各状態での編集は状態を変えずにイベントを発行する | (a) | 状態の表と遷移の表を実装に直し、自己遷移を加えた |
| 17 | Jobs | リースの期限を過ぎた `running` のジョブを別の `worker` が取得し直す遷移が、遷移の表にない | (a) | 自己遷移 `running → running` を加えた |
| 18 | Jobs | `jobs_unavailable` は Jobs の TypeSpec が宣言しているが、返すのは IdManagement のデータエクスポートだけで、Jobs の要件に書けない | (c) | 宣言の置き場所の誤りとして起票する。許容リストに残した。wi-12645 で起票した |
| 19 | Provisioning | 状態の表はタスクの再試行が新しいレコードを作ると書いていたが、実装は同じタスクを `dead_letter` から `pending` に戻す | (a) | 状態の表と遷移の表を実装に直した |
| 20 | Provisioning | `ProvisioningConnectionUpdated`、`ProvisioningConnectionDisabled`、`ProvisioningConnectionDeleted` を宣言しているが、発行する経路がない | (c) | 発行の欠落として起票する。許容リストに残した。wi-37490 で起票した |
| 21 | Provisioning | 例の付録は適用範囲の外、既存の接続、再試行できないタスクを個別のエラーと書いていたが、実装はどれも 409 `provisioning_conflict` を返す。`provisioning:read` だけの変更は 403 `insufficient_scope` を返す | (a) | 要件と例の付録を実装の応答に直した |
| 22 | System | `EmailSent` は `backend/shared/spec` の共有のイベントで、発行するのは Authentication と IdManagement であり、System の要件に書けない | (b) | 語彙の検査が `backend/shared/spec` を System に対応づけたことによる。許容リストに残した |
| 23 | Authentication | `AuthenticationStepCompleted`、`AuthenticationStepFailed`、`MfaChallengeIssued`、`MfaChallengeSucceeded`、`MfaChallengeFailed`、`SessionStarted`、`SessionRefreshed`、`SessionImpersonationStarted`、`SessionImpersonationEnded` を宣言し保持期間の表に載せているが、発行する経路がない。`SessionImpersonationStarted` が発行されないので、`impersonation` の必須のセキュリティ通知も送られない | (c) | 発行の欠落として起票する。REQ-AUTHENTICATION-031 はなりすましの通知を求めたままにし、許容リストに残した。wi-37490 で起票した |
| 24 | Authentication | WebAuthn の資格情報の登録、解除と、ステップアップ認証の開始と完了（`StepUpRequested`、`StepUpCompleted`、`step_up_failed`）を実装しているが、要件がなかった | (a) | REQ-AUTHENTICATION-038 と REQ-AUTHENTICATION-039 を加えた |
| 25 | Authentication | 外部 IdP の接続は、`Active` のまま信頼の根拠を更新すると `Disabled` に戻り、削除で消える。`Active` の接続の有効化は 400 `invalid_state`、存在しない接続の削除と関連付けていない解除は 204 を返す。`IdentityProviderConnectionActivated` と `IdentityProviderConnectionDisabled` は発行しない | (a) | 遷移の表に信頼の根拠の更新と削除（`Deleted`）を加え、二つのイベント名を遷移の契機の名前と明記した |
| 26 | Authentication | 失効済みの信頼済みデバイスの再失効は 204 を返し、イベントを発行しない | (a) | 遷移の表から `Revoked` の自己遷移を除き、マトリクスで「何もしない」とした |
| 27 | OAuth2 | 認可詳細の種類と MCP のリソースサーバーの管理 API（`type_exists`、`type_not_found`、`invalid_type`、`resource_exists`、`resource_server_not_found`）、authorization_details の要求と同意、リソースの指定（RFC 8707）を実装しているが、要件がなかった | (a) | REQ-OAUTH2-051、REQ-OAUTH2-052、REQ-OAUTH2-053、REQ-OAUTH2-054 を加えた |
| 28 | OAuth2 | デバイス認可の交換は `offline_access` を問わずリフレッシュトークンを発行し、REQ-OAUTH2-021 に反する | (c) | 実装の修正として起票する。wi-87724 で起票した |
| 29 | OAuth2 | 期限を過ぎた `user_code` の拒否は期限を確かめず、記録を `Denied` にする | (c) | マトリクスは `拒否：400 expired_token` とし、実装の修正として起票する。wi-87724 で起票した |
| 30 | OAuth2 | 撤回した同意も期限の切れた同意も、認可での再同意で `Granted` に戻る。撤回済みの同意の再撤回は 204 を返し `ConsentRevoked` を再び発行する。`Expired` は保存しない時刻の判定である | (a) | ConsentLifecycle の遷移の表に再同意と再撤回の遷移を加え、`Revoked` と `Expired` を終端から外した |
| 31 | OAuth2 | AuthorizationCodeFlow の仕様の遷移の表と実装の遷移の表が食い違っていた（実装は `Received` からの期限切れを持ち、`Authenticated` と `Consented` からの拒否を持たない） | (a) | 仕様の遷移の表を実装の遷移の表に合わせた |
| 32 | OAuth2 | 認可の処理は同意の判定を `Received` のまま行い、`ConsentPending` と `Consented` を経由しない | (c) | 実装の修正として起票する。遷移の表は同意の段を残した。wi-54733 で起票した |
| 33 | OAuth2 | `/authorize` はすべてのクライアントに PKCE を求めるが、モデルと判断はクライアントごとの `require_pkce` で定めるとしている | (c) | 実装をモデルへ合わせるか判断を改めるかを起票する。wi-61672 で起票した |
| 34 | OAuth2 | `AuthorizationDetailsRejected` を宣言しているが、発行する経路がない | (c) | 発行の欠落として起票する。許容リストに残した。wi-37490 で起票した |
| 35 | OAuth2 | リフレッシュトークンのローテーションは `RefreshTokenIssued` を発行しないが、EX-OAUTH2-006-01 は発行すると書いていた。EX-OAUTH2-005-01 のスコープはテストと異なり `offline_access` を欠いていた。ほかに例の付録の `AccessDeniedError`、`InvalidRequestError` が実装の応答と異なっていた | (a) | 例の付録を実装の応答とテストの要求に直した |
| 3 | ApiTokens | EX-APITOKENS-002-02、03 は AccessDeniedError と書いていたが、HTTP の境界では 401 `invalid_token` を返し、テストはポートの拒否を固定している | (a) | 要件に HTTP の応答を書き、例の付録は「主体を返さずに拒否する」に直した |

### 構文の検査の調整

- ツールの整形で `EARS_CONTEXTS` の一覧が複数行に折り返された後、一行を前提にした置換が黙って効かず、WsFederation から Seeding までの 6 Context が構文の検査の対象に入っていなかった。気付いた時点で一覧を直し、9 行の違反（応答の中の「なら」「場合」「とき」「〜ば」と、標識のない前置き）を直し、要件文から主体を消すと `check-spec` が失敗することを確かめた。以後は一覧の追加を確かめてから検査を走らせる。
- 状態遷移表（マトリクス形式）の検査は、状態の表と遷移の表の状態名から Markdown の記号（`_` を含む）を除いて読むのに、マトリクスの `→` の先はそのまま照合していたので、`dead_letter` のような名前を宣言していないと誤って報告した。テストを先に書き、マトリクスの先も同じ規則で読むよう直した。
- 状態の前置きに「〜を持つ間、」のような動詞に続く「間」を書くと、検査は「の間、」しか標識として認めず拒否した。言い換えると不自然になるので、動詞の終止形の語尾（う段の仮名と「い」）に続く「間、」も状態の標識とし、漢字に続く「期間、」「時間、」は標識としないことをテストで固定した。応答の中の条件の語にも同じ形を加えた。

## タスク

- [x] T001 [Spec] 各 Context の機能仕様を書き直す。
- [x] T002 [Spec] 状態機械に状態遷移表（マトリクス形式）を加える。
- [x] T004 [Tooling] 語彙の検査の対象を広げ、許容リストを減らす。
- [x] T005 [Tooling] 状態遷移表（マトリクス形式）の存在を検査で求める。
- [x] T006 [Verify] 変更を検証する。

## 検証

- `mise run check-spec`、`mise run check-unspecified-vocabulary`
- `mise run spec-diff` で、要件の差分が本文の書き直しと、状態遷移表の追加だけであること。
- `mise run verify`

## リスク

- 検査の対象を広げると、許容リストが膨らみ、読まれなくなる。
  Context を書き直すたびに対象へ加え、許容リストの項目をその作業の中で減らす。

## 完了

- **Completed At**: 2026-10-04
- **Summary**:
  `mise run spec-diff` によれば、要件を 9 件加え（REQ-APPLICATION-015、REQ-AUTHENTICATION-038、REQ-AUTHENTICATION-039、REQ-OAUTH2-051〜054、REQ-SAML-009、REQ-WSFEDERATION-006）、IdManagement と Tenancy 以外の 19 Context の要件 262 件の本文を、日本語の EARS の要件文に書き直した。
  20 個の状態機械の遷移の表を、実装が通る遷移に合わせて改め（同意の再同意と再撤回、IdP の接続の信頼の根拠の更新と削除、信頼済みデバイスの再失効、ジョブの再取得など）、すべての状態機械に状態遷移表（マトリクス形式）を加えた。
  `check-spec` は全 Context の要件文の構文と、すべての状態機械のマトリクスの存在を検査し、`check-unspecified-vocabulary` は全 21 Context を対象にした。
  未記載の振る舞いは 35 件を分類し、(c) の 11 件を 6 件の記録（wi-12645、wi-83307、wi-37490、wi-87724、wi-54733、wi-61672）に起票した。
  モデルベースのテストは、利用者の判断で wi-51017 へ移した。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（`EARS_CONTEXTS` に各 Context を加えた時点）
  - **Requirement**: N/A: 文書の書き直しであり、製品の振る舞いの要件を変えない。
  - **Observed Failure**: Context を構文の検査の対象に加えるたびに、前置きの標識の欠落、応答の中の条件（「場合」「限り」「とき」）、曖昧な語（「など」）を報告した。OAuth2 では `REQ-OAUTH2-035 response contains the condition 「場合」`、`REQ-OAUTH2-013 uses vague wording 「など」` などを観測した。
  - **Detection Reason**: 検査が要件文を一文ずつ読み、主体、標識、応答の中の条件を確かめるので、自由な散文に戻した要件文を区別できる。Authentication では故障を入れた要件文（主体のない文）を加えて拒否されることも確かめた。
- **Unit RED Evidence**:
  - **Test**: `tools/check/src/specification-doc.test.ts` の `state matrix > rejects a machine without a matrix`、`specification-rules.test.ts` の動詞に続く「間」のテスト、`specification-doc.test.ts` の下線を含む遷移先のテスト
  - **Requirement**: N/A: 検査の道具の変更である。
  - **Observed Failure**: マトリクスのない状態機械を受け入れていたので、`rejects a machine without a matrix` が期待したメッセージを得られずに失敗した。`dead_letter` を遷移先に持つマトリクスを、宣言していない状態として誤って報告した。
  - **Detection Reason**: マトリクスの欠落、下線を含む状態名、動詞に続く「間」の三つを、それぞれ受け入れ側と拒否側の表明で固定した。
- **Change-Resistance Results**:
  - 変更した Go のコードはない。検査の道具の変更（TypeScript）は、上の RED を GREEN にしたテストで固定した。変異ツールは Go だけを対象にするので、検査は手で故障を入れて確かめた。
  - `EARS_CONTEXTS` の一覧を整形が複数行に崩した後、一部の Context が検査の対象から外れていた事故を見つけ、故障を入れた要件文で各 Context が検査されることを確かめ直した。
  - 最初の `mise run verify` は、例の付録の `AccessDeniedError` を実装の応答（`insufficient_scope`）に直したことで、WS-Federation、Provisioning、SAML の 403 の `AccessDeniedError` を宣言する例がなくなり、`check-repository` の R4 で失敗した。`AccessDeniedError` は `admin` のロールを持たない利用者への拒否（`access_denied`）なので、その拒否を要件文と例（EX-WSFEDERATION-001-04、EX-SAML-005-04、EX-PROVISIONING-001-04）で宣言し、参照の 403 だけを確かめていた既存のテストを、削除の要求が `access_denied` で拒否され対象が残ることまで確かめるテストに強めた。役割の判定を外す誤実装では、削除が成功して対象が消えるので、三つのテストは失敗する。
- **Verification Results**:
  - `mise run check-spec` - 成功
  - `mise run check-unspecified-vocabulary` - 成功
  - `mise run check-work-items` - 成功
  - `mise run verify` - 成功（サンドボックスの外で実行し、PostgreSQL のテストを含む）
