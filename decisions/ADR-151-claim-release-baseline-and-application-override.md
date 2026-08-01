---
status: accepted
authors: [tn]
created_at: 2026-08-01
---

# ADR-151: claim release の fail-closed floor は属性可視性、上書きはアプリ単位の許可制にする

## コンテキスト

ADR-059 は宣言的・fail-closed な `ClaimMappingPolicy` を確定したが、`IssueClaims` の実装
(`backend/claimmapping/usecases/projection.go`) は `source_key` / NameID の source 属性を
tenant の属性可視性 (`IdManagement.UserAttributeDef.visibility`) と一切照合していない。
`ResolveUserAttributes` は `visibility=Private` の属性も含め User の全属性を無条件に
attrs へ積む。WS-Fed / SAML は RP/SP ごとの `ClaimMappingPolicy` を直接持つ
([[wi-63-federation-metadata-and-claims-mapping]]) が、この policy 自体は tenant の属性可視性
設定と無関係に任意の属性名を `source_key` に書ける。OIDC の ID Token / UserInfo は
`ClaimMapping` を使わず、`visibility=ClaimExposed` + scope + user consent の別経路
(`ClaimsForScopes`) で開示する。

[[wi-73-per-application-claim-release-override]] はここにアプリ単位の release 上書きを足す。
先例の [[ADR-081]] (sign-in policy のテナント既定 + アプリ上書き) にならう案を検討したが、
そのまま流用すると 2 つの問題が出る。(1) sign-in policy はテナント既定を「完全置換」でき、
デフォルトより弱くても警告のみで許可する。claim release で同じ完全置換を許すと、上書きが既定の
属性可視性を回避して未許可属性を漏らせてしまい、ADR-059 の fail-closed 原則に反する。
(2) wi-73 の動機例 (employeeNumber・部署を特定 SP だけに出す) が指す属性は
`org(...)` builtin (`backend/idmanagement/user/domain/attributes.go:42`) で
`visibility=SelfReadable` であり、これらは現行の OIDC 生成用 `ClaimExposed` 縛りには含まれない。
`ClaimExposed` を floor にすると wi-73 の動機例そのものが実現不能になる。

## 決定

`ClaimMapping` の claim 解決 (`IssueClaims` / `ResolveUserAttributes` 相当) に、tenant の
`UserAttributeDef` を必須入力として渡し、属性可視性を fail-closed floor として強制する。
新しい tenant 既定 policy エンティティは追加しない。

1. **floor は `visibility != Private` とする。** `Private` の属性は claim rule の `source_key`
   にも NameID/`sub` の source 属性にも使えず、tenant 既定・アプリ上書きのいずれからも解決できない
   (fail-closed)。`SelfReadable` / `AdminReadable` / `ClaimExposed` はいずれも管理者が明示的に
   claim rule へ写像すれば解放候補になる。これは既存の scope 主導・エンドユーザー同意ベースの
   `ClaimExposed` 限定公開 (OIDC 標準 claim、ADR-064 決定4) とは別の、管理者トラストベースの
   経路であり、AD FS の relying party trust claim rule と同じ性質を持つ。sub / email / name 等
   `User` の core field は `UserAttributeDef` に存在しないため常に解決対象にする (既存挙動を維持)。
   未知の custom attribute key は fail-closed で拒否する。
2. **reserved claim type は固定集合とし、`ClaimMappingRule.claim_type` の対象外にする。**
   `iss`/`aud`/`exp`/`iat`/`nbf`/`jti`/`azp`/`nonce`/`at_hash`/`c_hash`/`acr`/`amr`/`sid` は
   engine が発行する protocol 制御位置であり、mapping rule で上書き・追加できない。
   3 protocol 共通で同じ集合を使う。
3. **アプリ単位の release は Application が持つ既存の per-protocol `ClaimMappingPolicy`
   (WS-Fed `WsFedRelyingParty.ClaimPolicy` / SAML `SamlServiceProvider.ClaimPolicy`、新規追加する
   OIDC client 分) を「アプリの選択」として扱う。** テナント既定という別レイヤーの policy row は
   追加しない。1 Application は最大 1 protocol を持つため (ADR-064)、既存の per-protocol policy が
   実質的にすでにアプリ単位である。floor 適用によって、今まで書けてしまっていた
   `visibility=Private` 属性への参照だけが新たに拒否される。
4. **OIDC ID Token / UserInfo も同じ floor・同じ `IssueClaims` を通す。** scope/consent による
   ClaimExposed 属性の開示可否判定 (既存 `ClaimsForScopes`、OAuth2 が所有) はそのまま残し、
   加えて Application 単位の claim rule (本 WI で OIDC client にも追加する) を通した出力を
   ID Token / UserInfo にマージする。3 protocol が同じ `IssueClaims` 経由になる。
5. **既存 RP/SP の `claim_policy` は無変更で読み込めることを保つ。** floor 追加はふるまいの
   締め付けであり、`Private` 属性を参照していない既存設定は追加検証なしに動き続ける。

## 却下した代替案

- **ADR-081 と同じ「テナント既定 policy row + 完全置換の上書き」を新設する。** 既存の per-RP/SP
  `ClaimMappingPolicy` と二重の policy 表現になり、「どちらが正か」が曖昧になる。floor さえ
  属性可視性で締めれば、既存の per-application policy をそのままアプリ上書きとして使え、
  新しい permanent な tenant-default ストレージは不要。
- **floor を `visibility == ClaimExposed` のみにする。** wi-73 の動機例 (employeeNumber・部署)
  が `SelfReadable` であり実現できなくなる。属性最小化の趣旨は「絞る」ことであって
  「OIDC 標準 claim 相当だけに限定する」ことではない。
- **属性可視性チェックを admin API の入力検証だけに置き、`IssueClaims` 自体には持たせない。**
  API 検証だけでは、検証をバイパスする経路 (直接 DB 編集、将来の一括 import 等) で
  fail-closed が破れる。ADR-059 の「発行時点で必ず fail-closed」という原則に合わせ、
  発行 engine 自体に floor を持たせる。

## 影響

- `ClaimMapping`: `IssueClaims` 系 interface の入力に tenant の `UserAttributeDef` 一覧を追加し、
  `visibility=Private` の attribute source と reserved claim type を拒否する contract を明示する。
  `Tenancy.UserAttributeDef` / `AttrVisibility` への `depends_on` を追加する。
- `Application`: `ApplicationOidcConfig` / `ApplicationOidcConfigUpdateRequest` に
  `rules: ClaimMappingRule[]` と subject identifier (`sub`) の source 属性選択を追加し、
  WS-Fed/SAML と同じ claim release 編集面を OIDC にも揃える。`events.ApplicationClaimMappingUpdated`
  を追加する。
- `OAuth2`: ID Token / UserInfo 発行が `ClaimMapping.IssueClaims` の出力を、既存 scope 主導の
  `ClaimsForScopes` 出力とマージするよう変更する (キー衝突時は標準 claim / 既存 `ClaimsForScopes`
  を優先し、後方互換を保つ)。
- 既存 `saml_service_providers.claim_policy` / `wsfed_relying_parties.claim_policy` はスキーマ・
  データとも無変更。新しい tenant-default テーブルは追加しない。
