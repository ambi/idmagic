---
context: authentication
updated_at: 2026-07-27
---

# Architecture: authentication

## Overview

The `Authentication` context owns credential verification, MFA, login sessions, step-up, recovery,
and login-time federation. Independent capabilities are vertical feature slices with their own domain,
ports, use cases, and adapters; `module.go` remains the context composition boundary.

## Inbound identity federation

`federation/` is the identity-broker feature slice. It owns tenant-scoped upstream OIDC and SAML
connections, external-subject links, single-use login attempts, protocol validation, linking policy,
and the handoff to an IdMagic login session. Downstream SAML IdP and WS-Federation issuance remain in
their protocol contexts. Identity Management exposes credential-less user creation for JIT, but does
not own the login-time correlation policy. The ownership and rejected sourcing-context alternative are
recorded in [ADR-146](../../decisions/ADR-146-authentication-owned-identity-broker.md).

The broker first resolves an immutable external subject through a `FederatedIdentity`. Verified-email
linking is available only under an explicit connection policy, a verified upstream claim, and a unique
tenant-local match. JIT is separately enabled per connection and may be narrowed by an email-domain
allowlist. Explicit link and unlink operations require recent step-up, and the last usable sign-in
method cannot be removed.

Protocol adapters treat every upstream document as untrusted. OIDC uses saved HTTPS discovery
endpoints, Authorization Code with PKCE, state, nonce, issuer/audience/time checks, and a constrained
JWK algorithm. SAML uses a correlated AuthnRequest and validates XML signature, issuer, destination,
audience, subject confirmation, time, and replay. Unsolicited IdP-initiated responses and encrypted
assertions are outside the initial SAML adapter.

Client secrets are represented only by a `secret_reference`; the initial runtime resolver accepts
`env:NAME` references. Neither raw secrets nor upstream tokens/assertions are persisted or returned.
Public provider discovery exposes only active provider identifiers, display names, and protocols.

After callback validation the broker creates a normal Authentication session with `federated` in AMR.
OAuth authorization resumes through `/authorize/resume`, so application policy, required actions,
consent, and code issuance remain owned by OAuth2 rather than being duplicated in the broker.
