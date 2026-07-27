---
depends_on: []
status: completed
authors: ["tn"]
risk: high
created_at: 2026-06-20
---

# Inbound federation / Identity Broker を導入する

## Motivation

実運用 IdP では、自身が認証するだけでなく、外部 OIDC / SAML IdP、
social login、組織別 IdP discovery、JIT provisioning、account linking を扱う。
Keycloak の Identity Brokering、Okta / Google の workforce federation 相当の機能である。

## Scope
- **scl sections**:
  - `spec/contexts/authentication.yaml` の `standards` / `glossary` / `models` /
    `interfaces` / `states` / `authorization` / `objectives` / `scenarios` / `flows`
  - `spec/contexts/identity-management.yaml` の `models` / `interfaces` / `scenarios`
    （password credential を持たない federated User の作成契約）
- **decision**:
  - 新規 ADR: external identity provider、federated identity、account linking、 JIT provisioning の所有境界を定義する。自動 linking の条件と禁止条件を明記する。
- **scl**:
  - ExternalIdentityProvider / FederatedIdentity / AccountLinkingPolicy を追加する。
  - StartFederatedLogin / CompleteFederatedLogin / LinkExternalIdentity / UnlinkExternalIdentity を追加する。
  - IdP discovery interface を追加する。
- **go**:
  - OIDC RP adapter を実装し、外部 OIDC IdP の discovery / JWKS / nonce / state を検証する。
  - SAML SP adapter は SAML IdP WI の後続として追加できる形に port を切る。
  - social login は OIDC provider の preset として扱う。
  - JIT provisioning で User を作る場合、tenant policy と attribute mapping を必須にする。
  - account linking は step-up 済み session でのみ許可する。
- **ui**:
  - login 画面に tenant 設定済み external IdP の選択肢を表示する。
  - admin settings に IdP provider 登録 / mapping / discovery rule を追加する。
  - account portal に linked accounts を表示し、unlink を提供する。
- **documentation**:
  - README に OIDC external IdP 設定例、JIT provisioning の注意を書く。

## Out of Scope
- SCIM provisioning。別 WI。
- LDAP / AD / Kerberos。別 WI または将来判断。
- inbound SAML SP の完全実装は SAML library 選定後に段階化してよい。
- external IdP token の長期保管。

## Design
- **Ownership**: login-time federation は `Authentication` context が所有する。接続設定、
  外部 subject と local User の link、login attempt、JIT / linking policy は
  `backend/authentication/federation` feature slice に置く。`Saml` / `WsFederation` は
  downstream IdP / STS のまま維持し、inbound protocol adapter の domain ownership を
  持たない。
- **Trust model**: connection は tenant-scoped とし、`Draft -> Active <-> Disabled` の
  lifecycle を持つ。管理時に取得・検証した discovery / metadata の last-known-good
  endpoint と鍵だけを login request で利用し、request parameter 由来の URL は取得しない。
  OIDC は issuer 完全一致、HTTPS endpoint、authorization code + PKCE、state、nonce、
  ID Token の signature / audience / time を検証する。SAML は署名、Issuer、
  Destination、Audience、InResponseTo、時刻、response ID replay を検証する。
- **Correlation**: `FederatedIdentity` の一意性は
  `(tenant_id, provider_id, external_subject)` と `(tenant_id, provider_id, local_user_id)`
  の両方で担保する。既存 link を第一に解決し、email 一致による自動 link は
  `VerifiedEmail` policy、upstream の verified claim、tenant 内一意一致をすべて満たす場合
  だけ許可する。曖昧一致、未検証 email、policy 未設定は fail closed とする。
- **JIT provisioning**: provider ごとの明示 `jit_provisioning=true` と、subject /
  username を含む claim mapping、任意の email-domain allowlist が揃う場合だけ local User
  を作る。IdManagement は内部 published interface から password credential を持たない
  active User を作成し、通常 password login は password hash が無い user を常に拒否する。
  external token / assertion は永続化せず、正規化済み最小 claim と link のみを保存する。
- **Explicit linking**: account portal からの link / unlink は直近 5 分以内の step-up 済み
  session に限定する。最後の利用可能な認証手段を unlink して account をロックアウト
  させない。link の付け替えは暗黙に行わない。
- **Secrets and keys**: WI-97 の envelope encryption が未実装であるため、生 client secret
  は本 WI の DB / event / log に保存しない。管理 API は `secret_reference` だけを受け取り、
  runtime の `SecretResolver` が環境・外部 secret store から実値を解決する。API 応答には
  reference も secret も返さない。SAML certificate と OIDC JWKS は公開検証材として
  last-known-good を保持する。
- **Runtime handoff**: callback 検証後は Authentication の `SessionManager` で `federated`
  AMR の LoginSession を作成する。既存 OAuth authorization transaction があれば専用 resume
  endpoint へ渡し、direct admin/account login なら allowlist 済み `return_to` へ遷移する。
- **UI**: login 画面は public discovery API から active provider の表示名と protocol だけを
  取得する。admin settings は CRUD、metadata refresh/test、claim preview を提供し、
  credential 入力は write-only にする。account security は linked account の一覧と unlink
  を提供する。
- **Failure and observability**: state / RelayState は単発消費し、replay、provider disabled、
  issuer/audience mismatch、unsafe metadata、tenant mismatch を同じ fail-closed 境界で拒否する。
  成功・拒否・link / unlink・JIT を監査イベントへ記録し、secret / token / assertion は
  log と event payload に載せない。
- **Fuzzing decision**: OIDC discovery/JWKS/JWT と SAML metadata/assertion は未信頼の構造化
  入力であり認証判定に直結するため fuzz test を採用する。panic-free だけでなく、上限超過、
  重複 ID、署名対象の曖昧化、未知 algorithm を拒否することを検証する。

## Plan
1. `Authentication` SCL の standards / glossary / models / interfaces / states /
   authorization / objectives / scenarios / flows を先に更新し、`just check-scl` を通す。
   JIT 用に `IdentityManagement` SCL の credential-less User 作成契約も更新する。
   ownership と自動 linking / JIT の分岐理由を ADR に記録し、Authentication の
   `ARCHITECTURE.md` / `architecture.yaml` を feature slice の現状へ同期する。
2. Domain の失敗テストを先に追加し、connection lifecycle、claim mapping、
   FederatedIdentity の二重一意性、login attempt の単発消費を実装する。
   layer-local test を green にして Tasks の RED/GREEN 証跡を更新する。
3. Persistence の contract test を先に追加し、memory repository、PostgreSQL schema /
   query / repository、secret reference、tenant-scoped lookup、replay tombstone を実装する。
   `just sqlc-generate` が必要な場合は生成物を同期し、repository test を green にする。
4. OIDC adapter の失敗テストを先に追加し、SSRF-safe discovery/JWKS refresh、
   authorization code + PKCE request、state / nonce、ID Token 検証、claim normalization
   を実装する。malicious discovery/JWKS/JWT fuzz target を追加する。
5. SAML adapter の失敗テストを先に追加し、metadata 検証、AuthnRequest、RelayState /
   InResponseTo、XML signature、Destination / Audience / time / replay、attribute
   normalization を実装する。metadata/assertion fuzz target を追加する。
6. Broker use case の失敗テストを先に追加し、provider routing、既存 link 解決、
   verified-email linking、JIT provisioning、explicit link/unlink、SessionManager handoff、
   audit event を実装する。IdManagement には password credential を持たない federated User
   の internal creation use case を test-first で追加する。
7. HTTP contract test を先に追加し、public provider discovery / start / callback、
   admin CRUD / refresh / test / mapping preview、account linked identity API、
   OAuth authorization resume を配線する。memory / PostgreSQL bootstrap の両方を更新する。
8. UI unit test を先に追加し、login provider selector、admin provider settings、
   account linked accounts を実装する。README（English）へ OIDC upstream 設定例と
   JIT / auto-link の安全上の注意を追加する。
9. `just scl-render` で派生物を同期し、`just verify-go`、`just verify-ui`、
   `just verify`、protocol 別 E2E / fuzz smoke を通す。Tasks と検証証跡を更新し、
   Completion を追記して `work-items/done/` へ移動し、Conventional Commit を作成する。

## Tasks
- [x] T001 [ADR/SCL] inbound broker の context ownership、link policy、OIDC/SAML trust validation、routing interface/event/permission/scenario を決定し再生成した。
- [x] T002 [Domain] RED: federation domain test が未定義の connection / link / lifecycle 型で compile fail → tenant/provider/subject と provider/user の二重一意性、claim mapping、単発 attempt を実装して GREEN。
- [x] T003 [Persistence] RED: memory/PostgreSQL repository contract test が `NewRepositories` と本番 adapter 未実装で fail → tenant-scoped connection/link/attempt/replay と schema、write-only secret reference resolver を実装して GREEN。
- [x] T004 [OIDC Adapter] RED: client test が discovery / authorization / callback API 未実装で compile fail → SSRF-safe discovery/JWKS、PKCE/state/nonce、RS256 ID Token、issuer/audience/time と claim mapping を実装して GREEN。
- [x] T005 [SAML Adapter] RED: protocol test が AuthnRequest / response validator 未実装で compile fail → RelayState/InResponseTo、XML signature、Issuer/Destination/Audience/time/duplicate ID/replay と attribute mapping を実装して GREEN。
- [x] T006 [Broker Usecases] RED: broker test が link/JIT/session handoff use case 未実装で compile fail → existing link、verified-email、credential-less JIT、explicit link/unlink、federated AMR session と audit event を実装して GREEN。
- [x] T007 [Admin HTTP/UI] RED: HTTP contract は federation route が 404、UI unit test は provider selector/settings/linked identity が未表示 → public/admin/account API、OAuth resume、日英 UI、write-only secret、mapping preview を実装して GREEN。
- [x] T008 [Verify] malicious metadata/SSRF、issuer混同、replay、verified-email linking、disabled connection、tenant isolation を protocol adapter・repository・broker・HTTP integration test で検証した。
- [x] T009 [Security] OIDC discovery/JWKS/JWT と SAML assertion の fuzz target を追加し、panic-free と拒否既定を smoke 実行で検証した。

## Verification
- `just test-go`
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just lint-ui`
- `just build-ui`
- `just test-ui-e2e`
- `just test-go-fuzz ./backend/authentication/federation/protocol_oidc 1s`
- `just test-go-fuzz ./backend/authentication/federation/protocol_saml 1s`

## Risk Notes
federated login は account takeover の主要リスクになる。JIT provisioning と
account linking は便利だが危険なので、初期値は保守的にし、tenant admin が
明示設定した場合のみ自動化する。

## Completion

- **Completed At**: 2026-07-27
- **Summary**:
  - Authentication-owned inbound OIDC/SAML broker、tenant-scoped durable connection/link/attempt/replay storage、credential-less JIT provisioning、safe linking/unlinking、OAuth transaction resume を実装した。
  - Login、admin settings、account security の全表示を日本語・英語辞書へ接続し、通常の UI test は英語既定、明示的な `locale: 'ja'` のみ日本語検証とする開発ルールを RA レベルの `DEVELOPMENT.md` に追加した。
  - OIDC/SAML の未信頼入力検証、SSRF 防止、secret reference、step-up、account lockout 防止、replay 防止を adapter/use case/HTTP contract と fuzz test で固定した。
- **Verification Results**:
  - `just scl-render` - passed
  - `just check` - passed
  - `just verify-go` - passed
  - `just verify-ui` - passed (464 unit tests)
  - `just test-ui-e2e` - passed (22 browser scenarios)
  - `just verify` - passed
  - OIDC/SAML protocol fuzz smoke - passed
