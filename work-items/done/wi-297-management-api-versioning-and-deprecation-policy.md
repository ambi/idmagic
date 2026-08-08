---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-25
depends_on: []
initial_context:
  scl:
    System:
      - interfaces.BackendErrorResponse
    Tenancy:
      - interfaces.GetAdminSettings
    Sourcing:
      - interfaces.GetScimServiceProviderConfig
  decisions:
    - decisions/ADR-011-discovery-as-derived-artifact.md
    - decisions/ADR-136-scope-management-apis-by-owned-resource.md
    - decisions/ADR-137-unify-api-access-token-wire-format.md
  source:
    - backend/shared/http
    - backend/audit/handlers_http
    - backend/sourcing/scim/handlers_http
    - spec/idmagic.openapi.json
  tests:
    - backend/shared/http
  stop_before_reading:
    - backend/oauth2/token
    - frontend/src/features
---

# 管理 API の版管理と非推奨化ポリシーを定義し、外部契約として保証する

## Motivation

IdMagic は管理 API を大量に公開している (SCL の interface は全 context 合計で 270 件超、
うち管理 / 自己管理 API が大半)。SCIM API と管理 API は API アクセストークン
([[ADR-137-unify-api-access-token-wire-format]]) で外部から叩かれる**外部契約**である。

しかし **版管理と非推奨化の方針が存在しない**:

- 管理 API のパスに版が無く、破壊的変更を安全に入れる手段が無い。
  (SCIM は仕様上 `/scim/v2/` で版が入っているが、これは SCIM の版であって IdMagic の版ではない)
- 「いつ・どう予告して・いつ消すか」の規約が無い。結果として実質的に
  「破壊的変更をしない」か「予告なく壊す」の二択になる。
- OpenAPI (`spec/idmagic.openapi.json`) は SCL からの派生物として生成されているが、
  版間の差分や互換性の宣言を持たない。
- 非推奨を伝える仕組み (`Deprecation` / `Sunset` ヘッダ、応答内の警告) が無い。

これはエンタープライズ導入で具体的な障害になる:

1. **顧客の自動化が壊れる**。テナント側は管理 API で provisioning / 棚卸し / IaC を
   組む。予告なく応答形状が変わると、その自動化が沈黙して壊れる。
2. **アップグレードの可否を判断できない**。運用者は「この版に上げると自分たちの
   スクリプトが壊れるか」を知りたいが、答える材料が無い。
   [[wi-165-high-availability-and-failover-resilience-topology]] は
   スキーマとデプロイの N/N+1 互換を扱うが、**API 契約の互換は誰も扱っていない**。
3. **競合は明示している**。Okta は API のバージョニングと deprecation スケジュールを
   公開し、Entra は Graph の `v1.0` / `beta` を分離、Keycloak は Admin REST API の
   互換方針とリリースノートでの破壊的変更告知を運用している。

本 WI は「何を互換とみなすか」「どう版を付けるか」「どう非推奨にして、いつ消すか」を
決めて文書化し、機械的に検査できるところまで持っていく。

## Scope

- **decision**:
  - 新規 ADR (API 版管理と非推奨化): 互換の定義 (後方互換な変更 = フィールド追加 /
    任意パラメータ追加 / 新エンドポイント、破壊的変更 = フィールド削除・改名・型変更・
    必須化・エラーコード変更・既定値変更)、版の付け方 (パス版 `/admin/v1/` か
    ヘッダ版か。SCIM は仕様準拠の `/scim/v2/` を維持)、同時サポートする版の数と期間、
    非推奨の告知方法 (`Deprecation` / `Sunset` HTTP ヘッダ + OpenAPI の `deprecated` +
    リリースノート)、最小の非推奨期間、内部利用のみの API (フロントエンド専用の
    browser API) を外部契約から除外する境界、SCL の interface に版と安定性
    (stable / beta / internal) をどう表現するかを記録する。
- **scl**:
  - interface に安定性区分 (`stability: stable | beta | internal`) と非推奨情報
    (`deprecated_since` / `sunset_at` / 後継 interface 参照) を表現できるようにする。
  - `System` に「外部契約となる API は版と安定性を宣言する」制約を追加し、
    `BackendErrorResponse` のエラーコード安定性を明記する
    (エラーコードは破壊的変更の対象として扱う)。
  - 既存 interface を棚卸しして安定性を付与する。フロントエンド専用の browser API
    (`GetBrowserTransaction` / `SubmitBrowserConsent` / `SubmitBrowserLogin` 等) は
    `internal` として外部契約から外す。
  - `scenarios` に「非推奨 API を呼ぶと Deprecation / Sunset ヘッダが返る」を追加する。
- **go**:
  - 版付きルーティングを導入する。既存パスは現行版のエイリアスとして残し、
    破壊的変更が必要になったときに新版を切れる構造にする。
  - 非推奨 interface に `Deprecation` / `Sunset` ヘッダを付ける共通ミドルウェアを追加する。
  - 内部 (browser) API と外部契約 API のルーティング境界を明示する。
- **tooling**:
  - OpenAPI 生成に安定性・非推奨・版を反映する (`spec/idmagic.openapi.json`)。
  - `internal` な interface を OpenAPI の公開対象から外すか、明示的に区別する。
  - 破壊的変更を検知する検査を追加する: 前回リリースの OpenAPI と現在の OpenAPI を
    比較し、破壊的差分があれば版の変更または非推奨手続きを要求する。
    `just check-api-compat` として `justfile` に追加し、CI に組み込む。
    比較基準となる「前回リリースの OpenAPI」を repo 内に固定する。
- **documentation**:
  - README に API の安定性区分、版管理、非推奨スケジュール、互換の定義を追記する。
  - 非推奨中の API 一覧を機械生成できる形で置く。

## Out of Scope

- 実際の破壊的変更の実施。本 WI は「安全に変更できる仕組みと約束」を作るだけで、
  既存 API の形は変えない。
- OAuth2 / OIDC / SAML / SCIM の**プロトコル**エンドポイントの版管理。これらは標準が
  版を定義しており、IdMagic の版管理の対象外とする (discovery が正本)。
- SDK / クライアントライブラリの提供。
- Terraform provider 等の IaC 統合。→ [[wi-102-realm-declarative-config-export-import]] の
  宣言的設定が土台になる。
- API の rate limit / quota。→ [[wi-27-endpoint-rate-limit-and-bot-mitigation]]
- SCL 自体の spec_version 管理 (既に 3.0 として存在する)。

## Plan

- **既存パスを壊さないことを絶対条件にする**。版を導入する時点で URL を変えると、
  本 WI 自身が破壊的変更になる。現行パスを「v1 のエイリアス」として維持し、
  新版が必要になった時に初めて `/v2/` を切る。これで導入コストがゼロになる。
- **安定性の棚卸しが本体の作業**である。270 件超の interface を stable / beta / internal に
  分類する。ここで「フロントエンド専用の browser API を外部契約から外す」ことが最も
  効く。これらは UI と一体で変わるため外部契約にすると開発が固まる。
- **破壊的変更の検知を機械化する**。方針文書だけでは守られない。前回リリースの
  OpenAPI をベースラインとして repo に固定し、差分検査を CI に入れる。
  これにより「うっかり壊す」を構造的に防げる。RA の思想 (仕様が正本、検証で守る) と一致する。
- **エラーコードを契約に含める**。クライアントはエラーコードで分岐するため、
  エラーコードの変更は破壊的変更である。`BackendErrorResponse` のコード集合を
  契約として明示し、差分検査の対象にする。
- **非推奨は HTTP ヘッダで機械可読にする**。`Deprecation` と `Sunset` (RFC 8594 系) を
  付けることで、顧客側の監視が非推奨呼び出しを検知できる。ヘッダだけで済ませず、
  非推奨一覧をドキュメントに生成する。
- 未決定: 同時サポートする版の数 (2 版か 3 版か) と最小非推奨期間。運用負荷と顧客の
  移行期間のトレードオフなので、ADR で明示的に決める。第一候補は「同時 2 版・
  非推奨から削除まで最低 12 か月」。

## Tasks

- [x] T001 [Survey] 全 interface を棚卸しし、stable / beta / internal に分類した一覧を作る。
      browser API と管理 API の境界を明確にする。
      → 324 interface を分類 (stable 158 / internal 166)。境界基準は「API access token
      (`ManagementApiClient*`/`SelfApiClient*`/`Scim*` policy) で到達可能か、protocol-governed
      か、認証不要の公開資産・運用 endpoint か」。`/api/admin` 201 件中 116 件、`/api/account` 31 件中
      8 件は User principal のセッション認可のみで API access token 経路を持たず internal 認定
      (ADR-156 参照)。
- [x] T002 [ADR] API 版管理と非推奨化の ADR を起票する (互換の定義・版の付け方・
      同時サポート数と期間・告知方法・internal の除外・SCL での表現)。
      → [[ADR-156-api-versioning-and-deprecation-policy]]。
- [x] T003 [SCL] interface に `stability` と非推奨情報を表現できるようにし、T001 の分類を
      反映する。`System` に外部契約の制約と `BackendErrorResponse` のコード安定性を追加し、
      scenario を 1 件追加して `just check-scl` を通す。
      → `stability` (`stable|beta|internal`) を JSON Schema (`scl-v3.schema.json`) の Interface
      に必須 field として追加、324 interface 全件に反映。`deprecated_since`/`sunset_at`/`successor`
      を任意 field として追加し、`successor` 未解決参照・`sunset_at` の 12 か月下限・
      `sunset_at` without `deprecated_since` を `scl-semantics.ts` で機械検証 (RED:
      `missing-stability.json` フィクスチャで schema 違反を確認、
      および `resolves successor references and enforces the minimum deprecation period` テストで
      successor 未解決・12 か月下限違反を確認 → GREEN)。`System` に `InterfaceStability`/`Deprecation`
      glossary と `BackendErrorResponse` を `stability: stable` (エラーコードを契約に含める) へ、
      非推奨ヘッダの scenario を 1 件追加。組み込み RA/SCL tooling 自身の SCL (`tools/*/spec/scl.yaml`)
      13 interface も `stability: internal` を付与 (dev tooling の CLI、外部契約ではない)。
      `just check-scl` / `just test-tools` 通過。
- [x] T004 [Routing] 版付きルーティングを導入する。現行パスを現行版のエイリアスとして残し、
      既存の全テストが無変更で通ることを確認する。RED: 版付きパスと現行パスの両方で
      同じ応答になる handler テスト → GREEN。
      → `backend/shared/http/support_http/versioning.go`: `Echo.OnAddRoute` hook
      (`RegisterVersionAliases`) が `/api/admin`・`/api/account` 配下の全登録 route を
      `<prefix>/v1/...` へ自動的にミラーする。個々の `handlers_http.RegisterRoutes` は
      一切変更なし (path 文字列を書き換えていない)。RED: `TestRegisterVersionAliases_*`
      3 件 (`undefined: RegisterVersionAliases` で fail) → GREEN。tenant path-style
      (`/realms/:tenant_id/...`) でも group prefix の後に `/v1/` が挿入されることを
      `TestRegisterVersionAliases_HonorsGroupPrefix` で確認。
- [x] T005 [Deprecation] `Deprecation` / `Sunset` ヘッダを付ける共通ミドルウェアを追加する。
      RED: 非推奨マークした interface でヘッダが返るテスト → GREEN。
      → `backend/shared/http/support_http/deprecation.go`: `DeprecationHeadersMiddleware(scl)`
      が起動時ロード済みの `*spec.SCL` (`backend/cmd/idmagic/server.go` の `spec.LoadSCL()`)
      から `deprecated_since` 設定済み interface の (method, canonical path) 索引を構築し、
      一致した応答に `Deprecation` (RFC 9745 の HTTP-date 形式) と、`sunset_at` があれば
      `Sunset` (RFC 8594) を付与する。canonical path 化で tenant routing の 2 形式
      (`/realms/:tenant_id` 有無) と `/v1/` alias の両方を単一索引で吸収。RED:
      `TestDeprecationHeadersMiddleware` (`undefined: DeprecationHeadersMiddleware` で fail)
      → GREEN。`Register()` に `e.Use(support.DeprecationHeadersMiddleware(d.SCL))` を配線、
      `just test-go` / `just verify-go` 通過 (現行 SCL に `deprecated_since` を持つ
      interface は無いため、実運用の応答は無変更)。
- [x] T006 [OpenAPI] 生成器に安定性・非推奨・版を反映し、`internal` の扱いを決めて
      `spec/idmagic.openapi.json` を再生成する (`just scl-render`)。
      → `tools/scl-to-openapi/src/openapi.ts`: `stability`/`deprecated_since`/`sunset_at`/
      `successor` を `x-scl-stability` 等の拡張 field と OpenAPI 標準の `deprecated` へ反映。
      `/api/admin`・`/api/account` 配下の全 operation に `/v1/` alias path を生成
      (Go 側の path-only な `RegisterVersionAliases` と対称に、stability に関わらず path
      prefix だけで判定。`stability` は URL alias の有無ではなく T007 の互換検査対象を
      決める)。RED: `mirrors stable interfaces...`/`reflects stability and deprecation
      metadata...` 2 件 fail → GREEN。`just scl-render` で再生成し
      `TestAssembledRoutesMatchGeneratedOpenAPI` (既存の contract test、無変更) が通る
      ことを確認 (`just verify-go` 緑)。
- [x] T007 [Compat check] 前回リリースの OpenAPI をベースラインとして repo に固定し、
      破壊的差分を検知する `just check-api-compat` を実装する。エラーコード集合も比較対象に含める。
      RED: 意図的にフィールドを削除して検査が落ちることを確認 → 実装 → GREEN (削除は戻す)。
      → `spec/idmagic.openapi.baseline.json` を最初のリリースベースラインとして固定
      (この WI の完了時点のスナップショット。以降は release 時に更新)。
      `tools/check-api-compat/src/compat.ts`: `$ref` 解決 (循環ガード付き) を伴う再帰的
      schema diff で ADR-156 の互換定義を機械検証 — フィールド削除・改名 (削除側で検出)・
      型変更・必須化・既定値変更・エラーコード (`oneOf` の `$ref` 名集合) 削除・path/operation/
      parameter/response status 削除を breaking として検出し、フィールド追加・任意 parameter
      追加・新規 endpoint・エラーコード追加は無視する。単体テスト 15 件 GREEN
      (`compat.test.ts`)。`just check-api-compat` (`tools/check-api-compat/src/main.ts`)
      を新設し `justfile` の `verify`/`verify-serial` に組み込み。
      統合 RED: `LivenessProbe.output.status` を一時的に spec から削除して `just scl-render` →
      `just check-api-compat` が `GET /livez 200: field 'status' removed` で fail することを確認
      → SCL を復元 → GREEN。ツール自身の SCL (`tools/check-api-compat/spec/scl.yaml`) と
      `architecture.yaml` の contexts/modules 登録も追加 (`just check-architecture` 要求)。
- [x] T008 [CI] `check-api-compat` を `.github/workflows/idmagic-ci.yaml` に組み込む。
      → `verify` job の "Verify Core" step に `just check-api-compat` を追加。
      `justfile` の `verify`/`verify-serial` にも追加済みなのでローカルでも同じゲートがかかる。
- [x] T009 [Docs] README に安定性区分・版管理・非推奨スケジュール・互換の定義を追記し、
      非推奨一覧の生成先を明示する。
      → README.md に "API Stability, Versioning & Deprecation" 節を追加 (English)。
      stability tier・外部契約の判定基準・互換の定義・パス版管理・protocol endpoint の除外・
      Deprecation/Sunset ヘッダ・非推奨一覧を得る `jq` コマンド (`spec/idmagic.openapi.json` から
      機械生成、別リストを手で保守しない)・`just check-api-compat` とベースライン更新手順
      (release 時に `cp spec/idmagic.openapi.json spec/idmagic.openapi.baseline.json`) を記載。
- [x] T010 [Verify] 下記 Verification を緑にする。
      → 全項目 green。手動確認は `PERSISTENCE=memory` で実サーバを起動し curl で実施
      (下記 Completion 参照)。

## Verification

- `just check` / `just check-scl` / `just check-work-items` / `just check-ids`
- `just check-api-compat` (新設) — ベースラインに対して破壊的差分ゼロ
- `just scl-render` — OpenAPI が安定性・非推奨を反映して再生成される
- `just test-go` / `just verify-go` / `just verify-ui` — 既存のすべてが無変更で緑
  (本 WI は既存 API の形を変えないため、UI 側の変更は不要であることを確認する)
- 手動: 版付きパスと現行パスの両方で同一応答が返ることを確認する。
- 手動: 非推奨マークした 1 件の API を呼び、`Deprecation` / `Sunset` ヘッダを確認する。

## Risk Notes

版付きルーティングの導入で既存パスの挙動が変わると、本 WI 自体が破壊的変更になる。
現行パスをエイリアスとして維持し、「既存テストが無変更で通る」ことを完了条件にする。
安定性の分類を誤ると、内部 API を外部契約に格上げして開発を固めてしまう、あるいは
逆に顧客が使っている API を internal として気軽に壊せる状態にしてしまう。
browser API と管理 API の境界を SCL で明示し、分類根拠を ADR に残す。
互換検査のベースラインが古いまま放置されると検査が形骸化する。リリース時に
ベースラインを更新する手順を README に書き、更新忘れを検知できる形にする。
同時サポート版数を増やすと保守コストが線形に増える。ADR で上限を決め、
安易に版を増やさない方針を明記する。

## Completion

- **Completed At**: 2026-08-08
- **Summary**:
  [[ADR-156-api-versioning-and-deprecation-policy]] で互換の定義・パス版管理
  (`/api/admin`・`/api/account`、現行パス=v1)・同時2版・非推奨から最低12か月・
  `Deprecation`/`Sunset` ヘッダ・internal 境界 (API access token 到達可能性で判定) を決定した。
  全 324 interface に `stability` (stable 158 / internal 166) を付与し、`System` に
  `InterfaceStability`/`Deprecation` glossary と非推奨 scenario を追加、`BackendErrorResponse`
  のエラーコードを契約として `stable` にした。Go 側は `Echo.OnAddRoute` フックで
  `/api/admin`・`/api/account` 配下の全 route を `/v1/` へ自動ミラーする
  `RegisterVersionAliases`(既存 handler の登録コードは無改修) と、SCL の
  `deprecated_since`/`sunset_at` を読んで応答ヘッダを付与する `DeprecationHeadersMiddleware`
  を追加。`tools/scl-to-openapi` は同じ `/v1/` ミラーと `x-scl-stability` 等の拡張 field を
  生成に反映。新設 `tools/check-api-compat` は `$ref` 解決付き再帰 schema diff で
  ADR-156 の破壊的変更定義 (フィールド削除・改名・型変更・必須化・既定値変更・エラーコード削除・
  path/operation/parameter/response 削除) を検知し、`spec/idmagic.openapi.baseline.json`
  (このWI完了時点を最初のリリースベースラインとして固定) との差分を `just check-api-compat`
  として CI (`idmagic-ci.yaml`) と `just verify` に組み込んだ。README に運用手順を追記。
  **対応していないこと (Out of Scope の通り)**: 既存 API の実際の破壊的変更・形状変更は
  行っていない (現行パスは無変更で全既存テストが green)。OAuth2/OIDC/SAML/WS-Federation/
  SCIM/SharedSignals のプロトコル endpoint は IdMagic 独自の版管理の対象外のまま
  (discovery が正本、[[ADR-011-discovery-as-derived-artifact]])。SDK/クライアントライブラリ、
  Terraform provider 等の IaC 統合、rate limit/quota は本 WI の範囲外。SCL 自体の
  `spec_version` (3.0) は変更していない。`beta` stability は仕組みとして用意したが、
  該当する interface は現時点で無い。
- **Verification Results**:
  - `just check` / `just check-scl` / `just check-work-items` / `just check-ids` — green
  - `just check-api-compat` — ベースラインに対して破壊的差分ゼロ。統合 RED
    (`LivenessProbe.output.status` を一時削除→fail 確認→復元→GREEN) 済み
  - `just scl-render` — OpenAPI に `/v1/` alias・`x-scl-stability`・`deprecated` 等を反映して再生成
  - `just test-go` / `just verify-go` / `just verify-ui` — green (既存 UI 変更不要)
  - `just verify` (全体) — green
  - 手動: `PERSISTENCE=memory go run ./backend/cmd/idmagic` で実サーバを起動し、
    `/realms/default/api/admin/settings` と `/realms/default/api/admin/v1/settings` が
    同一 status (401) を返すことを curl で確認
  - 手動: `LivenessProbe` に `deprecated_since`/`sunset_at` を一時設定して再起動し、
    `/livez` の応答が `Deprecation: Thu, 01 Jan 2026 00:00:00 GMT` /
    `Sunset: Fri, 01 Jan 2027 00:00:00 GMT` ヘッダを返すことを curl で確認 → SCL を復元
