---
status: pending
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
      - models.TenantBranding
      - interfaces.GetTenantBranding
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
    - backend/tenancy/handlers_http
    - backend/shared/http
  tests:
    - backend/tenancy/handlers_http
    - backend/shared/http
  stop_before_reading:
    - backend/oauth2/token
    - backend/provisioning
affected_spec:
  - { context: Tenancy, kind: interface, element: ResolveTenant }
  - { context: Tenancy, kind: model, element: Tenant }
  - { context: OAuth2, kind: interface, element: GetOpenidConfiguration }
---

# テナント独自ドメイン (vanity domain) とホスト名ベースのテナント解決を導入する

## Motivation

現在のテナント解決は path prefix (`/realms/{realm}`) 単一方式である
([[ADR-033-tenant-resolution-via-path-prefix]])。ADR-033 は「subdomain 戦略は将来差し替え
可能な slot として残す」と明記し、`TenantResolver` interface は host / header 実装に置換可能
だが、実装は `PathPrefixTenantResolver` だけである。
[[wi-89-tenant-login-branding]] は custom domain / vanity URL を「インフラ範囲」として
明示的に Out of Scope にしており、どの WI もこれを扱っていない。

エンタープライズ IdP では独自ドメインが実質必須である:

- **Okta**: Custom Domain (`login.example.com`) が branding・メール内リンク・OIDC issuer の前提。
- **Entra ID**: verified custom domain がテナント identity の基礎で、B2C は custom domain 必須。
- **Keycloak**: hostname SPI で realm ごとの frontend URL / hostname を切り替えられる。

独自ドメインが無いことによる実害は「見た目」ではない:

1. **ブランド信頼性**: ユーザーは IdP ベンダのドメインで資格情報を入力させられる。フィッシング
   訓練で「正規ドメインを確認せよ」と教える運用と矛盾する。
2. **cookie / session の分離**: 全テナントが同一 origin を共有するため、ブラウザから見た
   session cookie の境界がテナント間で共通になる。
3. **WebAuthn の RP ID**: passkey は origin と RP ID に束縛される。現状 `WEBAUTHN_RP_ID` は
   プロセス全体で 1 つであり、テナント別ドメインを持つと passkey が成立しない。RP ID を
   ドメイン単位で解決できる構造が独自ドメインの前提になる。
4. **issuer 安定性**: OIDC `issuer` / SAML entityID がベンダドメインに固定されるため、
   将来のドメイン移行が全 RP の再設定を要する破壊的変更になる。

本 WI は「verified なテナント所有ドメインで到達でき、その origin で branding・cookie・
WebAuthn RP ID・protocol issuer が一貫する」状態を作る。

## Scope

- **decision**:
  - 新規 ADR (テナントドメインと issuer identity): host ベース解決を ADR-033 の resolver slot に
    追加する方式、path prefix 経路の同時サポート (後方互換)、`TenantDomain` の所有権検証方式
    (DNS TXT レコード)、issuer URL の決定規則 (custom domain 昇格時に issuer を切り替えるか、
    切り替えない場合の理由)、WebAuthn RP ID のドメイン別解決、cookie scope の決定規則を記録する。
  - issuer を切り替える場合の移行手順 (既発行 token の verify 互換期間、discovery の
    `issuer` claim 不一致による RP 側障害) を ADR の「影響」に明記する。
- **scl**:
  - `Tenancy` に `TenantDomain` model (hostname / state / verification_token / verified_at /
    is_primary) と `TenantDomainState` enum (Pending / Verified / Active / Disabled) を追加する。
  - `Tenancy.interfaces.ResolveTenant` の解決順序を「Host header の Verified/Active domain →
    path prefix `/realms/{realm}` → default tenant」に拡張する。未知 Host は fail-closed で
    path prefix 経路にのみフォールバックする (任意 Host を default tenant に落とさない)。
  - `Tenancy` に `RegisterTenantDomain` / `ListTenantDomains` / `VerifyTenantDomain` /
    `PromoteTenantDomainPrimary` / `DeleteTenantDomain` interface を追加する。
  - `states` に TenantDomainRegistered / TenantDomainVerified / TenantDomainActivated /
    TenantDomainDeleted event を追加する。
  - `scenarios` に「verified custom domain でのログイン」「未検証ドメインでのアクセス拒否」
    「custom domain 上の discovery が自ドメイン issuer を返す」「他テナントのドメインを
    登録しようとして拒否される」を追加する。
- **go**:
  - `HostTenantResolver` を追加し、既存 `PathPrefixTenantResolver` と合成する
    (chain resolver)。ドメイン → tenant の解決は高頻度パスなのでプロセス内キャッシュ
    (TTL + 明示 invalidation) を持つ。
  - `TenantDomain` の domain / repository / usecase (登録・DNS TXT 検証・primary 昇格・削除) を
    追加する。検証は `_idmagic-challenge.<hostname>` TXT に検証トークンを要求する。
  - hostname の正規化 (lowercase / IDNA punycode / trailing dot 除去) と、予約ドメイン
    (自身の既定ホスト、公開 suffix そのもの) の拒否を domain 層で行う。
  - protocol issuer / metadata URL 生成を「request が到達した tenant domain」から導出する
    共通ヘルパに集約し、OIDC discovery・SAML metadata・WS-Fed metadata・メール内リンクを
    そこに合わせる。
  - WebAuthn RP ID / origin をリクエストの tenant domain から解決できるようにする
    (現状の環境変数は「custom domain を持たないテナントの既定値」に降格する)。
- **http**:
  - Host ベースで到達したとき、path prefix 無しの `/authorize`・`/.well-known/*` 等が
    正しい tenant に解決されることを保証する route 構成にする。
  - `Vary: Host` / cache 制御と、Host に依存する応答 (discovery, metadata, branding) の
    キャッシュ境界を明示する。
- **ui**:
  - テナント設定にドメイン管理画面 (登録 / TXT レコード表示 / 検証実行 / primary 昇格 / 削除) を
    追加する。検証待ち・検証失敗の状態を明示する。
  - フロント側の API base URL 解決が Host ベース経路でも成立することを確認する。
- **documentation**:
  - README の Configuration に、独自ドメイン運用時の TLS 証明書 (ingress 責務)、DNS 設定、
    WebAuthn RP ID の注意、issuer 変更の影響を追記する。

## Out of Scope

- TLS 証明書の自動発行・更新 (ACME / cert-manager)。ingress / プラットフォーム責務とし、
  本 WI は「証明書が用意された前提で正しく解決する」ところまでを扱う。
- テナント別 CDN / WAF 設定。
- ワイルドカードサブドメイン自動割り当て (`{realm}.idmagic.example`) — 独自ドメインが
  成立すれば同じ resolver で表現できるため、必要になった時点で別 WI にする。
- メール送信ドメイン (SPF/DKIM/DMARC) のテナント委任。別 WI。
- 既存 path prefix 経路の廃止。後方互換として残す。

## Plan

- **resolver は合成で入れる**。既存 `PathPrefixTenantResolver` を残し、Host 一致を先に試す
  chain resolver にする。これにより既存 E2E / demo / seed が無変更で通り、回帰面を小さくできる。
- **fail-closed を守る**。未登録 Host や `Pending` 状態のドメインでは「tenant 解決成功」に
  しない。ここを fail-open にすると、任意の Host ヘッダで default tenant に到達できる
  tenant 境界の破りになる。resolver のテストで最初に落とすのはこのケースにする。
- **issuer 切り替えは段階化する**。第 1 段では「custom domain 上でも issuer は primary domain
  基準で 1 つに固定」を既定とし、テナントが明示的に `is_primary` を昇格させたときだけ
  issuer が変わる。issuer が変わると既発行 token の `iss` 検証と RP 設定が壊れるため、
  ADR に移行手順を書き、UI で警告を出す。
- **WebAuthn は既定値降格として扱う**。`WEBAUTHN_RP_ID` を消さず、「custom domain を持つ
  テナントはドメインから RP ID を導出、それ以外は環境変数」とする。passkey は既存ユーザーの
  ロックアウトに直結するため、ドメイン変更時に既存 credential が無効になることを UI で明示する。
- **DNS 検証はアプリから外向き HTTP を出さない**。DNS resolver 経由の TXT 参照のみを行い、
  任意 URL の fetch は行わない (SSRF 面を作らない)。検証は同期実行 (ボタン押下時) とし、
  失敗理由を構造化して返す。
- 未決定: 同一 hostname を複数テナントが登録しようとした場合の扱いは「グローバル一意制約 +
  先着」とする。cross-tenant で他テナントの verified ドメインを奪えないことをテストで固定する。

## Tasks

- [ ] T001 [SCL] `Tenancy` に TenantDomain / TenantDomainState model、ドメイン管理 interface 5 件、
      event 4 件、ResolveTenant の解決順序、scenario 4 件を追加し `just check-scl` を通す。
- [ ] T002 [ADR] テナントドメインと issuer identity の ADR を起票し、解決順序・fail-closed 規則・
      issuer 決定規則・WebAuthn RP ID 解決・cookie scope を記録する。
- [ ] T003 [Domain] hostname 正規化 (lowercase / IDNA / trailing dot)、予約ドメイン拒否、
      検証トークン生成、状態遷移を実装する。RED: 不正 hostname と不正遷移が落ちるテストを先に書く
      (scenario `Tenancy.custom_domain_registration`) → GREEN。
- [ ] T004 [Persistence] `tenant_domains` テーブル (hostname グローバル一意、tenant_id FK) を
      `infra/schema/postgres.sql` に追加し、memory / postgres repository を実装する。
      RED: 同一 hostname の二重登録が失敗するテスト → GREEN。
- [ ] T005 [Usecase] 登録 / DNS TXT 検証 / primary 昇格 / 削除の usecase と audit event 発火を
      実装する。RED: 未検証ドメインを primary に昇格できないテスト → GREEN。
- [ ] T006 [Resolver] `HostTenantResolver` と chain resolver を実装する。RED: 未登録 Host が
      default tenant に解決されない (fail-closed) テストを先に書く → GREEN。キャッシュの
      TTL / invalidation もテストする。
- [ ] T007 [Protocol] issuer / metadata URL / メール内リンク生成を tenant domain 由来の共通
      ヘルパに集約し、OIDC discovery・SAML metadata・WS-Fed metadata を追随させる。
      RED: custom domain 上の discovery が自ドメイン issuer を返すテスト → GREEN。
- [ ] T008 [WebAuthn] RP ID / origin をリクエスト tenant domain から解決する。RED: custom domain
      上で RP ID がそのドメインになるテスト → GREEN。環境変数は既定値として残す。
- [ ] T009 [HTTP] Host ベース経路の route 配線 (path prefix 無しの `/authorize` /
      `/.well-known/*`) と `Vary: Host` を追加する。RED: handler テスト → GREEN。
- [ ] T010 [UI] テナント設定にドメイン管理画面を追加する。TXT レコード表示・検証実行・
      primary 昇格の警告・削除確認を含める。RED: presentation logic の unit test → GREEN。
- [ ] T011 [Docs] README の Configuration と `infra/README.md` に独自ドメイン運用手順
      (DNS / TLS / WebAuthn / issuer 移行) を追記する。
- [ ] T012 [Verify] 下記 Verification を緑にする。`just scl-render` で派生物を再生成する。

## Verification

- `just check` / `just check-scl` / `just check-work-items` / `just check-ids`
- `just test-go` / `just test-go-race` / `just verify-go`
- `just verify-ui` / `just test-ui-unit`
- 手動: (1) ローカルで `/etc/hosts` に 2 つのホスト名を割り当て、それぞれ別テナントの
  ログイン画面と branding が出ることを確認する。(2) 未登録 Host で default tenant に
  到達できないことを確認する。(3) custom domain 上の
  `/.well-known/openid-configuration` の `issuer` が期待値であることを確認する。
  (4) custom domain 上で passkey を登録・認証できることを確認する。

## Risk Notes

テナント解決はすべてのリクエストの入口であり、fail-open な実装は tenant 境界の破り
(任意 Host で他テナントへ到達) に直結する。このため resolver の最初のテストを
「未登録 Host が解決されない」ことに固定し、既定を deny にする。
issuer 変更は既発行 token と RP 設定を壊す不可逆な運用変更なので、第 1 段では既定で
issuer を動かさず、明示的な primary 昇格操作にのみ結び付ける。
WebAuthn の RP ID 変更は既存 passkey を無効化しユーザーをロックアウトさせうる。
[[wi-143-admin-authenticator-reset-and-account-recovery]] の復旧導線が無い状態でドメインを
切り替えると復旧不能になるため、UI で影響を明示し、ADR に前提として記録する。
ホスト名解決はホットパスなのでキャッシュを持つが、削除・無効化が即時反映されないと
無効ドメインで到達し続ける。TTL を短く保ち、変更時に明示 invalidation を行う。
