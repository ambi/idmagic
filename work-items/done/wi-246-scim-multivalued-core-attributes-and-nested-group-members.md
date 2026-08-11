---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-18
depends_on: []
change_kind: feature
initial_context:
  scl:
    Sourcing:
      - standards.RFC7643.RFC7643-CORE-RESOURCES
      - standards.RFC7644.RFC7644-PATCH
      - interfaces.GetScimSchemas
      - interfaces.CreateScimUser
      - interfaces.UpdateScimUser
      - interfaces.PatchScimUser
      - interfaces.CreateScimGroup
      - interfaces.UpdateScimGroup
      - interfaces.PatchScimGroup
  source:
    - backend/sourcing/scim/domain/mutation.go
    - backend/sourcing/scim/domain/discovery.go
    - backend/sourcing/scim/usecases/users.go
    - backend/sourcing/scim/usecases/groups.go
  tests:
    - backend/sourcing/scim/domain/mutation_test.go
    - backend/sourcing/scim/domain/discovery_test.go
    - backend/sourcing/scim/handlers_http/resource_contract_test.go
  stop_before_reading:
    - frontend
affected_spec:
  - { path: spec/contexts/sourcing/SPECIFICATION.md, requirement: RFC7643-CORE-RESOURCES }
  - { path: spec/contexts/sourcing/SPECIFICATION.md, requirement: RFC7644-PATCH }
  - { path: spec/contexts/sourcing/SPECIFICATION.md, requirement: REQ-SOURCING-006 }
  - { path: spec/contexts/sourcing/main.tsp, symbol: IdMagic.Contract.GetScimSchemas }
  - { path: spec/contexts/sourcing/main.tsp, symbol: IdMagic.Contract.CreateScimUser }
  - { path: spec/contexts/sourcing/main.tsp, symbol: IdMagic.Contract.UpdateScimUser }
  - { path: spec/contexts/sourcing/main.tsp, symbol: IdMagic.Contract.PatchScimUser }
  - { path: spec/contexts/sourcing/main.tsp, symbol: IdMagic.Contract.CreateScimGroup }
  - { path: spec/contexts/sourcing/main.tsp, symbol: IdMagic.Contract.UpdateScimGroup }
  - { path: spec/contexts/sourcing/main.tsp, symbol: IdMagic.Contract.PatchScimGroup }
---

# SCIM core 属性の対応範囲を明示し、emails を単一 primary email へ決定的に投影する

## Motivation

wi-239 は `RFC7643-CORE-RESOURCES` を `adoption: partial` にし、User は単一
`Email *string`、Group は User member のみという意図的に狭い subset を実装した。一方、現在の
`emails` 処理は配列の先頭要素を無条件に採用するため、後続要素に `primary=true` の email や
`type=work` の email があっても誤った値を `idmanagement.User.Email` に保存する。また、
`phoneNumbers`、`addresses`、`members[].type == "Group"` の非対応範囲を protocol contract と
schema discovery で十分明確にしていないと、SCIM client の設定ミスを silent data loss にしてしまう。

RFC 7643 の core schema は、列挙された全 optional 属性の lossless 永続化や nested group の実装を
service provider に要求していない。実製品も自製品の user model に合わせて制限付きで写像している。

- Microsoft Entra ID の inbound SCIM schema は work address を 1 件、primary work email を 1 件、
  phone number を type ごとに 1 件へ制限している。
  <https://learn.microsoft.com/en-us/entra/identity/app-provisioning/entra-id-scim-api-schema-documentation>
- Keycloak は複数 `emails` の先頭だけを保存し、GET では単一の work/primary email を返す。
  SCIM Group の member type も常に `User` としている。
  <https://www.keycloak.org/docs/latest/server_admin/>
- Okta は中央 user profile と app ごとの app user profile を分け、schema discovery と mapping で
  接続先が実際に扱う属性だけを結ぶ。
  <https://developer.okta.com/docs/concepts/universal-directory/>

したがって本 WI は SCIM resource の lossless shadow store を追加しない。IdMagic の thin core と
attribute bag を維持し、`emails` を単一 email へ明示的・決定的に投影するとともに、未対応の複合属性と
nested group を正直に discovery・validation へ反映する。

## Scope

- CreateScimUser / UpdateScimUser / PatchScimUser の `emails` 配列を次の優先順位で単一
  `idmanagement.User.Email` へ投影する。
  1. `primary == true` の要素。
  2. primary がなければ、`type` が case-insensitive に `work` と一致する最初の要素。
  3. どちらもなければ、wire 配列順の最初の要素。
- `primary == true` が複数、要素が object でない、`value` が空または string でない、`primary` が
  boolean でない場合は `invalidValue` とし、User を変更しない。`type` の省略は許可する。
- email が保存されている User の SCIM 応答は、選択された 1 件だけを
  `{value: ..., type: "work", primary: true}` として返す。email がなければ `emails` を返さない。
  入力配列全体の lossless round-trip は保証しない。
- PUT で `emails` が省略された場合と、PATCH remove で `emails` 全体が削除された場合は、既存の完全置換／
  remove semantics に従い単一 email をクリアする。
- `phoneNumbers` と `addresses` は本 WI では追加しない。Create/PUT body で指定された場合は
  `invalidValue`、PATCH path で指定された場合は `invalidPath` とし、silent に破棄しない。
- Group member は `type` 省略または case-insensitive な `User` のみを受け付ける。
  `members[].type == "Group"` は `invalidValue` とし、Group-in-Group 関係を作らない。
- Group response の各 member は canonical な `type: "User"` を返す。
- GetScimSchemas は実装済み subset だけを公開する。Group `members.$ref` の `referenceTypes` は
  `User` のみにし、`phoneNumbers`、`addresses`、nested Group 対応を広告しない。
- `spec/contexts/sourcing/SPECIFICATION.md` の Design と normative scenarios、および関連 TypeSpec
  operation documentation に、projection 順序・正規化した応答・非対応属性のエラーを先に定める。

## Out of Scope

- 複数 email、phone number、address の全要素を SCIM 専用テーブルまたは shadow document に保存すること。
- `phoneNumbers`、`addresses` の canonical slot を `idmanagement.User.Attributes` に写像すること。
  実際の連携要求と source-of-truth semantics が確定した属性だけを、別 work item で追加する。
- nested group の保存、展開、循環検出、または `EffectiveRoles` への反映。
- enterprise/custom schema extension 属性([[wi-247-scim-enterprise-extension-schema]]で扱う)。
- 複数値属性への複合フィルタ (bracket 構文、例 `emails[type eq "work"]`) の filter/PATCH
  path 対応([[wi-248-scim-complex-value-filter-bracket-syntax]] が本 WI に依存する)。
- `photos`、`entitlements`、`roles`、`x509Certificates`、`ims` 等の希少な属性。

## Design

`idmanagement.User` と persistence schema は変更しない。SCIM は upstream representation をそのまま
保存する context ではなく、外部 source の表現を IdMagic の canonical User/Group へ適応する境界である。
そのため `emails` の多値構造を SCIM context に複製せず、選択規則を domain の pure function として一か所に
置き、POST/PUT/PATCH の全経路から利用する。応答は canonical User.Email から常に単一 work/primary entry を
再構成する。

Group-in-Group は IdMagic の認可・membership model に意味を持たないため、「送信元へ返すためだけの graph」も
保持しない。unsupported input を明示的に拒否することで、外部 IdP の mapping 設定ミスを検知可能にする。

過去に採択された [[ADR-162-scim-multivalued-attributes-stay-in-scim-context]] の
「`idmanagement.User` を変更しない」という境界は維持するが、SCIM 補助テーブルを作る案は採用しない。
[[ADR-163-scim-nested-group-member-is-representational-only]] の representational-only graph も採用しない。
両 ADR は実装前の判断履歴として read-only のまま残し、実装後の durable current design は owning
`SPECIFICATION.md` に記述する。

### Rejected Alternatives

- **SCIM context に multi-valued 補助テーブルを追加する**: lossless SCIM profile storage という製品要件がなく、
  CRUD/PATCH、順序、primary 一意性、削除、監査を canonical User と二重管理するコストに見合わない。
- **`idmanagement.User` に複合 multi-valued 型を追加する**: 認証、通知、管理 UI、CSV、data export まで
  SCIM 固有の cardinality を波及させる。
- **`User.Attributes` の StringArray に押し込む**: element ごとの `type` / `primary` や address sub-attributes を
  表現できず、lossless という目的を満たさない。
- **nested group を認可に使わず保存だけする**: 利用者が意味のある所属関係と誤認しうる graph と、循環検出・
  深さ上限・削除整合性だけが残る。

## Tasks

- [x] T000 [Spec] `spec/contexts/sourcing/SPECIFICATION.md` の Design と normative scenarios に、
      email projection 順序、単一 work/primary 応答、unsupported `phoneNumbers` / `addresses` / Group member
      のエラー、discovery の subset を定め、関連 TypeSpec operation documentation を同期する。
- [x] T001 [Domain] RED: email 選択規則、複数 primary、不正な element/value/type、fallback 順序、
      unsupported core attributes、Group member type の table-driven test を追加する。
      RED は `TestProjectCanonicalEmailPriority` / `TestProjectCanonicalEmailRejectsInvalidValues`
      (`REQ-SOURCING-006`) が未実装 symbol で失敗することを確認した。`TestParseUserWriteRejectsUnsupportedComplexAttributes`、
      `TestParseGroupWriteMemberType` / `TestParseGroupPatchOpsRejectsGroupMemberType` (`REQ-SOURCING-005`) も追加した。
- [x] T002 [Usecase/Adapter] RED: POST/PUT/PATCH/GET の各経路で同じ projection と canonical response が
      適用され、invalid input が atomic に拒否される HTTP contract test を追加する。
      `TestScimCreateUserResourceContract`、`TestScimUpdateUserFullReplace`、
      `TestScimPatchUserResourceContract` (`REQ-SOURCING-006`) と `TestScimGroupResourceContract`
      (`REQ-SOURCING-005`) に contract cases を追加した。
- [x] T003 [Domain] email projection を pure function として実装し、mutation parser と PATCH usecase の
      先頭要素直接参照を置き換える。User/Group の永続化 model と DB schema は変更しない。
- [x] T004 [Discovery] GetScimSchemas が対応 subset だけを返し、Group member の reference type を User に
      限定するよう実装・test を同期する。
- [x] T005 [Verify] `just spec-render`、`just check-spec`、
      `just test-go-package ./backend/sourcing/scim/...`、`just verify-go`、`just check` を実行する。

## Verification

- `just spec-render`
- `just check-spec`
- `just test-go-package ./backend/sourcing/scim/...`
- `just verify-go`
- `just check`
- 手動: 先頭が home、後続が `primary=true` の emails で User を作り、primary の値だけが保存され、GET が
  単一 work/primary entry を返すことを確認する。
- 手動: primary なしで work が後続にある場合は work、work もない場合は先頭が選ばれることを確認する。
- 手動: 複数 primary、`phoneNumbers`、`addresses`、`members[].type == "Group"` が明示的な SCIM error となり、
  User/Group が部分更新されないことを確認する。

## Risk Notes

配列先頭から意味ベースの選択へ変えるため、これまで偶然先頭値に依存していた client では保存される email が
変わりうる。ただし `primary`、次に canonical `work` を優先する方が SCIM metadata と IdMagic の単一 email の
意味に一致する。選択規則と正規化応答を specification と discovery に明記し、全 mutation 経路で同じ pure
function を使うことで差異を防ぐ。

Entra ID や Okta で `phoneNumbers` / `addresses` を明示的に mapping した連携は本 WI 後に error となる。
これは silent data loss より安全であり、実需要が判明した時点で canonical slot と source-of-truth semantics を
仕様化して追加する。nested Group も同様に fail-fast とし、IdMagic が意味を持たない graph を受け入れない。

## Completion

- **Completed At**: 2026-08-11
- **Summary**:
  `REQ-SOURCING-006` を追加し、SCIM emails を全 element 検証後に primary、work、wire order の順で
  単一 `User.Email` へ投影する current design と HTTP contract を specification-first で確定した。
  POST/PUT/PATCH は同じ domain pure function を利用し、複数 primary、不正 value/type/primary を
  `invalidValue` で mutation 前に拒否する。GET/LIST/各 mutation response は email がある場合だけ
  単一 `{type: "work", primary: true}` entry を返す。`phoneNumbers` / `addresses` は POST/PUT で
  `invalidValue`、PATCH で `invalidPath` とし、silent discard を廃止した。Group member は type 省略または
  User のみに限定し、response と schema discovery は User を canonical type / reference type として返す。
  User/Group の canonical model、DB schema、認可計算は変更せず、SCIM 専用 profile table、shadow document、
  Group-in-Group graph は追加していない。multi-valued email/phone/address の lossless storage、phone/address の
  canonical attribute mapping、nested group の保存・展開・認可継承、enterprise/custom schema、bracket
  filter/PATCH path は Out of Scope のままである。
- **Verification Results**:
  - RED: `just test-go-package ./backend/sourcing/scim/domain` は `ProjectCanonicalEmail` 未実装で失敗
  - `just spec-render` - passed
  - `just check-spec` - passed
  - `just check-api-compat` - passed (release baseline は未変更)
  - `just test-go-package ./backend/sourcing/scim/...` - passed
  - `just verify-go` - passed (lint 0 issues、全 Go race tests)
  - `just check` - passed
  - `just verify` - passed (Go/UI/tooling/spec/API compatibility の全並列 check)
  - `git diff --check` - passed
