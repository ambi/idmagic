---
status: completed
authors: [tn]
risk: high
created_at: 2026-07-24
depends_on: []
---

# Application と単一 protocol 設定を通常の外部キーで正規化する

## Motivation

`Application` は運用者が接続・割当・監査する上位概念であり、OAuth2 client、SAML service
provider、WS-Federation relying party のいずれか 1 つを protocol 設定として持つ。現状は
`applications.bindings` JSONB が protocol natural key (`client_id` / `entity_id` / `wtrealm`) を
opaque に保持するため、参照先の存在、tenant 一致、重複所属を DB が保証できず、逆引きも tenant 内
Application の全件走査になる。

一方、DCR、system client、tenant-level federation 設定など、Application Catalog に載らない
protocol record は有効である。空に近い Application を自動生成せず、protocol record が catalog に
所属する場合だけ nullable `application_id` で明示する。

既存 protocol identity と主キーを維持しながら、1 Application = 0 または 1 protocol、作成後の
protocol 種別不変、Application 削除時の protocol 設定削除を relational constraint と use case
lifecycle へ揃える。

## Scope

- **decision**:
  - ADR-066 / wi-69 の 0..N opaque binding 判断を置き換える ADR を追加する。
  - 1 Application は weblink（protocol なし）または OAuth2 / SAML / WS-Fed のいずれか 1 種類とし、
    protocol 種別・record の付け替えは許可しない。
  - catalog 外 protocol record は `application_id = NULL` で許可する。
- **scl**:
  - `spec/contexts/application.yaml` の glossary / models / constraints / interfaces / scenarios /
    authorization / flows を単一 optional protocol へ変更する。
  - `spec/contexts/oauth2.yaml`、`spec/contexts/saml.yaml`、
    `spec/contexts/ws-federation.yaml` の protocol entity に optional `application_id` を追加し、
    同一 tenant・一意所属を定義する。
  - `ProtocolBinding[]`、attach / detach interface と event を廃止し、Application 作成時に
    protocol を確定する。Application response は optional な単一 `protocol` を返す。
- **schema**:
  - `clients` を `oauth2_clients` に改名し、参照 FK・索引・SQL・sqlc 生成物を追随させる。
  - `applications` に nullable `protocol_type` を追加する。weblink は NULL、service は oauth2、
    federated は oauth2 / saml / wsfed のいずれかを要求する。
  - 3 protocol table に既存主キーを維持した nullable・unique `application_id` を追加する。
    storage discriminator と composite FK で Application / protocol の tenant・type 一致を保証し、
    Application 削除は `ON DELETE CASCADE` で protocol record も削除する。
  - 本機能は未リリースのため旧 Application データの移行互換は設けず、
    `applications.bindings` を宣言的 schema から直接削除する。
- **domain / use cases / adapters**:
  - Application domain を optional 単一 protocol に変更し、kind / protocol_type の不変条件を持たせる。
  - protocol record を catalog 外として準備し、Application insert と relation 設定を Postgres では
    同一 transaction にして、意味のない Application を残さない。
  - linked protocol record の低レベル削除は conflict とし、Application 削除へ誘導する。
    catalog 外 record の低レベル CRUD と gate bypass は維持する。
  - Application lookup / access gate / display name resolver は protocol record の
    `application_id` を索引で解決し、JSON 走査を廃止する。
  - memory adapter、seed/import、管理 UI を同じ単一 protocol 契約へ揃える。
- **derived artifacts**:
  - SCL HTML / JSON Schema / OpenAPI と sqlc 生成物を再生成する。

## Out of Scope

- 1 Application に複数 protocol を持たせること。
- 作成後に protocol 種別または protocol record を付け替えること。
- OAuth2 `client_id`、SAML `entity_id`、WS-Fed `wtrealm` など wire identity の改名。
- `oauth2_clients` / `saml_service_providers` / `wsfed_relying_parties` を
  `*_applications` へ一律改名すること。
- `provisioning_*` の改名。ADR-128 により protocol-neutral Provisioning core である。
- Entra federation profile の別 aggregate 化。

## Plan

1. SCL と ADR で単一 protocol、catalog 外 record、削除・認可 semantics を確定する。
2. Domain / use case / adapter の順に test-first で optional 単一 protocol へ移行する。
3. 宣言的 schema、context-local SQL、sqlc を最終形へ直接更新する。
4. HTTP / UI / seed を単一 protocol response と immutable protocol lifecycle へ揃える。
5. 派生成果物を同期し、全検証後に Completion を記録して done へ移す。

## Tasks

- [x] T001 [SCL] Application / OAuth2 / Saml / WsFederation の models・constraints・interfaces・
  scenarios・authorization・flows を単一 protocol 契約へ更新し、SCL を検証する。
- [x] T002 [Decision] ADR-138 で通常の nullable FK、catalog 外 record、単一 protocol、
  immutable lifecycle、削除 semantics と却下案を記録する。
- [x] T003 [Domain] RED: `TestValidateApplicationSingleProtocolContract` が新型・不変条件未実装で
  compile fail することを先に確認し、optional 単一 protocol domain を実装した。
- [x] T004 [UseCase] RED: `TestDeleteAdminOAuth2ClientRejectsApplicationOwnedClient` が
  `ErrProtocolOwnedByApplication` 未実装で fail することを先に確認し、catalog 外 gate bypass、
  Application cascade、linked protocol 直接削除 conflict を実装した。
- [x] T005 [Persistence] RED: schema constraint test が `oauth2_clients` と複合 FK 未実装で fail
  することを先に確認し、nullable unique FK、tenant/type 不一致拒否、indexed reverse lookup、
  cascade、catalog 外 record の Postgres / memory test と repository を実装した。
- [x] T006 [Schema] 未リリース前提で旧データ互換を設けず、JSON binding 列を削除して
  relational schema を最終形へ直接更新する。
- [x] T007 [HTTP/UI] RED: 単一 protocol response、attach/detach route 不在、linked delete conflict の
  handler/UI test を先に失敗確認し、HTTP / UI / seed を更新する。
- [x] T008 [Derived/Verify] SCL / sqlc 派生物を再生成し、全検証を通す。
- [x] T009 [Complete] Completion と test-first 証跡を記録し、done へ移して commit する。

## Verification

- `just yaml-check-scl`
- `just scl-render`
- `just yaml-check-work-items`
- `just check-ids`
- `just sqlc-generate`
- `just test-go`
- `just verify-go`
- `just verify-ui`
- `just test-ui-e2e`
- `just verify`
- Postgres: existing protocol PK、nullable unique application FK、tenant/type 一致、cascade、
  indexed reverse lookup を確認する。

## Risk Notes

high。公開 Application response、SCL、3 protocol context、認可 gate、宣言的 schema、sqlc、seed、
UI にまたがる破壊的変更である。未リリースのため旧 schema / Application データとの互換は持たず、
最終形へ直接変更する。

外部入力の新規 parser はなく、複雑文法・認証入力解釈も追加しないため fuzz/property test は
採用しない。認可回帰は catalog 外 bypass、catalog 内未割当拒否、割当済み許可、disabled 拒否を
明示テストする。

## Completion

- **Completed At**: 2026-07-24
- **Summary**: `clients` を `oauth2_clients` へ改名し、Application を optional な単一 protocol
  契約へ変更した。3 protocol table は既存 identity を維持した nullable unique
  `application_id` と固定 discriminator を持ち、複合 FK が tenant / protocol type を保証する。
  Application insert と relation 設定は transaction で commit し、削除は protocol row へ cascade
  する。catalog 外 record は NULL のまま有効で、linked record の低レベル削除は 409 とした。
  JSON bindings、attach/detach API、複数形 response/UI を削除し、単一 `protocol` projection へ揃えた。
- **Affected Guarantees State**: 1 Application は weblink の protocol なし、または
  federated/service の単一かつ作成後不変な protocol を持つ。同一 Application への複数 protocol、
  tenant/type 不一致、linked protocol の直接削除を fail-closed で拒否する。catalog 外 protocol は
  Application/assignment gate を要求しない。`provisioning_*` は protocol-neutral のため維持する。
- **Verification Results**:
  - `just yaml-check-scl` / `just scl-render` — passed（23 SCL file、HTML / JSON Schema / OpenAPI 再生成）
  - `just sqlc-generate` — passed
  - `just verify-go` — passed（lint 0 issues、全 Go package race test green）
  - `just verify-ui` — passed（77 test files / 425 tests、typecheck・lint・production build green）
  - `just test-ui-e2e` — passed
  - `just check-ids` / `git diff --check` — passed
  - `just verify` — 本 Work Item の SCL は valid。既存
    `work-items/done/wi-216-dynamic-group-rule-builder-ui.md` の completion metadata 不足だけで
    repository-wide work-item check は失敗（本 WI の変更対象外）。
- **Evidence**:
  - 実行日: 2026-07-24
  - 実行環境: macOS local workspace、実行主体: Codex
  - 対象ソース版: commit 前 working tree
  - 保存先: repository 内の SCL、ADR-138、domain/usecase/adapter tests、schema、生成済み artifacts
  - RED 証跡: domain 新型未定義 compile failure、schema rename/FK assertion failure、
    linked OAuth2 client conflict sentinel 未定義 compile failure を各実装前に確認
