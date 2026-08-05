---
status: accepted
authors: [tn]
created_at: 2026-07-04
superseded_by: [ADR-144]
---

# ADR-033: テナント解決は `/realms/{tenant_id}` パスプレフィックスで行う

## コンテキスト

ADR-032 で導入したテナント集約をリクエストに紐付ける手段を選ぶ必要がある。
主な選択肢は次の三つ:

1. **サブドメイン** `acme.idp.example.com` — Auth0 のカスタムドメイン等で
   定番。テナント単位の TLS / DNS / CDN 設定が要求される。
2. **パスプレフィックス** `/realms/{id}/...` — Keycloak が標準採用。
   既存 TLS 終端と DNS のままで動く。
3. **ヘッダ** (`X-Tenant-Id`) — 内部 API 向け。ブラウザフローでは
   伝搬が難しく、Redirect URI / `iss` claim との整合性が破綻する。

idmagic はブラウザフロー (`/authorize` `/login` `/consent`) を扱うため、
`iss` claim と Discovery メタデータがテナントごとに一意である必要がある。
3 は機構的に不適。1 vs 2 のトレードオフを評価した。

## 決定

すべてのプロトコルルートを `/realms/{tenant_id}/...` 配下に配置し、`iss` claim と Discovery の
`issuer` はこの prefix から組み立てる。サブドメインは per-tenant TLS/DNS を要求して開発・CI・
K8s ingress のセットアップを重くし、ヘッダはブラウザフロー (303 redirect) を跨いで伝搬できず
`iss` claim との整合が崩れるため、いずれも不採用とした。`TenantResolver` は path / subdomain /
ヘッダのいずれでも差し替え可能な interface とし、本 ADR はその初期実装として path-prefix
resolver を採用する。テナント CRUD (`/admin/tenants/...`) は `system_admin` が default
control-plane tenant に所属する (ADR-032 §6) ため `/realms/default/admin/tenants/...` に置き、
root へ cookie path を広げずに済ませる。

> **ADR-144 による撤回:** 未 prefix ルートを `default` テナントへフォールバックさせる項と、
> `LEGACY_BARE_ISSUER` escape hatch は [[ADR-144-tenant-canonical-location-and-host-based-resolution]]
> で撤回した。未 prefix 経路は `default` テナントの第 2 ロケーションであり、「1 テナント = 1
> 正規ロケーション」の不変条件に反すると判明したため。現在のテナント解決とルーティングの詳細は
> [`backend/tenancy/ARCHITECTURE.md`](../backend/tenancy/ARCHITECTURE.md) を正とする。

## 影響

- `bootstrap/config.ts` の `issuer` は **base URL** に意味が変わる
  (`{base}` を保存し、リクエスト時に `/realms/{id}` を後置)。
  既存 `ISSUER` env が `https://idp.example.com/realms/default` の形に
  なっていた場合、移行ガイドが必要。
- Hono のルートマウントが二重になる: `/realms/:id/...` と `/` (default
  フォールバック)。コード上は単一ルート群を二度マウントする方法と、
  middleware で path-rewrite する方法がある。本実装は前者を採用する
  （middleware で書き換えると downstream route のテスト容易性が落ちる）。
- Redirect URI 検証ロジックは無影響。RP が `iss` を pin している場合は
  ADR の `LEGACY_BARE_ISSUER` で 1 リリース猶予を与える。
- Discovery メタデータの `issuer` フィールドがリクエスト URL に依存して
  動的になる。CDN / 上流キャッシュは tenant prefix を含む URL ごとに
  キャッシュキーを分ける必要がある（Vary ヘッダではなく URL で分離）。

## 却下した代替案

- **サブドメイン解決のみ採用**: 開発・CI・K8s ingress のセットアップが
  重くなり、Phase 4 のスコープを超える。ホストアプリで TLS を per-tenant
  配備する SaaS では subdomain が望ましいが、現状の demo / dev workflow
  との両立を優先する。
- **ヘッダ (`X-Tenant-Id`) 解決**: ブラウザフロー (303 redirect) で
  伝搬できず、`iss` claim と URL の整合が崩れる。OIDC RP 実装が
  サーバから受け取った `iss` URL を JWKS / discovery 取得に再利用する
  ため、URL に tenant が現れない設計は事実上採れない。
- **URL prefix + iss を bare のまま固定**: iss = URL prefix の不一致は
  RP 側の検証ライブラリ (ex: `oidc-client-ts`) がデフォルトで弾く。
  単一 IdP インスタンスを RP から見て複数 IdP に見せる目的に反する。
