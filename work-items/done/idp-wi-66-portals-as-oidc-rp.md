---
id: idp-wi-66-portals-as-oidc-rp
title: "管理コンソールとアカウントポータルを自分自身の IdP の OIDC RP にする"
created_at: 2026-06-27
authors: ["tn"]
status: completed
risk: high
---

# Motivation
現状、管理コンソール (`/admin/*`) とアカウントポータル (`/account/*`) の React SPA は
OIDC ではなく IdP 自身のファーストパーティ・ブラウザセッション (HttpOnly セッション
cookie、`POST /api/auth/login` で発行) でログインしている。`/api/admin/*` と
`/api/account/*` はサーバ側でセッション cookie から `sub` を解決し、ロール / actor
境界 (ADR-031 / ADR-038 / ADR-042) で認可している。つまり両 SPA は「RP」ではなく IdP に
内蔵された first-party アプリで、IdP が発行するトークンを一切消費していない。

本 WI は両 SPA を **自分自身の IdP の OIDC RP** に作り変える。authorization_code +
PKCE で IdP の `/authorize` にリダイレクトし、`/callback` で code を `/token` に交換して
access token (RFC 9068 JWT, ADR-012) を取得し、`Authorization: Bearer` で
`/api/admin/*` / `/api/account/*` を呼ぶ。API 側はセッションではなく Bearer トークンを
検証する resource server になる。

これは Keycloak が admin console を master realm の OIDC クライアント
(`security-admin-console`) として保護しているのと同じ「自己ドッグフーディング」構成で、
デモ IdP として「IdP を本来の使い方 (OIDC RP) で自分自身に対して使う」ことを示す価値が
ある。RP 側の authorization_code / PKCE / token 取得 / silent renew / RP-initiated
logout という、これまで IdP 側からは見えていなかった経路を実装で示せる。

ユーザ判断 (本 WI の起点):
  - トークンの持ち方: **ピュア SPA RP** を選択。トークンはブラウザが保持し、
    `/api/{admin,account}/*` を Bearer (RFC 9068) のリソースサーバ化する。
    本リポジトリの `ui/ARCHITECTURE.md` が掲げる no-token-in-JS 方針は本 WI で更新する
    (ADR-061)。
  - 対象範囲: **管理 + アカウント両方**の SPA を OIDC RP 化し、ログイン経路を統一する。

# Scope
- **decision**:
  - 新規 ADR-061: 「ファーストパーティ portal を自分自身の IdP の OIDC RP にする」。 pure SPA RP を採用しトークンをブラウザ保持にする判断、`/api/{admin,account}/*` の Bearer resource server 化、ブラウザに格納するトークンの種別と寿命 (短命 access token + refresh rotation, ADR-004/012)、ブートストラップ・ロックアウト緩和 (循環依存対策として first-party セッションログインを緊急経路として残す) を確定する。 `ui/ARCHITECTURE.md` の no-token-in-JS 方針の更新もここで明記する。
- **scl**:
  - ファーストパーティ portal を表す OIDC クライアントと scope (`idmagic.admin` / `idmagic.account`) を SCL に追記する。両 client は public + PKCE 必須 + first-party (consent skip) とする。Bearer で保護される admin / account interface の前提 (caller が access token を提示する) を反映する。
- **go**:
  - `core.Deps.ResolveAuthentication` に Bearer 経路を追加する。`Authorization: Bearer` があれば `TokenIntrospector.IntrospectAccessToken` (既存) で JWT access token を 検証し、active かつ要求 scope を含むなら `AuthenticationContext` を組み立てる。 セッション cookie 経路は緊急ログインのため当面併存させる (dual-mode)。
  - first-party SPA クライアント `idmagic-admin-console` / `idmagic-account-portal` を seed する (public, authorization_code + PKCE, redirect_uri = `/callback`, scope = `openid profile idmagic.admin` / `openid profile idmagic.account`)。
  - first-party クライアントは consent をスキップする (Client に first_party フラグ、 または seed 済み granted consent)。fail-open しないよう scope は固定。
- **http**:
  - `/api/admin/*` と `/api/account/*` の認可を Bearer 必須に切り替える (緊急セッション 経路は残す)。401 応答は `WWW-Authenticate: Bearer` を返す。
  - discovery の `scopes_supported` に `idmagic.admin` / `idmagic.account` を広告する。
- **ui**:
  - SPA に最小の OIDC RP クライアント (PKCE verifier/challenge/state/nonce 生成、 `/authorize` への redirect、`/callback` での code→token 交換、token を sessionStorage に保持、`request()` への Bearer 付与、prompt=none / refresh による silent renew) を実装する。
  - `loadPageData` の admin / account 分岐を、未トークン時は `/login` ではなく `/authorize` へ誘導する形に変える。`/callback` を実トークン交換に作り替える (現状 CallbackPage は静的デモ表示)。
  - RP-initiated logout (`/end_session` + token revoke) を sign-out 導線に接続する。
- **documentation**:
  - README と `ui/ARCHITECTURE.md` に「両 portal は自分自身の IdP の OIDC RP」構成、 トークン保持方針の変更、緊急セッションログイン経路を記す。

# Out of Scope
- BFF (サーバ側 token 保持) 方式。ユーザ判断で pure SPA RP を採用したため扱わない。
- 外部 (サードパーティ) RP の新規追加。既存 demo-client の扱いは変えない。
- DPoP による SPA トークンの sender-constraint 化。必要なら追加 WI とする。

# Verification
- `go build ./...` (in: idmagic)
- `go test ./...` (in: idmagic)
  - reason: Bearer 解決 (active/scope/テナント不一致) と既存セッション経路の併存の境界。
- `golangci-lint run ./...` (in: idmagic)
- `bun --cwd idmagic/ui typecheck`
- `bun --cwd idmagic/ui build`
- 手動: 管理者で `/admin` を開き `/authorize`→ログイン→`/callback`→token 取得→ `/api/admin/*` が Bearer で通ること、トークン無効化後に 401 となること、 緊急セッションログインで復旧できることを確認する。

# Risk Notes
IdP が自分自身を認証する循環依存により、OIDC クライアント設定 / 署名鍵の破壊が
管理コンソールのロックアウトに直結する。緊急 first-party セッションログイン経路を
必ず残す。pure SPA はトークンをブラウザに置くため XSS によるトークン窃取が直接の
資格情報漏洩になる。短命 access token・rotation refresh・CSP / no-store の徹底で
リスクを抑える。移行中は Bearer とセッションの dual-mode となるため、両経路の認可が
同じロール / actor 境界を通ることをテストで担保する。

# Completion
- **Completed At**: 2026-06-27
- **Summary**:
  管理コンソール (`/admin/*`) とアカウントポータル (`/account/*`) を、自分自身の IdP の
  OIDC RP (authorization_code + PKCE、pure SPA) として認証する構成へ移行した。
  `/api/{admin,account}/*` は RFC 9068 アクセストークンを検証する resource server と
  なり、Bearer の `sub` を既存のロール / actor 境界 (ADR-031 / 038 / 042) に通す。
  ファーストパーティ public クライアント `idmagic-admin-console` / `idmagic-account-portal` を
  seed し、`first_party` フラグで consent をスキップする。SPA には最小の OIDC RP
  クライアント (`api/oidc`) を実装し、`/authorize` リダイレクト・`/callback` での
  code→token 交換・refresh による silent renew・RP-initiated logout を行う。OIDC 設定
  破壊時のロックアウトを避けるため、first-party セッションログインを緊急経路として残し
  `ResolveAuthentication` を Bearer/セッション dual-mode とした。resource server は
  portal scope (`idmagic.admin` / `idmagic.account`) を要求し、cross-portal 利用を fail-closed で
  拒否する。
- **Verification Results**:
  - `go build ./...` (in: idmagic)
    - result: ok
  - `go test ./...` (in: idmagic)
    - result: ok
  - `golangci-lint run ./...` (in: idmagic)
    - result: 0 issues
  - `bun --cwd idmagic/ui run lint && bun --cwd idmagic/ui typecheck && bun --cwd idmagic/ui build`
    - result: ok
  - 手動 (ローカル実機): idmagic-admin-console で authorize(consent skip)→login→token (scope idmagic.admin offline_access, refresh_token あり)→/api/admin/users 200、Bearer 無し 401。refresh_token grant で silent renew→再取得 token で 200。idmagic.account token は /api/admin/* 401・/api/account/profile 200 (cross-portal 拒否)。
- **Affected Guarantees State**:
  - admin/account authorization: Bearer (active + portal scope) を要求し、role/actor 境界に通す。scope 不足・cross-portal は 401 (live + unit test)
  - bootstrap availability: first-party セッションログインを緊急経路として残し dual-mode
  - token exposure: access token 600s + offline_access による rotation refresh。no-store 維持
  - audit: OIDC ログイン (code / token 発行) は既存監査で記録
