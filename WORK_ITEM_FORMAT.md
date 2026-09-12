# 作業項目フォーマット

作業項目は、一つの意味上の変更を説明、設計、実装、検証する作業単位である。
未完了の項目は `work-items/`、完了または中止した項目は `work-items/done/` に置く。
ファイル名には `wi-<連番>-<ケバブケースの題名>.md` を使う。

作業項目は、タスクリスト、変更固有の設計文書、実装履歴も兼ねる。
完了時点でも有効な結論は、TypeSpec またはその種類の内容を所有する正準文書へ反映しなければならない。

```markdown
---
status: pending
authors: [name]
risk: low
reversibility: reversible # 任意。このテンプレートを写さず、変更ごとに判断する
created_at: 2026-01-01
priority: p1
depends_on: []
change_kind: feature
evidence_policy: risk-based-v3 # 着手後は必須
documentation_impact: # 着手後は必須
  level: release_note
  reason: 新たにサポートする機能をリリースの読者へ知らせる必要がある。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-999-start-task.md }
initial_context: # 起票時ではなく着手時に記入する
  specification: [docs/contexts/system/scenarios.feature.md#REQ-SYSTEM-001]
  typespec: [Product.System.Operations.StartTask]
  source: [backend/system]
  tests: [backend/system]
  stop_before_reading: [frontend]
affected_spec:
  - { path: docs/contexts/system/scenarios.feature.md, requirement: REQ-SYSTEM-001 }
  - { path: spec/contexts/system/main.tsp, symbol: Product.System.Operations.StartTask }
primary_use_cases: # feature、bugfix、standards.md の変更では着手後に必須
  - id: start-task
    requirement: REQ-SYSTEM-001
    observable_result: 呼び出し元がタスクの実行開始を観測できる。
    unit_test: { path: backend/system/usecases/start_task_test.go, name: TestStartTask_REQ_SYSTEM_001, task: test-go-race }
    e2e_test: { path: backend/system/e2e_test.go, name: TestE2E_StartTask_REQ_SYSTEM_001, task: test-go-race }
    unit_fault_model: ユースケースが開始コマンドを発行しない。
    e2e_fault_model: 構成済みの経路がハンドラーとユースケースを接続しない。
maturity_evidence: # 成熟度の昇格を検出した場合は完了時に必須
  - feature: start-task-v1
    from: preview
    to: supported
    security: セキュリティレビューで、対象ユースケースに未解決の統制不足がないことを確認した。
    compatibility: 既存の preview 設定は移行せずに引き続き受理される。
    documentation: docs/releases/changes/wi-999-start-task.md
---

# 意味上の変更を表す一文

## 動機
変更が必要な理由を書く。

## 対象範囲
- 変更に含める仕様と実装を書く。

## 対象外
- 変更から明示的に除外する作業を書く。

## 設計
採用する設計、考慮事項、採用しない代替案を書く。

## 計画
実装順序、移行、未解決の問いを書く。
作るものを変え得る問いは、実装前にすべて解決する。

## タスク
- [ ] T001 [Spec] 仕様を更新する。
- [ ] T002 [Acceptance] 観測可能な境界で受け入れ RED を確認する。
- [ ] T003 [App] 単体 RED を確認し、GREEN にしてからリファクタリングする。
- [ ] T004 [Verify] 変更を検証する。

## 検証
- `mise run verify`

## リスク
リスクと緩和策を書く。
```

`priority`（`p0`〜`p3`）と `depends_on` は別の問いに答える。
`depends_on` は先に完了すべき項目を示し、機械検査によって作業順を制約する。
`priority` は、依存関係に妨げられていない項目のうち何を先に扱うべきかを示す参考値である。
未設定なら順位を付けていないことを表す。

`risk` と `reversibility` も別の問いに答える。
`risk` は変更を誤った場合の被害を、`reversibility` は判断を後から取り消せるかを表す。
両者は独立している。
レプリカ構成、キャッシュ方針、画面配置は影響が大きくても元に戻せる場合がある。
一方、通信フォーマット、識別子の意味、公開済みスキーマ、破棄した鍵、割り当てた `REQ` 番号は、小さな変更に見えても取り消せない。
判断を戻すためにリポジトリ外の利用者が保存済み、送信済み、または信頼済みのものを変える必要がある場合は `irreversible` とする。

`reversibility` 自体は必要な証拠を選ばない。
後から読む人が、まだ選び直せる判断と、今後も維持すべき判断を区別できるように記録する。
`reversible` としても `risk` が求める証拠は緩和しない。
このフィールドは導入前の記録を有効に保つため任意であり、未記入は可逆を意味せず、評価していないことを意味する。

項目を `in_progress` にするときは `evidence_policy: risk-based-v3` を追加する。
リスクは完了までに必要な証拠を決めるが、push、merge、本番操作、外部システムの変更を許可するものではない。
作業の権限は項目の起票によって与えられるため、別の承認記録は持たない。
プロダクトの振る舞い、公開契約、採用する設計境界、タスク分割を変え得る問いは実装前に解決する。
実装中に規範の変更が必要だと分かった場合は仕様作業へ戻り、実装を通すためにシナリオを弱めてはならない。
リスクと証拠の対応は[仕様先行の開発ワークフロー](docs/development/specification-first-workflow.md#4-証拠の要件)が定める。

`affected_spec` is required for `feature`, `bugfix`, and `operations` items. It directly references a
normative scenario/standard ID or a TypeSpec symbol. Changes with no specification impact (`refactor`,
`docs`, `tooling`, or `maintenance`) may use:

```yaml
spec_impact: { kind: none, reason: "A concrete reason." }
```

`initial_context` is the reading list one agent starts from. Write it when the item moves to
`in_progress`, not when it is filed: a list written for a backlog item rots before the work begins, and a
reading list that points at moved or deleted files is worse than none. A pending item needs only
Motivation, Scope, and Out of Scope to be useful.

Once the item is `in_progress`, `mise run check-work-items` resolves that list: every path must exist, and a
`docs/contexts/<context>/scenarios.feature.md#REQ-<CONTEXT>-NNN` entry must name a scenario the document declares.

`affected_spec` is resolved for every record, completed ones included, because it indexes the normative
element the change touched rather than what someone read at the time. When a normative element moves to a
different file, repoint those references; when it is retired, the retired heading keeps them resolving.

For medium and larger changes, make `Design` and `Plan` concrete. For changed core logic, name the principal
domain data types and operation signatures, and identify time, randomness, identifier generation,
configuration, persistence, notification, and other effects at the boundary where they enter or leave the
calculation. Domain, Use Cases, and Adapters tasks must retain the corresponding tests and normative scenario
ID as self-evidence.

Every item that enters `in_progress` declares one `documentation_impact` and keeps the same structured field
through completion. `level` is one of `none`, `release_note`, `upgrade_note`, `deprecation_notice`, or
`removal_notice`. `none` requires a concrete reason and no release-document references. Every other level
declares the planned release-document paths before implementation; completion requires those paths to exist,
name the work item, and link to an `affected_spec` requirement or TypeSpec symbol. A release note lives at
`docs/releases/changes/wi-<id>.md`; an upgrade note lives at
`docs/releases/upgrades/wi-<id>.md`. `upgrade_note`, `deprecation_notice`, and `removal_notice` require both
kinds because a reader needs the noteworthy delta and the action or compatibility information. The checker
derives a minimum from the change kind, normative specification diff, TypeSpec deprecations, and feature-
registry maturity diff. An author may select a stronger level, never a weaker one.

When the feature-registry diff promotes `experimental` to `preview` or `preview` to `supported`, completion
also records one `maturity_evidence` entry for each promoted feature. The entry names the exact transition,
the security-check result, either compatibility or migration information, and the release-document path that
shows the new maturity. The applicable item still declares `primary_use_cases`; maturity evidence does not
replace its Unit RED, E2E RED, or fault-injection results. Completed records written before this contract are
history and are not reinterpreted.

For `feature`, `bugfix`, and any item whose `affected_spec` references a requirement in a `standards.md`, add
`primary_use_cases` before implementation. Each entry declares one central successful route with a stable
kebab-case `id`, its exact `REQ-*` or standards requirement, the final `observable_result`, Unit and E2E test
references, and the distinct plausible fault each test must detect. A test reference contains the repository-
relative `path`, stable `name`, and required `mise` or CI `task`. The test may be absent while the item is
`in_progress`; at completion the checker requires the file and identifier, the requirement in the test source,
and reachability through the declared standard task. Do not use input acceptance, enum validation, line
coverage, or a directly constructed lower-level component as an E2E result when production uses a wider entry
and composition path.

`risk-based-v3` applies this contract to newly started work. Completed `risk-based-v1` and `risk-based-v2`
records remain valid history. An applicable item that was already `in_progress` at adoption moves to v3 and
adds the plan; no completed record is rewritten.

When the work is complete, set `status` to `completed`, append the following section, and move the file
to `work-items/done/`:

```markdown
## 完了
- **Completed At**: 2026-01-01
- **Summary**:
  The semantic difference introduced by the work, read from `mise run spec-diff` rather than recalled.
- **Acceptance RED Evidence**:
  - **Test**: The failing test or check observed before implementation.
  - **Requirement**: `REQ-CONTEXT-NNN`, or `N/A: <reason>` for work with no normative product requirement.
  - **Observed Failure**: The expected failure that was actually observed.
  - **Detection Reason**: Why the assertion distinguishes a plausible wrong implementation from the required
    behavior. When an acceptance boundary is inapplicable, identify the alternate check that actually failed.
- **Unit RED Evidence**:
  - **Test**: The failing test or check observed before implementation.
  - **Requirement**: `REQ-CONTEXT-NNN`, or `N/A: <reason>` for work with no normative product requirement.
  - **Observed Failure**: The expected failure that was actually observed.
  - **Detection Reason**: Why the assertion distinguishes a plausible wrong implementation from the required
    inner behavior. When a unit boundary is inapplicable, identify the alternate check that actually failed.
- **Change-Resistance Results**:
  For medium risk and above, record the representative incorrect implementation, diff mutation, or explicit
  fault injection and whether the tests detected it. For high and critical pure-logic changes, one
  representative is not enough: mutate the changed logic systematically or inject explicit faults across it,
  and record the equivalent mutations and the limits of the method rather than hiding them. A mutation tool
  produces the systematic half — it rewrites the tokens that are already there — and the mutations that add,
  remove, or redirect behavior stay hand-written; the split, and how to read the survivors without scoring
  them, is in
  [docs/development/specification-first-workflow.md](docs/development/specification-first-workflow.md).
- **Verification Results**:
  - `mise run verify` - passed
```

The Acceptance and Unit RED fields above are the completion shape for work without a primary-use-case
requirement. Applicable `risk-based-v3` items use the following field instead; they may retain additional
Acceptance or Unit evidence for non-primary behavior, but it does not replace this evidence:

```markdown
- **Primary Use Case Evidence**:
  - id: start-task
    unit_red: TestStartTask_REQ_SYSTEM_001 failed because no start command was emitted.
    e2e_red: TestE2E_StartTask_REQ_SYSTEM_001 failed because the configured route produced no running task.
    unit_fault_injection: Removing command emission made TestStartTask_REQ_SYSTEM_001 fail.
    e2e_fault_injection: Disconnecting the route made TestE2E_StartTask_REQ_SYSTEM_001 fail.
```

The `id` must match a `primary_use_cases` plan entry. Every planned entry needs exactly one completion entry;
each RED and fault-injection result is a non-empty observation, not a future instruction.
