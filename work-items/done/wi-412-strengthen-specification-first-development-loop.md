---
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-23
priority: p1
depends_on: [wi-410-adopt-risk-based-agentic-discipline]
change_kind: tooling
evidence_policy: risk-based-v2
approval:
  by: tn
  at: 2026-08-23
  scope: "仕様工程とスキルの同期、Acceptance REDとUnit REDの分離、型と効果の設計、GREEN後のRefactorを既存の文書体系と検査へ組み込む。"
  baseline: 230305245ba1f463c873c5eaebb67999a731a739
initial_context:
  specification: [DEVELOPMENT.md, WORK_ITEM_FORMAT.md]
  typespec: []
  source: [.agents/skills/new-work-item/SKILL.md, .agents/skills/spec-change/SKILL.md, .agents/skills/update-design/SKILL.md, .agents/skills/implement-work-item/SKILL.md, tools/check/src/main.ts, tools/check/schemas/work-item.schema.json, mise.toml]
  tests: [tools/check/src/lib.test.ts, tools/check/src/work-item-markdown.test.ts, tools/check/src/mise-config.test.ts]
  stop_before_reading: [backend, frontend, spec, docs/contexts]
spec_impact: { kind: none, reason: "製品の外部契約と実行時の振る舞いを変えず、開発方法の正本、エージェント向けスキル、作業項目の証拠契約と検査を更新する。" }
---

# 仕様先行開発に受け入れRED、型と効果の設計、リファクタリングを組み込む

## Motivation

`DEVELOPMENT.md` の影響元と現在の工程を比較すると、DDD、Ports and Adapters、Modular Monolith、Agentic Discipline は文書、配置、検査へ具体化されている一方、Specification-Driven Development、ATDD、BDD、Functional Design、Type-First Development、TDD、XP の一部は参考文献の説明にとどまっている。

実際、仕様変更を担う `spec-change` と `update-design` は廃止済みの `spec/**/SPECIFICATION.md` を参照し、作業項目の証拠契約は受け入れ試験と単体試験のREDを区別せず、実装手順はGREEN後のリファクタリングを独立した工程として扱わない。

開発方法の正本とエージェントが実行する手順を同期し、外部から観察できる振る舞いと内部の論理を別々に失敗させ、型と効果を設計してから最小実装とリファクタリングへ進む循環に改める必要がある。

## Scope

- `DEVELOPMENT.md` に、実装を左右する未決事項の解消、Acceptance RED、Unit RED、GREEN、Refactor の循環を定める。
- `DEVELOPMENT.md` と `WORK_ITEM_FORMAT.md` に、変更する中核論理のデータ型と操作シグネチャ、暗黙の入出力、計算とアクションの分離を設計時に確認する規則を追加する。
- `risk-based-v2` を追加し、今後着手する項目と `wi-412` 以降の完了項目では受け入れREDと単体REDの証拠を別々に要求する。既存の `risk-based-v1` 完了記録は変更しない。
- `new-work-item`、`spec-change`、`update-design`、`implement-work-item` を現在の文書体系と新しい開発循環へ同期する。
- リポジトリ固有スキルが現在の正本配置と証拠方針を参照していることを検査する `mise` タスクを追加し、標準検証へ組み込む。
- Markdown解析、JSON Schema、開発フロー検査をテスト先行で変更する。

## Out of Scope

- TypeSpec、規範シナリオ、API、製品コードの振る舞い変更。
- 規範シナリオとは別のGherkin正本またはOpenSpec CLIの導入。
- 既存コード全体の純粋関数化、識別子型の一括置換、不変データへの一括移行。
- すべての変更へのE2E試験、変異試験、複雑度上限の一律な強制。
- 完了済みの `risk-based-v1` 作業項目の書き換え。

## Design

### Evidence policy

`risk-based-v2` は番号にかかわらず今後 `in_progress` へ移る作業項目と、`wi-412` 以降に完了する作業項目へ適用する。

製品の振る舞いを変える作業では、Acceptance RED が規範シナリオを外部境界から検証し、Unit RED が変更する中核論理を狭い境界で検証する。

どちらかが適用不能な文書、道具、リファクタリングでは、対象の欄に理由と、代わりに実際に失敗させた最小検査を記録する。

既存の `risk-based-v1` は履歴としてスキーマに残し、意味を遡及変更しない。

### Design loop

中規模以上の変更は、実装内容を変える未決事項を承認または実装より前に解消する。

変更する中核論理について、作業項目の `Design` にデータ型、主要な操作シグネチャ、時刻、乱数、識別子生成、設定、永続化、通知などの効果境界を記録する。

実装は Acceptance RED、Unit RED、最小のGREEN、GREENを保つRefactorの順に進め、次の層または振る舞いへ移る。

### Guidance consistency

開発方法の正本は `DEVELOPMENT.md`、`SPECIFICATION_FORMAT.md`、`WORK_ITEM_FORMAT.md` であり、スキルはその手順を実行する薄い指示として保つ。

検査は、`spec-change` と `update-design` が現在の `docs/` と `spec/**/*.tsp` を指し、`implement-work-item` が最新の証拠方針と四段階の循環を要求することを確認する。

一般的なMarkdownリンク検査や自然言語全体の同値性判定は本作業項目へ含めない。

### Types and effects

検査の入力は `GuidanceDocument { file, source }`、出力は `AgentGuidanceFinding { file, message }` とし、`verifyAgentGuidance(documents): AgentGuidanceFinding[]` をファイルシステムに依存しない決定的な計算にする。

対象スキルの一覧は検査論理から `agentGuidanceFiles` として公開し、CLI側に重複させない。ファイル読み込み、標準出力と標準エラー、終了コードは `check-agent-guidance.ts` の効果境界に閉じ込める。

作業項目スキーマは `evidence_policy` を `risk-based-v1 | risk-based-v2` とし、後者の完了証拠を `acceptance_red_evidence` と `unit_red_evidence` の別オブジェクトで表す。Markdown解析は見出しラベルをこの二つのキーへ写像し、既存の `red_evidence` 経路を履歴互換として残す。

### Open questions

実装内容を変える未決事項はない。

受け入れ境界が存在しない道具変更にも二つの証拠欄を要求するが、適用不能理由と代替REDを許可することで、製品変更と同じ試験形態を強制しない。

## Plan

最初に `risk-based-v2` の構造化証拠とスキル同期検査の失敗試験を追加する。

次に開発方法の正本を更新し、Markdown解析、スキーマ、スキル、`mise` タスクを順に同期する。

最後に代表的な古いスキル内容と片方のRED証拠を欠く作業項目が拒否されることを確認し、独立レビューと全体検証を通す。

## Tasks

- [x] T001 [Spec] 開発方法の正本に未決事項の解消、二系統のRED、型と効果の設計、GREEN後のRefactorを定める。
- [x] T002 [Acceptance] `mise run check-work-items` で `risk-based-v2` が旧enumに拒否されるAcceptance REDを確認した。
- [x] T003 [App] `mise run test-tools` でスキル検査モジュール不在、v2スキーマ、二系統の証拠、Markdown解析のUnit REDを確認し、GREEN後に検査対象一覧とファイルI/Oを分離した。
- [x] T004 [Guidance] リポジトリ固有スキルを現在の正本配置、新しい循環、適用不能時の代替REDへ同期した。
- [x] T005 [Verify] 変更耐性確認、独立レビュー、局所検証、全体検証を完了した。

## Verification

- `mise run test-tools`
- `mise run typecheck-tools`
- `mise run check-work-items`
- `mise run check-ids`
- `mise run check-agent-guidance`
- `mise run check-command-map`
- `mise run verify`

## Risk Notes

開発工程全体へ適用される変更なのでリスクは medium とする。

証拠欄を増やすだけでは試験品質は上がらず、低リスクの文書変更へ過剰な試験を要求すると形骸化するため、二系統の責任を明確にしつつ適用不能理由と実際に失敗した代替検査を許可する。

スキル内容の検査を特定の全文一致にすると正当な文章改善まで拒否するため、現在の正本と工程を表す少数の安定した標識だけを検査する。

## Completion

- **Completed At**: 2026-08-23
- **Summary**:
  開発方法の正本とリポジトリ固有スキルを、未決事項の解消、Acceptance RED、Unit RED、GREEN、Refactor、型と効果の設計へ同期し、`risk-based-v2` とガイダンス検査で今後のドリフトを防ぐようにした。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-work-items`
  - **Requirement**: N/A: 製品の規範要件を変えない開発用道具の変更である
  - **Observed Failure**: `wi-412` が指定した `risk-based-v2` を旧スキーマが許可せず、`evidence_policy` のenum違反として拒否した。
  - **Detection Reason**: 標準の作業項目検査が新しい証拠契約を利用者から見える境界で受理できないことを直接検出した。
- **Unit RED Evidence**:
  - **Test**: `mise run test-tools`
  - **Requirement**: N/A: 製品の規範要件を変えない開発用道具の変更である
  - **Observed Failure**: ガイダンス検査モジュールが存在せず、v2スキーマ、二系統のRED証拠、Markdown解析を要求する試験が失敗した。
  - **Detection Reason**: スキーマ、解析、純粋なガイダンス検査を別々に失敗させ、各実装経路の欠落を識別した。
- **Post-Approval Changes**:
  `mise run spec-diff 230305245ba1f463c873c5eaebb67999a731a739` は規範仕様の変更なしと報告した。承認範囲の変更はない。
- **Independent Verification**:
  実装に関与していない fresh-context agent `wi412_independent_review` が規約軸と仕様軸をレビューした。初回所見を修正後に二回再レビューし、最終結果は Standards 0件、Spec 0件だった。
- **Change-Resistance Results**:
  廃止済みの `SPECIFICATION.md` と `risk-based-v1`、開発循環の逆順を持つガイダンスfixtureを拒否した。AcceptanceまたはUnitの片方を欠くv2完了記録と、要件IDの後ろに余分な文字を持つ両証拠fixtureも拒否した。
- **Verification Results**:
  - `mise run test-tools` - 194件成功
  - `mise run typecheck-tools` - 成功
  - `mise run check-work-items` - 411件成功
  - `mise run check-ids` - 411件成功
  - `mise run check-agent-guidance` - 成功
  - `mise run check-command-map` - 成功
  - `mise run spec-diff 230305245ba1f463c873c5eaebb67999a731a739` - 規範仕様の変更なし
  - `mise run verify` - 成功
