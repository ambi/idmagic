---
status: completed
authors: [tn]
risk: low
created_at: 2026-08-23
priority: p2
depends_on: []
change_kind: tooling
initial_context:
  specification: [DEVELOPMENT.md, WORK_ITEM_FORMAT.md]
  typespec: []
  source: [.agents/skills/implement-work-item/SKILL.md]
  tests: []
  stop_before_reading: [backend, frontend, spec, docs/contexts]
spec_impact: { kind: none, reason: "Agentic Discipline と現行開発フローの適合性を評価し、開発方法の改善候補だけを記録する作業項目であり、製品の外部契約、振る舞い、保証は変更しない。" }
---

# Agentic Discipline を IdMagic の開発フローへ取り入れる価値を評価する

## Motivation

Robert C. Martin が公開している Agentic Discipline の実践と、その影響を受けた Disciplined Agentic Engineering を調べ、IdMagic の仕様先行開発に不足している規律があるかを判断する。

## Scope

- 指定された O'Reilly、`empire-2025`、SwarmForge、Disciplined Agentic Engineering の一次資料または公式説明を調べる。
- `DEVELOPMENT.md`、`WORK_ITEM_FORMAT.md`、`implement-work-item` と比較し、既存要素、欠けている要素、導入リスク、改善候補を分ける。
- 採用する原則と、採用しない実装形態を決める。

## Out of Scope

- `DEVELOPMENT.md`、作業項目形式、検査、スキルの実装変更。
- O'Reilly の購読者向け映像を未視聴のまま内容推定すること。
- 変異テスト、CRAP、Gherkin、SwarmForge を IdMagic に導入すること。

## Design

調査日：2026-08-23

### Decision

Agentic Discipline の中心にある「人間が振る舞いと設計上の制約を所有し、エージェントの成果を実行可能な証拠で制約する」という考えは、IdMagic に取り入れる価値がある。
ただし、IdMagic はすでに仕様先行、安定した要求識別子、作業項目、RED の確認、内側から外側への実装、段階的な検証、依存境界の機械検査を備えており、Agentic Discipline への全面移行は必要ない。
不足しているのは、承認済みの振る舞いを実装中の都合で変えない仕組み、実装者から独立した検証、テスト自体が欠陥を検出できることの確認、作業リスクに応じた人間の承認点である。
この四点を既存の TypeSpec、規範シナリオ、`REQ-<CONTEXT>-NNN`、`mise` ゲートへ足すのが適切であり、別系統の Gherkin 仕様、固定的な多エージェント編成、全変更への一律な変異テストは導入しないほうがよい。

### Evidence Boundary

O'Reilly のコースページから確認できたのは、全6回、5時間34分、中級、Robert C. Martin と Justin Martin による講座であり、エージェント開発の生産性を得ながらコード品質、構造、アーキテクチャを維持する規律、ワークフロー、道具と、複数エージェントの編成を扱うという公式概要までである。
各回の本文と映像は購読が必要であり、本調査では視聴していないため、以下の具体的な原則をコースで語られた内容として直接帰属させない（[O'Reilly コースページ](https://learning.oreilly.com/course/clean-ai-agentic/9780135968819/)）。

Robert C. Martin 自身の公開実践として、`empire-2025` はコミット [`62cb8b5`](https://github.com/unclebob/empire-2025/tree/62cb8b5e134d9802e933de381cd8efa41e94516b)、SwarmForge の共通部分はコミット [`ba50410`](https://github.com/unclebob/swarm-forge/tree/ba50410637ee36b97b314a19ca9a2ee716469ef8)、四役の実行可能な編成はコミット [`615a083`](https://github.com/unclebob/swarm-forge/tree/615a08365a861f936395315624e1e5ef3b312466) を調べた。
Disciplined Agentic Engineering（DAE）は Robert C. Martin のリポジトリではなく、README も `empire-2025` から受け継いだ ATDD と変異テストに加え、自律性レベル、独立検証、引き継ぎ契約、成果物オントロジーなどを独自に追加したと明記しているため、DAE の全項目を Robert C. Martin 本人の考えとしては扱わない（[DAE README、コミット `1adbf3c`](https://github.com/swingerman/disciplined-agentic-engineering/blob/1adbf3c7d83e27b9ccb13a40c93d4dcb23ab6dfc/README.md)）。

指定された資料は方法論の説明と実装例であり、対照実験や生産性の測定結果ではない。
したがって、「品質を保ったまま生産性が上がる」という効果は公式概要と各リポジトリの主張であって、本調査が実証した事実ではない。

### Provenance

| 実践 | Robert C. Martin の一次資料 | DAE で一般化または追加された内容 |
|---|---|---|
| 振る舞いを先に固定する | 人間が所有する受け入れシナリオ、失敗する受け入れ試験と単体試験の二系統 | 意図、受け入れ条件、Gherkin、設計計画を段階成果物として承認する |
| テスト品質を調べる | 変更モジュールの差分変異、CRAP、Gherkin 変異 | 差分リスク評価とアーキテクチャ適合性を同じ検証工程へ置く |
| 役割を分離する | SwarmForge の仕様、実装、整理、設計、堅牢化、品質保証 | 検証者と実装者が異なることを成果物オントロジーで検査する |
| 作業を再開可能にする | worktree と検証済み Git 引き継ぎ | 工程入口の引き継ぎゲート、進捗の現在位置、推奨する次工程を構造化する |
| 人間の権限を残す | 仕様引き継ぎを操作盤で承認する | 機能ごとの自律性レベル、機密経路の上書き、設計と外向き書き込みの常時承認 |

この表の右列は DAE の独自追加を含み、Robert C. Martin 本人の主張として帰属させない。

### Practices Confirmed in the Public Sources

#### Behavior Contract Before Implementation

`empire-2025` の `AGENTS.md` は、新規または変更される振る舞いごとに受け入れシナリオを書き、それが失敗することを確認してから、失敗する単体テストと実装へ進むよう要求している。
既存の受け入れテストを変えるには明示的な許可が必要であり、変換できないシナリオも望まれる振る舞いを示す失敗テストとして残す（[`empire-2025/AGENTS.md`](https://github.com/unclebob/empire-2025/blob/62cb8b5e134d9802e933de381cd8efa41e94516b/AGENTS.md)）。

同リポジトリの受け入れ試験フレームワークは、Given、When、Then 形式の文書を中間表現と実行可能な試験へ変換し、失敗を元のファイルと行へ結び付ける。
設計原則は内部状態ではなく利用者が観察する振る舞いを検証することであり、乱数などの非決定性は観察可能な結果を記述したうえで固定する（[Acceptance Test Framework](https://github.com/unclebob/empire-2025/blob/62cb8b5e134d9802e933de381cd8efa41e94516b/plans/permanent/acceptance-test-framework.md)）。

SwarmForge の四役編成では、仕様担当が利用者の意図を実装詳細へ漏らさず決定的な Gherkin にし、実装担当は承認済みの仕様から振る舞いの小片ごとに単体テストを先に失敗させ、受け入れ試験と単体試験の双方を通す。
実装担当が生成された受け入れ試験を単体試験の代用にすることも禁じている（[仕様担当](https://github.com/unclebob/swarm-forge/blob/615a08365a861f936395315624e1e5ef3b312466/swarmforge/roles/specifier.prompt)、[実装担当](https://github.com/unclebob/swarm-forge/blob/615a08365a861f936395315624e1e5ef3b312466/swarmforge/roles/coder.prompt)）。

#### Deterministic Guardrails

DAE は、長い作業で弱まり得るプロンプト上の規則より、非ゼロの終了コードを返す決定的なゲートを重視する。
前工程の完了した引き継ぎ、正しいブランチ、成果物間の対応、現在位置を各工程の入口で検査し、作業の完了条件と次の工程を構造化した引き継ぎへ記録する（[DAE README](https://github.com/swingerman/disciplined-agentic-engineering/blob/1adbf3c7d83e27b9ccb13a40c93d4dcb23ab6dfc/README.md)）。

SwarmForge も役割ごとの専用 worktree、検証済みの引き継ぎファイル、受信側が実行する定型ヘルパーを使い、エージェントが別のブランチや手書きの通知手順へ逸脱しにくくしている。
実行可能な編成は二役、四役、六役に分かれ、作業規模に応じて仕様、実装、整理、設計、堅牢化、品質保証の責任を分ける（[SwarmForge README](https://github.com/unclebob/swarm-forge/blob/ba50410637ee36b97b314a19ca9a2ee716469ef8/README.md)、[共通 Workflow Rules](https://github.com/unclebob/swarm-forge/blob/ba50410637ee36b97b314a19ca9a2ee716469ef8/swarmforge/constitution/articles/workflow.prompt)、[Handoff Rules](https://github.com/unclebob/swarm-forge/blob/ba50410637ee36b97b314a19ca9a2ee716469ef8/swarmforge/constitution/articles/handoffs.prompt)）。

#### Test Defect-Detection Ability

`empire-2025` は変更した各モジュールへ差分変異テストをかけ、未検出の変異を殺し、CRAP 指標を8以下にするよう求める（[`empire-2025/AGENTS.md`](https://github.com/unclebob/empire-2025/blob/62cb8b5e134d9802e933de381cd8efa41e94516b/AGENTS.md)）。
SwarmForge の四役編成はこれを分担し、整理担当が CRAP、重複、変異箇所数を調べ、設計担当が差分変異テストと Gherkin のソフト変異を順番に実行する（[整理担当](https://github.com/unclebob/swarm-forge/blob/615a08365a861f936395315624e1e5ef3b312466/swarmforge/roles/refactorer.prompt)、[設計担当](https://github.com/unclebob/swarm-forge/blob/615a08365a861f936395315624e1e5ef3b312466/swarmforge/roles/architect.prompt)）。

DAE はこの考えを、変更箇所だけを対象にした変異テスト、差分に対する変更リスク評価、アーキテクチャ適合性検査として一般化し、検証者が実装者と異なることも成果物間の制約として検査する（[DAE README](https://github.com/swingerman/disciplined-agentic-engineering/blob/1adbf3c7d83e27b9ccb13a40c93d4dcb23ab6dfc/README.md)）。

#### Autonomy and Human Decisions

DAE は機能ごとに低、中、高の自律性レベルを記録し、セキュリティや課金などの経路ではプロジェクト全体の上限より厳しくできるようにする。
設計案の確認と、既定ブランチへの反映、プルリクエスト、エージェント設定の自己変更、本番環境への書き込みは自律性レベルにかかわらず人間の許可を要求する（[DAE README](https://github.com/swingerman/disciplined-agentic-engineering/blob/1adbf3c7d83e27b9ccb13a40c93d4dcb23ab6dfc/README.md)）。

SwarmForge の四役編成でも仕様担当から実装担当への引き継ぎは操作盤上の人間の承認を通り、役割ごとに所有する判断と実行してはならない検査を分けている（[SwarmForge README](https://github.com/unclebob/swarm-forge/blob/ba50410637ee36b97b314a19ca9a2ee716469ef8/README.md)、[仕様担当](https://github.com/unclebob/swarm-forge/blob/615a08365a861f936395315624e1e5ef3b312466/swarmforge/roles/specifier.prompt)）。

### Current IdMagic Coverage

| Agentic Discipline の狙い | IdMagic の現状 | 評価 |
|---|---|---|
| 振る舞いを先に固定する | `DEVELOPMENT.md` は TypeSpec と `docs/` の規範シナリオを実装より先に変更し、安定した `REQ-<CONTEXT>-NNN` を与える | すでに強い |
| 仕様、実装、試験の対応を残す | 作業項目の `affected_spec` と `initial_context`、テスト名の要求識別子、`spec-where`、生成される追跡可能性ページが対応を作る | すでに強い |
| RED から始める | `WORK_ITEM_FORMAT.md` の標準タスクは RED の確認を含み、`implement-work-item` は振る舞いの層ごとに RED とテスト名、要求識別子を記録する | すでにあるが `DEVELOPMENT.md` のループ表からは見えにくい |
| 小さな検証から広げる | 層ごとの最小試験から最後の `mise run verify` へ広げる検証の段階がある | すでに強い |
| 規則を機械的に強制する | `check-work-items`、`check-spec`、`check-boundaries`、`check-api-compat`、`check-ids`、拒否制御の検査などを `mise` タスクとして持つ | すでに強い |
| 観察可能な効果を検証する | 拒否について応答だけでなく、起きなかった副作用を読み戻して検証する | Agentic Discipline の公開例より具体的な強み |
| 再開可能な作業記録 | 作業項目が動機、範囲、設計、計画、タスク、リスク、完了証拠を持ち、`initial_context` が読む範囲を限定する | ほぼある |
| 独立した検証 | 実装者とは別の検証者を要求する工程は `DEVELOPMENT.md` にない | 欠けている |
| テストの欠陥検出能力 | RED は確認するが、差分変異や意図的な欠陥注入を一般の完了条件にはしていない | 一部だけある |
| リスク別の人間承認 | 作業項目に `risk` はあるが、リスクと仕様承認、設計承認、自律実行の関係を `DEVELOPMENT.md` は定めていない | 欠けている |

IdMagic は Agentic Discipline の出発点より先に進んでいる部分が多い。
特に、Gherkin を新しい正本として追加すると、TypeSpec と種類別の規範文書が担う「一つの事実に一つの置き場所」という現行設計を壊す。

調査時点のツリーでは、`docs/**/scenarios.md` にある重複除去後の要求識別子 251 件に対し、`backend` と `frontend` の試験コードが明示的に名前を挙げる要求識別子は 51 件だった。
この差は残る 200 件が未試験であることを示さず、識別子を付けずに同じ振る舞いを検証している試験もあり得るが、一般の規範シナリオと実行可能な証拠の対応を機械的に確認できない範囲が大きいことは示している。
しかし、`wi-362-req-test-traceability-and-lookup` は要求識別子の注記ゲートが試験の存在や意味ではなく注記忘れだけを検査すると判断し、追跡ページと `spec-where` に限定したため、本調査もこの数だけを根拠に一般的な注記ゲートを再提案しない。

### Broader Influences and References

IdMagic の開発フローは、次の方法論のいずれかをそのまま実装したものではなく、各一次資料にある規律をリポジトリ固有の仕様体系、作業リスク、`mise` ゲートへ組み合わせたものである。
とりわけ、リスク別の承認点、承認後の仕様差分、独立検証を DDD、Clean Architecture、TDD、BDD、ATDD の原著へ帰属させず、Agentic Discipline と DAE の公開実践を参考に IdMagic が追加する運用上の決定として扱う。

| 方法論または実践 | 一次資料または公式説明 | IdMagic で参考にした具体的な規則 |
|---|---|---|
| Agentic Discipline | Robert C. Martin と Justin Martin の [Clean AI: Agentic Discipline](https://www.oreilly.com/videos/clean-ai-agentic/9780135968819/) 公式概要、Robert C. Martin の [`empire-2025/AGENTS.md`](https://github.com/unclebob/empire-2025/blob/62cb8b5e134d9802e933de381cd8efa41e94516b/AGENTS.md) と [SwarmForge](https://github.com/unclebob/swarm-forge/tree/ba50410637ee36b97b314a19ca9a2ee716469ef8) | 承認対象の振る舞いを実装より先に確定すること、受け入れ試験と単体試験の RED を別々に確認すること、変更した論理の試験が欠陥を検出できるかを確かめること、規模に応じて責任を分離することを、既存の要求識別子とゲートへ取り込む。 |
| Disciplined Agentic Engineering | Swingerman の [Disciplined Agentic Engineering](https://github.com/swingerman/disciplined-agentic-engineering/blob/1adbf3c7d83e27b9ccb13a40c93d4dcb23ab6dfc/README.md) | `risk` に応じた人間の承認点、実装者から独立した検証、構造化した完了証拠と再開可能な引き継ぎの参考にするが、これらを Robert C. Martin 本人の提案とは扱わない。 |
| Specification-Driven Development | GitHub Spec Kit の [Specification-Driven Development](https://github.com/github/spec-kit/blob/27f50f7e6b618ea14d74dd4037f9e7c60218b16c/spec-driven.md) | 仕様を主成果物として、要求と受け入れ基準から実装計画、タスク、試験、コードへ進む考えを、TypeSpec と規範文書を先に変更して作業項目から実装と検証を駆動する流れの参考にする。ただし、IdMagic はコード全体を仕様から再生成される成果物とはみなさず、Spec Kit の形式とコマンドも採用しない。 |
| OpenSpec | Fission-AI の [OpenSpec](https://github.com/Fission-AI/OpenSpec/blob/f1b521dffac38ed6638689cd28b0c204b1eef0f1/README.md) | 成果物に沿った `proposal`、`specs`、`design`、`tasks`、`apply`、`archive` は、作業項目で動機、仕様差分、設計、タスク、実装、完了を一つの変更として扱う流れと共通する。一方、IdMagic は OpenSpec の `openspec/changes` 形式、手書きの仕様差分、`openspec` CLI を採用せず、現在状態の TypeSpec と種類別文書、Git から導出する `spec-diff`、`mise` タスクを使う。 |
| Domain-Driven Design | Eric Evans の [Domain-Driven Design Reference](https://www.domainlanguage.com/wp-content/uploads/2016/05/DDD_Reference_2015-03.pdf) | 仕様と実装を境界づけられたコンテキスト単位に置き、その境界内では TypeSpec、規範シナリオ、コード、会話に共通するユビキタス言語を使う。 |
| Clean Architecture | Robert C. Martin の [The Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html) | 業務規則を UI、データベース、ウェブの詳細から隔離し、ドメインの振る舞いとユースケースから外側のアダプターへ実装しながら層ごとに検証する。依存方向の原則自体が実装順序を直接規定するとは解釈しない。 |
| Screaming Architecture | Robert C. Martin の [Screaming Architecture](https://blog.cleancoder.com/uncle-bob/2011/09/30/Screaming-Architecture.html) | 最上位構造がフレームワーク名ではなく対象領域とユースケースを表すという考えを、コンテキスト名と種類別の規範文書からシステムの意図をたどれる配置へ反映する。 |
| Modular Monolith | Simon Brown の [Modular monolith and package by component](https://simonbrown.je/modular-monolith/) | 単一のソースツリーと配置単位を維持しながら、コンテキストと機能を明示的なモジュール境界にし、公開面を小さく保ち、パスと依存検査で越境を拒否する現在の構造を支える。 |
| Microservice Architecture | James Lewis と Martin Fowler の [Microservices](https://martinfowler.com/articles/microservices.html) | 業務能力に沿った境界、疎結合、局所的な所有権と、分散による相互作用、整合性、運用上の費用をコンテキスト境界の設計判断へ反映する。これは境界設計への影響であり、IdMagic が現在コンテキストごとの独立配置を採用していることを意味しない。 |
| Functional Design | Eric Normand の [Grokking Simplicity](https://www.manning.com/books/grokking-simplicity) | 不変なデータ、決定的な計算、副作用のあるアクションを分ける実践を、純粋なドメインまたはユースケースの論理と外側のアダプターの分離へ反映する。Go と TypeScript の全体を純粋関数型にする規則とは扱わない。 |
| Type-First Development | Tomas Petricek の [Why type-first development matters](https://tomasp.net/blog/type-first-development.aspx/) | データ型と関数の形を早期の設計・対話手段にする考えを、TypeSpec のモデルと契約を依存する実装より先に定義する流れへ反映する。ただし、型は観察可能な振る舞いを表す規範シナリオや試験の代わりにはしない。 |
| Extreme Programming | Kent Beck の [Extreme Programming Explained](https://www.informit.com/store/extreme-programming-explained-embrace-change-9780321278654) | 小さな変更、短いフィードバック、単純な設計、継続的統合、リファクタリングを日常の開発ループとして扱う。エージェント作業の承認境界は IdMagic が追加する規律であり、XP の規則として帰属させない。 |
| Test-Driven Development | Kent Beck の [Test Driven Development: By Example](https://www.informit.com/store/test-driven-development-by-example-9780321146533) | 新しい実装より先に失敗する自動試験を確認し、小さい RED、GREEN、リファクタリングの循環を回すため、失敗した試験名、その失敗理由、検出する誤実装を RED の証拠として残す。 |
| Behavior-Driven Development | Dan North の [Introducing BDD](https://dannorth.net/blog/introducing-bdd/) | 要求を利用者が観察できる振る舞いとして共通語彙で表し、規範シナリオと受け入れ基準を実行可能な証拠へ結び付ける。別の Gherkin 正本を追加する根拠にはしない。 |
| Acceptance Test-Driven Development | Robert C. Martin と Grigori Melnik の [Tests and Requirements, Requirements and Tests: A Möbius Strip](https://gmelnik.com/papers/IEEE_Software_Moebius_GMelnik_RMartin.pdf)（[DOI `10.1109/MS.2008.24`](https://doi.org/10.1109/MS.2008.24)） | 規範シナリオを実装前に具体化し、外から観察できる受け入れ試験で振る舞いを検証する。TypeSpec、規範文書、試験には異なる責任があるため、試験だけを唯一の仕様とはしない。 |
| Hexagonal Architecture (Ports and Adapters) | Alistair Cockburn の [Hexagonal architecture: the original 2005 article](https://alistair.cockburn.us/hexagonal-architecture) | アプリケーションを UI やデータベースから隔離し、目的を表すポートと技術固有のアダプターを分けることで、内側の振る舞いから外側へ実装し、単体から統合へ検証を広げられるようにする。 |

これらの参照は、IdMagic の各規則の由来と判断材料を示すためのものであり、各方法論への完全準拠や、方法論の効果を本リポジトリで実証したことを意味しない。

### Proposed Changes to `DEVELOPMENT.md`

#### Behavior Evidence and Independent Verification

現在の「仕様を先に変更する」と「内側から外側へ実装する」の間に、「承認対象となる振る舞いを確定し、RED を記録する」工程を明示する。
低リスクの変更では作業者がそのまま進め、中リスク以上では規範シナリオと設計上の選択を人間が確認し、確認後に振る舞いを変える必要が生じたら仕様工程へ戻る。

実装工程のゲートは単に層ごとの試験を通すことではなく、要求識別子、失敗したテスト名、なぜ妥当な誤実装で失敗するのかを作業項目へ残すことにする。
これは `implement-work-item` にすでにある要求を `DEVELOPMENT.md` から見えるようにする変更であり、新しい文書体系を増やさない。

完了とコミットの前には「独立検証」を追加する。
中リスク以上、または認証、認可、テナント境界、暗号、プロトコル互換性、永続化移行に触れる変更では、実装を担当していない人または新しいコンテキストのエージェントが、規範差分と実装差分を照合し、試験が仕様の言い換えではなく誤実装を検出するかを確認する。
独立検証は新しい設計を始める工程ではなく、逸脱を発見したら仕様または実装の所有工程へ差し戻す工程に限定する。

#### Change-Resistance Verification

現行の検証の段階へ、最後の全体検証より前に「変更した論理の試験品質を確かめる」段を追加する。
最初は `risk: high` の Go の純粋なドメインまたはユースケース層を一つ選び、差分変異テストまたは一時的な欠陥注入で、対象となる単体試験が落ちることを確認する試行で十分である。

この段階は全ファイルへ固定の CRAP 上限や変異得点を課すものではない。
同値変異、道具の成熟度、実行時間、UI と外部アダプタの扱いを測定してから、対象範囲と失敗条件を決める必要がある。
IdMagic の「失敗し得る最も安いゲートから始める」という方針に合わせ、変更箇所だけを対象にし、`mise` タスクとして再現可能にする。

Google の大規模な実務データを使った研究でも、従来の全体変異テストは計算量と人間の確認費用が高く、変更行だけを対象にして非生産的な変異を絞る方法が実用化の条件になっているため、差分に限定する方針には Agentic Discipline の事例とは独立した根拠がある（[Practical Mutation Testing at Scale](https://arxiv.org/abs/2102.11378)）。

#### Risk-Based Approval

新しい自律性の列挙を増やすより、既存の作業項目の `risk` を使うほうが単純である。
`DEVELOPMENT.md` に次の対応を規定すると、エージェントが承認要否を会話ごとに推測しなくて済む。

| `risk` | 仕様と設計の承認 | 独立検証 | 変更耐性の確認 |
|---|---|---|---|
| `low` | 作業者が仕様差分を記録して継続 | 任意 | RED の記録 |
| `medium` | 規範シナリオまたは公開契約の差分を人間が確認 | 必須 | 代表的な誤実装で試験が落ちることを確認 |
| `high` | 仕様と設計を人間が承認してから実装 | 必須 | 差分変異または欠陥注入を完了証拠へ記録 |

この対応は外部システムへの書き込み、プッシュ、マージ、本番操作の許可を代替しない。
それらは作業リスクとは別の権限であり、常に明示的な許可を要する。

#### Observable Post-Approval Specification Changes

受け入れシナリオを一律に編集禁止にすると、誤りを見つけたときに正しい仕様へ直せなくなる。
代わりに、中リスク以上では承認時点の Git 参照を作業項目へ記録し、完了時の `mise run spec-diff <ref>` に承認後の規範変更があれば、理由と再承認を完了証拠へ要求する。
これにより、実装を通すために仕様を緩める変更と、実装中に見つかった正当な仕様訂正を同じ禁止規則で扱わずに済む。

### Required Supporting Changes

`DEVELOPMENT.md` の記述だけでは規律がプロンプトへ戻ってしまうため、文書変更と同時に最小限の機械化が必要になる。

- `WORK_ITEM_FORMAT.md` の完了証拠へ、RED のテスト名と要求識別子、中リスク以上の承認基準点、独立検証結果、実施した場合の変異または欠陥注入結果を追加する。
- `check-work-items` は、中リスク以上の完了項目に必要な証拠の欄が存在することと、承認後の仕様差分がある場合に理由が記録されていることを検査する。証拠の意味までは構文検査で保証できないため、独立検証が内容を読む。
- 独立検証には既存の `code-review` に相当する読み取り専用の工程を使い、仕様適合とリポジトリ標準適合を分けて報告する。
- 変異テストを試す場合も基礎コマンドを直接呼ばず、固定したバージョンと対象範囲を `mise.toml` に置く。

### Rejected Adoption Options

#### Separate Gherkin Source of Truth

IdMagic では TypeSpec と種類別の規範文書が正本であり、要求識別子から試験を追跡できる。
同じ振る舞いを `.feature` に書き直すと二つの正本が生まれるため、実行可能な受け入れ試験を増やす場合も、既存の規範シナリオを参照する試験として書くか、既存の正本から生成する必要がある。

#### Fixed Multi-Agent Topology for Every Change

SwarmForge は規模別に二役、四役、六役を使い分けており、公開資料自体も一つの編成を全作業へ適用していない。
IdMagic でも、低リスクの局所変更まで仕様担当、実装担当、整理担当、設計担当へ分けると、引き継ぎと統合の費用が変更本体を上回る。
役割分離は中リスク以上の独立検証と、並行可能な大きい作業に限定する。

#### Universal CRAP Threshold and Mutation Testing

CRAP の数値は複雑度とカバレッジを組み合わせた警告としては使えるが、全言語と全層へ同じ上限を置く根拠は指定資料に示されていない。
変異テストも生き残った変異が欠陥と同義ではなく、同値変異の判定には人間の時間を使う。
まず高リスクの純粋な論理へ限定した試行を行い、検出した欠陥、所要時間、同値変異率を記録して採用範囲を決める。

#### Fetching Latest Quality Tools at Startup

SwarmForge の共通規則は、起動時に CRAP、変異、重複検査の最新版を各 GitHub リポジトリから取得するよう求める（[Engineering Rules](https://github.com/unclebob/swarm-forge/blob/ba50410637ee36b97b314a19ca9a2ee716469ef8/swarmforge/constitution/articles/engineering.prompt)）。
これは `mise.toml` を単一のバージョン、環境、コマンド対応表とする IdMagic の再現性と供給網の方針に反するため、道具の目的だけを採用し、バージョンは `mise` で固定する。

## Plan

1. `DEVELOPMENT.md` のループ表へ RED の証拠とリスク別の独立検証を明記し、既存の `risk` を承認方針へ結び付ける。
2. `WORK_ITEM_FORMAT.md` と `check-work-items` に中リスク以上の承認基準点、独立検証、承認後の仕様差分の証拠を追加する。
3. 高リスクの純粋な Go ロジック一箇所で差分変異または欠陥注入を試し、検出力、所要時間、同値変異を測る。
4. 試行で費用に見合う結果が出た場合だけ、対象をセキュリティ制御、認可判断、状態遷移、プロトコル解析へ広げる。

この順序なら、IdMagic がすでに持つ仕様先行と文脈節約を保ったまま、Agentic Discipline から不足している検証の独立性とテスト品質の証拠だけを追加できる。

実装する場合は本作業項目を再利用せず、第一段階の文書と形式変更を一つの新しい作業項目として起票し、変異テストの試行は別の評価作業項目へ分ける。

## Tasks

- [x] T001 [Research] 指定された四つの資料を一次資料優先で調べ、O'Reilly の未確認範囲を明示する。
- [x] T002 [Compare] 現行の仕様先行、RED、検証ゲート、文脈節約と対応付ける。
- [x] T003 [Decision] 選択的な採用を推奨し、別の Gherkin 正本、固定的な多エージェント編成、一律の品質閾値、最新版道具の起動時取得を退ける。
- [x] T004 [Record] 改善候補、導入リスク、段階的な導入順序、出典を本作業項目へ記録する。

## Verification

- `mise run check-work-items`
- `mise run check-ids`
- 手動：出典ごとに Robert C. Martin の一次資料と DAE 独自の追加を区別し、O'Reilly の購読者向け内容を推定で補っていないことを確認する。

## Risk Notes

本作業項目は評価だけを記録し、製品コードと開発フローを変更しないため、直接の変更リスクは低い。
提案を実装する場合の主なリスクは、既存の正本と Gherkin の二重化、低リスク変更への過剰な承認待ち、変異テストの同値変異と実行時間、役割分割による引き継ぎ費用、固定されていない品質道具による再現性と供給網の悪化であり、Plan の分割と試行で評価する。

## Completion

- **Completed At**: 2026-08-23
- **Summary**: Agentic Discipline の原則は IdMagic に選択的に取り入れる価値があると判断した。現行フローは仕様先行、RED、段階的検証、機械ゲートをすでに備えるため、改善対象はリスク別の人間承認、承認後の仕様差分、独立検証、変更した論理の試験品質の証拠に限定する。実装は別作業項目とし、本作業項目では行っていない。
- **Verification Results**:
  - `mise run check-work-items`：passed（408 files）。
  - `mise run check-ids`：passed（408 record IDs）。
