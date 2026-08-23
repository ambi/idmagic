---
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-11
depends_on:
  - wi-355-replace-scl-architecture-ledgers-and-adrs
change_kind: tooling
spec_impact:
  kind: none
  reason: This work item reorganizes the development method and its documentation tooling without changing product behavior.
initial_context:
  source:
    - REGENERATIVE_ARCHITECTURE.md
    - SPECIFICATION_FORMAT.md
    - SPECIFICATION_CORE_LANGUAGE.md
    - WORK_ITEM_FORMAT.md
    - AGENTS.md
    - ARCHITECTURE.md
    - backend/*/ARCHITECTURE.md
    - frontend/ARCHITECTURE.md
    - docs/requirements.md
    - docs/contexts/*/requirements.md
    - spec/main.tsp
    - docs/contexts/*/main.tsp
    - spec/tspconfig.yaml
    - tools/ra
    - tools/check
    - .agents/skills
  tests:
    - tools/ra/src
    - tools/check/src
  stop_before_reading:
    - backend/*/*.go
    - frontend/src
---

# 開発方式を specification-first workflow として再定義し仕様・設計文書体系を簡素化する

## Motivation

specification、architecture ledger、traceability manifest、新規 ADR 運用を廃止した後の開発方式は、
TypeSpec と requirements を先に更新し、実装・テスト・生成契約を追従させる軽量な
Spec-driven Development (SDD) である。現在の `Regenerative Architecture` という名称と
`REGENERATIVE_ARCHITECTURE.md` は、実際には存在しない自己再生機構や重い architecture
management を想起させ、方法論の理解コストを不必要に上げる。

残す価値のある原則を特定したうえで、方法論、文書、CLI、skills の名称を実態に合う最小の
語彙へ揃える必要がある。

同時に、current-state の仕様と設計が `docs/contexts/*/requirements.md` と実装側の
`*/ARCHITECTURE.md` に離れており、同じ context を理解するために往復が必要になっている。
ファイル名の casing も不統一で、context 概要をどちらへ書くかが曖昧なため、概要を欠くか
重複させるかになりやすい。シナリオは通常の英文と制御語の区別や alternative の対応先も
不明瞭である。さらに、生成 OpenAPI は API が `default` tag に集中して context 単位で把握
しづらく、仕様・設計全体を横断して閲覧できる HTML view もない。

名称だけを変えるのではなく、情報の正本、配置、文法、機械検証、閲覧方法までを一つの
lightweight SDD として再設計し、コンテキストコストと管理コストを実際に減らす必要がある。

## Scope

- 現行手順のうち SDD として維持すべき保証と、RA 固有語として削除できる概念を整理する。
- `Regenerative Architecture`、`RA`、`ra` CLI、関連ファイル名・skill 名の改称案を比較する。
- `REGENERATIVE_ARCHITECTURE.md` を lightweight SDD の短いガイドへ置換または改称する方針を決める。
- requirements と architecture を一つの context 文書へ統合する案と、同じ場所に置いた別文書と
  して維持する案を比較し、root/context 共通の canonical layout と filename casing を決める。
- canonical context 文書を `spec/` に集約する案と、対応する source directory に置く案を、
  discovery、編集時の近接性、複数実装言語、link stability、tooling cost の観点で比較する。
- context overview の正本を一か所に定め、requirements と architecture 間の重複をなくす。
- context 文書の見出し順を、overview、glossary、normative rules、standards、state transitions を
  scenarios より先に読める構成へ統一する。
- scenario keyword と step/alternative reference の小さな文法を決め、通常の英文から明確に
  区別でき、各 alternative の対応先を機械的にも人間にも判別できるようにする。
- TypeSpec operation を context ごとの OpenAPI tag に分類し、`default` に全 API が集中しない
  source convention と検証を追加する。
- lightweight SDD guide、root/context の current-state 文書、TypeSpec/OpenAPI reference を
  navigation 付きの一つの静的 HTML view として生成できるようにする。
- 不要になった `SPECIFICATION_CORE_LANGUAGE.md` を削除し、current references を修復する。
- `REGENERATIVE_ARCHITECTURE.md` と `SPECIFICATION_FORMAT.md` の汎用的な方法論・format 説明から
  `IdMagic` など特定 application の名称を除く。
- work item、specification-first workflow、current-state design record の関係を簡潔に定義する。
- 選んだ名称を docs、AGENTS、tools、skills、README に一貫して反映する実装範囲を確定する。

## Out of Scope

- Product requirements or TypeSpec contract changes.
- specification、architecture ledger、traceability manifest、必須 ADR の再導入。
- 外部の包括的 SDD framework の導入。
- HTML view の hosting、公開サイト運用、全文検索 service の導入。
- 過去の completed work item や ADR 本文を current-state 文書へ混在させること。

## Design

### Canonical context document

少なくとも次の案を比較し、一つを選ぶ。

1. requirements と architecture を `SPECIFICATION.md` のような単一文書へ統合する。
2. `REQUIREMENTS.md` と `ARCHITECTURE.md` を同じ directory に置き、責務境界を明示する。
3. 分離配置を維持し、index/link だけで接続する。

案 3 は移行量が小さい一方、現在の往復コストと overview ownership を根本的に解決できない
可能性が高い。案 1 は discovery と overview の重複防止に優れる一方、大きな context で文書が
肥大化しやすい。案 2 は関心を分離できるが、どちらを先に読むか、共通知識をどちらが所有するかを
追加規約なしには解決できない。ファイル数そのものではなく、1 context を理解・変更する際に読む
正本の数、重複、参照更新数、machine check の複雑さで評価する。

配置は application source tree を特定言語の backend directory と同一視しない。backend、frontend、
worker、外部実装など複数 consumer があり得るため、`spec/` 集約と source-adjacent 配置のどちらが
context ownership を最も安定して表現できるかを決める。選択後は root と全 context に同じ原則を
適用し、概要は canonical document に一度だけ書く。

**Decision:** 案 1 を採用し、root は `docs/SPECIFICATION.md`、各 context は
`docs/contexts/<context>/SPECIFICATION.md` を唯一の current-state 文書とする。TypeSpec と同じ
言語非依存の `spec/` tree に置くことで backend、frontend、worker、将来の別言語実装から等距離に
保つ。source-adjacent 文書を重ねる代わりに、生成 HTML の context navigation と repository search
で code からの discovery を支える。既存の requirements と architecture の内容は失わず統合し、
source-side `ARCHITECTURE.md` は削除する。

### Document order and scenario grammar

canonical context document は概要を入口とし、glossary を直後に置く。standards と state transition
tables を scenarios より前に置き、scenarios は詳細な振る舞い例として末尾側に配置する。
normative requirement の本文と scenario を同一視するか分離するかも決め、見出し順を checker で
検証できるようにする。

scenario は `ACTOR`、`GIVEN`、`WHEN`、`THEN`、`ALT` の uppercase keyword を使い、colon を必須に
しない。番号で離れた step を参照する案と、対象 step の直下へ `ALT` を入れ子にする案を比較する。
旧 SCL の `at` が保持していた対応情報を移行時に失わず、複数 step に関係する分岐、precondition
failure、scenario-wide alternative の表現も決める。checker は orphan alternative を拒否する。

**Decision:** section order は Overview、Glossary、Standards、State Transitions、Authorization
Boundary、Design、Scenarios とする。`REQ-*` scenario heading 自体を一つの観測可能な必須挙動と
定義し、同じ意味を繰り返す `The system SHALL ...` 行は要求しない。`SHOULD` / `MAY` の強度が必要な
記述は scenario と混ぜず、説明または standards policy として明示する。

独立した Requirements section の削除は operation 名の一致だけでは判断しない。旧 interface
requirements 325件を本文と条件まで意味監査した結果は次のとおりだった。

- 公開 operation 301件の説明本文は、空白を正規化した比較ですべて TypeSpec `@doc` と一致した。
- うち219件には追加の事前・事後条件がなく、TypeSpec との完全な二重管理だった。
- 82件には追加条件があったため、名前一致を理由に削除せず、下表の current-state 正本へ被覆を
  確認した。HTTP shape ではなく観測可能な成否は Scenarios、状態不変条件は State Transitions、
  tenant/role/scope 境界は Authorization Boundary、構造・event・内部不変条件は Design、値制約と
  operation semantics は TypeSpec が所有する。
- TypeSpec operation ではない内部 interface 24件は、説明と入力・結果 invariant を Design の
  `Internal Interfaces` に保持した。

| Context | 条件付き公開 interface | Current-state coverage |
|---|---:|---|
| api-tokens | 2 | TypeSpec、Scenarios |
| application | 8 | TypeSpec、Authorization Boundary、Scenarios |
| audit | 1 | Authorization Boundary、Scenarios |
| authentication | 4 | TypeSpec model constraints、Design、Scenarios |
| identity-governance | 2 | State Transitions、Design、Scenarios |
| identity-management | 18 | TypeSpec、State Transitions、Design、Scenarios |
| oauth2 | 8 | TypeSpec、Standards、Authorization Boundary、Scenarios |
| provisioning | 5 | TypeSpec、State Transitions、Scenarios |
| saml | 5 | TypeSpec、Standards、Scenarios |
| sharedsignals | 1 | State Transitions、Scenarios |
| signing-keys | 3 | Design、Scenarios |
| sourcing | 12 | TypeSpec、Authorization Boundary、Scenarios |
| tenancy | 6 | TypeSpec、Design、Scenarios |
| workloadidentity | 2 | TypeSpec、Authorization Boundary、Scenarios |
| ws-federation | 5 | TypeSpec、Standards、Scenarios |

この監査で Requirements だけに残る規範内容が0件になったため、独立 section は持たない。今後も
Requirements を機械的に Scenario へ言い換えるのではなく、意味の owning section に置く。移行先を
示せない規範内容が発生した場合は、独立 section の再導入を禁じず、その時点で再評価する。

scenario keyword は colon なしの `ACTOR`、`GIVEN`、`WHEN`、`THEN`、`ALT` とする。step 番号による
間接参照は使わず、alternative を対象となる `WHEN` または `THEN` の入れ子 list item として直下に
置く。たとえば `THEN SP は証明書を返す` の代替結果は、その子要素として
`ALT 証明書が利用不能である → 証明書を返さずエラーを返す` と書く。Markdown の構造自体が
旧 SCL の extension `at` に相当する対応関係を保持するため、番号を追わず局所的に理解できる。

`GIVEN` は trigger より前から成立する状態・前提だけ、`WHEN` は system under specification を
作動させる操作・入力・外部 event、`THEN` はその後に観測できる結果だけを表す。既存 step の先頭を
一律に `WHEN` へ変換せず、244 scenario を個別に読み、必要なら一つの文に混在していた操作と結果を
分割する。複数操作を含む flow では複数の `WHEN` を許す。checker は意味そのものを自然言語から
推測せず、keyword の casing、`WHEN` / `THEN` の存在、`ALT` の入れ子と `→` を検証する。

### Generated views

TypeSpec では context namespace または明示 decorator から安定した OpenAPI tag を生成する。
共通 schema は必要なら独立 category として扱うが、operation を無差別に `default` に残さない。

HTML は別の手書き正本を作らず、canonical Markdown と TypeSpec の生成物から再生成する。
root から context overview、glossary、standards、state transitions、requirements/scenarios、API reference
へ辿れる navigation を持たせる。生成物は review 用の derived artifact とし、通常の source editing
や verification に hosted service を要求しない。

**Decision:** owning context namespace に明示的な OpenAPI tag を付ける。静的 HTML は
`DEVELOPMENT.md`、`SPECIFICATION_FORMAT.md`、root/context `SPECIFICATION.md` と生成 OpenAPI を一つの
navigation にまとめ、`just spec-render` で再生成する。方法論に固有名や略称を付けず、
specification-first development workflow と呼ぶ。方法論専用 CLI は設けず、公開操作は既存の
`just` recipe に限定する。

## Plan

現行フローを「必須」「有用だが任意」「歴史的残骸」に分類する。名称候補ごとに意味の正確さ、
検索性、既存 path/command の移行コストを比較し、最小の概念集合を選ぶ。同時に、代表的な root、
backend-owned context、frontend/shared context を使って文書統合・配置案を試し、読むファイル数、
重複行、cross-directory reference、checker/renderer の特別規則を比較する。

選択した layout と scenario grammar を全 context へ機械的に移行し、その後 TypeSpec tagging と統合
HTML renderer を追加する。結論は current-state の仕様・設計文書へ直接反映し、却下案の長期保存を
目的とした新規 ADR は作らない。path rename では active work item、skills、tool tests、Markdown link
を同時に更新し、historical records は link compatibility が必要な場合だけ最小限に触る。

## Tasks

- [x] T001 [Analysis] 現行 method 文書・CLI・skills が実際に提供する保証を一覧化する。
- [x] T002 [Analysis] 統合/分離と spec-central/source-adjacent の候補を代表 context で試し、context cost、重複、参照数、tooling cost を比較する。
- [x] T003 [Decision] 固有名を持たない specification-first workflow、canonical document layout/casing/location、command/skill naming を決める。
- [x] T004 [Decision] overview ownership、section order、uppercase scenario keyword、nested alternative の文法を決める。
- [x] T005 [Docs] root と全 context の仕様・設計文書を選択した layout、順序、scenario grammar へ移行する。
- [x] T006 [Tooling] document checker を新しい path、section order、scenario step/alternative integrity に対応させる。
- [x] T007 [Cleanup] `SPECIFICATION_CORE_LANGUAGE.md` を削除して current references を修復し、汎用 method/format 文書から application 固有名を除く。
- [x] T008 [TypeSpec] 全 operation に context tag を付け、生成 OpenAPI で context ごとに分類されることを検証する。
- [x] T009 [Docs UI] current-state の仕様・設計・TypeSpec API reference をまとめた navigation 付き静的 HTML を生成する recipe を追加する。
- [x] T010 [Docs] workflow 文書、AGENTS、README、format 文書、work-item references の用語と path を同期する。
- [x] T011 [Tooling] 方法論専用 CLI を廃止し、公開操作を `just` recipe、内部 discovery を `tools/workspace` に整理する。
- [x] T012 [Spec] 全244 scenario の `GIVEN` / `WHEN` / `THEN` を意味で個別監査し、番号参照を nested `ALT` へ移行する。
- [x] T013 [Verify] scenario semantics の個別監査後に repository checks、tool tests、OpenAPI tag check、HTML link check、stale terminology/path search を通す。

## Verification

- `just check`
- `just test-tools`
- `just typecheck-tools`
- `just check-spec`
- `just check-api-compat`
- `just render-spec-docs`
- generated OpenAPI に operation の `default` tag がなく、全 operation が owning context tag を持つこと
- generated HTML から root と全 context の overview、glossary、standards、states、scenarios、API reference へ到達できること
- scenario checker が `WHEN` / `THEN` の欠落、top-level または深すぎる `ALT`、`→` のない `ALT` を拒否すること
- current-state references に `SPECIFICATION_CORE_LANGUAGE.md` と旧 requirements path が残っていないこと
- generic method/format documents に `IdMagic` または他の application 固有名が残っていないこと
- `rg -n "Regenerative Architecture|REGENERATIVE_ARCHITECTURE|\\bra\\b" AGENTS.md README.md *.md tools .agents/skills`

## Risk Notes

主なリスクは、機械的な改称や文書移動で過去の work item・ADR link を壊すこと、requirements と
architecture の統合で巨大な文書を作ること、一般名 `SDD` に寄せすぎて repository 固有の最小ルール
が曖昧になることである。TypeSpec tag の追加で released OpenAPI の operation semantics を変えない
こと、HTML renderer を新たな独自 specification format にしないことにも注意する。履歴ファイル名の
互換性と現在形の用語は分けて扱う。

## Completion

- **Completed at**: 2026-08-11
- **Summary**:
  Regenerative Architecture / SCL 固有の方法論・CLI・ledger を、TypeSpec と root/context
  `SPECIFICATION.md` を正本とする specification-first development workflow へ簡素化した。
  requirements と source-side architecture 文書を context ごとの単一文書へ統合し、OpenAPI の
  context tag と navigation 付き HTML view を追加した。独立 Requirements section は本文・条件の
  意味監査で他の正本へ被覆できることを確認したうえで削除した。

  scenario 文法は番号参照を廃止し、`ALT` を対象の `WHEN` / `THEN` の直下に置く形式にした。
  全244 scenario を個別に読み、`GIVEN` は事前状態、367件の `WHEN` は操作・入力・外部 event、
  564件の `THEN` は観測結果となるよう再分類・分割した。242件の `ALT` は対象 step の子要素へ
  配置し、番号付き step は0件になった。
- **Verification results**:
  - `just check-spec` passed for 21 specification documents, 244 scenario IDs, 311 operations, and 17 API tags.
  - `just test-tools` passed (97 tests).
  - `just typecheck-tools` passed.
  - `just check-api-compat` passed against `spec/idmagic.openapi.baseline.json`.
  - `just verify` passed, including Go tests/lint and frontend tests/build.
