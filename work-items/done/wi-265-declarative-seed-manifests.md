---
status: completed
authors: [tn]
risk: high
created_at: 2026-07-20
depends_on: [wi-236-environment-aware-idempotent-seeding]
change_kind: feature
completion:
  completed_at: 2026-07-20
  summary: "bootstrap/development/test/performance の seed desired state を versioned YAML manifest へ移し、strict include loader、env/file secret reference、CLI/startup path 選択、production file-only policyを導入した。"
  verification:
    - "just yaml-check"
    - "just test-go-fuzz ./backend/seeding/adapters/manifest 3s"
    - "just verify-go"
    - "just verify"
    - "4 profile default manifest dry-run"
  affected_guarantees_state:
    - "既存 dry-run、idempotent reapply、manual drift conflict、production bootstrap policy を維持する。"
    - "manifest と diagnostics は秘密値を保持せず、dry-run でも secret reference の解決可能性を検証する。"
    - "performance synthetic user は password を持たない Disabled user になる。"
  evidence:
    - id: "manifest-red-green"
      kind: "test"
      procedure: "Domain / UseCase / Adapter tests を先にコンパイル失敗させ、manifest validation、materialization、strict loader、secret resolver 実装後に green を確認"
      result: "passed"
    - id: "manifest-fuzz"
      kind: "test"
      procedure: "just test-go-fuzz ./backend/seeding/adapters/manifest 3s"
      result: "passed; 1854 executions, no panic"
    - id: "repository-verification"
      kind: "test"
      procedure: "just verify"
      result: "passed"
initial_context:
  scl: { Seeding: [models.SeedRequest, interfaces.SeedData, scenarios.環境別の明示profileが選択される] }
  source: [backend/seeding, backend/cmd/internal/bootstrap, backend/cmd/idmagic-seed]
  tests: [backend/seeding, backend/cmd/internal/bootstrap]
  stop_before_reading: [frontend]
affected_spec:
  - { context: Seeding, kind: model, element: SeedManifest }
  - { context: Seeding, kind: model, element: SeedSecretReference }
  - { context: Seeding, kind: model, element: SeedRequest }
  - { context: Seeding, kind: interface, element: SeedData }
---

# seed profile を YAML manifest と secret reference で宣言可能にする

## Motivation
現在の seed resource は Go の composition code に直接記述されているため、resource の追加・環境差分・
秘密値の注入方法を変更するたびに再ビルドが必要であり、運用設定とプログラムの責務も混在している。
一方で YAML に秘密値そのものを置く設計は repository、ログ、dry-run 出力からの漏えいを招く。
型付き manifest と fail-closed な secret reference を導入し、既存の dry-run、冪等性、drift、
production policy を維持したまま seed 定義を外部化する。

## Scope
- `spec/contexts/seeding.yaml` の `glossary`、`models.SeedManifest`、
  `models.SeedSecretReference`、`models.SeedRequest`、`interfaces.SeedData`、
  `scenarios`、`objectives.SeedScalability`
- `backend/seeding/domain` の manifest / secret reference の純粋な検証規則
- `backend/seeding/usecases` の manifest materialization と secret resolver port
- `backend/seeding/adapters/manifest` の strict YAML loader、include 解決、env/file resolver
- `backend/cmd/internal/bootstrap` の contributor を manifest 駆動へ移行
- `backend/cmd/idmagic-seed` と起動時 seed の manifest path 選択
- `seed/manifests` の bootstrap/development/test/performance 既定 manifest
- `ARCHITECTURE.md`、ADR、README、just recipe の同期

## Out of Scope
- Kubernetes Secret、Vault、AWS/GCP/Azure secret manager 固有 provider
- remote URL からの manifest/include 読み込み
- manifest に存在しない既存 DB resource の prune/delete
- Go plugin や任意テンプレートによる resource kind の動的拡張
- GitOps controller、manifest hot reload、実行履歴/checkpoint table

## Plan
- YAML は DB fixture ではなく、既存 record context の公開 command surface へ渡す型付き desired state とする。
- root manifest は明示 `--manifest` / `SEED_MANIFEST` を優先し、未指定時は profile ごとの
  repository default を使う。manifest の profile と request profile は一致を必須にする。
- strict decode、未知 key/重複 logical key/schema version mismatch/include cycle/root 外 path を
  書き込み前に拒否する。include はローカル相対 path のみに限定し、merge key、任意 template、
  environment substitution は受理しない。
- secret は `{provider, locator, version}` の参照だけを manifest に持つ。development/test では
  env/file、staging/production では file のみを許可する。file は regular file、64 KiB 以下、
  NUL なしとし、末尾の改行 1 個だけを除く。dry-run でも解決可能性を検証するが値は plan/log/error
  に出さない。
- performance user は generator 定義で決定的に作り、login-disabled として既知 password を持たせない。
- 既存 resource ID と logical key を manifest へ移し、同一 manifest の再適用 no-op と drift policy を保つ。

## Tasks
- [x] T001 [SCL/Decision] SCL、ADR-132、Architecture、派生成果物を先行更新する。
- [x] T002 [Domain] manifest と secret reference の検証を実装する。
      RED: domain validation tests を先に fail 確認
      (`models.SeedManifest` / `models.SeedSecretReference`) → GREEN。
- [x] T003 [UseCase] request と materialized manifest の整合、production provider policy を実装する。
      RED: usecase tests を先に fail 確認
      (`interfaces.SeedData`、scenario `manifestとrequestのprofile不一致は拒否される`) → GREEN。
- [x] T004 [Adapter] strict YAML/include loader と env/file secret resolver を実装する。
      RED: adapter tests を先に fail 確認
      (scenario `不正manifestは書き込み前に拒否される`、
      `productionではenv secret providerを拒否する`) → GREEN。
- [x] T005 [Infrastructure] contributor、CLI、startup、既定 manifest、README、just recipe を接続し、
      Go の hard-coded desired data を除去する。
- [x] T006 [Verify] 全 profile の dry-run、`just yaml-check`、`just verify-go`、`just verify` を通す。

## Verification
- `just yaml-check`
- `just test-go`
- `just verify-go`
- `just seed development bootstrap dry_run`
- `just seed development development dry_run`
- `just seed test test dry_run`
- `just seed development performance dry_run`
- `just verify`

## Risk Notes
YAML/include と secret locator は外部の未信頼入力であり、path traversal・循環・秘密漏えいが高リスク。
include の再帰構造には fuzz test を採用し、任意 bytes に対して panic せず root 外を読まないことを確認する。
secret file の固定規則は組み合わせ爆発を伴わないため property test は採用せず、table-driven 境界テストで
regular file、size、NUL、newline、production provider policy を確認する。
