---
status: completed
authors: [tn]
risk: high
created_at: 2026-07-25
depends_on: []
change_kind: feature
initial_context:
  scl:
    Tenancy:
      - interfaces.ResolveTenant
      - models.Tenant
      - models.TenantEndpointStyle
      - interfaces.SetTenantEndpointStyle
    OAuth2:
      - interfaces.GetOpenidConfiguration
      - interfaces.GetOauthAuthorizationServer
    Saml:
      - interfaces.PublishSamlMetadata
  decisions:
    - decisions/ADR-033-tenant-resolution-via-path-prefix.md
    - decisions/ADR-085-tenant-uuid-key-and-realm-identifier.md
    - decisions/ADR-096-tenant-branding-value-and-logo-storage.md
  source:
    - backend/tenancy/domain
    - backend/tenancy/usecases
    - backend/shared/http/support_http
    - backend/shared/http/server_http
  tests:
    - backend/shared/http/support_http
    - backend/shared/http/server_http
    - backend/tenancy/domain
  stop_before_reading:
    - backend/oauth2/token
    - backend/provisioning
affected_spec:
  - { context: Tenancy, kind: model, element: Tenant }
  - { context: Tenancy, kind: model, element: TenantEndpointStyle }
  - { context: Tenancy, kind: interface, element: ResolveTenant }
  - { context: Tenancy, kind: interface, element: SetTenantEndpointStyle }
  - { context: OAuth2, kind: interface, element: GetOpenidConfiguration }
---

# テナントごとに正規ロケーション (path / subdomain) を選べるようにし、host ベースのテナント解決を導入する

## Motivation

現在のテナント解決は path prefix (`/realms/{realm}`) 単一方式である
([[ADR-033-tenant-resolution-via-path-prefix]])。ADR-033 は「subdomain 戦略は将来差し替え
可能な slot として残す」と明記し、`TenantResolver` は host 実装に置換可能だが、
実装は path prefix だけである。

path prefix 方式そのものは正当で、Keycloak が採用し、ワイルドカード DNS も
ワイルドカード証明書も要求しない。捨てる理由はない。問題は「path prefix **しか**
選べない」ことで、テナント固有 origin を前提とする機能が成立しない:

1. **cookie の境界**: 全テナントが同一 origin を共有するため、ブラウザから見た
   session cookie の分離は path scope に依存し、`__Host-` prefix を使えない。
2. **WebAuthn の RP ID**: passkey は origin と RP ID に束縛される。現状
   `WEBAUTHN_RP_ID` はプロセス全体で 1 つで、テナント別 origin を持てない。
3. **ブランド信頼性**: ユーザーは常にベンダのホスト名で資格情報を入力する。
4. **issuer の移行性**: 将来 origin を変える余地が構造として無い。

エンタープライズ IdP は例外なくテナント固有 origin を持つ (Okta の org subdomain、
Entra ID の `<name>.onmicrosoft.com`、OneLogin の `<company>.onelogin.com`、
Auth0 の tenant domain)。本 WI は **テナントごとに正規ロケーションを選べる**状態を作り、
その origin で branding・cookie・WebAuthn RP ID・protocol issuer が一貫するようにする。

顧客所有のカスタムドメイン (`login.example.com`) は同じ resolver の上に載るため、
[[wi-299-tenant-custom-domain]] に分割する。本 WI はその土台を作る。

## Scope

- **decision**:
  - 新規 ADR: 不変条件「1 テナント = 1 正規ロケーション = 1 issuer」、`endpoint_style`
    の 2 値と issuer / cookie / RP ID の導出規則、解決順序と fail-closed 規則、
    `TENANT_BASE_DOMAIN` 未設定時に `subdomain` を選択不能にする決定、
    ADR-033 §2 (未 prefix ルートを default へ) と §3 (`LEGACY_BARE_ISSUER`) の撤回、
    realm を immutable に確定する決定を記録する。
- **scl**:
  - `Tenancy` に `TenantEndpointStyle` enum (`Path` | `Subdomain`) と
    `Tenant.endpoint_style` (既定 `Path`) を追加する。
  - `Tenancy` に `SetTenantEndpointStyle` interface を追加する。`endpoint_style` は
    `Tenant` の属性であり `TenantLifecycle` の遷移ではないため、state machine は追加しない。
  - `Tenancy.interfaces.ResolveTenant` の解決順序を
    「Host (`{realm}.{TENANT_BASE_DOMAIN}`) → path prefix `/realms/{realm}` → 404」に
    差し替え、解決したテナントの `endpoint_style` と到達経路が一致しない場合は 404 と
    明記する。未知 Host / 未知 realm は fail-closed で default に落とさない。
  - `Tenant.realm` を immutable とし、DNS ラベル準拠の制約と予約語を反映する。
  - `scenarios` に「subdomain テナントへのサブドメイン経由ログイン」「未知サブドメインが
    default に到達しない」「path テナントがサブドメインで 404」「subdomain テナントが
    path prefix で 404」「両 style の discovery が取得元 URL と一致する issuer を返す」
    を追加する。
- **go**:
  - `backend/shared/http/support_http/tenant_middleware.go` に host 解決を追加し、
    解決後に `endpoint_style` との一致を確認する。
  - bare route group / `ResolveDefaultTenant` / `LEGACY_BARE_ISSUER` / `bare` 分岐を
    削除する。bare path は default テナントの第 2 ロケーションであり不変条件に反する。
    `GET /` のみ default テナントの正規 root へ 302 リダイレクトする。
  - hostname の正規化 (lowercase / trailing dot 除去 / port 除去) と、realm の
    DNS ラベル検証・予約語拒否を domain 層で行う。**新規作成にのみ適用**する。
  - `tenantIssuer(base, realm)` を `endpoint_style` から正規ロケーションを導出する
    共通ヘルパに置き換え、OIDC discovery・SAML metadata・WS-Fed metadata・
    メール内リンクをそこに合わせる。
  - cookie 名と path を `endpoint_style` から決める。`subdomain` は `__Host-` prefix +
    `Path=/`、`path` は現行のまま `Path=/realms/{realm}`。
  - WebAuthn RP ID / origin は `subdomain` テナントではリクエストホストから解決し、
    `path` テナントでは `WEBAUTHN_RP_ID` を使う。
- **http**:
  - Host ベースで到達したとき、path prefix 無しの `/authorize`・`/.well-known/*` 等が
    正しい tenant に解決される route 構成にする。
  - `Vary: Host` と、Host に依存する応答 (discovery, metadata, branding) の
    キャッシュ境界を明示する。
- **ui**:
  - テナント設定に `endpoint_style` の変更操作を追加し、RP 再設定と passkey 再登録が
    必要になることを警告する。
  - route / API 呼び出しは `frontend/src/router.tsx` の `basepath` と
    `frontend/src/api/core.ts` の `tenantBasePath` が既に両方式に対応しているため変更しない。
- **documentation**:
  - README の Configuration に `TENANT_BASE_DOMAIN`、`endpoint_style`、
    ワイルドカード DNS / TLS 証明書 (ingress 責務)、WebAuthn RP ID の注意、
    style 変更の影響を追記する。

## Out of Scope

- 顧客所有のカスタムドメイン (`login.example.com`) と DNS TXT による所有権検証。
  同じ resolver と不変条件の上に `endpoint_style` の第 3 の値として載るため、
  [[wi-299-tenant-custom-domain]] で扱う。
- TLS 証明書の自動発行・更新 (ACME / cert-manager)。ingress / プラットフォーム責務とし、
  本 WI は「ワイルドカード証明書が用意された前提で正しく解決する」ところまでを扱う。
- テナント別 CDN / WAF 設定。
- メール送信ドメイン (SPF/DKIM/DMARC) のテナント委任。別 WI。
- self-service なテナント作成と、その際の realm 自動採番ポリシー。
  現状テナント作成は `system_admin` 専用のため名前先取り問題は発生しない。
- 既存 realm の再検証と realm rename。realm は immutable に確定する。

## Plan

- **不変条件を先に固定する**。「1 テナント = 1 正規ロケーション = 1 issuer」。
  テナントは自分の正規ロケーションからのみ到達でき、他方の経路では 404 になる。
  この 1 点により、同一テナントが 2 つの origin を持つことから生じる issuer の
  多義性・cookie 名の衝突・discovery の非適合がすべて発生しなくなる。
  `{ISSUER}/realms/acme/.well-known/openid-configuration` が
  `issuer: {ISSUER}/realms/acme` を返す構成は OIDC Discovery 1.0 §4.3 /
  RFC 8414 §3.3 に完全に適合する。非適合になるのは「両方から到達できるのに issuer が
  片方」の場合だけである。
- **path style を残す**。ワイルドカード証明書を用意できない構成と単一ホスト運用を
  一級市民として維持する。`TENANT_BASE_DOMAIN` 未設定なら `subdomain` は選択不能とし、
  既定は `path` なので既存の開発・テスト・demo 環境は無変更で通る。
- **fail-closed を守る**。未登録 Host や未知 realm では「tenant 解決成功」にしない。
  ここを fail-open にすると、任意の Host ヘッダで default tenant に到達できる
  tenant 境界の破りになる。resolver のテストで最初に落とすのはこのケースにする。
- **未リリースなので互換措置は残さない**。ADR-033 §2 の「未 prefix ルートを default へ」
  と §3 の `LEGACY_BARE_ISSUER` は撤回して削除する。移行手順は不要。
- **`endpoint_style` 変更は破壊的操作として扱う**。issuer が変わるため既発行 token の
  `iss` 検証と RP 設定が壊れ、RP ID が変わるため既存 passkey が無効になる。
  専用 API に分離し、UI で影響を明示する。Okta が custom domain の issuer 切り替えに
  ついて同じ影響 (SAML/WS-Fed metadata URL 更新、OIDC アプリ設定更新、passkey 再登録) を
  明記している。
- **realm を DNS ラベルとして扱う**。現行パターン `^[a-z0-9][a-z0-9-]{0,62}$` は
  `acme-` (hyphen 終端) と `xn--foo` (偽 IDNA A-label) を通し、`www` / `api` / `login` を
  予約していない。`subdomain` を選んだ時点で realm がホスト名になるため厳格化するが、
  適用は新規作成のみとし既存 realm は再検証しない。
- 却下: **path style 全廃**。ワイルドカード DNS / 証明書を必須にし、単一ホスト運用と
  ローカル開発のハードルを上げる。Keycloak が path style を採る理由がそのまま残る。
- 却下: **Okta 型の dynamic issuer** (リクエスト origin をそのまま issuer にする)。
  同一テナントが複数 issuer を持ち、片方で発行した token がもう片方で検証できない。

## Tasks

- [x] T001 [SCL] `Tenancy` に `TenantEndpointStyle` / `Tenant.endpoint_style` /
      `SetTenantEndpointStyle` を追加し、`Tenant.realm` を
      immutable + DNS ラベル準拠に更新、`ResolveTenant` の解決順序を差し替え、
      scenario 5 件を追加して `just check-scl` を通す。
- [x] T002 [ADR] 不変条件・解決順序・fail-closed 規則・issuer / cookie / RP ID の導出規則・
      ADR-033 §2/§3 の撤回・却下案を ADR に記録する。
- [x] T003 [Domain] `TenantEndpointStyle` と realm の DNS ラベル検証・予約語を実装する。
      RED: `acme-` / `xn--foo` / `www` が新規作成で落ち、`TENANT_BASE_DOMAIN` 未設定で
      `subdomain` を選べないテストを先に書く (scenario `Tenancy.tenant_endpoint_style`)
      → GREEN。既存 realm の再検証はしない。
- [x] T004 [Persistence] `tenants` テーブルに `endpoint_style` 列を追加し
      (`infra/schema/postgres.sql`)、memory / postgres repository を追随させる。
      RED: 既定値 `path` で読み書きできるテスト → GREEN。
- [x] T005 [Resolver] `tenant_middleware.go` に host 解決を追加し、解決後に
      `endpoint_style` との一致を確認する。bare group / `ResolveDefaultTenant` /
      `LEGACY_BARE_ISSUER` / `bare` 分岐を削除し、`GET /` の 302 を追加する。
      RED: 未登録 Host が default tenant に解決されないテストを**最初に**書く
      (scenario `Tenancy.host_based_tenant_resolution`) → GREEN。
      続けて「path テナントがサブドメインで 404」「subdomain テナントが path で 404」。
- [x] T006 [Protocol] `tenantIssuer` を `endpoint_style` 由来の共通ヘルパに置き換え、
      OIDC discovery・SAML metadata・WS-Fed metadata・メール内リンクを追随させる。
      RED: 両 style で discovery の `issuer` が取得元 URL と一致するテスト
      (scenario `Tenancy.canonical_issuer_per_endpoint_style`) → GREEN。
- [x] T007 [Cookie] cookie 名と path を `endpoint_style` から決めるヘルパを
      `support_http` に追加し、`SetCookie` 呼び出し (oauth2 / csrf / wsfederation / saml) を
      経由させる。RED: subdomain テナントで `__Host-` prefix が付き `Domain` 属性が無く、
      path テナントは現行のままというテスト → GREEN。
- [x] T008 [WebAuthn] `subdomain` テナントは RP ID / origin をリクエストホストから解決し、
      `path` テナントは `WEBAUTHN_RP_ID` を使う。RED: subdomain テナントで RP ID が
      フルホストになるテスト → GREEN。
- [x] T009 [HTTP] Host ベース経路の route 配線 (path prefix 無しの `/authorize` /
      `/.well-known/*`) と `Vary: Host` を追加する。RED: handler テスト → GREEN。
- [x] T010 [UI] テナント設定に `endpoint_style` 変更操作を追加し、RP 再設定と passkey
      再登録が必要である警告を出す。RED: presentation logic の unit test → GREEN。
- [x] T011 [Docs] README の Configuration と `infra/README.md` に `TENANT_BASE_DOMAIN` /
      `endpoint_style` / ワイルドカード DNS / TLS / WebAuthn / style 変更の影響を追記する。
- [x] T012 [Verify] 下記 Verification を緑にする。`just scl-render` で派生物を再生成する。

## Verification

- `just check` / `just check-scl` / `just check-work-items` / `just check-ids`
- `just test-go` / `just test-go-race` / `just verify-go`
- `just verify-ui` / `just test-ui-unit` / `just test-ui-e2e`
- 非退行: `TENANT_BASE_DOMAIN` 未設定 (全テナント `path`) の状態で上記がすべて緑であること。
  bare path を使っていた箇所のみ `/realms/default` に直す。
- 手動 (Chrome。`*.localhost` は loopback に解決されるので `/etc/hosts` 不要):
  1. `TENANT_BASE_DOMAIN=localhost just dev` で
     `http://localhost:5173/realms/default/` が従来どおり動く。
  2. acme を `endpoint_style=subdomain` に変更すると `http://acme.localhost:5173/` が
     acme の branding を出し、同時に `http://localhost:5173/realms/acme/` が 404 になる。
  3. `http://unknown.localhost:5173/authorize` が default tenant に到達しない。
  4. 両 style の `/.well-known/openid-configuration` の `issuer` が、どちらも
     取得元 URL と一致する。
  5. `http://acme.localhost:5173` で passkey を登録・認証でき、RP ID が `acme.localhost`。
  6. acme の session cookie に `__Host-` prefix が付き `Domain` 属性が無く、
     default の cookie は現行のまま `Path=/realms/default`。

## Risk Notes

テナント解決はすべてのリクエストの入口であり、fail-open な実装は tenant 境界の破り
(任意 Host で他テナントへ到達) に直結する。このため resolver の最初のテストを
「未登録 Host が解決されない」ことに固定し、既定を deny にする。

不変条件「1 テナント = 1 正規ロケーション」を破ると、同一テナントが 2 つの issuer を
持ちうる状態になり、discovery が OIDC Discovery 1.0 §4.3 に違反する。解決経路と
`endpoint_style` の一致確認を resolver から外さないこと。

`endpoint_style` の変更は issuer と RP ID を同時に変える不可逆に近い運用変更である。
既発行 token の `iss` 検証、RP 設定、既存 passkey がすべて壊れる。WebAuthn の RP ID 変更は
ユーザーのロックアウトに直結し、[[wi-143-admin-authenticator-reset-and-account-recovery]]
の復旧導線が無い状態で切り替えると復旧不能になるため、UI で影響を明示し ADR に前提として
記録する。

bare route と `LEGACY_BARE_ISSUER` の削除は、それらに依存する既存テスト・demo・
docker-compose を落とす。未リリースのため利用者影響は無いが、変更範囲が広いので
T005 の時点で一括して直す。

Host / realm の解釈は固定長の DNS ラベルと単純な suffix 照合であり、再帰・組合せ爆発・
認可文法の解釈を含まない。fuzz/property test は採用せず、境界値・正規化・fail-closed の
表形式テストで検証した。

## Completion

- **Completed At**: 2026-07-26
- **Summary**: Added per-tenant path/subdomain canonical locations, fail-closed host resolution, canonical protocol URLs, endpoint-style administration, host-only cookies, request-derived WebAuthn RPs, and deployment guidance.
- **Verification Results**:
  - `just scl-render` - passed
  - `just check` - passed
  - `just verify` - passed
- **Evidence**:
  - Domain RED → GREEN: `TestSetEndpointStyle` (scenario `Tenancy.tenant_endpoint_style`) first failed for the missing `SetEndpointStyle` use case, then passed.
  - Resolver RED → GREEN: `TestUnknownSubdomainDoesNotResolveToDefaultTenant` (scenario `Tenancy.host_based_tenant_resolution`) asserts fail-closed host routing and canonical-location mismatch rejection.
  - Protocol / cookie / WebAuthn: `TestSubdomainTenantResolvesFromHost`, `TestTenantCookieScope`, and `TestResolveRPForRequest` cover the issuer, `__Host-` cookie scope, and request-derived RP behavior.
