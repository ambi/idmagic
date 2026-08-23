---
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-08-08
depends_on: []
---

# SCL の error model へ HTTP status と envelope 形式を導入し RFC 9110 準拠の 400/422 に再分類する

## Motivation

`backend/shared/http/support_http/response.go` の `WriteBrowserError` を
RFC 9457 Problem Details に移行する決定 (ADR-154,
`decisions/ADR-154-rfc-9457-problem-details-for-http-errors.md`) をしたが、
SCL の `kind: error` model には HTTP status code や response envelope 形式を
表す field が無く、`errors: [...]` で名前を挙げるのみである。RA は SCL-first
(spec を先に変更してから実装する) を要求するため、Problem Details 移行の
実装に着手する前に、まず SCL 言語自体に status / envelope を表現する語彙を
追加し、その語彙で全 error model を実際に分類しておく必要がある。

現状 422 (`Unprocessable Entity`) はリポジトリ全体で 1 箇所
(`quota_exceeded`) しか使われておらず、「構文は正しいが内容で処理できない」
系のビジネスルール違反 (`invalid_role`, `password_reuse`,
`ssf_stream_event_types_required` など) が軒並み 400 に丸められている。
RFC 9110 に沿った意味論への再分類も、この SCL 変更のタイミングで行う
(400/422 は per-error の `status` field 1 個の値として決まるため、
envelope 変更と切り離すと後から再度 75+ 個を洗い直すことになる)。

## Scope

- `SPECIFICATION_CORE_LANGUAGE.md` §3.2 (`Model` field 表) — `kind: error`
  model に `status` field (Integer, MUST) を追加する記述。
- `SPECIFICATION_CORE_LANGUAGE.md` §3.3.1 (`Binding` kind 別必須/任意 field
  表) — `kind: http` に任意 field `error_format`
  (`problem_details` (既定) | `oauth2` | `scim` | `set_delivery`) を
  追加する記述。
- `tools/check/schemas/scl-v3.schema.json` — `Model` (`additionalProperties:
  false`) と `Binding` の `http` 分岐に対応する field を追加。
- `docs/contexts/*.yaml` の全 19 ファイル・75+ 個の `kind: error` model へ
  `status` を付与。
- `spec/contexts/oauth2.yaml`、`spec/contexts/sourcing.yaml` (SCIM)、
  RFC 7591 Dynamic Client Registration を持つ binding への `error_format`
  設定。
- 共通 `InvalidRequestError` (400) に紛れ込んでいた Go 側のビジネスルール
  違反コード (`invalid_role`、`password_reuse` 等) を、業務ルールごとの
  専用 `kind: error` model (`status: 422` または状態競合系は `409`) として
  個別に定義し、該当 interface の `errors: [...]` へ配線する
  (`identity-management.yaml`、`authentication.yaml`、`tenancy.yaml`、
  `sharedsignals.yaml`、`workloadidentity.yaml`、`oauth2.yaml` の 6 ファイル、
  37 個の新規 error model)。
- `backend/shared/spec/loader.go` の `Model` struct (Go 側で SCL YAML を
  strict decode する独自の型定義。実装中に判明: SCL の語彙は
  TS 側の JSON Schema (`tools/check/schemas/scl-v3.schema.json`) と
  TS 型 (`tools/scl-to-html/src/types.ts`) だけでなく、この Go struct にも
  三重に反映する必要がある) への `Status` field 追加。

## Out of Scope

- `backend/shared/http/support_http` の Problem Details 実装、および
  各ハンドラの `WriteBrowserError` 呼び出し箇所 (約 390 箇所) の移行。
  別 work item「HTTP エラーレスポンスを RFC 9457 Problem Details へ
  移行する」で行う (本 work item に依存させる)。
- 新しい公開 `type` URI の実際のドキュメントページ発行。
- `backend/application/handlers_http/admin_category_handler.go` の
  カテゴリ名必須・存在しないカテゴリ割当の 2 条件。Go 側がそもそも
  `"invalid_request"` という汎用コードしか使っておらず、他の 37 件と異なり
  区別可能な文字列コードが存在しないため、SCL 側だけで新しい error model を
  作っても Go の実際の挙動と乖離する。Go に新しいコードを追加する変更と
  セットでなければ意味がなく、それは wi-326 (Go 実装) 側の scope とする。

## Design

- `status` は `kind: error` model の必須 field とし、その error を返す
  すべての interface に共通のカノニカルな HTTP status を 1 つ持たせる
  (同じ error が複数 interface から返されても意味は変わらないため、
  interface 側ではなく model 側に持たせる)。
- `error_format` は `Binding` (`kind: http`) の任意 field とし、既定は
  `problem_details`。OAuth2 / SCIM / DCR のように envelope 形式が
  プロトコル仕様で決まっている endpoint だけ明示的に上書きする。
  実装の結果、`error_format: oauth2` を設定したのは
  `spec/contexts/oauth2.yaml` の `writeOAuthError` を実際に使う 12
  interface (`RegisterClient` (RFC 7591 DCR も同じ形式を再利用)、
  `Authorize`、`EndSession`、`PushAuthorizationRequest`、`Token`、
  `Introspect`、`Revoke`、`UserInfo`、`PostUserInfo`、
  `DeviceAuthorization`、`GetOpenidConfiguration`、
  `GetOauthAuthorizationServer`、`GetProtectedResourceMetadata`)。
  `ResumeFederatedAuthorization`/`CheckSessionIframe` 等は Go 実装が
  実際には generic `WriteBrowserError` を使っていたため対象外とした
  (RFC 6750 は Bearer エラーの body 形式まで規定しないため、
  `UserInfo` 以外の Bearer 保護 endpoint は Problem Details 化してよい
  という設計判断も併せて確認できた)。`error_format: scim` は
  `spec/contexts/sourcing.yaml` の SCIM 15 binding すべてに設定した。
  `error_format: set_delivery` は `spec/contexts/sharedsignals.yaml` の
  inbound SET receiver 1 binding に設定した。
- SharedSignals inbound SET receiver (`POST /ssf/streams/:id/events`) は
  調査の結果、RFC 8935 §2.3 がエラー応答形式を明示的に規定していることを
  確認した: SET の解析・検証・認証エラー時は HTTP 400、
  `Content-Type: application/json`、body は IANA SET Error Code レジストリの
  値を持つ `err` と人間可読な `description` の 2 フィールドを **MUST** で
  含む固定形式 (Problem Details とは非互換)。このため `error_format` に
  `set_delivery` を追加し、このエンドポイントの binding に設定する。
  同じ SharedSignals でも stream 管理 (admin CRUD) API にはこの制約はなく
  `problem_details` のままとする。
- 実装の結果判明したこと: SCL の `kind: error` model は当初想定より粗い
  粒度で、多くのコンテキストが `InvalidRequestError`/`AccessDeniedError`/
  `QuotaExceededError` 等の共通 model を「published language stub」として
  再公開する形になっている (元コンテキストで定義した意味を継承する)。
  Design 節に書いた `admin_user_handler.go` の `invalid_role` 等の Go 側
  詳細コードは、当初 SCL 上は個別 model 化されておらず汎用
  `InvalidRequestError` (400) に含まれていた。事前調査で洗い出した候補
  (14 ファイル・約 40 箇所) について実際の Go handler と SCL interface の
  対応を突き合わせ、業務ルールごとに専用 model (`status: 422`、状態競合系は
  `409`) を新設し、該当 interface の `errors: [...]` へ配線した (下記の
  「新設した granular error model」参照)。唯一
  `admin_category_handler.go` の 2 条件のみ、Go 側に区別可能な文字列コード
  自体が存在しなかったため対象外とした (Out of Scope 参照)。
- 新設した granular error model (37 個、6 ファイル):
  `identity-management.yaml` — `InvalidRoleError`(422),
  `SelfDeleteForbiddenError`(422), `SelfDisableForbiddenError`(422),
  `UserNotPendingDeletionError`(409, 状態競合),
  `RestoreGracePeriodExpiredError`(409, 状態競合),
  `InvalidUserAttributeError`(422), `InvalidRequiredActionError`(422),
  `GroupNameRequiredError`(422), `InvalidDynamicGroupRuleError`(422),
  `AgentNameRequiredError`(422), `AgentOwnerRequiredError`(422),
  `AgentOwnerNotFoundError`(422, 参照整合性), `InvalidEmailError`(422),
  `EmailUnchangedError`(422), `InvalidExportColumnsError`(422),
  `InvalidExportTargetError`(422), `InvalidExportFilterError`(422)。
  `authentication.yaml` — `PasswordReuseError`(422),
  `MfaEnrollmentNotAllowedError`(422),
  `AuthenticatorResetNotAllowedError`(422)。
  `tenancy.yaml` — `PolicyOverrideWeakerError`(422),
  `InvalidUserAttributeSchemaError`(422)。
  `sharedsignals.yaml` — `SsfStreamEventTypesRequiredError`(422),
  `SsfStreamEventTypeInvalidError`(422),
  `SsfTransmitterDeliveryEndpointInvalidError`(422),
  `SsfTransmitterAudienceRequiredError`(422),
  `SsfReceiverTrustedIssuerInvalidError`(422),
  `SsfReceiverJwksRequiredError`(422),
  `SsfReceiverAcceptedAudiencesRequiredError`(422)。
  `workloadidentity.yaml` — `WorkloadTrustBundleJwksRequiredError`(422),
  `WorkloadTrustBundleNameRequiredError`(422),
  `WorkloadTrustBundleIssuerRequiredError`(422),
  `WorkloadTrustBundleAudiencesRequiredError`(422),
  `WorkloadTrustBundleInvalidTtlError`(422),
  `AgentWorkloadBindingAgentNotFoundError`(422, 参照整合性),
  `AgentWorkloadBindingPatternRequiredError`(422)。
  `oauth2.yaml` — `InvalidAuthorizationDetailTypeError`(422)。
  「必須値が空」系はすべて 422 とした (JSON としては解釈できており、
  field の内容が業務ルールを満たさない、という RFC 9110 の 422 の定義に
  合致するため)。「対象の現在の状態がこの操作を許さない」系
  (`UserNotPendingDeletionError`、`RestoreGracePeriodExpiredError`) だけは
  既存の `WorkflowRevisionConflictError` 等と同じ 409 (状態競合) に分類した。
- 最終的な status 割り当ては、各 model の description に既に記された
  RFC 節番号・引用 (例: `ApplicationOwnedProtocolError` は
  「HTTP 409 とし」と明記済み) と、対応する既存 Go 実装
  (`backend/oauth2/handlers_http/errors.go` の `writeOAuthError`、
  `backend/shared/http/support_http/auth.go` の
  `WriteAccessTokenError`/`WriteAdminAccessError`、
  `backend/shared/http/support_http/error_handler.go` の
  quota_exceeded 422、`backend/saml/handlers_http/admin_idp_profile_handler.go`
  の `profile_in_use` 409 等) を優先して決定した。主なもの:
  `InvalidRequestError`=400、`AccessDeniedError`=403、
  `QuotaExceededError`/`ClientSecretLimitExceededError`/
  `ClaimReleaseDeniedError`/`ProvisioningSubjectNotInScopeError`/
  `SeedRejectedError`=422、`WorkflowRevisionConflictError`/
  `DataKeyStillReferencedError`/`JobLeaseLostError`/
  `ApplicationOwnedProtocolError`/`ProvisioningConnectionAlreadyExistsError`/
  `ProvisioningDeliveryNotRetryableError`/`IdPProfileInUseError`/
  `DefaultIdPProfileError`/`SeedConflictError`=409、
  `ProvisioningConnectionNotFoundError`/`ProvisioningDeliveryNotFoundError`=404、
  `InvalidClientError`/`InvalidTokenError`/`InvalidDpopProofError`/
  `WorkloadAttestationRejectedError`=401、
  `InsufficientScopeError`=403、`ServerError`=500、
  `DataKeyUnavailableError`=503 (依存先 KMS 到達不能などの一時的障害)、
  RFC 8935 が固定する `SecurityEventRejectedError`=400
  (`error_format: set_delivery`)、`ScimProtocolError`=400
  (SCIM は個々の応答で `payload.status` が実際の値を持つため、
  model 側の `status` は代表値)。OAuth2 の各プロトコルエラー
  (`InvalidGrantError` 等) は RFC 6749 §5.2 の規定通り 400
  (`invalid_client` のみ 401)。
- 採用しなかった代替: `status` を interface 側 (`errors: [...]` の各要素
  ごと) に持たせる案 — 同じ error 名が複数 interface から返るケースで
  値がぶれる余地があり、model 側に一元化する方が単純。

## Plan

1. `SPECIFICATION_CORE_LANGUAGE.md` と `tools/check/schemas/scl-v3.schema.json`
   を更新し、`just verify-spec` 相当のチェックで言語拡張自体が壊れていない
   ことを確認する (既存 `docs/contexts/*.yaml` は `status` 未設定のまま
   一時的に検証が落ちる可能性があるため、`status` を MUST にする変更は
   全 error model への付与とセットでコミットする)。
2. SharedSignals inbound SET receiver の RFC 8935 エラー形式を調査する。
3. `docs/contexts/*.yaml` を 1 ファイルずつ (または関連するまとまり単位で)
   `status` を付与し、都度 `just verify-spec` を通す。
4. OAuth2 / SCIM / DCR / (必要なら SSF receiver) の該当 binding に
   `error_format` を設定する。
5. `tools/scl-to-openapi` を、単一 `default` レスポンスではなく status 別
   レスポンス + `error_format: problem_details` の場合は
   `application/problem+json` content-type を生成するよう対応させる。
6. `just scl-render` で派生物 (`spec/idmagic.openapi.json` 等) を再生成する。

## Tasks

- [x] T001 [SCL] `SPECIFICATION_CORE_LANGUAGE.md` に `status`/`error_format`
      の言語定義を追記した (§3.2 Model field 表、§3.3.1 Binding http 行、
      および `error_format` 各値の説明段落)。
- [x] T002 [SCL] `tools/check/schemas/scl-v3.schema.json` を対応する
      JSON Schema 変更で更新した (`Model.status` MUST-if-kind-error、
      `Binding.http.error_format` enum)。実装中に判明した
      `backend/shared/spec/loader.go` の Go 側 strict decoder にも
      `Model.Status` を追加した (Scope 参照)。
- [x] T003 [Research] SharedSignals inbound SET receiver の RFC 8935
      エラー形式規定の有無を調査する。→ RFC 8935 §2.3 が `{err,
      description}` 固定形式・400・`application/json` を MUST で規定して
      いることを確認。`error_format: set_delivery` を新設して対応する。
- [x] T004 [SCL] `docs/contexts/*.yaml` の全 `kind: error` model へ
      `status` を付与した (37 種類のユニークな error model 名、
      19 コンテキストファイル + `tools/ra`/`tools/scl-to-openapi` の
      自己記述 SCL を含む 21 ファイル・77 箇所)。Design 節に最終
      マッピングを記載。
- [x] T005 [SCL] OAuth2 (12 interface, RFC 7591 DCR 含む)/SCIM (15
      binding)/SSF receiver (1 binding, `set_delivery`) の
      `kind: http` binding へ `error_format` を設定した。
- [x] T005b [SCL] `InvalidRequestError` に紛れていたビジネスルール違反
      (14 ファイル・約40箇所) の Go handler → SCL interface 対応を
      調査 (Explore agent 委譲) した上で、37 個の専用 error model を
      新設し (`status: 422`、状態競合系は `409`)、6 コンテキストファイルの
      該当 interface の `errors: [...]` へ配線した。
      `admin_category_handler.go` の 2 条件のみ Go 側に区別可能な
      文字列コードが無く対象外 (Out of Scope 参照)。
- [x] T006 [App] `tools/scl-to-openapi` を status 別レスポンス生成・
      `error_format` 別 content-type 対応に更新した (RED→GREEN:
      `openapi.test.ts` に「エラーを status でグルーピングする」
      「error_format ごとに content-type を切り替える」テストを先に追加し
      fail を確認してから実装。scenario 相当参照:
      `models.status`/`bindings.error_format`)。
- [x] T007 [Verify] `just scl-render` で派生物を再生成した。
      `spec/idmagic.openapi.json` の diff を確認: 例えば `/token` は
      400/401/403/422 に分かれ全て `application/json`、
      `/scim/v2/Users` は 400 `application/scim+json`、
      `/ssf/streams/{stream_id}/events` は 400 `application/json`、
      それ以外の generic endpoint は `application/problem+json` になった。
- [x] T008 [Verify] `just verify` を通した。

## Verification

- `just verify-spec`
- `just scl-render` 後の生成物 diff レビュー (特に
  `spec/idmagic.openapi.json` のエラーレスポンス定義)
- `just verify`

## Risk Notes

75+ 個の error model を一つ一つ 400/422 判定するのは、業務ロジックの
実際の意味を取り違えるリスクがある。本 work item の Design 節に挙げた
候補リストはあくまで出発点であり、各ハンドラの実装コードを読んで
個別に確認すること。誤って 400 のままにしても RFC 9457 移行自体は
壊れないが、クライアント側のエラー種別判定が不正確になる。

共通 `InvalidRequestError` に紛れていたビジネスルール違反コードは、
`admin_category_handler.go` の 2 条件を除き 37 個の専用 error model として
SCL に反映した。ただし SCL 側の `status` が現時点の Go 実装 (引き続き
`WriteBrowserError(..., http.StatusBadRequest, ...)` で 400 を返す) と
一時的に乖離する — 実際の HTTP レスポンスが 422/409 になるのは、
`wi-326` が該当 390 箇所の呼び出しを Problem Details 実装へ移行する
タイミング。SCL-first の通常の流れ (spec が先行し実装が追従する) であり、
今回追加した他の大半の error model (Go 側が既に正しい status を返している
もの) との違いとして明記しておく。`DataKeyUnavailableError` (503) と
`WorkloadAttestationRejectedError` (401) は対応する Go ハンドラでの
実装が見当たらず、model の description から RFC 9110 の意味論で判断した
推定値である。`ScimProtocolError` の `status` (400) は代表値であり、
実際の SCIM エラー応答は `payload.status` で個別に変わる
(RFC 7644 §3.12 の仕様通り)。

## Completion

- **Completed At**: 2026-08-08
- **Summary**:
  SCL 言語 (`SPECIFICATION_CORE_LANGUAGE.md`) に `kind: error` model の
  `status` field (MUST) と `kind: http` binding の `error_format` field
  (`problem_details` 既定 | `oauth2` | `scim` | `set_delivery`) を追加し、
  `tools/check/schemas/scl-v3.schema.json` と Go 側の strict decoder
  (`backend/shared/spec/loader.go`) を対応するスキーマ変更で更新した。
  19 の bounded context ファイルおよび `tools/ra`/`tools/scl-to-openapi`
  自己記述 SCL、合計 21 ファイル・77 個の `kind: error` model すべてに
  `status` を付与し (RFC 9110 に沿って 400/401/403/404/409/422/500/503 を
  各 model の意味論に応じて割り当て、既存 Go 実装のステータスコードと
  整合させた)、OAuth2 (12 interface, RFC 7591 DCR 含む)・SCIM (15
  binding)・SharedSignals inbound SET receiver (1 binding, RFC 8935 の
  固定形式を新設の `set_delivery` として表現) の `kind: http` binding へ
  `error_format` を設定した。`tools/scl-to-openapi` を、単一の `default`
  エラーレスポンスではなく HTTP status 別のレスポンス + `error_format` に
  応じた content-type (`application/problem+json`/`application/json`/
  `application/scim+json`) を生成するよう変更し (test-first: RED→GREEN)、
  `just scl-render` で `spec/idmagic.openapi.json` 等の派生物を再生成した。
  さらに、共通 `InvalidRequestError` (400) に紛れ込んでいた Go 側の
  ビジネスルール違反コード (`invalid_role`、`password_reuse` 等、14 ファイル・
  約40箇所) を Go handler → SCL interface の対応関係を洗い出した上で
  37 個の専用 error model (`status: 422`、状態競合系は `409`) として
  個別にモデリングし、6 コンテキストファイルの該当 interface の
  `errors: [...]` へ配線した。ADR-154 は本 work item の実装により
  `suggested` から `accepted` に更新した。

  **未対応の範囲 (Out of Scope として明記)**:
  - `admin_category_handler.go` の 2 条件 (カテゴリ名必須・存在しない
    カテゴリ割当) のみ、Go 側に区別可能な文字列コードが無いため対象外
    (Go 側の変更とセットで `wi-326` にて対応)。
  - 新設した 37 個の granular error model のうち、Go 側の対応する
    ビジネスルール違反コードは引き続き `WriteBrowserError(...,
    http.StatusBadRequest, ...)` で 400 を返している。SCL は正しい
    status (422/409) を宣言済みだが、実際の HTTP レスポンスが追従するのは
    `wi-326` が該当呼び出し箇所を Problem Details へ移行するタイミングに
    なる (SCL-first の意図した一時的な spec/実装間の乖離)。
  - `backend/shared/http/support_http` の Problem Details 実装と約 390
    箇所のハンドラ移行そのものは `wi-326` の scope。
- **Verification Results**:
  - `just verify` - passed (test-go / lint-go / test-ui-unit / build-ui /
    check / typecheck-tools / test-tools / lint-ui / traceability-strict /
    format-check-ui すべて green)
  - `bun test` (tools/scl-to-openapi, tools 全体) - passed、新規 RED→GREEN
    テスト 3 件 (status によるグルーピング、複数 error の同一 status への
    集約、`error_format` ごとの content-type 切り替え) を含む
  - `just scl-render` 後の `spec/idmagic.openapi.json` diff を目視確認
