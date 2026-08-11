---
context: frontend
updated_at: 2026-07-26
---

# Architecture: frontend

## Overview

The React UI is a separate build artifact from the Go API, joined into one origin by a gateway. It owns
the hosted authentication surfaces (login, consent, device), the admin console, and the account portal.

This document is the design record for that boundary: how the SPA and the API divide responsibility,
which browser protections apply, how routing and UI conventions are fixed, and why. The machine-checked
module boundaries are inferred from paths and forbidden imports; run instructions and verification commands live in
`README.md`.

## Deployment boundary

React and Go are separate build artifacts and separate services.

```text
Browser
  |
  | same origin
  v
Gateway / static server (Caddy, Nginx, CDN + proxy, etc.)
  |-- /login, /consent, /device, /status, /admin/* -> React SPA
  `-- /api/* and OAuth/OIDC endpoints                -> Go
```

Caddy is the reference configuration, not a required runtime. Any gateway that preserves the
same-origin boundary, TLS, headers, and routing contract can replace it.

## Authorization transaction

The Go service keeps the complete OAuth authorization request server-side. Its internal UUID is
stored only in a short-lived `HttpOnly`, `Secure` in HTTPS, `SameSite=Lax` transaction cookie.
It is not included in HTML, URLs, or JavaScript-readable application state.

The SPA calls `GET /api/auth/transaction` to obtain only display data such as the screen kind,
client name, and requested scopes. Login and consent commands resolve the transaction from the
cookie.

## Browser protections

- Session and authorization transaction cookies are `HttpOnly`.
- State-changing UI APIs require a double-submit CSRF cookie and `X-CSRF-Token` header.
- State-changing browser APIs require an `Origin` header matching the configured public issuer.
- Consent verifies that the current login session subject matches the authorization transaction.
- Authorization requests expire after ten minutes and completed requests cannot be reused.
- OAuth redirect URIs, PKCE values, scopes, and client identifiers are read from server-side state.
- UI API responses use `Cache-Control: no-store` and never return credentials or internal request IDs.

## API boundary

Browser-facing authentication APIs live under `/api/auth/*`. OAuth/OIDC protocol endpoints retain
their standard paths. Management APIs live under `/api/admin/*` and self-service APIs under
`/api/account/*`; both use explicit authorization policies independently from the login
transaction APIs.

## Admin console and account portal as OIDC RPs

The admin console (`/admin/*`) and account portal (`/account/*`) authenticate as OIDC relying
parties of the IdP itself, using `authorization_code` + PKCE against the IdP's own `/authorize`
and `/token`. They are registered as first-party public clients with fixed UUID `client_id`s
(admin `…0022`, account `…0023`, mirrored in `src/api/oidc.ts` and the bootstrap seed) whose
consent screen is skipped because the resource owner is the IdP user.

Because they are pure SPA RPs, the access token is held in the browser (`sessionStorage`) and sent
as `Authorization: Bearer` to `/api/{admin,account}/*`, which validate it as RFC 9068 resource
servers. This is a deliberate departure from a strict "no tokens in JavaScript" posture; it is
bounded by short-lived access tokens (600 s), `Cache-Control: no-store`, and keeping tokens out of
URLs, logs, and the DOM. The first-party session login (`POST /api/auth/login`) is retained as an
emergency bootstrap path so a broken OIDC client/key configuration cannot lock administrators out.

## Client-side routing

The SPA uses TanStack Router for client-side navigation with file-based routes under
`src/routes/`. The Vite router plugin generates `src/routeTree.gen.ts` and applies automatic code
splitting, including route loaders and components. Route files follow the request path structure:
`admin/route.tsx` and `account/route.tsx` are thin layout routes that render `<Outlet>`, while
`admin/index.tsx`, `account/index.tsx`, and leaf route files own their own `loader`, API requests,
page component, and path params. Detail pages that should not render through a
list page use TanStack Router's trailing underscore convention, for example
`admin/users_/$sub.tsx` for `/admin/users/$sub`. Files prefixed with `-` are route-local helpers
and are excluded from route generation. Internal admin/account navigation uses `<Link>`, so moving
between pages does not reload the document or re-fetch every page's data — only the target route's
loader runs.
The OIDC login guard (`ensureLoggedIn`) runs inside the loader, so it applies to both the initial
load and in-app navigation. Auth-flow transitions (login/consent/callback and the OIDC redirects)
remain full-page navigations by nature. The rendered page kind is asserted to the DOM via
`<meta name="idmagic:page">` for E2E.

## Design Guidelines

- **Prioritize Trust**: Establish a secure experience through calm color palettes, clear service identification, and transparent descriptions of current operations and security states, reassuring users during authentication decisions.
- **Present Critical Information First**: Clear information hierarchies display the page title, requesting party, shared information, next action, and cancellation procedures up-front.
- **Simplify Critical Actions**: Limit screens to a single primary action, visually distinguishing it from reject/cancel operations. Avoid modifying OAuth/OIDC form names, submission values, and transition contracts for UI-specific reasons.
- **Accessibility as Standard**: Support keyboard navigation, visible focus indicators, sufficient color contrast, explicit labels, appropriate `aria-*` attributes, and animations respecting reduced-motion preferences.
- **Density for Enterprise Use**: Avoid excessive animations or consumer-focused decorations. Maintain a structured layout using consistent spacing, typography, borders, and state colors.
- **Responsive Without Data Loss**: Display supplementary details on desktops while prioritizing authentication operations on mobile without omitting service identification or safety warnings.
- **Consistency via Shared Components**: Utilize local components conforming to Tailwind CSS, Radix UI, and shadcn/ui. Avoid ad-hoc implementations of colors, border-radii, focus rings, or disabled states.

## Admin Console Policy

The Admin Console's information design is inspired by directory-centric systems like Keycloak, Okta, and Google Cloud IAM. It uses a left-hand navigation sidebar to identify management targets, displays search, status, and major permissions in a high-density table format, and presents detailed views and modification options within the same context. Destructive operations (such as deletion or disabling) are visually separated from standard read-only views.

- **Tables as Command Centers**: Use list views as the primary workspace, allowing users to search, filter, and review status, MFA config, and roles at a glance.
- **Verify Before Modifying**: Enable users to inspect principal IDs, authentication status, and assigned permissions in detail panes before committing changes.
- **Explicit Permission Modification**: Avoid inline editing for sensitive role changes. Use dedicated configuration screens displaying differences (additions/deletions) before confirmation.
- **Visible Danger Actions**: Highlight dangerous actions with clear descriptions and appropriate warning colors to prevent accidental execution.
- **Secure Credentials**: Display client secrets exactly once upon creation. Re-confirm client deletion only after reviewing affected systems.
- **Scalable Architecture**: Structure navigation to accommodate future modules like groups, applications, and audit logs. Unimplemented features must not appear interactive.
- **Consistent Layout**: Maintain a unified structure across the console using `AdminShell` (headers, sidebars, breadcrumbs, content widths, and action placements).
- **Unauthorized Link Fallback**: Redirect unauthenticated direct requests to `/admin/*` to `/login`, returning to the original target destination upon successful login. Allowed redirection targets are constrained to the current realm's `/admin` path.

*References:*
- [Keycloak Server Administration Guide](https://www.keycloak.org/docs/latest/server_admin/)
- [Okta Manage users](https://help.okta.com/en-us/content/topics/users-groups-profiles/usgp-people.htm)
- [Google Cloud IAM access management](https://cloud.google.com/iam/docs/granting-changing-revoking-access)

## UI Library Selection

The UI foundation balances accessibility and design consistency without relying on complex, pre-packaged themes.

| Library | Role | Selection Rationale |
| --- | --- | --- |
| React + TypeScript | UI and type-safe views | Maintains clear component boundaries and state management from simple login screens to the administrative console. |
| Vite | Dev server and production build | Fast, straightforward generation of static bundles that can be served via API gateways or CDNs. |
| Tailwind CSS | Design tokens and styling | Enables consistent styling (states, responsiveness, accessibility) while preserving enterprise branding controls. |
| Radix UI | Accessible headless primitives | Accessible keyboard handling and ARIA compliance decoupled from visual presentation. |
| Local Components (shadcn/ui layout) | Buttons, Inputs, Labels, Cards, Alerts | Maintained within the repository for easy audit and customization, minimizing runtime dependency overhead. |
| TanStack Router | Type-safe routing | Safe translation of page metadata from the Go backend to target UI views. |
| TanStack Table | Administrative data grid | Separates sorting, filtering, and pagination logic from UI presentation. (Reserved for user/client tables; currently unused in the 4 core login screens). |
| Tabler Icons | Vector icons | Consistent line weights and extensive library to serve as visual aids for states and actions rather than mere decoration. |
| Class Variance Authority / Clsx / Tailwind Merge | Class merging | Type-safe styling variants and runtime merging of conflicting Tailwind classes. |
| Biome | Linter and formatter | Rapid automated enforcement of syntax, style, and code quality guidelines. |

Priorities are accessibility, bundle size, maintainability, design ownership, and preserving API contracts. Introduce new libraries only when existing tools cannot satisfy specific requirements.

## UI navigation and consistency policy

The admin console and account portal follow a set of strict UI consistency and navigation guidelines, applied to Entra federation and to external identity providers (`/admin/identity-providers`).

1. **Detail-then-Edit Navigation Policy**
   - For resource creation or editing, the UI must separate the read-only view (detail) from the write/edit view.
   - The user is first presented with a read-only detail view of the resource configuration, with an explicit "Edit" button that navigates to a dedicated edit route (e.g., `/admin/users/$id/edit` or `/account/profile/edit`).
   - Modals should not be used for primary resource creation or editing; they must use dedicated routed pages to ensure predictable browser "Back" button behavior and deep-linking capabilities.
2. **List-View Action Unification**
   - Action buttons (Detail, Edit, Delete, etc.) in table list views must be visible directly in each row rather than hidden under dropdown/kebab menus.
   - Destructive actions (such as deletion) must use red-toned buttons (`variant="outline" tone="danger"`).
3. **Dynamic Page Titles**
   - Every page must have a dynamic and context-aware browser tab title (e.g., "ユーザー | IdMagic 管理コンソール") defined via the `PAGE_TITLES` map in `src/routes/-page.tsx` and evaluated by the `PageMarker` component.
4. **Terminology Unification**
   - The UI must use the term "監査イベント" (Audit Event) instead of "監査ログ" (Audit Log) to maintain consistency with the underlying specification (`AuditEvent`/`audit_events`).

## Container / Presentation component split

New `*Page.tsx` files (and refactors of existing ones) follow a container/presentation split so that UI rendering can be unit-tested apart from data fetching and side effects:

1. **Split by meaning, not by file.** The exported `XxxPage` function stays a thin container: it owns `useState`, API calls, and effects, and lays out the page's `*Shell` wrapper directly. Do not wrap an entire page in a single `XxxPresentation` twin that re-receives every piece of container state as a prop — that only relocates the same complexity behind an extra layer.
2. **Extract at the section boundary.** Pull out a presentational component for each self-contained unit that benefits from isolated testing — a form with its own validation (e.g. `DefaultPolicyFormPresentation` in `AdminSignInPolicyPage.tsx`), an item list (`PasskeyList`), or a card with interactive state (`TotpEnrollmentForm` in `AccountSecurityPage.tsx`). Purely static, read-only markup can stay inline in the container; it does not need its own component.
3. **Keep presentational props small.** A presentational component should take only the props its own section needs (typically well under 10), plus callbacks for the actions it triggers — never the container's entire state object. If a component's prop list balloons because a page has several independent sections, split it further into one component per section instead of widening the props.
4. **No side effects in presentational components.** They receive data and callbacks and render; `fetch`/`api.*` calls, `useEffect`, and navigation stay in the container (or in a small section-local container, e.g. `DefaultPolicyCard`, when a section manages its own state before delegating to a pure form).
5. **Test what was extracted.** Each extracted presentational component and any pure helper function (date formatting, validation, derived-value calculators) gets a Vitest/Testing Library unit test. Components that wrap `AccountShell`/`AdminShell`/`AuthShell` need a router context to render (those shells use TanStack Router's `Link`); use the `renderWithRouter` test helper (`src/test/renderWithRouter.tsx`) for those instead of skipping the test.

## Design Decisions

- Design rationale, the machine-checked module ledger, and run/verification instructions are kept in
  Architecture and run instructions are kept separate because they answer different questions, rather than one combined
  document ([ADR-143](../decisions/ADR-143-second-layer-design-ledger-decision-split.md)).
- The admin console and account portal are first-party OIDC relying parties of the IdP itself, as pure
  SPA RPs holding the access token in the browser rather than behind a BFF
  ([ADR-061](../decisions/ADR-061-first-party-portals-as-oidc-rp.md)).
- Internally generated id columns, including the admin/account portals' fixed `client_id`s, are typed as
  `UUID` rather than `TEXT`
  ([ADR-084](../decisions/ADR-084-postgres-column-type-policy.md)).
- The admin console and account portal follow a fixed set of UI consistency rules: detail-then-edit
  navigation instead of inline/modal editing, in-row list actions instead of kebab menus, dynamic
  per-page browser tab titles, and "監査イベント" instead of "監査ログ" as the audit terminology
  ([ADR-086](../decisions/ADR-086-ui-navigation-consistency-and-page-title-policy.md)).
- The tenant-wide default sign-in policy applies to an application as an override, not a composed floor,
  of that application's own policy — the precedent that motivated this file's container/presentation
  component split, first applied to the admin UI built for that decision
  ([ADR-081](../decisions/ADR-081-tenant-default-sign-in-policy-composition.md)).
