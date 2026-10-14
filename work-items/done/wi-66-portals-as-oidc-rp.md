---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 管理コンソールとアカウントポータルを自分自身の IdP の OIDC RP にする

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

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
