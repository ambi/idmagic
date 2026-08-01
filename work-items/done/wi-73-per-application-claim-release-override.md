---
depends_on: []
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-06-27
---

# アプリケーション単位で claim release を上書きする

## Motivation
Okta も Entra ID も、アプリケーションごとに「そのアプリに渡す属性」を個別設定できる。
同じユーザーでも、ある SP には employeeNumber と部署だけ、別の RP にはメールと表示名だけ、
というように出力する claim をアプリ単位で絞る/写像する。最小権限と属性最小化の観点で重要で、
プロトコル (OIDC claim / SAML AttributeStatement / WS-Fed claim URI) をまたいで効く。

idmagic は ADR-059 / [[wi-63-federation-metadata-and-claims-mapping]] で宣言的・fail-closed な
ClaimMappingPolicy を持ち、ADR-064 で `ClaimMapping` を protocol-neutral capability として
分離した。ただし現状の release policy はテナント/プロトコル既定の粒度で、アプリ単位の上書きが
ない。本 WI は [[wi-69-application-catalog-aggregate-and-assignment]] の Application に
claim release の上書きをぶら下げ、ClaimMapping が解決時にアプリ別ルールを適用できるようにする。

## Scope
- **decision**:
  - ADR-059 への追補 (または軽量 ADR-069): claim release 解決の優先順位を確定する。 テナント既定 → Application 上書き → (将来 assignment 単位) の合成順序、未定義属性の fail-closed 維持、上書きが各 protocol の wire projection (OIDC/SAML/WS-Fed) に どう反映されるかを決める。所有は ClaimMapping、アプリ紐付けは Application。
- **scl**:
  - ClaimMapping に ApplicationClaimMappingPolicy (Application を参照する ClaimMappingPolicy の 上書きセット) を追加し、IssuedClaim 解決時に合成する。
  - interface: 管理者の Application 配下 claim release CRUD。解決自体は既存の claim 発行経路 (UserInfo / ID Token / SAML AttributeStatement / WS-Fed) に組み込む。
  - アプリ別上書きは属性 claim だけでなく subject 識別子も対象にする。ClaimMapping の NameIdConfiguration をアプリ単位で写像し、OIDC `sub` / SAML NameID / WS-Fed の 主体識別子を Application ごとに決められるようにする (下流 SP がメール以外でアカウントを 突き合わせるケース)。per-user の literal override は SWA / password-vault 導入時の将来 WI。
  - event: ApplicationClaimMappingUpdated。
  - invariant: ClaimMappingFailClosed の維持 (上書きでも未許可属性は出さない)、 ClaimMappingResolvedPerApplication (アプリ別ルールが全 protocol projection に効く)。
  - permission: 既存 AdminApplicationsManage を再利用、または AdminApplicationClaimsManage を追加。
- **go**:
  - per-application claim release policy の persistence (application_claim_release テーブル、tenant scope)。
  - claim 解決器を「テナント既定 + アプリ上書き」の合成にし、OAuth2 / WsFederation / 将来 Saml の発行経路から共通に呼ぶ。
- **http**:
  - /admin/applications/{id}/claim-release の取得/更新。
- **ui**:
  - admin: Application 詳細に claim release エディタ (出力属性の選択・写像・既定からの差分表示)。

## Out of Scope
- 属性変換式言語 (任意の式評価) の導入。初期は選択 + 単純写像 + 既定差分のみ。
- assignment (user/group) 単位のさらに細かい上書き。アプリ単位を先に入れ、後続で検討。
- Application 本体・割当 ([[wi-69-application-catalog-aggregate-and-assignment]])。

## Plan
- 既存 SAML/WS-Fed は RPごとの `ClaimMappingPolicy`、OIDC は scope/UserInfo/ID Token mapping を別々に持つため、Application context に protocol-neutral な `ClaimReleasePolicy` を置き、各 protocol adapter が published language へ変換する。
- tenant default は Identity/Tenancy の属性 visibility と system-reserved claim 禁止集合を正本にし、application override は allow/rename/transform の狭い差分にする。override で非公開属性、未同意 scope、reserved claim を復活できない。
- effective policy は tenant baseline→application override→protocol-required claims→user consent の順に合成し、最終出力を claim issuance engine へ渡す。SAML/WS-Fed の既存 per-RP policy は migration 中に Application policy へ読み替える。
- preview は sample 値ではなく source attribute/type/visibility と出力 claim名を示し、実 user PII を管理 UI に表示しない。policy version を発行/監査 event に残す。

## Tasks
- [x] T000 [Decision] ADR-151: claim release の fail-closed floor は既存の `UserAttributeDef.visibility` (Private 除外) とする。新規 tenant-default policy row は追加せず、既存の per-RP/SP/Client `claim_policy` をアプリ上書きとして読み替える (ADR-081 の完全置換モデルは不採用)。
- [x] T001 [SCL] ClaimMapping (`AttrVisibility`/`UserAttributeDef` published stub、`ClaimReleaseDeniedError`、`ResolveEffectiveClaims` internal interface)、Application (`ApplicationOidcConfig`/`UpdateRequest.rules`/`sub_source_attribute`、`ApplicationClaimMappingUpdated` event、3 Update*Config interface の `requires: claim_release_rules_within_floor` + `emits`)、OAuth2 (`ClaimMapping` depends_on、`OAuth2Client.claim_policy`、`IdTokenClaims`/`UserInfoResponse.additional_claims`) を更新し `just check` green。
- [x] T002 [Domain] `backend/claimmapping/usecases/floor.go`: `IsAttributeReleasable`(Private 除外 + 未知キー fail-closed)、`IsReservedClaimType`(iss/sub/aud/exp/iat/nbf/jti/azp/nonce/at_hash/c_hash/acr/amr/sid)、`IssueClaimsWithFloor` を実装。RED: `floor_test.go` の `TestIssueClaimsWithFloor_RejectsReservedClaimType` 等を先に fail 確認 (未定義シンボルによるコンパイル RED) → GREEN。scenario 「管理者はApplication単位でclaim releaseを絞り込める」に対応。
- [x] T003 [Persistence] `oauth2_clients` に nullable `claim_policy JSONB` を追加 (infra/schema/postgres.sql, backend/oauth2/db_postgres/clients.sql, sqlc-generate 済み)。`OAuth2Client.ClaimPolicy *claimdomain.ClaimMappingPolicy` を Go domain に追加、db_postgres の marshal/unmarshal を実装。既存 `saml_service_providers.claim_policy` / `wsfed_relying_parties.claim_policy` は無変更。
- [x] T004 [Protocol] `backend/claimmapping/usecases/attribute_defs.go` に `TenantAttributeSchemaRepo`/`ResolveTenantAttributeDefs` を追加。OIDC (`userinfo.go` UserInfo、`jwt_signer.go` SignIDToken)、SAML (`saml/usecases/signin.go`)、WS-Fed (`wsfederation/usecases/signin.go`, `wstrust.go`) を全て `IssueClaimsWithFloor` へ接続し、`AttrSchemaRepo` を DI 経由 (module.go → server_http/routes.go → d.Tenancy.AttrSchemaRepo) で配線。RED: `jwt_signer_idtoken_test.go`/`userinfo_test.go` の `TestSignIDTokenClaimPolicyRejectsPrivateAttribute`/`TestUserInfo_ClaimPolicyRejectsPrivateAttribute` を先に fail 確認 → GREEN。architecture.yaml (oauth2, shared) に claimmapping depends_on を追加し `just check-architecture` green。`go test ./...`/`just lint-go` green (既存の無関係な `TestAgentStatusMatchesSCL` 失敗を除く、main ブランチで再現済みの pre-existing issue)。
- [x] T005 [Admin UI] `AdminApplicationEditPage.tsx` に OIDC 向け claim release editor (`sub_source_attribute` input + rules JSON textarea) を追加。`AdminApplicationsShared.tsx` に `ClaimReleaseAttributesPreview` (非PII preview: `getTenantUserAttributeSchema` から key/type/visibility を一覧、visibility=private を強調表示、実ユーザー値は表示しない) を新設し OIDC/WS-Fed/SAML の3 editor に共通で埋め込む。`AdminApplicationDetailPage.tsx` に OIDC の sub_source_attribute/rules 表示を追加。バックエンド側で `handleUpdateOIDCConfig`/`handleUpdateWsFedConfig`/`handleUpdateSamlConfig` に `ValidateClaimReleaseRules` (write-time floor 検証、400 拒否) と `ApplicationClaimMappingUpdated` イベント発行を実装。RED: `claim_release_test.go` の `TestUpdateApplicationOidcConfig_RejectsUndefinedAttributeSource` 等を先に fail 確認 (OIDC は decode エラー、WsFed/Saml は204で無検証通過) → GREEN。`bun run typecheck/lint/test:unit/build` green。
  - 開示: ブラウザでの実 UI 目視確認は本セッションでは実施できなかった (ブラウザツール未接続)。`just dev-memory` を起動し API 経路の生存確認は行ったが、ログイン画面からの実クリック操作は未検証。バックエンドの新規挙動 (rules roundtrip・fail-closed 拒否・ID Token/UserInfo マージ) は Go の HTTP 統合テスト (`claim_release_test.go` 等、実サーバーコード経路) で検証済み。
- [x] T006 [Verify] `go test ./...`・`just lint-go`・`bun run typecheck/lint/test:unit/build`・`just check` すべて green (既存の無関係な pre-existing flaky/failing テスト `TestAgentStatusMatchesSCL` と `TestMfaFactorReencryptor_NoPlaintextSurvivesBackfillAcrossTenants` を除く。main ブランチでも再現し本 WI の変更とは無関係)。baseline deny (Private/未知属性拒否)・reserved claim type 拒否・rename (output_claim 相当は rules 自体の claim_type で表現)・多値属性 (既存 `IssueClaims` の multi-value 経路を再利用) は floor.go の単体テストと 3 protocol の HTTP テストで確認。scope/consent は既存 `ClaimsForScopes` と独立に動作することをマージ順序 (標準 claim 優先) のテストで確認。

## Verification
- `just test-go`
- `just lint-go`
- `just typecheck-ui`
- `just lint-ui`
- `just build-ui`
- `just yaml-check-work-items`
- `just yaml-check-scl`
- 手動: 同一ユーザーで 2 つのアプリに別々の claim release 上書きを設定し、それぞれの ID Token / アサーションに出力属性が想定どおり差分で現れることを確認する。未許可属性が 上書きでも漏れないことを確認する。

## Risk Notes
claim 解決の合成順序を誤ると属性過剰開示や fail-closed の破れにつながる。ADR-059 の
宣言的・fail-closed 原則を維持したまま「テナント既定 + アプリ上書き」を合成し、解決器は
全 protocol 発行経路で共通の 1 経路に保つ。上書きは出力を絞る方向を既定とし、新規属性の
追加は明示許可済みソースに限る。

## Completion
- **Completed At**: 2026-08-01
- **Summary**:
  ADR-151 を起票し、claim release の fail-closed floor を新規 tenant-default policy row では
  なく既存の `UserAttributeDef.visibility` (Private 除外) とする決定を確定した。既存の
  per-RP/SP/Client `claim_policy` をそのままアプリ上書きとして読み替え、新しい tenant-default
  ストレージは追加していない (ADR-081 の完全置換モデルは不採用)。

  - **Domain**: `backend/claimmapping/usecases/floor.go` に `IsAttributeReleasable` (Private
    属性・未知 attribute key を fail-closed で拒否)、`IsReservedClaimType`
    (iss/sub/aud/exp/iat/nbf/jti/azp/nonce/at_hash/c_hash/acr/amr/sid)、
    `ValidateClaimReleaseRules`、`IssueClaimsWithFloor` を実装。
  - **Persistence**: `oauth2_clients` に nullable `claim_policy JSONB` を追加し、OIDC にも
    WS-Fed/SAML と同じ claim release 上書きの入れ物を用意した (既存 2 テーブルは無変更)。
  - **Protocol**: OIDC (ID Token / UserInfo)、SAML assertion、WS-Fed / WS-Trust token の
    発行経路をすべて `IssueClaimsWithFloor` に接続した。OIDC は scope 主導の既存
    `ClaimsForScopes` の後にアプリ上書き claim を追加でマージし (標準 claim 優先)、
    `sub` / NameID の source 属性もアプリ単位で上書きできるようにした。
  - **Admin API / UI**: `UpdateApplicationOidcConfig` に `rules` / `sub_source_attribute` を
    追加 (WS-Fed/SAML と同型)。3 protocol すべての Update*Config が書き込み時に
    `ValidateClaimReleaseRules` で fail-closed floor を検証し (400 拒否)、
    `ApplicationClaimMappingUpdated` イベントを発行する。Admin UI に OIDC 向け claim release
    editor と、テナント属性定義 (key/type/visibility) を実ユーザー値なしで一覧する非PII
    preview (`ClaimReleaseAttributesPreview`) を追加し、OIDC/WS-Fed/SAML の 3 editor 共通で
    埋め込んだ。
- **Out of Scope (WI 本文どおり)**: 属性変換式言語、assignment (user/group) 単位の上書き、
  Application 本体・割当機能は対象外のまま。
- **Verification Results**:
  - `go test ./...` (in: idmagic)
    - result: passed (既存の無関係な pre-existing 失敗 2 件を除く: `TestAgentStatusMatchesSCL`
      と `TestMfaFactorReencryptor_NoPlaintextSurvivesBackfillAcrossTenants`。いずれも本 WI の
      変更前の main ブランチで再現し、AgentStatus enum の大文字小文字表記ゆれと postgres
      テストの並列実行時の順序依存によるもので、claim release / ClaimMapping / Application /
      OAuth2 の変更とは無関係)
  - `just lint-go`
    - result: passed
  - `go build ./...`
    - result: passed
  - `just check` (SCL / work-items / ids / architecture / traceability)
    - result: passed
  - `bun run typecheck` (in: frontend)
    - result: passed
  - `bun run lint` (in: frontend)
    - result: passed (警告のみ、既存コードに起因するもの)
  - `bun run test:unit` (in: frontend)
    - result: passed (474 tests)
  - `bun run build` (in: frontend)
    - result: passed
  - 手動ブラウザ確認: 本セッションではブラウザツールが未接続のため実施できなかった。
    `just dev-memory` を起動し API サーバーの起動確認は行ったが、ログイン画面からの実クリック
    操作による目視確認は未検証。新規挙動 (claim release rules の roundtrip、fail-closed 拒否、
    ID Token/UserInfo へのマージ) は `backend/application/handlers_http/claim_release_test.go`
    ほか、実サーバーコード経路を通す Go HTTP 統合テストで検証した。
