---
depends_on: []
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-08-30
priority: p0
change_kind: feature
affected_spec:
  - { path: docs/contexts/system/scenarios.md, requirement: REQ-SYSTEM-016 }
  - { path: docs/contexts/system/scenarios.md, requirement: REQ-SYSTEM-017 }
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

`FeatureDefinition` は `ID`、`Version`、`Maturity`、`DefaultEnablement`、`Dependencies`、`UpdatePolicy` を持つ不変値とする。登録は composition root で静的に行い、設定ファイルやデータベースから型や依存関係を追加しない。成熟度は `experimental`、`preview`、`supported`、`deprecated` とし、実験的機能は既定無効、非推奨機能は新規の既定有効化を禁止する。

機能の依存は有向グラフとして解決するが、一般的な依存注入器にはしない。`ResolveFeatures` は有効化された機能の依存閉包を求め、明示的に無効化された依存が必要なら設定エラーにする。エラーは全件を集約し、`REQ-SYSTEM-016` と同じく副作用のある初期化前に返す。

機能一覧の生成物は人が編集する正本にしない。正準文書と標準対応表は製品の約束を所有し、registry は実行時選択と更新影響を所有する。両者の対応を必要とする機能だけ、安定した規範 ID または標準 ID を任意の `SpecificationReference` として持つ。

## Plan

1. `REQ-SYSTEM-016` と `REQ-SYSTEM-017`、機能の成熟度を所有する正準文書を先に更新する。
2. domain 型、registry 検証、機能解決を Unit RED から実装する。
3. 起動時 Config と composition root へ接続し、不正な組合せが副作用前に失敗する E2E を追加する。
4. 実行時メタデータと設定リファレンスを同じ registry から生成する。
5. 既存機能から、実験的または更新方針を明示する必要がある最小集合だけを登録する。

## Tasks

- [ ] T001 [Spec] 機能成熟度、既定有効化、依存、更新方針と起動時拒否を仕様化する。
- [ ] T002 [Domain] `FeatureDefinition` と `ResolveFeatures` を、重複、循環、依存欠落、明示無効の Unit RED から実装する。
- [ ] T003 [Config] 機能選択を起動時の集約検証へ接続し、副作用前に全エラーを返す。
- [ ] T004 [Metadata] 有効機能、成熟度、版、更新方針を配備前検査が読める版付きメタデータへ生成する。
- [ ] T005 [Docs] 既存の設定リファレンスへ機能の成熟度と更新影響を重複なく反映する。
- [ ] T006 [E2E] 正式な起動経路で既定値、明示有効化、依存拒否、警告を検査する。
- [ ] T007 [Verify] 主要ユースケースの Unit/E2E RED と故障注入を記録し、標準検証を通す。

## Verification

- `mise run check-spec`
- `mise run test-go-package ./backend/cmd/internal/bootstrap`
- `mise run check-config-reference`
- `mise run verify`
- 実験的機能の暗黙有効化、存在しない依存、循環、依存の明示無効化を一つずつ与えると起動前に失敗する。
- registry から機能を削除または更新方針を変えると、生成メタデータと参照の乖離検査が失敗する。

## Risk Notes

リスクは medium。registry が製品機能の新しい総覧へ膨張すると、標準対応表、API 契約、テナント設定の責任を奪い、変更のたびに中央ファイルを編集する構造になる。実行時に選択できる機能と更新影響だけを登録し、静的に常時提供する機能は列挙しない。
