---
context: seeding
updated_at: 2026-08-11
---

# Seeding Specification

## Overview

環境別の seed profile、投入計画、dry-run、適用の安全方針を所有する運用 bounded context。
各業務データの意味と永続化は各 record context に残し、Seeding は published command
surface を通じて依存順の実行を調整する。profile は明示選択であり、production では
bootstrap 以外を fail-closed で拒否する。

The `Seeding` context is an operations bounded context owning `SeedProfile`, `SeedRequest`,
`SeedPlan`, environment policy, drift policy, and application order. It does not own the meaning,
validation, or persistence of seeded resources — those stay in each record context (IdManagement,
Authentication, OAuth2, Application, Saml, WsFederation), reached through their existing idempotent
command surfaces. This split keeps environment safety and application order centralized while
avoiding duplicated invariant checks; the rejected alternative of scattering profiles across record
contexts loses that single point of cross-context safety verification.

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| SeedProfile | seed の内容と生成規則を表す明示選択の profile。bootstrap は稼働に必要な最小データだけ、development/test は既知のサンプル、performance は非機密の合成データを表す。環境名から暗黙に選ばない。 | seed profile |
| SeedPlan | 現在状態と profile manifest を比較して作る、redacted な変更計画。dry-run と apply は同じ計画規則を使う。 | seed plan |
| SeedDrift | seed 管理対象の logical key に対する現在値が manifest の canonical value と異なる状態。既定では手動変更を上書きせず conflict として停止する。 | drift |
| BootstrapSeed | first-party client など、サービス稼働に必要な最小データ。デモ資格情報やサンプル tenant data を含まない。 |  |
| SeedOperator | 明示した profile を plan または apply するローカル運用者または自動化主体。 |  |
| SeedManifest | seed resource と決定的 generator を宣言する versioned YAML desired state。DB fixture ではなく各 record context の公開 command surface への入力となる。 | seed manifest |
| SeedSecretReference | manifest が秘密値そのものの代わりに保持する provider、locator、version の組。解決値は plan、log、error に現れない。 | secret reference |

## Design

### Internal Interfaces

#### SeedData
seed を同一の決定的 planner で計画し適用する内部運用 interface。apply は各 record context の published command surface だけを呼び、直接 SQL fixture で不変条件を迂回しない。
- Input invariant: manifest_schema_supported(input.request)
- Input invariant: manifest_profile_matches_request(input.request)
- Input invariant: manifest_paths_are_local_and_contained(input.request)
- Input invariant: input.request.environment in ['staging', 'production'] implies manifest_secret_providers(input.request) == ['file']
- Result invariant: input.request.mode == 'dry_run' implies persistent_state_unchanged()
- Result invariant: reapply_same_request_is_noop(input.request)
- Result invariant: input.request.environment == 'production' && input.request.profile == 'bootstrap' implies production_safe_redirect_uris(input.request.first_party_redirect_uris)
- Result invariant: seed_plan_and_diagnostics_exclude_secret_values(output.plan)

### Environment policy and planning

A profile is never inferred from environment name — it must be given explicitly by request/CLI.
Production accepts only the `bootstrap` profile; `demo`/`test`/`performance` are rejected
fail-closed before any write, so a misrouted request cannot seed demo credentials into production.
Dry-run and apply share the same planner, and re-applying the same manifest/generator-seed/secret
version is a no-op — manual drift is a conflict by default, with explicit reconcile left as a
separate, later contract. Application uses bounded, dependency-ordered batches of idempotent
commands rather than one cross-context transaction; the `performance` profile's batch size defaults
to 250 and caps at 1,000. Rather than a dedicated checkpoint table, the same request replays
deterministically from logical keys/IDs derived from profile and generator seed, serialized by an
in-process mutex per request key and, across processes, a PostgreSQL advisory lock on the existing
connection.

### Seed manifests and secret references

`models.SeedManifest` is a versioned, strictly-decoded YAML desired state, converted by the
`manifests_yaml` adapter into domain types before reaching the existing per-resource contributors;
domain and usecases never see the parser or filesystem/env APIs directly. `include` resolves only
local relative paths under the manifest root, bounded by depth and total-count limits — YAML merge
keys, templating, remote URLs, and env-var expansion are excluded from the grammar to avoid path
traversal and injection surfaces. Secret values are never written into a manifest; they are
referenced through `models.SeedSecretReference`, whose `env` provider is available everywhere but
whose `file` provider is the only one permitted in staging/production. Dry-run validates that a
reference resolves without ever passing the materialized value into a plan, log, or error.

### Design Decisions

- `Seeding` is a separate operations context that owns environment policy, drift policy, and
  application order across record contexts through their existing idempotent command surfaces,
  rather than scattering seed profiles across each record context and losing the single point of
  cross-context safety verification
  ([ADR-118](../../../decisions/ADR-118-extract-environment-aware-seeding-context.md)).
- Seed manifests are versioned, strictly-decoded YAML with a restricted `include`/secret-reference
  grammar (no merge keys, templating, remote URLs, or literal `${ENV}` expansion), and re-applying
  the same manifest/generator-seed/secret version replays deterministically rather than relying on a
  dedicated checkpoint table
  ([ADR-132](../../../decisions/ADR-132-use-versioned-seed-manifests-and-secret-references.md)).

## Scenarios

### REQ-SEEDING-001: 環境別の明示profileが選択される
- ACTOR SeedOperator
- GIVEN SeedOperator が environment と profile を明示している
- WHEN SeedOperator が SeedData を dry_run で呼ぶ
- THEN planner は environment policy に許可された manifest だけを選ぶ
- THEN 応答は redacted な SeedPlan を返し永続状態を変更しない

### REQ-SEEDING-002: 明示manifestまたはprofile既定manifestが選択される
- ACTOR SeedOperator
- GIVEN SeedOperator が environment と profile を明示している
- WHEN SeedOperator が明示 manifest path を指定して SeedData を呼ぶ
  - ALT manifest path が未指定である → loader は profile ごとの repository default manifest を選ぶ
- THEN loader は指定 path の manifest と contained include を strict decode する
- THEN planner は manifest の typed desired resource を計画する

### REQ-SEEDING-003: manifestとrequestのprofile不一致は拒否される
- ACTOR SeedOperator
- GIVEN SeedOperator が request と異なる profile の manifest を指定している
- WHEN SeedOperator が SeedData を呼ぶ
- THEN SeedData は secret 解決と書き込みの前に SeedRejectedError で拒否する

### REQ-SEEDING-004: 不正manifestは書き込み前に拒否される
- ACTOR SeedOperator
- GIVEN manifest に未知 key、重複 logical key、未対応 schema version、include cycle、または root 外 path がある
- WHEN SeedOperator が SeedData を呼ぶ
- THEN loader は secret 解決と書き込みの前に SeedRejectedError で拒否する
- THEN 診断には秘密値を含めない

### REQ-SEEDING-005: productionではenv secret providerを拒否する
- ACTOR SeedOperator
- GIVEN environment が production である
- GIVEN manifest が env secret provider を参照している
- WHEN SeedOperator が SeedData を dry_run または apply で呼ぶ
- THEN SeedData は secret 解決と書き込みの前に SeedRejectedError で拒否する
- THEN 永続状態は変更されない

### REQ-SEEDING-006: 同一seedの再適用はno-opになる
- ACTOR SeedOperator
- GIVEN 同じ manifest、generator seed、secret version で seed が apply 済みである
- WHEN SeedOperator が同じ SeedRequest を再度 apply する
- THEN SeedPlan の全 operation は noop である
- THEN password history と created_at / updated_at は変更されない

### REQ-SEEDING-007: productionでdemoまたはperformance profileは拒否される
- ACTOR SeedOperator
- GIVEN environment が production である
- WHEN SeedOperator が development または performance profile を指定して SeedData を呼ぶ
- THEN SeedData は書き込み前に SeedRejectedError で拒否する
- THEN 既知のデモ資格情報は作成されない

### REQ-SEEDING-008: production bootstrapには明示redirect URIが必要である
- ACTOR SeedOperator
- GIVEN environment が production である
- GIVEN profile が bootstrap である
- WHEN SeedOperator が first_party_redirect_uris を指定して SeedData を apply する
  - ALT redirect URI が未指定、localhost、または http URI である → SeedData は書き込み前に SeedRejectedError で拒否する
- THEN first-party client は指定 URI だけを redirect URI として持つ

### REQ-SEEDING-009: manual driftは上書きせずconflictになる
- ACTOR SeedOperator
- GIVEN seed 管理対象の logical key が手動変更されている
- WHEN SeedOperator が対応する profile を apply する
- THEN SeedData は SeedConflictError を返す
- THEN 手動変更は維持される

### REQ-SEEDING-010: 部分失敗後に同じrequestを再実行すると収束する
- ACTOR SeedOperator
- GIVEN SeedData の apply が一部の operation を完了した後に失敗している
- WHEN SeedOperator が同じ SeedRequest を再度 apply する
- THEN 完了済み logical key は no-op と判定される
- THEN 未完了の logical key だけが適用され、重複なく目的状態へ収束する
