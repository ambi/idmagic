---
status: pending
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

- [ ] T001 [Survey] 全 interface を棚卸しし、stable / beta / internal に分類した一覧を作る。
      browser API と管理 API の境界を明確にする。
- [ ] T002 [ADR] API 版管理と非推奨化の ADR を起票する (互換の定義・版の付け方・
      同時サポート数と期間・告知方法・internal の除外・SCL での表現)。
- [ ] T003 [SCL] interface に `stability` と非推奨情報を表現できるようにし、T001 の分類を
      反映する。`System` に外部契約の制約と `BackendErrorResponse` のコード安定性を追加し、
      scenario を 1 件追加して `just check-scl` を通す。
- [ ] T004 [Routing] 版付きルーティングを導入する。現行パスを現行版のエイリアスとして残し、
      既存の全テストが無変更で通ることを確認する。RED: 版付きパスと現行パスの両方で
      同じ応答になる handler テスト → GREEN。
- [ ] T005 [Deprecation] `Deprecation` / `Sunset` ヘッダを付ける共通ミドルウェアを追加する。
      RED: 非推奨マークした interface でヘッダが返るテスト → GREEN。
- [ ] T006 [OpenAPI] 生成器に安定性・非推奨・版を反映し、`internal` の扱いを決めて
      `spec/idmagic.openapi.json` を再生成する (`just scl-render`)。
- [ ] T007 [Compat check] 前回リリースの OpenAPI をベースラインとして repo に固定し、
      破壊的差分を検知する `just check-api-compat` を実装する。エラーコード集合も比較対象に含める。
      RED: 意図的にフィールドを削除して検査が落ちることを確認 → 実装 → GREEN (削除は戻す)。
- [ ] T008 [CI] `check-api-compat` を `.github/workflows/idmagic-ci.yaml` に組み込む。
- [ ] T009 [Docs] README に安定性区分・版管理・非推奨スケジュール・互換の定義を追記し、
      非推奨一覧の生成先を明示する。
- [ ] T010 [Verify] 下記 Verification を緑にする。

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
