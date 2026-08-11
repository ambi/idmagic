---
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-11
depends_on: [wi-356-reframe-ra-as-lightweight-spec-driven-development]
change_kind: tooling
spec_impact:
  kind: none
  reason: >-
    仕様・設計の正本にある現在設計と説明を補足し、その派生 HTML の情報構造と表現を改善する変更であり、
    製品の API wire contract、認証・認可、状態遷移、利用者向け挙動は変更しない。
initial_context:
  source:
    - WORK_ITEM_FORMAT.md
    - SPECIFICATION_FORMAT.md
    - spec/main.tsp
    - spec/SPECIFICATION.md
    - spec/contexts/*/SPECIFICATION.md
    - spec/contexts/*/*.tsp
    - spec/generated/openapi/*.json
    - tools/render-spec-docs/src/main.ts
    - tools/render-spec-docs/src/render.ts
    - tools/package.json
    - tools/README.md
    - justfile
    - README.md
  tests:
    - tools/render-spec-docs/src/render.test.ts
    - tools/check/src
  stop_before_reading:
    - backend
    - frontend/src
---

# 生成仕様 HTML を分割し、図・scenario・API・model を人が読める仕様サイトにする

## Motivation

`just spec-render` が生成する `spec/generated/docs/index.html` は、method 文書、IdMagic 全体の仕様、
全 bounded context の仕様、API reference、全 model を約 1.1 MB の単一ページへ連結している。
左 navigation はあるが、目的の context を開いても前後の全内容を同時に読み込むため、現在位置と文書境界を
把握しづらく、長い API/model 表は横断的な参照にも個別要素の理解にも向いていない。

内容と表現にも次の不整合がある。

- Method 配下の `ワークアイテム` だけが日本語で、本文も他の method 文書と表示言語が揃っていない。
- root 文書の `Repository Specification` は repository の管理仕様に見え、製品名に依存しない
  whole-system の仕様・現在設計の入口であることが伝わらない。
- `Context Map` は context と Go package の一覧表だけで、DDD の context map が示す bounded context 間の
  upstream/downstream 関係、連携、境界パターンを図として表していない。
- canonical `SPECIFICATION.md` に historical ADR への link が多数残り、生成 HTML が廃止済みの ADR を
  current documentation の一部として提示している。現在も必要な判断理由は仕様本体にあるべきである。
- state transition は規範表だけで、遷移経路を視覚的に追える図がない。
- scenario の `ACTOR` / `GIVEN` / `WHEN` / `THEN` / `ALT` が通常文と同じ表示で、前提、trigger、結果、
  分岐を走査しづらい。
- 独自 API 一覧は OpenAPI `operationId` を意味の曖昧な `Operation` 列として表示する一方、OpenAPI 自体が持つ
  parameter、authentication、request/response の情報を再現できていない。標準契約を不完全に再実装している。
- Models は OpenAPI components schema だけを schema 名、property 名の comma-separated list、説明からなる巨大な表に
  している。API に露出しない TypeSpec model を含む仕様全体の model catalog になっておらず、property ごとの型、
  必須性、制約、説明、model 間の関係も確認できない。
- typography、余白、階層、responsive navigation、focus 表示などが最低限で、仕様を継続的に読む画面としての
  情報密度と可読性が不足している。

生成物を別の正本にはせず、Markdown、TypeSpec、生成 OpenAPI から再現可能なまま、読む単位と情報表現を
人の理解に合わせた静的 specification site へ改善する。

## Scope

- `WORK_ITEM_FORMAT.md` の H1、見出し、本文、例を英語へ統一し、Method navigation では
  `Work Item Format` と表示する。
- `spec/SPECIFICATION.md` の root title を application 名に依存しない `Whole-System Specification` とし、
  repository 管理情報ではなく全 bounded context を横断する specification/current design であることを Overview と
  navigation で明示する。
- root `Context Map` に bounded context 間の domain relationship を表す Mermaid graph と legend を追加し、
  現行の responsibility 一覧は詳細参照として残す。
- root/context `SPECIFICATION.md` にある `decisions/ADR-*` 参照を意味監査する。現在も有効な設計・理由は owning
  `SPECIFICATION.md` の Design へ短く自己完結して移し、historical link は canonical specification と生成 HTML
  から除く。`decisions/` の履歴ファイル自体は変更しない。
- `spec/generated/docs/index.html` を入口とし、Method、whole-system specification、bounded context、API reference、model を
  意味単位の複数 HTML page に分割する。すべてのページは相互 navigation と stable deep link を持つ。
- canonical state-transition table を入力に Mermaid state diagram を生成し、同じ page に表と図を表示する。
  手書きの遷移表と手書きの図を二重管理しない。
- Markdown 内の Mermaid fence を検証して安全に図として表示し、Context Map を graph として閲覧可能にする。
- scenario keyword を意味別の badge/color/spacing で表示し、nested `ALT` と親 step の関係を視覚化する。
- API の request/response/authentication/schema 表示を独自実装せず、生成 OpenAPI を入力とする標準的な
  OpenAPI-native viewer に委譲する。仕様サイトは API reference への navigation と source OpenAPI への link を提供する。
- OpenAPI Models とは別に、TypeSpec program の全 model/enum/union/scalar を context ごとに閲覧できる model catalog を
  生成する。API 到達性にかかわらず property、必須性、型、constraint、説明、継承・参照関係を表示し、API に露出する
  symbol にはその区別を付ける。
- model/property の説明が TypeSpec 側で欠けている場合は、wire contract を変えない `@doc` 等の metadata を owning
  TypeSpec に追加する。
- responsive layout、読みやすい本文幅と typography、明確な heading hierarchy、breadcrumb/current navigation、
  keyboard focus、light/dark theme、print 時の基本可読性を整える。
- generated page/assets の到達性、内部 link、escaping、Mermaid、OpenAPI viewer integration、model catalog を自動検証する。
- `tools/` の実装から application 名・module path・OpenAPI filename の固定値を除き、compiler source location、
  `go.mod`、標準 directory 内の単一 artifact discovery から対象を決める。同じ specification-first layout を持つ
  application で tool source を変更せず再利用できるようにする。

## Out of Scope

- 製品 API の route、request/response shape、status code、authentication、authorization の変更。
- 製品の状態機械や scenario の意味変更。
- generated HTML の hosting、公開サイト、server-side search、analytics の導入。
- completed work item や `decisions/` 配下の historical ADR 本文の翻訳・削除・改変。
- source import graph や全実装依存 edge を Context Map に転記する architecture ledger の再導入。
- 独自 Markdown dialect、独自 API schema、手書き HTML を新たな正本にすること。
- OpenAPI operation/request/response/schema renderer の自作と、Go struct を specification model として収集すること。
- frontend application の component/design system を documentation renderer と共有すること。
- TypeSpec に既存する大量の説明欠落を symbol 名から推測して一括生成すること。欠落は catalog で明示し、
  domain knowledge に基づく説明追加は owning specification change として扱う。

## Design

### Information architecture

次の案を比較する。

1. 現在の単一長大 HTML を維持し、CSS だけを改善する。
2. 単一 HTML 内で JavaScript により document を切り替える client-side view にする。
3. `index.html` を入口とする複数の静的 HTML page を生成する。

案 1 は読み込み量、document 境界、API/model の長大化を解消しない。案 2 は一つの配布物を保てるが、初期 payload は
残り、navigation/history/accessibility を JavaScript で再実装する必要がある。案 3 を採用し、概ね次の単位で
出力する。

```text
spec/generated/docs/
  index.html
  method/<document>.html
  specification/index.html
  contexts/<context>/index.html
  api/index.html
  models/index.html
  models/<symbol>.html
  assets/*
```

正確な filename escaping は renderer 内で一か所に定義する。`index.html` は landing/navigation とし、全 canonical
document、API reference、model が link graph 上で到達可能でなければ生成を失敗させる。page の基本内容と link は
JavaScript なしでも利用可能にし、filter、menu toggle などは progressive enhancement とする。外部 CDN や
network access を閲覧・生成の必須条件にしない。

既存の heading anchor は可能な限り維持し、文書間 relative Markdown link と fragment を新しい出力先へ解決する。
旧単一 page anchor の完全互換 shim は持たず、generated artifact 内の新しい link 整合を保証する。

### Whole-system specification and Context Map

root document は `Repository Specification` ではなく `Whole-System Specification` と呼ぶ。application/product 名を
title に含める案は repository ごとに表示語が変わり method として不安定なため採らない。`Root Specification` は配置を、
`Shared Specification` は shared kernel/capability を想起させ、内容の範囲を表さない。`System Specification` は既存の
bounded context `System` と区別しづらい。`Whole-System` は全 bounded context を包含する範囲を直接表し、個別 context の
`<Context> Specification` と対になる。navigation では root を最初に配置し、context 個別仕様との包含関係を明確にする。

Context Map は package/module dependency graph ではなく、DDD の domain relationship を表す。各 bounded context を
node とし、実際に存在する published interface/event/protocol に基づく主要な upstream/downstream relationship と
integration pattern を edge label で示す。意味のない全 pair や source import edge は載せない。graph の近くに
relationship label の legend と、詳細責務を読む一覧を置く。実装時に current interfaces と context ownership を監査し、
推測の edge は追加しない。

graph の Mermaid source は `spec/SPECIFICATION.md` に置き、current design の一部として review 可能にする。renderer は
Mermaid syntax error を生成時に検出し、図を安全な SVG または repository-local asset として表示する。

### ADR retirement in current documentation

historical ADR の title/link をそのまま非表示にする renderer filter は採らない。それでは current rationale が link の
向こうにだけ残り、Markdown と HTML の意味が分岐するためである。root/context `SPECIFICATION.md` の各 ADR reference を
監査し、現在も有効な conclusion と短い rationale を owning Design に自己完結させた後で link を削除する。歴史的経緯だけの
reference は移植しない。

canonical specification から `decisions/` への link が再流入しない checker/test を追加する。work item から historical
evidence を参照することと、ADR archive を read-only で保持することは妨げない。

### State diagrams and scenarios

state transition の規範的な正本は既存の `From / Event / Guard / To / Effects` table のままとする。renderer は section 内の
各 state-machine table を解析し、同じ row から Mermaid state diagram を構成する。edge label には Event と Guard を、詳細には
Effects を失わず対応付ける。開始/終了状態を表から一意に導けない場合は勝手に補わず、通常 state/edge のみを描く。

table parser が期待する列や state を解釈できない場合は図を黙って省略せず生成を失敗させる。これにより table と diagram の
drift を作らない。`SPECIFICATION_FORMAT.md` には、表が規範正本で図が同じデータからの derived view であることを追記する。

scenario は source grammar を変えず、list item の先頭 token を renderer で分類する。`ACTOR`、`GIVEN`、`WHEN`、`THEN`、
`ALT` は色だけに依存しない label、形、indent/connector を持ち、通常の uppercase word は誤分類しない。nested `ALT` は親の
`WHEN`/`THEN` と同じ visual group に置く。screen reader で元の文と順序が失われない HTML semantics を維持する。

### API reference and model catalog

API の完全な presentation model を独自 renderer で再構築する案は採らない。OpenAPI は operation、parameter、security、
request body、response、schema の標準表現を既に持ち、独自実装は仕様追加のたびに表示漏れを作る。生成済み
`spec/generated/openapi/` で一意に発見した生成 OpenAPI JSON をそのまま入力にする OpenAPI-native viewer を
`api/index.html` に統合し、
request/response/authentication/schema/`operationId` の表示と navigation は viewer に委譲する。raw OpenAPI JSON への link も
残す。

viewer は OpenAPI version compatibility、offline/local asset、deep link、accessibility、bundle size、active maintenance を
比較した。Swagger UI を採用する。公式 static distribution が OpenAPI 3.1、operation/security/request/response/schema、
deep link を一つの local bundle で扱え、独自 API renderer を持たずに済むためである。Redoc/Scalar も候補だが、同じ契約の
表示を複数導入する利点がないため採用しない。外部 CDN は使わず、viewer 固有の annotation を TypeSpec/OpenAPI に持ち込まない。viewer が利用不能でも raw
OpenAPI へ到達できるようにする。仕様サイト自身が持つ API 固有ロジックは page shell、source link、context から API reference
への導線までに限定する。

一方、model catalog は API reference の Models section と同一視しない。TypeSpec compiler program から project source に
宣言された model、enum、union、scalar と property を収集し、HTTP operation から到達しない domain/specification model
も含める。`Operations` namespace の transport wrapper は Swagger UI に委ねて重複収集せず、Go struct は実装詳細なので
収集しない。catalog は検索、namespace、symbol kind、documentation、template/継承、
property の optionality/type/default/constraint、他 symbol への参照を表示し、symbol 間を link する。OpenAPI に emit された symbol
には `API-exposed` のような区別と API reference への導線を付けるが、OpenAPI schema を model catalog の入力正本にはしない。

model/property description の欠落は renderer が推測で補わず、重要な宣言は TypeSpec の `@doc` を仕様側で補う。recursive reference
は無限展開せず link で表す。同名 symbol は完全修飾 TypeSpec 名で識別し、URL slug collision を生成時に検出する。

tooling の application 非依存性は文字列置換ではなく入力境界で作る。TypeSpec symbol の所有は root namespace 名ではなく
compiler の `project` source location、OpenAPI current/baseline は標準 directory の一意な JSON、Go import prefix は
`go.mod` の module 宣言から導出する。曖昧な複数 artifact は推測で選ばず fail closed にする。

### Visual and implementation boundaries

renderer は Markdown と TypeSpec symbol metadata を presentation model に正規化してから page template へ渡す。OpenAPI の
意味解釈は OpenAPI-native viewer に委譲する。path/link/symbol reference の
解決、HTML escaping、page chrome を小さな責務に分け、巨大な template string 一つへ追加し続けない。共通 CSS/必要最小限の
JS は fingerprint 不要の repository-local asset とし、generated site を directory ごとコピーして閲覧できるようにする。

見た目は装飾量より走査性を優先する。本文幅、line height、heading spacing、code/table overflow、sticky navigation、breadcrumb、
current item、method badge、property required marker、focus ring を共通 token で整える。色は scenario/method の唯一の意味伝達手段に
せず、狭い viewport では navigation を折りたたみ、本文や API field が viewport 全体の横 scroll を発生させない。

## Plan

最初に canonical documents、state tables、scenario grammar、TypeSpec symbol と OpenAPI viewer integration、現行 internal links を
inventory 化し、representative fixture と期待する page graph を固定する。次に `spec/SPECIFICATION.md`、各 owning context の Design、
`SPECIFICATION_FORMAT.md`、`WORK_ITEM_FORMAT.md` を specification-first に更新する。

その後 renderer を source parsing、presentation model、link graph、page rendering、asset emission に分け、まず multi-page
navigation と link rewrite を実装する。Mermaid/state diagram と scenario rendering を加えた後、OpenAPI-native viewer と
TypeSpec model catalog を統合する。最後に visual polish を行い、情報欠落を CSS で隠していないことを fixture と生成結果の
両方で確認する。

generated files は ignored artifact のままとし、source、tests、recipe だけを commit 対象にする。新しい ADR は作らず、採用案と
却下案、移行中に判明した制約はこの work item の Design/Completion に残す。

## Tasks

- [x] T001 [Inventory] canonical document/ADR link/state table/scenario、全 TypeSpec symbol、OpenAPI viewer 候補、現行 output/link を棚卸しし、代表 fixture と欠落情報を記録する。
- [x] T002 [Spec] `Whole-System Specification` title/Overview と DDD Context Map graph を `spec/SPECIFICATION.md` に追加し、現在も有効な ADR rationale を owning root/context Design へ移して historical link を除く。
- [x] T003 [Method] `WORK_ITEM_FORMAT.md` を例を含め英語へ統一し、`SPECIFICATION_FORMAT.md` に multi-page generated view、derived state diagram、canonical spec から ADR archive を参照しない規則を反映する。
- [x] T004 [TypeSpec] API 到達性にかかわらず model/enum/union/scalar/property の説明欠落を監査し、欠落を catalog に明示する。domain knowledge なしの推測文は生成しない。
- [x] T005 [Renderer] source parsing、presentation model、link graph、page template、asset emission を分離し、index/Method/root/context/API/model の multi-page output と relative link rewrite を実装する。TypeSpec/OpenAPI/Go module discovery は application 名に依存させない。
- [x] T006 [Diagram] canonical Mermaid fence を検証・表示し、state-transition table から Mermaid state diagram を一意に生成して表と併記する。
- [x] T007 [Scenario] `ACTOR` / `GIVEN` / `WHEN` / `THEN` / nested `ALT` を semantic markup と accessible style で表示する。
- [x] T008 [API] OpenAPI-native viewer を比較・選定して repository-local asset として統合し、raw OpenAPI link と context からの導線を追加して独自 operation/schema renderer を撤去する。
- [x] T009 [Models] TypeSpec compiler program から API 非公開 symbol を含む model catalog、property details、symbol 間 link、API-exposed marker を生成する。
- [x] T010 [Design] responsive navigation、breadcrumb、content hierarchy、typography、light/dark/print、keyboard focus を整える。
- [x] T011 [Tests] multi-page manifest、全 internal link、ADR archive 非参照、Mermaid/state rows、scenario semantics、OpenAPI viewer integration、全 TypeSpec symbol/model constraints、escaping の自動テストを追加する。
- [x] T012 [Docs] `README.md` と `tools/README.md` の generated site の入口・再生成方法・正本ではないことを新しい出力構造へ同期する。
- [x] T013 [Verify] specification、work item、tool unit/typecheck、renderer、全体 verification を通し、代表的な desktop/narrow viewport の生成結果を目視確認する。
- [x] T014 [Polish] Mermaid の edge/node contrast、unconditional Guard 表記、dark theme 内の Swagger UI contrast を実生成で調整する。

## Verification

- `just check-work-items`
- `just check-ids`
- `just check-spec`
- `just test-tools`
- `just typecheck-tools`
- `just spec-render`
- `just check`
- `just verify`
- `rg -n -i 'idmagic' tools --glob '!node_modules/**'`
- `spec/generated/docs/index.html` から全 Method 文書、`Whole-System Specification`、全 bounded context、OpenAPI reference、全 TypeSpec model symbol へ link のみで到達でき、生成物内の全 relative link/fragment が解決すること。
- Method navigation と本文に `ワークアイテム` が残らず、`Work Item Format` と英語本文が表示されること。
- canonical root/context `SPECIFICATION.md` と generated HTML に `decisions/ADR-*` link がなく、必要な current rationale が owning Design 内で自己完結していること。
- Context Map が bounded context node、実在する主要 relationship edge、DDD relationship label/legend を持つ graph として表示され、責務一覧も参照できること。
- 全 canonical state-transition table に同じ row から生成した diagram があり、state/event/guard/effect の対応を失わないこと。
- 全 scenario keyword が semantic class/label を持ち、nested `ALT` が親 `WHEN`/`THEN` と視覚的・DOM 構造上対応すること。
- OpenAPI-native viewer が生成 OpenAPI を直接読み、authenticated/public operation、parameter、request body、response、schema を閲覧でき、raw OpenAPI JSON にも到達できること。
- API operation から到達しない fixture を含む `Operations` namespace 外の TypeSpec model/enum/union/scalar が catalog に現れ、required/optional property、型、default、constraint、継承・参照関係が正しく表示・link されること。
- `tools/` の tracked source/manifest/lockfile に application 名・固定 module path がなく、別名 fixture と標準 directory discovery の test が通ること。
- JavaScript と network access がなくても Method/specification本文、navigation、model catalog を読め、API page から raw OpenAPI JSON へ到達できること。bundled viewer を有効にした通常閲覧では API details を操作できること。
- narrow viewport で page 全体の意図しない横 scroll がなく、keyboard だけで主要 navigation と content link を辿れること。

## Risk Notes

multi-page 化では relative link と fragment の破損、schema 名/path の衝突、generated output の stale file が主なリスクになる。
出力前に page manifest を確定し、同一 manifest で link rewrite と到達性検査を行う。出力 directory の置換は generated artifact
だけを対象にし、source を巻き込まない安全な生成手順にする。

Mermaid を browser runtime/CDN に任せると offline 閲覧、supply-chain、content security が不安定になる。生成時 validation と
repository-local renderingを優先し、dependency を追加する場合は tools scope に閉じる。Markdown/API の文字列は必ず escape し、
diagram label や example を raw HTML として注入しない。

ADR reference の除去は情報消失を招き得るため、機械的な一括削除をしない。各 link の現在有効な conclusion/rationale を先に
owning Design へ写した証跡を T001/T002 に残す。Context Map は全 source dependency の台帳に膨張させず、domain relationship と
integration boundary に限定する。

OpenAPI viewer dependency は bundle size、更新、脆弱性、offline asset のリスクを持つため、T001/T008 の比較結果と採用理由を
この work item に残す。TypeSpec compiler API への model catalog の結合は compiler upgrade で壊れ得るため、symbol extraction を
adapter に閉じ、API 非公開 symbol を含む fixture で固定する。

TypeSpec の説明補足が generated OpenAPI baseline の wire compatibility を変えないことを確認する。visual snapshot だけに依存せず、
model presentation data と semantic HTML、OpenAPI viewer integration の unit test を主とする。

## Completion

- **Completed at**: 2026-08-11
- **Summary**:
  単一の長大 HTML を 767-page の静的 specification site に置き換え、Method、whole-system、各 bounded
  context、Swagger UI API reference、検索可能な TypeSpec model catalog を独立して閲覧できるようにした。
  DDD Context Map と transition table 由来の Mermaid diagram、scenario keyword の semantic styling、
  responsive/light/dark/print layout、全 generated link の到達性検査を追加した。

  root 文書は application 名に依存しない `Whole-System Specification` とし、current specification から
  historical ADR link を除去して現行 rationale を Design に残した。モデル監査では `Operations` transport
  wrapper 1,805件を Swagger UI に委譲し、それ以外の宣言型740件を property 型・必須性・constraint・参照と
  ともに catalog 化した。既存 TypeSpec には539 symbol page、合計2,220箇所の description 欠落があったため、
  推測文は生成せず `No description.` として明示した。

  `tools/` から application 名と固定 module/OpenAPI path を除去した。TypeSpec ownership は compiler の
  project source location、OpenAPI current/baseline は標準 directory の一意 artifact、Go import root は
  `go.mod` から導出する。別名 fixture と曖昧 artifact の fail-closed test を追加した。

  目視 feedback 後、Mermaid の edge/arrow と node border を light/dark theme ごとの高 contrast 色へ
  固定し、無条件遷移の Guard 91件を `""` から `—` へ正規化した。Swagger UI は dark page 内でも
  自身の配色が崩れない light color-scheme surface に隔離した。
- **Verification results**:
  - `just check-spec` passed for 24 documents, 311 operations, 17 API tags, and 740 catalog symbols.
  - `just test-tools` passed (102 tests).
  - `just typecheck-tools` passed.
  - `just check-api-compat` passed against the released OpenAPI baseline.
  - `just spec-render` generated 767 linked pages and validated all Mermaid sources.
  - `rg -n -i 'idmagic' tools --glob '!node_modules/**'` returned no matches.
  - `just verify` passed, including Go tests/lint and frontend tests/build.
  - Desktop and compact Quick Look previews confirmed the generated hierarchy, navigation, content width,
    focusable links, and responsive styling structure.
