---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p3
depends_on: [wi-26063-move-remaining-contexts-to-the-feature-and-design-layout, wi-71372-rewrite-tenancy-specifications-in-the-requirement-format]
change_kind: docs
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
---

# 残りの Context の機能仕様を、EARS 形式の要件だけで書く方式へ書き直す

## 動機

wi-26063 は、IdManagement と Tenancy 以外の 19 の Context を機能仕様と内部設計の構造へ移すが、規則の内容と書き方は変えない。
wi-17076 で定めた書き方は、構成の移行の後に別に適用する必要がある。
二つの書き方が並ぶ間は、読み手は Context ごとに読み方を変えなければならず、`check-unspecified-vocabulary` も IdManagement の外では働かない。

## 対象範囲

- wi-26063 で移した各 Context の機能仕様を、`user` と同じ方式で書き直す。
  要件の ID は変えず、本文を EARS 形式に改める。
- 状態機械を持つ Context（Authentication、DataKeys、IdentityGovernance、Jobs、OAuth2、Provisioning、SharedSignals、SigningKeys、WorkloadIdentity）に状態遷移表（マトリクス形式）を加え、wi-86874 の方式でモデルベースのテストを加える。
- 書き直した Context を `check-unspecified-vocabulary` の対象に加え、許容リストを減らす。
- すべての Context を書き直した後、状態遷移表（マトリクス形式）の存在を `check-spec` で求め、`SPECIFICATION_FORMAT.md` の記述に検査済みの印を付ける。
- 書き直しで見つけた未記載の振る舞いを、仕様にない振る舞いの分類に従って扱い、(c) は起票する。

## 対象外

- 構成の移行。
  wi-26063 が扱う。
- (c) に分類した振る舞いの実装の変更。

## 設計

一つの作業で書き直す Context の数と順序は、着手時に、wi-83002 と wi-71372 で測った一件あたりの費用から決める。

## タスク

- [ ] T001 [Spec] 各 Context の機能仕様を書き直す。
- [ ] T002 [Spec] 状態機械に状態遷移表（マトリクス形式）を加える。
- [ ] T003 [Test] モデルベースのテストを加える。
- [ ] T004 [Tooling] 語彙の検査の対象を広げ、許容リストを減らす。
- [ ] T005 [Tooling] 状態遷移表（マトリクス形式）の存在を検査で求める。
- [ ] T006 [Verify] 変更を検証する。

## 検証

- `mise run check-spec`、`mise run check-unspecified-vocabulary`
- `mise run spec-diff` で、要件の差分が本文の書き直しと、状態遷移表の追加だけであること。
- `mise run verify`

## リスク

- 検査の対象を広げると、許容リストが膨らみ、読まれなくなる。
  Context を書き直すたびに対象へ加え、許容リストの項目をその作業の中で減らす。
