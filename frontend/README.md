# IdMagic UI

The authorization UI of `idmagic` aims to provide a modern, compliant, easy-to-use, and high-visual-quality authentication and identity management experience suitable for enterprise environments.

## Implementation Guidelines

When modifying the UI, run the following verification checks:
```bash
bun run lint
bun run typecheck
bun run build
```

When API contracts are modified, run Go HTTP E2E tests to verify that cookies, CSRF protection, OAuth redirects, and JSON schemas remain correct.

While the Vite CLI utilizes `#!/usr/bin/env node`, dev and build scripts execute JS entries directly via `bun`. This unifies running processes under `bun .../vite.js` without requiring a Node.js runtime.

## E2E Smoke Tests

Verify the SPA's golden path (`/authorize → login → consent → callback`) by running:
```bash
bun run test:e2e
```

The runner uses `bun test` and the built-in `Bun.WebView` (using WKWebView on macOS and Chrome via CDP on Linux/Windows), eliminating the need for heavy browser automation frameworks or manual driver downloads.

The test suite (`tests/e2e/`) automatically manages the lifecycle of:
1. **Go API**: Starts in `memory` mode on port `:8081` (`ADDR=:8081 ISSUER=http://localhost:5173`) to match the browser origin and pass CSRF checks.
2. **Vite Dev Server**: Starts on port `:5173`, proxying `/authorize` and `/api` requests to `8081`.
3. **Mock Callback Server**: Starts on port `:3000` to receive the auth code at the development seed's external demo client's `redirect_uri` (`http://localhost:3000/callback`; client ID `00000000-0000-4000-8000-000000000021`).

This setup validates client routing (`meta[name="idmagic:page"]`) and ensures that `code` and `iss` parameters are preserved during cross-origin redirects (RFC 9207). Requires only `go` and `bun` in your `PATH`.

## Localization (UI Display Languages)

The hosted authentication, account, and admin UI support Japanese (`ja`) and English (`en`) only.
English is the product default. Set `VITE_DEFAULT_LOCALE=ja` or `VITE_DEFAULT_LOCALE=en` at
application startup to choose the final fallback when neither an explicit nor a browser locale is
available; an unset or invalid value falls back to English.

Add user-visible copy to the dictionary that is local to its feature (for example,
`frontend/src/features/auth-flow/LoginPage.i18n.ts`) and provide both locale values in
the same change. Use `defineDictionary` so TypeScript rejects missing or extra keys;
run `just verify-ui` before committing. Do not add another locale without a separately
specified product decision. Translate only stable backend error codes in the receiving
UI dictionary; render an unknown backend message unchanged, because backend error text
is intentionally English-only.
