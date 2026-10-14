---
status: completed
archived_from: 92e556a5d23d6cfc329c4005fe9c71f351945f13
---

# ゲートウェイの経路許可リストが実行時の経路表からずれていることを検出し、現在のずれを塞ぐ

The normative diff against `main` adds one scenario, `REQ-SYSTEM-021`: the reference frontend gateway's
route allowlists are compared against the assembled route table, and drift is reported instead of being
served as the SPA. Four routes the allowlists did not carry now reach the API through both the host form
and the `/realms/{tenant_id}` path form: the Shared Signals receiver
`/ssf/streams/{stream_id}/events`, the OIDC `check_session_iframe` at `/session/check`, the application
icon URL at `/application-icons/{application_id}/{id}`, and `/authorize/resume`, which the Caddyfile
alone was missing because its `/authorize` entry matches exactly while the Vite key is a prefix. Whether
a route may pass the gateway is now classified beside the route registration, defaulting to forbidden
for infrastructure and unclassified routes, which keeps the unauthenticated metrics endpoint off the
public entry point without anyone naming it. `mise run check-gateway-routes` joins `mise run check`.

全文は `git show 92e556a5d23d6cfc329c4005fe9c71f351945f13:<このファイルのパス>` で読む。
