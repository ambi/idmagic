---
status: pending
authors: [tn]
risk: high
created_at: 2026-07-26
depends_on: [wi-285-tenant-endpoint-style-and-host-based-resolution]
change_kind: feature
initial_context:
  scl:
    Tenancy:
      - interfaces.ResolveTenant
      - models.Tenant
      - models.TenantEndpointStyle
      - models.TenantQuota
  decisions:
    - decisions/ADR-033-tenant-resolution-via-path-prefix.md
    - decisions/ADR-134-tenant-resource-quotas.md
  source:
    - backend/tenancy/domain
    - backend/tenancy/usecases
    - backend/tenancy/handlers_http
    - backend/shared/http/support_http
  tests:
    - backend/tenancy/domain
    - backend/tenancy/usecases
    - backend/shared/http/support_http
  stop_before_reading:
    - backend/oauth2/token
    - backend/provisioning
affected_spec:
  - { path: spec/contexts/tenancy/models.tsp, symbol: IdMagic.Contract.TenantEndpointStyle }
  - { path: spec/contexts/tenancy/models.tsp, symbol: IdMagic.Contract.Tenant }
  - { path: spec/contexts/tenancy/models.tsp, symbol: IdMagic.Contract.TenantQuota }
  - { path: spec/contexts/tenancy/requirements.md, requirement: REQ-TENANCY-020 }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Contract.SetTenantEndpointStyle }
---

# 顧客が所有するカスタムドメインを DNS TXT で検証し、テナントの正規ロケーションにする

## Motivation

[[wi-285-tenant-endpoint-style-and-host-based-resolution]] でテナントは
`{realm}.{TENANT_BASE_DOMAIN}` という**ベンダ側の**サブドメインを正規ロケーションに
選べるようになる。しかしそれはベンダのドメインであり、次が満たせない:

1. **ブランド信頼性**: ユーザーは自社ドメイン (`login.example.com`) ではなく
   ベンダドメインで資格情報を入力する。「正規ドメインを確認せよ」というフィッシング
   訓練の運用と矛盾する。
2. **ベンダロックイン**: OIDC `issuer` / SAML entityID がベンダドメインに固定され、
   将来の移行が全 RP の再設定を要する破壊的変更になる。

エンタープライズ IdP では独自ドメインが実質必須である。Okta は Custom Domain を
branding・メール内リンク・OIDC issuer の前提とし、Entra ID は verified custom domain を
テナント identity の基礎に置く。Auth0 は 1 テナント 1 カスタムドメインを TXT 検証で扱う。

wi-285 が確立した不変条件「1 テナント = 1 正規ロケーション = 1 issuer」の上に、
`endpoint_style` の第 3 の値として `custom_domain` を載せる。resolver と issuer 導出、
cookie、WebAuthn RP ID の仕組みは wi-285 のものをそのまま使う。

## Scope

- **decision**:
  - 新規 ADR: `TenantDomain` の所有権検証方式 (DNS TXT)、`Pending` を予約にしない
    設計とその理由、apex ドメインを拒否する理由、quota による登録数制御、
    hostname のグローバル一意制約の範囲を記録する。
- **scl**:
  - `Tenancy` に `TenantDomain` model (hostname / state / verification_token /
    verified_at) と `TenantDomainState` enum (`Pending` | `Verified` | `Active` |
    `Disabled`) を追加する。
  - `TenantEndpointStyle` に `CustomDomain` を追加する。
  - `Tenancy` に `RegisterTenantDomain` / `ListTenantDomains` / `VerifyTenantDomain` /
    `PromoteTenantDomain` / `DeleteTenantDomain` interface を追加する。
  - `TenantQuota` / `TenantUsage` に `custom_domains` を追加する。
  - `states` に TenantDomainRegistered / TenantDomainVerified / TenantDomainActivated /
    TenantDomainDeleted event を追加する。
  - `scenarios` に「verified custom domain でのログイン」「未検証ドメインでのアクセス拒否」
    「custom domain 上の discovery が自ドメイン issuer を返す」「他テナントの verified
    ドメインを奪えない」を追加する。
- **go**:
  - `TenantDomain` の domain / repository / usecase (登録・DNS TXT 検証・昇格・削除) を
    追加する。検証は `_idmagic-challenge.<hostname>` TXT に検証トークンを要求する。
  - hostname の正規化 (lowercase / IDNA punycode / trailing dot 除去)、apex 拒否、
    予約ドメイン (自身の既定ホスト、`TENANT_BASE_DOMAIN` 配下、public suffix そのもの) の
    拒否を domain 層で行う。
  - wi-285 の resolver に custom domain の解決を追加する。ドメイン → tenant の解決は
    高頻度パスなのでプロセス内キャッシュ (TTL + 明示 invalidation) を持つ。
- **ui**:
  - テナント設定にドメイン管理画面 (登録 / TXT レコード表示 / 検証実行 / 昇格 / 削除) を
    追加する。検証待ち・検証失敗の状態を明示する。
- **documentation**:
  - README の Configuration に、独自ドメイン運用時の TLS 証明書 (ingress 責務)、
    DNS 設定、WebAuthn RP ID の注意、issuer 変更の影響を追記する。

## Out of Scope

- TLS 証明書の自動発行・更新 (ACME / cert-manager)。ingress / プラットフォーム責務とし、
  本 WI は「証明書が用意された前提で正しく解決する」ところまでを扱う。
- テナント別 CDN / WAF 設定。
- メール送信ドメイン (SPF/DKIM/DMARC) のテナント委任。別 WI。
- 1 テナントに複数の custom domain を持たせること。不変条件「1 テナント = 1 正規
  ロケーション」を保つため、有効な custom domain は 1 つに限る。

## Plan

- **wi-285 の不変条件を崩さない**。custom domain は `endpoint_style = custom_domain` の
  テナントの唯一の正規ロケーションであり、そのテナントは `{realm}.{TENANT_BASE_DOMAIN}`
  でも `{ISSUER}/realms/{realm}` でも 404 になる。issuer は custom domain の origin。
- **`Pending` を予約にしない**。同一 hostname の `Pending` は複数テナントで並存でき、
  グローバル一意制約は `Verified` / `Active` にのみ効かせる (部分 unique index)。
  `Pending` は TTL (7 日) で失効させる。「登録するだけで他人のドメインを人質に取れる」
  squatting 面を作らないため。所有証明が必須なので、自由入力・グローバル一意で
  名前の先取り問題は構造的に発生しない。
- **apex ドメインを拒否する** (`login.example.com` は可、`example.com` は不可)。
  Okta / OneLogin と同じ制約。apex CNAME の DNS 制約を避け、顧客の Web サイトごと
  IdP に向ける事故を防ぐ。
- **トライアル抑止は既存 quota 機構に乗せる**。ADR-134 の resource に `custom_domains` を
  追加し既定 1 とする。トライアルは override で 0 にすれば登録自体が塞がる
  (Auth0 の「有料プランのみ」に相当)。新しい plan / tier 概念を持ち込まない。
- **DNS 検証はアプリから外向き HTTP を出さない**。DNS resolver 経由の TXT 参照のみを行い、
  任意 URL の fetch は行わない (SSRF 面を作らない)。検証は同期実行 (ボタン押下時) とし、
  失敗理由を構造化して返す。
- **昇格は破壊的操作**。wi-285 の `SetTenantEndpointStyle` と同じ扱いで、issuer 変更と
  passkey 無効化の警告を UI に出す。

## Tasks

- [ ] T001 [Spec] `TenantDomain` / `TenantDomainState` / interface 5 件 / event 4 件 /
      `TenantEndpointStyle.CustomDomain` / quota `custom_domains` / scenario 4 件を追加し
      `just check-scl` を通す。
- [ ] T002 [ADR] 所有権検証方式・`Pending` 非予約・apex 拒否・quota・一意制約の範囲を
      ADR に記録する。
- [ ] T003 [Domain] hostname 正規化 (lowercase / IDNA / trailing dot)、apex 拒否、
      予約ドメイン拒否、検証トークン生成、状態遷移を実装する。
      RED: 不正 hostname と不正遷移が落ちるテストを先に書く
      (scenario `Tenancy.custom_domain_registration`) → GREEN。
- [ ] T004 [Persistence] `tenant_domains` テーブルを `infra/schema/postgres.sql` に
      追加する。`Verified` / `Active` のみを対象とする部分 unique index にする。
      memory / postgres repository を実装する。
      RED: 同一 hostname の二重 verify が失敗し、二重 Pending は許されるテスト → GREEN。
- [ ] T005 [Usecase] 登録 / DNS TXT 検証 / 昇格 / 削除の usecase と audit event 発火、
      quota enforce、`Pending` の TTL 失効 (登録時 lazy purge) を実装する。
      RED: 未検証ドメインを正規ロケーションに昇格できないテスト → GREEN。
- [ ] T006 [Resolver] wi-285 の resolver に custom domain 解決を追加し、キャッシュ
      (TTL + 明示 invalidation) を入れる。RED: 他テナントの verified ドメインで
      到達できないテスト → GREEN。削除直後に到達不能になることもテストする。
- [ ] T007 [UI] ドメイン管理画面を追加する。TXT レコード表示・検証実行・昇格の警告・
      削除確認を含める。RED: presentation logic の unit test → GREEN。
- [ ] T008 [Docs] README と `infra/README.md` に独自ドメイン運用手順 (DNS / TLS /
      WebAuthn / issuer 移行) を追記する。
- [ ] T009 [Verify] 下記 Verification を緑にする。`just spec-render` で派生物を再生成する。

## Verification

- `just check` / `just check-scl` / `just check-work-items` / `just check-ids`
- `just test-go` / `just test-go-race` / `just verify-go`
- `just verify-ui` / `just test-ui-unit`
- 手動: (1) ローカルで `/etc/hosts` にカスタムドメインを割り当て、検証済みにした
  テナントのログイン画面と branding が出ることを確認する。(2) 未検証ドメインで
  到達できないことを確認する。(3) custom domain 上の
  `/.well-known/openid-configuration` の `issuer` が取得元 URL と一致することを
  確認する。(4) custom domain 上で passkey を登録・認証できることを確認する。

## Risk Notes

hostname のグローバル一意制約を `Pending` にまで広げると、所有していないドメインを
先に登録して正当な所有者を締め出せる。部分 unique index の対象を `Verified` / `Active` に
限る設計を崩さないこと。

ホスト名解決はホットパスなのでキャッシュを持つが、削除・無効化が即時反映されないと
無効ドメインで到達し続ける。TTL を短く保ち、変更時に明示 invalidation を行う。

custom domain への昇格は issuer と WebAuthn RP ID を同時に変える。既発行 token の
`iss` 検証、RP 設定、既存 passkey がすべて壊れる。
[[wi-143-admin-authenticator-reset-and-account-recovery]] の復旧導線が無い状態で
切り替えると復旧不能になるため、UI で影響を明示する。

DNS TXT 参照は外部ネットワークに依存する。resolver の応答遅延がリクエストを
ブロックしないよう、検証は同期の管理操作に限定し、テナント解決のホットパスでは
DNS を引かない。
