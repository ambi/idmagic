---
status: completed
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
- [x] T001 [Decision] 新キー名を `user_id` に確定。互換方針: 本システムは未リリースであり
      永続データの後方互換を考慮する必要がないため、読み替え shim / 一括書き換えのいずれも
      実装しない (ユーザー判断、着手時に確認)。dev seed data (`backend/cmd/internal/bootstrap/federation.go`)
      は新キー名に直接書き換えた。
- [x] T002 [SCL] claim-mapping.yaml の `ResolveEffectiveClaims` 説明を `user_id` に更新した。
      oauth2.yaml の "sub" 言及は全て ID Token/UserInfo wire claim を指しており内部キーの
      言及はなかったため変更なし。
- [x] T003 [Domain] `AttrSub` を `AttrUserID`（値 `"user_id"`）に改名し、`coreAttributeKeys` を
      追随させた。`floor_test.go` / `attributes_test.go` は定数参照のため自動追随、
      `policy_test.go` / `saml_handler_test.go` / `wsfed_handler_test.go` /
      `application_handler_test.go` / `userinfo_test.go` / `jwt_signer_idtoken_test.go`
      (Design 記載外だが影響範囲として発見) の直書きリテラルを `user_id` に更新した。
      振る舞い変更ではなく値の張り替えのため red-green サイクルは適用せず、
      `go test ./...` で green を確認した。
- [x] T004 [Compat] 未リリースのため互換シムは実装しない方針に決定 (T001)。
      当初実装した read-time shim (`IssueClaimsWithFloor` 手前での旧キー読み替え) は
      ユーザー指示により削除し、単純なリネームとした。
- [x] T005 [UI] `DEFAULT_NAMEID_SOURCE` / `CORE_ATTRIBUTE_KEYS` を `user_id` に更新し、
      `coreAttributeSubLabel` を `coreAttributeUserIdLabel` に改名した (ラベル文言自体は
      既に protocol-neutral な「ユーザー ID」/「User ID」だったため変更なし)。
      `CreateApplicationDialog.tsx` は現行コードでは `AdminApplicationCreatePage.tsx` に
      統合されており、`DEFAULT_NAMEID_SOURCE` 経由で追随した。
- [x] T006 [Verify] `go test ./...`、`just lint-go`、`just verify-ui`
      (format-check/lint/test:unit/build)、`just check` を実行しグリーンを確認した。

## Verification
- `go test ./...`
- `just lint-go`
- `bun run typecheck`
- `bun run lint`
- `bun run test:unit`
- `bun run build`
- `just check`

## Risk Notes
内部キー名は OIDC/SAML/WS-Federation の claim 発行の中核 (attrs map のルックアップキー) に
使われており、改名を誤ると既存アプリの NameID / sub 解決が壊れ、サインインが失敗し得る
(fail-closed の設計上、キー不一致は「値が見つからない」という形で顕在化し、
`required: true` の場合は発行拒否になる)。テストと手動確認を通じてから適用する。

未リリースであることを着手時に確認し、既存永続データとの互換性リスクは対象外と判断した
(T001)。将来リリース後に同種のキー改名を行う場合は、この Risk Notes の懸念が再び有効になる
ことに注意する。

## Completion

- **Completed At**: 2026-08-01
- **Summary**:
  `AttrSub`（値 `"sub"`）を `AttrUserID`（値 `"user_id"`）へ改名し、`ResolveUserAttributes` /
  `coreAttributeKeys` / SCL (`claim-mapping.yaml`) / frontend (`DEFAULT_NAMEID_SOURCE`,
  `CORE_ATTRIBUTE_KEYS`, `coreAttributeUserIdLabel`) / dev seed data
  (`backend/cmd/internal/bootstrap/federation.go`) / 関連テストの直書きリテラルを一括で
  更新した。OIDC の ID Token/UserInfo wire claim `"sub"` および `reservedClaimTypes["sub"]`
  は Out of Scope のとおり変更していない。
  本システムは未リリースのため、既存永続データ (JSONB) との後方互換は対象外と判断し、
  当初実装した read-time 互換 shim (`IssueClaimsWithFloor` 手前での旧キー読み替え) は
  ユーザー指示により削除し、単純なリネームとした (T001/T004、Design の互換方針からの逸脱)。
  **対応していないこと**: `spec/contexts/oauth2.yaml` の `OAuth2Client.claim_policy` 説明文
  ("NameID 相当の source 属性を sub の source として使う") は、内部キーではなく OIDC の
  ID Token wire claim `sub` を指す記述であると判断し、変更していない。
  test-first からの逸脱 (self-attest): T003 は振る舞い変更ではなく定数値とテストリテラルの
  張り替えであり、新規の RED-GREEN サイクルは適用しなかった。
- **Verification Results**:
  - `go test ./...` - passed。
  - `just lint-go` - passed (0 issues)。
  - `just verify-ui` (format-check/lint/test:unit/build) - passed (523 tests)。
  - `just check` - passed。
