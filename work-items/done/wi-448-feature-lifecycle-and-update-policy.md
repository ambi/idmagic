---
depends_on: []
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-08-30
priority: p0
change_kind: feature
evidence_policy: risk-based-v3
initial_context:
  specification:
    - docs/contexts/system/scenarios.md#REQ-SYSTEM-016
    - docs/contexts/system/scenarios.md#REQ-SYSTEM-017
  typespec:
    - IdMagic.Contract.FeatureRuntimeMetadata
    - IdMagic.OAuth2.Operations.HealthHttpResponse
  source:
    - docs/contexts/system/decisions.md
    - docs/contexts/system/internals.md
    - docs/structure.md
    - docs/deployment.md
    - backend/cmd/internal/bootstrap/config.go
    - backend/cmd/internal/bootstrap/configreference.go
    - backend/cmd/internal/bootstrap/deps.go
    - backend/cmd/idmagic/server.go
    - backend/shared/http/support_http/deps.go
    - backend/shared/http/server_http/health_handler.go
  tests:
    - backend/cmd/internal/bootstrap/config_test.go
    - backend/cmd/internal/bootstrap/configreference_test.go
    - backend/shared/http/server_http/routes_e2e_test.go
  stop_before_reading: [backend/application, frontend, infra]
affected_spec:
  - { path: docs/contexts/system/scenarios.md, requirement: REQ-SYSTEM-016 }
  - { path: docs/contexts/system/scenarios.md, requirement: REQ-SYSTEM-017 }
  - { path: spec/contexts/system/models.tsp, symbol: IdMagic.Contract.FeatureRuntimeMetadata }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.HealthHttpResponse }
primary_use_cases:
  - id: resolve-feature-selection-before-startup
    requirement: REQ-SYSTEM-016
    observable_result: Operator の明示指定、既定値、依存閉包が正式な起動設定経路で一つの有効機能集合へ解決され、不正な選択は副作用のある初期化前に集約拒否される。
    unit_test: { path: backend/cmd/internal/bootstrap/features_test.go, name: TestResolveFeatures_REQ_SYSTEM_016, task: test-go-race }
    e2e_test: { path: backend/cmd/internal/bootstrap/features_e2e_test.go, name: TestLoadFeatureConfig_REQ_SYSTEM_016, task: test-go-race }
    unit_fault_model: 明示的に無効化した依存を resolver が有効化して成功を返す。
    e2e_fault_model: 起動設定アダプターが FEATURES_ENABLE を resolver へ渡さず、Operator の選択を無視する。
  - id: publish-resolved-feature-metadata
    requirement: REQ-SYSTEM-017
    observable_result: Operator が /health を読むと、形式の版と有効機能の識別子、版、成熟度、更新方針だけが返る。
    unit_test: { path: backend/cmd/internal/bootstrap/features_test.go, name: TestFeatureResolutionMetadata_REQ_SYSTEM_017, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/routes_e2e_test.go, name: TestHealthReportsFeatureMetadata_REQ_SYSTEM_017, task: test-go-race }
    unit_fault_model: 解決結果のメタデータが無効な機能を含めるか、更新方針を欠落させる。
    e2e_fault_model: composition root から HealthInfo への配線が feature metadata を捨て、/health が空または旧形式を返す。
---

# 機能の成熟度、依存、有効化、更新方針を一つの正本で管理する

## Motivation

IdMagic は API の安定性、標準の採否、個別の機能フラグ、起動時設定を別々に管理している。この分離自体は正しいが、実行時に有効な機能について、成熟度、版、依存機能、既定の有効化、ローリング更新の可否を同時に検証する正本がない。そのため、実験的な機能を意図せず既定有効にすること、依存機能だけを無効にすること、停止が必要な変更を通常のローリング更新へ流すことを起動前に拒否できない。

Keycloak の feature profile から閉じた登録と依存検証を採用するが、成熟度名、既定値、機能一覧は IdMagic 自身の仕様と検証から決める。API の `stable` と `beta`、標準対応表、テナント単位の設定を一つの概念へ潰さない。

## Scope

- `FeatureID`、`FeatureVersion`、`FeatureMaturity`、`DefaultEnablement`、`UpdatePolicy`、`FeatureDefinition` を閉じた型として定義する。
- `FeatureRegistry` を製品ビルドが持つ唯一の実行時機能一覧とし、重複 ID、同じ未版名の競合、存在しない依存、依存循環、不正な既定有効化をビルドまたは起動前に拒否する。
- `ResolveFeatures(registry, explicitEnable, explicitDisable)` を決定的な計算にし、明示指定、既定値、依存を解決して有効な機能集合または全検証エラーを返す。
- `UpdatePolicy` は少なくとも `rolling`、`recreate_on_version_change`、`recreate_always` を持ち、後続の配備前互換性検査が利用できるよう生成メタデータへ含める。
- 実験的またはプレビューの機能を明示的に有効化した事実、非推奨機能を利用している事実を、秘密情報を含まない起動ログと運用メタデータで可視化する。
- `ConfigurationReference` の既存生成経路を維持し、機能選択設定に必要な説明、成熟度、更新影響だけを同じ正本から生成する。

## Out of Scope

- テナント管理者が任意の機能を動的に登録する plugin 機構。
- API の `stable` と `beta`、標準の採否、UI の表示可否を `FeatureMaturity` だけで自動決定すること。
- 既存の全機能を遡及的に機能フラグ化すること。
- 配備済み版との互換性を判定する CLI と異版混在テスト。後続 work item が扱う。

## Design

`FeatureID`、`FeatureVersion`、`FeatureMaturity`、`DefaultEnablement`、`UpdatePolicy` は閉じた値とし、`FeatureDefinition` は `ID`、未版名、版、成熟度、既定の有効化、依存する `FeatureID`、更新方針、任意の仕様参照を持つ不変値とする。`FeatureRegistry` は `[]FeatureDefinition`、`ResolveFeatures(registry FeatureRegistry, explicitEnable, explicitDisable []FeatureID) (FeatureResolution, error)` は有効な定義、起動警告、`FeatureRuntimeMetadata` を返す。登録は composition root で静的に行い、設定ファイルやデータベースから型や依存関係を追加しない。成熟度は `experimental`、`preview`、`supported`、`deprecated` とし、実験的機能は既定無効、非推奨機能は新規の既定有効化を禁止する。

機能の依存は有向グラフとして解決するが、一般的な依存注入器にはしない。`ResolveFeatures` は有効化された機能の依存閉包を求め、明示的に無効化された依存が必要なら設定エラーにする。エラーは全件を集約し、`REQ-SYSTEM-016` と同じく副作用のある初期化前に返す。

設定は `LoadFeatureConfig(loader *ConfigLoader, registry FeatureRegistry) FeatureResolution` が `FEATURES_ENABLE` と `FEATURES_DISABLE` を読み、検証エラーを既存の `ConfigLoader` へ返す。このアダプターより内側の解決計算は環境、時刻、乱数、永続化、通知へ依存しない。起動ログは解決結果の警告だけを出力し、`/health` と ConfigurationReference は同じ解決結果と registry から導出する。

機能一覧の生成物は人が編集する正本にしない。正準文書と標準対応表は製品の約束を所有し、registry は実行時選択と更新影響を所有する。両者の対応を必要とする機能だけ、安定した規範 ID または標準 ID を任意の `SpecificationReference` として持つ。既存の常時提供機能は遡及登録せず、現在の製品 registry は空でも有効とする。

## Plan

1. `REQ-SYSTEM-016` と `REQ-SYSTEM-017`、機能の成熟度を所有する正準文書を先に更新する。
2. domain 型、registry 検証、機能解決を Unit RED から実装する。
3. 起動時 Config と composition root へ接続し、不正な組合せが副作用前に失敗する E2E を追加する。
4. 実行時メタデータと設定リファレンスを同じ registry から生成する。
5. 既存機能から、実験的または更新方針を明示する必要がある最小集合だけを登録する。

## Tasks

- [x] T001 [Spec] `REQ-SYSTEM-016` / `REQ-SYSTEM-017` と TypeSpec に機能成熟度、既定有効化、依存、更新方針、起動時拒否、版付き運用メタデータを仕様化し、`mise run check-spec`、`mise run check-api-compat`、`mise run check-boundaries` を通した。
- [x] T002 [Domain] `TestResolveFeatures_REQ_SYSTEM_016` は `FeatureRegistry` が存在しないコンパイル失敗を、`TestFeatureResolutionMetadata_REQ_SYSTEM_017` はメタデータ型が存在しないコンパイル失敗を Unit RED として観測した。重複、循環、依存欠落、既定値、明示無効、警告を含む `FeatureDefinition` と `ResolveFeatures` を実装し、`mise run test-go-package ./backend/cmd/internal/bootstrap` を通した。
- [x] T003 [Config] `LoadFeatureConfig` で `FEATURES_ENABLE` / `FEATURES_DISABLE` を既存の `ConfigLoader` へ接続し、registry と選択の全エラーを副作用前の集約検証へ返した。
- [x] T004 [Metadata] 解決済み機能だけを依存順で持つ `FeatureRuntimeMetadata` と形式版 `1` を生成し、全プロセスの警告と API の `/health` へ接続した。
- [x] T005 [Docs] `RenderFeatureRegistryReference` を同じ registry から生成し、選択可能機能が無いビルドも明示した。`mise run generate-config-reference` 後に `mise run check-config-reference` を通した。
- [x] T006 [E2E] `TestLoadFeatureConfig_REQ_SYSTEM_016` は `LoadFeatureConfig` が存在しないコンパイル失敗を、`TestHealthReportsFeatureMetadata_REQ_SYSTEM_017` は HTTP 用メタデータと配線が存在しないコンパイル失敗を E2E RED として観測した。正式な起動設定経路と `/health` の最終観測結果を接続し、両パッケージのテストを通した。
- [x] T007 [Verify] 主要ユースケースの Unit/E2E RED と 4 件の故障注入を記録し、`mise run verify` を終了コード 0 で通した。

## Verification

- `mise run check-spec`
- `mise run test-go-package ./backend/cmd/internal/bootstrap`
- `mise run check-config-reference`
- `mise run verify`
- 実験的機能の暗黙有効化、存在しない依存、循環、依存の明示無効化を一つずつ与えると起動前に失敗する。
- registry から機能を削除または更新方針を変えると、生成メタデータと参照の乖離検査が失敗する。

## Risk Notes

リスクは medium。registry が製品機能の新しい総覧へ膨張すると、標準対応表、API 契約、テナント設定の責任を奪い、変更のたびに中央ファイルを編集する構造になる。実行時に選択できる機能と更新影響だけを登録し、静的に常時提供する機能は列挙しない。

## Completion

- **Completed At**: 2026-08-31
- **Summary**:
  `mise run spec-diff` は `normative specification change against main` を返し、`REQ-SYSTEM-016` と `REQ-SYSTEM-017` を変更した。TypeSpec には `FeatureMaturity`、`FeatureRuntimeMetadata`、`FeatureUpdatePolicy`、`RuntimeFeatureMetadata` を追加した。製品ビルドが所有する静的な機能 registry、既定値・明示指定・依存閉包を決定的に解決する純粋計算、起動前の集約拒否、起動警告、設定リファレンス、形式版付き `/health` メタデータを同じ解決結果へ接続した。
- **Primary Use Case Evidence**:
  - id: resolve-feature-selection-before-startup
    unit_red: "TestResolveFeatures_REQ_SYSTEM_016 は `FeatureRegistry`、`FeatureDefinition`、`ResolveFeatures` が未定義のためコンパイルに失敗した（0 pass、build failed）。"
    e2e_red: "TestLoadFeatureConfig_REQ_SYSTEM_016 は `LoadFeatureConfig` が未定義のためコンパイルに失敗した（0 pass、build failed）。"
    unit_fault_injection: "明示的に無効化した依存の拒否条件を一時的に到達不能にすると、`TestResolveFeatures_REQ_SYSTEM_016` が「明示的に無効化した依存を必要とするのに成功した」と失敗した。変異を復元後、対象パッケージは通過した。"
    e2e_fault_injection: "`FEATURES_ENABLE` の読取結果を一時的に空へ置換すると、`TestLoadFeatureConfig_REQ_SYSTEM_016` が有効機能 ID の不一致で失敗した。変異を復元後、対象パッケージは通過した。"
  - id: publish-resolved-feature-metadata
    unit_red: "TestFeatureResolutionMetadata_REQ_SYSTEM_017 は実行時メタデータ型と `FeatureResolution.Metadata` が未定義のためコンパイルに失敗した（0 pass、build failed）。"
    e2e_red: "TestHealthReportsFeatureMetadata_REQ_SYSTEM_017 は `HealthInfo.Features` と HTTP 用機能メタデータ型が未定義のためコンパイルに失敗した（0 pass、build failed）。"
    unit_fault_injection: "解決済みメタデータから `UpdatePolicy` を一時的に欠落させると、`TestFeatureResolutionMetadata_REQ_SYSTEM_017` が空の更新方針と期待値の差を検出した。変異を復元後、対象パッケージは通過した。"
    e2e_fault_injection: "`/health` 応答から `features` を一時的に除くと、`TestHealthReportsFeatureMetadata_REQ_SYSTEM_017` が空のメタデータと期待値の差を検出した。変異を復元後、対象パッケージは通過した。"
- **Change-Resistance Results**:
  中リスク変更として、依存拒否条件、起動設定アダプター、更新方針メタデータ、HTTP 応答配線を個別に壊す 4 件の変異を与えた。主要ユースケースごとの単体テストと E2E テストがそれぞれ想定した誤実装を検出し、全変異の復元後に対象パッケージと標準検証が通過した。
- **Verification Results**:
  - `mise run verify` - passed (exit 0)
  - `mise run test-go-package ./backend/cmd/internal/bootstrap` - passed
  - `mise run test-go-package ./backend/shared/http/server_http` - passed
  - `mise run lint-go` - passed（0 issues）
  - `mise run check-spec` - passed
  - `mise run check-api-compat` - passed
  - `mise run check-boundaries` - passed
  - `mise run check-config-reference` - passed
  - `mise run check-contract-drift` - passed
  - `mise run check-status-drift` - passed
  - `mise run check-work-items` - passed
  - `mise run check-ids` - passed
  - `mise run spec-diff` - `REQ-SYSTEM-016` と `REQ-SYSTEM-017` を変更し、4 件の TypeSpec 宣言を追加
