---
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-08-01
depends_on: []
---

# ClaimMapping の内部属性キー "sub" を protocol-neutral な名前に改名する

## Motivation
`backend/claimmapping/usecases/projection.go` の `AttrSub = "sub"` は、`ResolveUserAttributes`
が `attrs["sub"] = user.ID` として詰める内部属性マップのキー名であり、OIDC / SAML / WS-Federation
の claim 発行が共通で参照する ([[ADR-059]] / [[wi-63-federation-metadata-and-claims-mapping]])。
"sub" は本来 OIDC/JWT の ID Token subject claim を指す語彙であり、これを「ユーザー自身の識別子」
を表す protocol-neutral な内部キー名として転用したのが今の設計。[[wi-73-per-application-claim-release-override]]
で NameID / sub source 属性を選択式 UI にした際、SAML/WS-Fed の管理画面にも "sub" という
OIDC 由来の語がそのまま選択肢として出てしまい、レビューで指摘を受けた。

この転用自体は wi-73 が持ち込んだものではなく ADR-059/wi-63 からの既存設計だが、UI 化によって
初めて利用者の目に触れる形で顕在化した。内部キー名を protocol-neutral な名前
(例: `user_id`) に改名し、OIDC の実際の wire claim 名 "sub" (ID Token / UserInfo の出力、および
reserved claim type 集合の一員としての "sub") とは明確に区別する。

## Scope
- `spec/contexts/claim-mapping.yaml`: `interfaces.ResolveEffectiveClaims` の説明にある
  "User の core field (sub / email / ...)" の記述を新しいキー名に更新する。
- `spec/contexts/oauth2.yaml`: `OAuth2Client.claim_policy` 等の説明文中、内部キーとしての
  "sub" 言及があれば更新する (wire claim としての "sub" 言及は変更しない)。

## Out of Scope
- OIDC が実際に発行する ID Token / UserInfo の `sub` wire claim 名。これは RFC 7519 / OIDC Core
  が定める語彙であり変更しない。
- `backend/claimmapping/usecases/floor.go` の `reservedClaimTypes["sub"]`
  (claim_type としての予約語) — これは内部属性キーとは無関係の別の "sub" であり対象外。
- attribute visibility floor のロジック自体の変更 (wi-73 / ADR-151 で確定済み、本 WI は
  キーの命名だけを扱う)。

## Design
- 新しい内部キー名の候補: `user_id` (User 集約の識別子であることが明確、他の属性キー
  (`email` / `name` / `roles` 等) と語彙のトーンが揃う)。決定は着手時に確定する。
- `backend/claimmapping/usecases/projection.go` の `AttrSub` 定数の**値**を変更する
  (定数名 `AttrSub` は Go 内部の識別子でありユーザーに見えないため、名前ごと変えるかは
  着手時に判断してよい。値の変更が本質)。
- 既存データとの互換性: `saml_service_providers.claim_policy` /
  `wsfed_relying_parties.claim_policy` / `oauth2_clients.claim_policy` の JSONB に、
  `source_attribute: "sub"` または `rules[].source_key: "sub"` として既に永続化された値が
  存在し得る (dev seed data 含む)。schema convergence (psqldef) は列定義だけを見るため
  JSONB の中身までは書き換えない。次のいずれかを選ぶ:
  1. 読み込み時に旧キー名 "sub" を新キー名へ読み替える一時的な互換 shim を
     `ResolveUserAttributes` 呼び出し側 (または `IssueClaimsWithFloor` 手前) に置き、
     一定期間後に外す。
  2. 起動時 / 一括スクリプトで該当 JSONB 値を書き換える。
  実データの有無 (dev 環境のみか、何らかの永続環境が既にあるか) を着手時に確認して選ぶ。
- 影響範囲 (2026-08-01 時点の調査結果、着手時に再確認すること):
  - `backend/claimmapping/usecases/projection.go` (`AttrSub` 定数、`ResolveUserAttributes`)
  - `backend/claimmapping/usecases/floor.go` (`coreAttributeKeys` の `AttrSub` エントリ。
    `reservedClaimTypes` の "sub" は触らない)
  - テストの直書きリテラル: `backend/claimmapping/usecases/floor_test.go`、
    `backend/wsfederation/domain/attributes_test.go`、
    `backend/saml/handlers_http/saml_handler_test.go`、
    `backend/claimmapping/domain/policy_test.go`、
    `backend/wsfederation/handlers_http/wsfed_handler_test.go`、
    `backend/application/handlers_http/application_handler_test.go`
  - frontend: `frontend/src/features/admin-applications/AdminApplicationsShared.tsx` の
    `DEFAULT_NAMEID_SOURCE` 定数と、`AdminApplicationEditPage.tsx` /
    `CreateApplicationDialog.tsx` の呼び出し箇所。

## Plan
1. 新キー名を確定する (`user_id` を軸に検討)。
2. `backend/claimmapping` の定数・ロジックを変更し、対象テストを新キー名に合わせて更新する
   (test-first: 新キー名を前提にした RED を先に確認してから実装を合わせる、または既存テストの
   期待値を新キー名へ張り替えた上で green を維持することを確認する)。
3. 既存 JSONB データの互換方針 (Design 参照) を選び実装する。
4. frontend の `DEFAULT_NAMEID_SOURCE` と表示ラベルを更新する。
5. SCL の該当記述を更新し `just check` を通す。

## Tasks
- [ ] T001 [Decision] 新キー名を確定し、既存 JSONB データの互換方針 (読み替え shim か
      一括書き換えか) を決める。
- [ ] T002 [SCL] claim-mapping.yaml / oauth2.yaml の内部キー言及を更新する。
- [ ] T003 [Domain] `AttrSub` 定数値と `coreAttributeKeys` を変更し、
      `floor_test.go` 等の対象テストを新キー名に合わせる。
- [ ] T004 [Compat] 既存永続データ (saml_service_providers / wsfed_relying_parties /
      oauth2_clients の claim_policy JSONB) に対する互換方針を実装する。
- [ ] T005 [UI] `DEFAULT_NAMEID_SOURCE` とラベル表示を更新する。
- [ ] T006 [Verify] `go test ./...`、`bun run typecheck/lint/test:unit/build`、`just check`。

## Verification
- `go test ./...`
- `just lint-go`
- `bun run typecheck`
- `bun run lint`
- `bun run test:unit`
- `bun run build`
- `just check`
- 手動: 既存の (改名前に作成された) WS-Fed/SAML アプリの NameID 解決が改名後も壊れないことを
  確認する。

## Risk Notes
内部キー名は OIDC/SAML/WS-Federation の claim 発行の中核 (attrs map のルックアップキー) に
使われており、改名を誤ると既存アプリの NameID / sub 解決が壊れ、サインインが失敗し得る
(fail-closed の設計上、キー不一致は「値が見つからない」という形で顕在化し、
`required: true` の場合は発行拒否になる)。テストと手動確認を通じてから適用する。
