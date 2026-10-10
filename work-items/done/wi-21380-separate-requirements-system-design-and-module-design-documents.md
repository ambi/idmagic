---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-10
priority: p2
depends_on: []
change_kind: docs
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 文書の区分、配置、生成サイトの URL を整理する変更であり、製品の利用者と運用者が観測する振る舞い、設定、API は変わらない。
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - tools/workspace/src/document-layout.ts
    - tools/check/src/specification-doc.ts
    - tools/check/src/check-terminology.ts
    - tools/render-docs/src/main.ts
    - tools/render-docs/src/render.ts
  tests: [tools/check/src/specification-doc.test.ts, tools/render-docs/src/render.test.ts]
  stop_before_reading: [backend, frontend/src]
spec_impact:
  kind: none
  reason: "文書の区分名、配置、索引の表、生成サイトの URL と目次を整理するだけで、外部から観測できる振る舞い、TypeSpec の契約、REQ-* と標準 id の文言と番号は変えない。"
---

# 文書を要件、全体設計、モジュール設計の三区分に分け、索引、配置、語彙、生成サイトを一つの体系にそろえる

## 動機

文書の区分と配置が、区分名、ディレクトリ、生成サイトの目次の間で一致しない。
読み手は、ある内容が要件なのか設計なのか、全体のものかモジュールのものかを、区分名からも置き場所からも判断できない。

| 箇所 | 問題 |
| --- | --- |
| `docs/README.md` | 冒頭の「製品の目的と範囲はプロダクト概要に示す」は、直下の表の要求文書の行と重なる |
| `docs/README.md` | 「リファレンス」の行だけ入口へのリンクがない |
| `docs/README.md`、`docs/requirements/` | requirement を「要求」と訳しており、一般語の要求（依頼、リクエスト）と区別できない。機能仕様の `REQ-*` はすでに「要件」と呼んでいる |
| 生成サイトの目次 | 要求文書が「設計文書」の枝の下に表示される。ディレクトリは `docs/requirements/` と `docs/design/` に分かれている |
| 生成サイトの URL | システムの文書は `site/docs/...`、モジュールの文書は `site/domain/...`、フォーマットは `site/format/...`（ディレクトリは `formats/`）に出力され、リポジトリのパスと対応しない |
| `site/traceability/index.html` | 規則とテスト参照の一覧であり、被覆の判定は `mise run traceability-strict` と仕様差分の検査が行う。読み手が文書から使う場面がない |
| `docs/design/README.md` など 3 文書 | 冒頭が「このディレクトリには」であり、文書ガイドが定める「この文書は」に従っていない |
| `docs/design/README.md` | 列名「話題」は、正式な設計記述の用語に対応しない |
| `docs/design/README.md` | 「設計判断」と「リスク」は `architecture/` の中の文書なのに、アーキテクチャと同列の行になっている |
| `docs/design/README.md` | 行の順序が生成サイトの目次の順序と異なる（インフラストラクチャの位置） |
| `docs/design/README.md` | 「アーキテクチャ」だけ「〜設計」が付かない。表の右列にリンクと説明が混在する |
| `docs/design/README.md`、`docs/domain/README.md` | 「設計文書」と「モジュール設計」の関係が分からず、設計文書の索引からモジュール設計へたどれない |
| モジュールの `design/README.md`（21 文書） | 同じ「話題」の表であり、上の三つの問題がすべて当てはまる |
| `docs/requirements/functional.md` | 機能の表が 4 分類しかなく、利用目的の表と軸が異なる。粒度が粗い理由が書かれていない |
| `docs/requirements/functional.md` | 「担当する仕様」の列が実際にはモジュール設計の文書を指す |
| `docs/requirements/quality.md` | セキュリティとアクセシビリティの義務を `docs/domain/standards.md` と各モジュールに委ねており、文書ガイド §4.2 が定める「品質の義務は `quality.md` が一次情報」と食い違う |
| `docs/domain/` | 区分名は「モジュール設計」なのにディレクトリ名が `domain` であり、`docs/domain/README.md` がその不一致を説明する一文を置いている |
| `docs/domain/` 直下の全体文書 | `glossary.md`、`standards.md`、`scenarios.feature.md` はシステム全体の要件、`structure.md` は全体設計であり、モジュール設計の区分に置かれている |
| `docs/domain/structure.md`、`docs/design/application/` | フロントエンド設計はあるがバックエンド設計がなく、バックエンドの構成はモジュール設計の区分の `structure.md` にある。フロントエンドのコンポーネント構造も `structure.md` と `frontend.md` に分かれている |
| `docs/domain/structure.md`、`docs/design/architecture/logical.md` | 両方に「アーキテクチャ様式」の節がある |
| `docs/design/architecture/constraints.md`、`strategy.md` | どちらも表一つの小さな文書で、内容はアーキテクチャ設計の概要にあたる |
| `docs/domain/glossary.md` | Module の定義が、モジュール設計の文書の単位と同じものかを明示していない |
| `docs/formats/specification-format.md` | 名前は仕様の形式だが、`docs/` 全体の配置図、設計判断と設計領域と脅威モデルの書式、生成サイトの規則まで扱う。文書ガイドは配置図をこの文書へ委ねており、内容の担当と配置の分担がねじれている |

## 対象範囲

- `docs/` の最上位の区分、区分名、索引の表、ディレクトリ配置を、下の設計に合わせて変える。
- requirement の訳語を「要件」に統一する。
- 「話題」を「設計領域」に改め、設計領域の集合と順序を変える。システムとモジュールの設計の索引をすべて書き換える。
- `docs/domain/` を `docs/modules/` へ移し、全体文書を要件文書と全体設計文書へ移す。
- `structure.md` を分解し、バックエンド設計を新設する。
- 機能要件と品質要件の文書を書き直す。
- 生成サイトの出力パス、目次、リファレンスの索引を変え、トレーサビリティのページを削除する。
- `specification-format.md` を、配置と共通規則を文書ガイドへ、設計の書式を新設する `design-format.md` へ分け、仕様の書式だけを残す。
- 上の変更が触れる検査（`tools/check`、`tools/workspace`、`tools/render-docs`）、skills、`AGENTS.md`、文書ガイド、仕様フォーマット、作業項目フォーマットを更新する。

## 対象外

- `api-guidelines.md` と `design-guidelines.md` の記述の改善、設計文書からコードへの参照の整理、Aggregate の扱い。wi-89430 が扱う。
- `spec/contexts/` と `REQ-<CONTEXT>-NNN` の `CONTEXT` という名前。TypeSpec の配置と要件 ID の文法に関わり、138 ファイルに及ぶため、別の作業項目で扱う。
- 汎用部分と IdMagic 固有部分の分離（wi-99632）。この作業項目で移すパスを前提に、あちらの対象ファイルを読み替える。
- 完了済みの作業項目と `docs/releases/` の本文の書き換え。

## 設計

### 参照する正式な文書

| 出典 | 借りるもの |
| --- | --- |
| ISO/IEC/IEEE 29148:2018 | 利害関係者要件、システム要件、要件仕様書の区別。上位の要件を構成要素へ割り当て、詳細な要件を下位で定める構造 |
| ISO/IEC/IEEE 42010:2022 | アーキテクチャ記述、関心事、ビュー、判断とその根拠をアーキテクチャ記述の一部とする扱い |
| arc42 | 第 2 章の制約、第 4 章の解決戦略、第 8 章の横断的概念、第 9 章の設計判断、第 11 章のリスクと技術的負債、第 12 章の用語集 |
| SWEBOK Guide V4 | Software Requirements、Software Architecture、Software Design の知識領域の区別。アーキテクチャ（全体の構造）と、構成要素ごとの設計の区別 |
| ISO/IEC 25010:2023 | 品質特性の網羅。セキュリティとインタラクション能力（旧版の使用性。アクセシビリティを含む）を品質要件の特性として扱う |

### 最上位の区分

`docs/README.md` の表を次の区分にする。
左の列は区分名を入口へのリンクにし、右の列に内容を書く。
冒頭の「製品の目的と範囲はプロダクト概要に示す」の文は削除する。

| 区分 | ディレクトリ | 内容 |
| --- | --- | --- |
| 要件文書 | `docs/requirements/` | プロダクト概要、機能要件、品質要件、外部規範への準拠、システム横断シナリオ、用語集 |
| 全体設計文書 | `docs/design/` | アーキテクチャ設計と、アプリケーション、データ、インフラストラクチャ、セキュリティ、信頼性、性能、オブザーバビリティ、検証の各設計 |
| モジュール設計文書 | `docs/modules/` | モジュールごとの責務、公開契約、機能仕様、内部設計 |
| 開発文書 | `docs/development/` | 変更なし |
| 運用文書 | `docs/operations/` | 変更なし |
| リファレンス | `spec/` | TypeSpec から生成した API リファレンスとモデルカタログ |
| フォーマット | `docs/formats/` | 変更なし |

- 区分名は、兄弟の区分（開発文書、運用文書）に合わせて「〜文書」で終える。
- 「全体設計」と「モジュール設計」は、SWEBOK V4 のアーキテクチャと構成要素の設計の区別に対応する。全体設計は構成要素への割り当てとシステムに共通する機構を、モジュール設計は割り当てを受けた一つのモジュールの内側を定める。
- リファレンスの行のリンク先は TypeSpec の入口 `spec/main.tsp` とする。生成サイトでは、描画器がこのリンクをリファレンスの索引へ解決する。リポジトリで読む人は一次情報へ、サイトで読む人は生成したビューへ着く。

採らない案：

| 案 | 採らない理由 |
| --- | --- |
| 要件文書を設計文書の下に置いたまま区分名だけ直す | 要件は設計の入力であり、29148 と 42010 のどちらも要件と設計の記述を別の情報項目とする。ディレクトリも分かれている |
| 区分名を「PRD」にする | PRD はプロダクトの企画文書を指すことが多く、品質要件、外部規範、横断シナリオを含む現在の範囲より狭い |
| リファレンスの行を表から外す | 区分の一覧から生成ビューの存在が読めなくなる |

### requirement の訳語

requirement は「要件」と訳す。
`docs/` の「要求文書」「機能要求」「品質要求」「システム要求」などを、文脈を読んで「要件」に置き換える。
「要求」がリクエストや依頼の意味で使われている箇所は変えないため、一括置換は使わない。
`check-terminology` に、置き換え後に戻らないための語を加える（「要求文書」「機能要求」「品質要求」）。

機能仕様の `REQ-*` はすでに「要件」と呼んでいるので、システムの機能要件からモジュールの要件までが同じ語でつながる。

### 要件文書の構成

| 文書 | 内容 |
| --- | --- |
| `requirements/README.md` | 区分の索引 |
| `requirements/product-overview.md` | 変更なし |
| `requirements/functional.md` | 機能要件。下で定める |
| `requirements/quality.md` | 品質要件。下で定める |
| `requirements/standards.md` | `docs/domain/standards.md` を移す。システム全体として準拠する外部規範 |
| `requirements/scenarios.feature.md` | `docs/domain/scenarios.feature.md` を移す。一つのモジュールでは起こせない振る舞いの規範シナリオ |
| `requirements/glossary.md` | `docs/domain/glossary.md` を移す。モジュールを跨いで意味が固定される語 |

用語集は arc42 では第 12 章にあたるが、ここで定める語は業務と外部契約の語彙であり、29148 の要件仕様書が定義の節に置く内容である。
設計の区分に置くと、要件文書が設計の区分の用語を参照する逆向きの依存が生じるため、要件文書に置く。

### 機能要件

`functional.md` は、システムの機能要件を機能群の単位で列挙し、各機能群をモジュールへ割り当てる文書とする。
29148 のシステム要件にあたり、個々の振る舞いは割り当て先のモジュールの機能仕様が `REQ-*` で定める。
粒度が粗いのはこの割り当ての構造による。冒頭でこのことを明示する。

現在の「主な機能」と「担当する仕様」の二つの表は、一つの表にまとめる。

| 列 | 内容 |
| --- | --- |
| 機能群 | 利用目的の単位の名前 |
| 利用者が行えること | 主な機能を一文から数文で書く。現在の「主な機能」の表の内容（認証要素、上流と下流の同期、上流 IdP との連携）はここへ入れる |
| 実現するモジュール | モジュール設計文書へのリンク |

### 品質要件

`quality.md` を、品質要件のすべての特性について入口となる一次情報にする。
現在の「品質特性の網羅状況」の節を、ISO/IEC 25010:2023 の特性ごとの表に書き換える。

| 列 | 内容 |
| --- | --- |
| 品質特性 | 25010:2023 の特性 |
| 要件 | この製品に求める水準。数値目標がある特性は既存の SLO、CAP の ID を参照する |
| 規範と検証 | 規範 id を定める文書（`requirements/standards.md`、モジュールの `standards.md`）と、検証設計の該当箇所 |

- セキュリティとインタラクション能力（アクセシビリティ）は、この表の行として要件を書く。WCAG 2.2 のような外部規範の条項 id は `requirements/standards.md` が定め、`quality.md` はその規範への準拠を要件として宣言する。
- 冒頭の「セキュリティとアクセシビリティの義務は全体の標準仕様と各モジュールの標準仕様で定める」と、末尾近くの同じ趣旨の文は削除する。
- 規範 id を `quality.md` へ移す案は採らない。`standards.md` の id は `//spec:covers` と標準の検査が参照する形式であり、移すと検査と既存テストの参照を変える必要がある。

### モジュール設計文書の配置

`docs/domain/` を `docs/modules/` へ移す。
`docs/modules/` の直下には `README.md` とモジュールのディレクトリだけを置く。

| 現在 | 移動先 |
| --- | --- |
| `docs/domain/<module>/` | `docs/modules/<module>/` |
| `docs/domain/README.md` | `docs/modules/README.md`。「`docs/domain/` は既存の文書パスであり」の一文と「システム全体」の節を削除する |
| `docs/domain/glossary.md`、`standards.md`、`scenarios.feature.md` | 要件文書へ（上の表） |
| `docs/domain/structure.md` | 下の「バックエンド設計と structure.md の分解」 |

完了済みの作業項目の `affected_spec` は旧パスのまま解決できなければならない。
`tools/check/relocated-spec-paths.json` はファイルごとの対応を列挙する形式であり、281 の完了記録が `docs/domain/` を参照するため、ディレクトリの接頭辞の対応（`docs/domain/` → `docs/modules/`）を一つ書ける形に広げる。
個別の移動（`scenarios.feature.md` など）は既存の形式で足す。

### 用語集の Module

Module の定義に、次の対応が一対一であることを書く。

- 論理アーキテクチャの責務表の一行
- `docs/modules/<module>/` のモジュール設計文書
- `backend/` の実装のモジュール

「モジュール設計文書」はこの単位ごとの文書の集まりを指す、と明記する。
TypeSpec の `spec/contexts/<context>/` との対応も責務表が定めることは変えない。

### 全体設計文書の索引と設計領域

「話題」を「設計領域」に改める。
文書ガイド §1 がすでに「領域別設計」「設計領域」の語で同じものを呼んでいるので、新しい語を足さずに既存の語へそろえる。

採らない案：

| 案 | 採らない理由 |
| --- | --- |
| 関心事（42010 の concern） | 42010 の関心事は利害関係者がシステムについて抱く問いであり、表の行（アプリケーション、データ、インフラストラクチャ）は文書の区分である。アーキテクチャの README がすでに各ビューの関心事の列でこの語を使っており、意味が重なる |
| 観点（viewpoint） | 42010 の観点はビューを作る規約であり、データやセキュリティの設計文書の区分とは異なる |
| 章（arc42 の chapter） | arc42 の章はアーキテクチャ記述の目次であり、インフラストラクチャや検証の設計を含まない |

設計領域の集合と順序を次のとおりにする。
文書ガイド §1 の軸（実現対象、次に複数の実現対象へ作用する設計領域、最後に検証）に合わせた順序である。

| 順序 | 設計領域 | システム | モジュール |
| --- | --- | --- | --- |
| 1 | アーキテクチャ設計 | `architecture/` | `architecture.md` |
| 2 | アプリケーション設計 | `application/` | `application.md` |
| 3 | データ設計 | `data/` | `data.md` |
| 4 | インフラストラクチャ設計 | `infrastructure/` | `infrastructure.md` |
| 5 | セキュリティ設計 | `security/` | `security.md` |
| 6 | 信頼性設計 | `reliability/` | `reliability.md` |
| 7 | 性能設計 | `performance/` | `performance.md` |
| 8 | オブザーバビリティ設計 | `observability/` | `observability.md` |
| 9 | 検証設計 | `verification/` | `verification.md` |

- 設計判断とリスクは設計領域から外す。arc42 は第 9 章と第 11 章を、42010 は判断と根拠を、どちらもアーキテクチャ記述の一部とする。システムでは `architecture/decisions.md` と `architecture/risks.md` のまま、アーキテクチャ設計の索引から参照する。モジュールでは `design/decisions.md` と、置いている場合は `design/risks.md` のまま、`design/README.md` の設計領域の表の下に「設計の記録」の表を置いて参照する。
- 採らない案は、`decisions.md` と `risks.md` を `docs/design/` の直下へ出すことである。判断とリスクがアーキテクチャ記述から外れ、上の二つの出典と食い違う。
- 生成サイトの目次、`DESIGN_TOPICS`（改名して設計領域の定数にする）、`SYSTEM_DOCUMENT_DIRECTORIES` の `docs/design/*` の並び、仕様フォーマットの語彙の表を、すべてこの順序にする。目次は `SYSTEM_DOCUMENT_DIRECTORIES` の順に従うので、並びの一次情報は設計領域の定数とし、`SYSTEM_DOCUMENT_DIRECTORIES` はそこから並べる。

索引の表は、システムとモジュールで次の形にそろえる。

```markdown
| 設計領域 | 内容 |
| --- | --- |
| [アーキテクチャ設計](architecture/README.md) | システム境界、制約、解決戦略、論理構成、実行構成、配置、設計判断、リスク |
| データ設計 | 該当なし：理由 |
```

- 左の列は設計領域の名前とし、文書があればその文書へのリンクにする。
- 右の列は内容の説明とし、文書がなければ「該当なし：」に続けて理由を書く。
- 検査は、左の列がリンクであることと、右の列が「該当なし：」で始まることの、ちょうど一方を求める。

`docs/design/README.md` の冒頭は「この文書は」で書き出し、全体設計とモジュール設計の関係を一段落で書き、モジュール設計文書へリンクする。

### アーキテクチャ設計の概要

`constraints.md` と `strategy.md` を `architecture/README.md` へ統合し、README をアーキテクチャ設計の概要と索引にする。

| 節 | 内容 | 対応する arc42 の章 |
| --- | --- | --- |
| 制約 | 現在の `constraints.md` の表 | 第 2 章 |
| 解決戦略 | 現在の `strategy.md` の表 | 第 4 章 |
| 文書 | 現在の索引の表に、設計判断とリスクを含める | — |

arc42 は第 2 章と第 4 章を短く書くことを想定しており、どちらも表一つで足りている。
独立した文書のままにすると、概要を読むために三つの文書を開く必要がある。
`strategy.md` の「Modular Monolith」の行は `logical.md#アーキテクチャ様式` を参照しており、統合後もその参照を保つ。

### バックエンド設計と structure.md の分解

`docs/domain/structure.md` の節を、内容の種類に応じて次へ移し、`structure.md` は削除する。

| 現在の節 | 移動先 |
| --- | --- |
| ディレクトリ、開発工程との対応 | `docs/development/` のリポジトリ構成の文書 |
| 技術構成 | `architecture/README.md` の制約（言語とツールチェーン）。重複する内容は一つにする |
| モジュールの内部構造、パッケージの三種類、モジュール間の依存規則、公開範囲と `internal/`、検査と負債、モジュール間イベント、HTTP ルーティング | 新設する `docs/design/application/backend.md`（バックエンド設計） |
| フロントエンドのコンポーネント構造 | `docs/design/application/frontend.md` |
| アーキテクチャ様式 | `architecture/logical.md` の同名の節へ統合する |

- バックエンド設計はフロントエンド設計と同じ構成（全体構成、コードの置き場所と依存の向き、テスト）を基本とし、`structure.md` にない内容は書き足さない。
- `structure.md#モジュール間の依存規則` などのアンカーへのリンクは、すべて新しい場所へ張り替える。
- `boundary-fitness.ts`、`agent-guidance.ts`、`spec-diff.ts`、`document-layout.ts` が `structure.md` のパスを参照しているので、新しいパスに合わせる。

### 生成サイト

| 変更 | 内容 |
| --- | --- |
| 出力パス | `docs/` 配下の Markdown は、すべてリポジトリのパスと同じ相対パスで `site/docs/<path>.html`（README は `index.html`）に出力する。`docs/README.md` は従来どおり `site/index.html` にも出力する |
| 目次 | 区分を `docs/README.md` の表と同じ順序と名前で並べる。要件文書を独立した区分にし、全体設計文書の中は設計領域の順にする |
| リファレンス | `spec/main.tsp` へのリンクを `reference/index.html` へ解決する |
| トレーサビリティ | `traceability/index.html`、`traceabilityPage`、目次のリンク、`traces.ts` と `collectTraces` を削除する |

- `api/`、`models/`、`reference/` は TypeSpec の生成ビューであり、`docs/` に対応する Markdown がないので最上位に残す。
- 出力パスをリポジトリのパスにそろえると、URL から元の Markdown が分かり、文書の間のリンクの変換も接頭辞を足すだけになる。
- 製品は未リリースであり、生成サイトの旧 URL からの転送は設けない。
- トレーサビリティのページを削除しても、`//spec:covers` の被覆の検査、`mise run traceability-strict`、`mise run spec-diff` は変わらない。ページを読む利用者はなく、検査の結果はこれらのタスクの出力で読む。

### 形式文書の分担

`specification-format.md` は、仕様の書式に加えて、`docs/` 全体の配置、設計の書式、生成サイトを扱っている。
この repo の「仕様」は、要素が外へ約束すること（責務、公開契約、要件、品質、TypeSpec）であり、それをどう実現するかを書く設計とは別の情報項目である。
ISO/IEC/IEEE 29148 は要件仕様を、ISO/IEC/IEEE 42010 はアーキテクチャ記述を、それぞれ別の情報項目として扱う。
名前と中身を一致させるため、形式文書を次の三つに分ける。

| 文書 | 扱う内容 |
| --- | --- |
| `formats/documentation-guide.md`（文書ガイド） | どの内容をどの文書に書くか、`docs/` の配置図、一次情報文書に共通する規則（H1、`README.md` による境界の宣言）、一次情報文書に置かない内容、生成ビュー |
| `formats/specification-format.md`（仕様フォーマット） | 仕様の用語、TypeSpec の範囲、モジュールの仕様（`README.md`、`glossary.md`）、機能仕様、仕様の木、状態遷移、標準仕様、要件と例、旧形式の扱い |
| `formats/design-format.md`（設計フォーマット、新設） | 設計ガイドライン、設計領域と索引、`design/decisions.md`、機能の設計、認可設計、脅威モデル |

- 配置図を読む検査（`check-document-layout.ts`）は、文書ガイドの配置図を読む。
- 節の番号を振り直すため、`specification-format.md#<節>` へのリンクは移した先へ張り替える。

採らない案：

| 案 | 採らない理由 |
| --- | --- |
| `specification-format.md` を「文書フォーマット」へ改名するだけにする | 仕様と設計の書式が一つの文書に残り、29148 と 42010 の区別に対応しない。配置図と内容の担当が文書ガイドと二か所に分かれたままになる |
| 配置図を `specification-format.md` に残す | 配置図は内容の担当を図にしたものであり、文書ガイドの担当である。仕様の書式の文書が設計や運用の文書の配置を定めることになる |

### 冒頭の書き出し

`docs/design/README.md`、`docs/design/architecture/README.md`、`docs/design/application/README.md` の「このディレクトリには」を「この文書は」に改める。
文書ガイドの規則はすでにあるので、`check-terminology` に、文書の最初の段落の「このディレクトリ」「本書は」「このファイルは」を拒否する規則を足す。

## 計画

1. 規則を先に書き換える。文書ガイド（§2、§4.1、§4.2、§4.3、§4.4）、仕様フォーマット（§1 の配置、設計領域の語彙、索引の形式）、作業項目フォーマットと仕様先行の開発ワークフローの `docs/domain` の例を、新しい区分、パス、語彙に合わせる。
2. 検査と描画器を新しい配置に合わせる。テストを先に新しい期待へ変え、失敗を確かめてから実装を変える。
3. ファイルを `git mv` で移す。`docs/domain/` から `docs/modules/` への移動と、全体文書の移動を同じコミットにし、リンクの張り替えを同じコミットに含める。
4. 文書の本文を書き直す（`docs/README.md`、`docs/design/README.md`、各モジュールの `design/README.md`、`architecture/README.md`、`functional.md`、`quality.md`、`backend.md`、`glossary.md` の Module）。
5. 「要求」から「要件」への置き換えを、一件ずつ文脈を読んで行う。

未解決の問いはない。

## タスク

- [x] T001 [Spec] 文書ガイド、仕様フォーマット、作業項目フォーマット、仕様先行の開発ワークフロー、`AGENTS.md`、skills の区分名、パス、設計領域の語彙を更新する。
- [x] T002 [Tooling] `document-layout.ts`、`specification-doc.ts`（設計領域の定数と索引の検査）、`check-terminology`、`relocated-spec-paths.json` の接頭辞の対応、`structure.md` を参照する検査を、テストを先に変えてから更新する。
- [x] T003 [Tooling] 描画器の出力パス、目次、リファレンスのリンクの解決を変え、トレーサビリティのページを削除する。
- [x] T004 [Docs] `docs/domain/` を `docs/modules/` へ、全体文書を `docs/requirements/` へ移し、リンクを張り替える。
- [x] T005 [Docs] `structure.md` を分解し、`backend.md` を新設し、`frontend.md` と `logical.md` へ統合する。
- [x] T006 [Docs] `constraints.md` と `strategy.md` を `architecture/README.md` へ統合する。
- [x] T007 [Docs] `docs/README.md`、`docs/design/README.md`、21 のモジュールの `design/README.md` の索引を新しい形式で書き直す。
- [x] T008 [Docs] `functional.md` と `quality.md` を書き直し、用語集の Module の定義を改める。
- [x] T009 [Docs] 「要求」を「要件」に置き換える。
- [x] T011 [Docs] `specification-format.md` を文書ガイド、仕様フォーマット、設計フォーマットへ分け、配置図の検査と参照を新しい文書へ向ける。
- [x] T010 [Verify] 検証する。

## 検証

- `mise run test-tools`
- `mise run check-links`
- `mise run check-terminology`
- `mise run check-spec`
- `mise run check-work-items`
- `mise run check-rendered-docs`
- `mise run verify`
- 生成サイトで、目次の区分と順序が `docs/README.md` の表と一致すること、`site/docs/modules/` に各モジュールが出力されること、リファレンスの行が `reference/index.html` へ着くことを目で確かめる。

## リスク

| リスク | 緩和策 |
| --- | --- |
| 移動したパスへのリンクの張り替え漏れ | `mise run check-links` がアンカーまで検査する。移動とリンクの張り替えを同じコミットにする |
| 完了済みの作業項目の `affected_spec` が解決できなくなる | 接頭辞の対応を足し、`mise run check-work-items` で全記録の解決を確かめる |
| 「要求」の置き換えで、リクエストの意味の「要求」まで変える | 一括置換をせず、差分を一件ずつ読む |
| wi-99632 と同じファイルを動かし、あちらの対象範囲の記述が古くなる | この作業項目の完了後に、wi-99632 の対象範囲のパスを読み替える |

## 完了
- **完了日**: 2026-10-10
- **要約**:
  `mise run spec-diff -- main` は「no normative specification change against main」を報告した。`REQ-*`、標準 id、TypeSpec の宣言は変わっていない。
  文書の区分を要件文書（`docs/requirements/`）、全体設計文書（`docs/design/`）、モジュール設計文書（`docs/modules/`）に分けた。`docs/domain/` は `docs/modules/` へ移した。直下にあった用語集、全体の標準仕様、システム横断シナリオは要件文書へ移した。`structure.md` は、新設したバックエンド設計、論理アーキテクチャ、フロントエンド設計、開発文書のリポジトリ構成へ分けた。
  設計の索引の「話題」は「設計領域」に改めた。設計判断とリスクは索引の行から外し、設計の記録として参照する形にした。設計領域の順序は一か所の定数で定め、索引の検査と生成サイトの目次が同じ順に従う。
  形式文書は、文書ガイド（配置と共通規則）、仕様フォーマット、新設した設計フォーマットの三つに分けた。理由の文書は `format-rationale.md` に改名し、同じ三つの構成に並べ直した。制約と解決戦略はアーキテクチャ設計の README に統合した。
  機能要件は、機能群と実現するモジュールの一つの表にした。品質要件には、ISO/IEC 25010:2023 の品質特性ごとの要件の表を置いた。requirement の訳語は「要件」にそろえ、`check-terminology` が旧い区分名と「このディレクトリには」「話題の索引」を拒否する。
  生成サイトは `docs/` の Markdown をリポジトリと同じパスへ出力する。目次は `docs/README.md` の区分の順に並べ、`spec/main.tsp` へのリンクをリファレンスの索引へ解決する。トレーサビリティのページは削除した。走査の処理は brief だけが使うので `tools/brief/` へ移した。
  完了記録の `affected_spec` は、`relocated-spec-paths.json` に足した接頭辞の対応（`docs/domain/` → `docs/modules/`）で解決する。完了記録とリリース文書の本文は書き換えていない。ただし、リンク検査を通すため、移したファイルへのリンク先だけは張り替えた。
  設計の索引と同じ問題が残っていた箇所も直した。対象は、設計領域の README 7 件の書き出し、IdManagement の機能の `design.md` 10 件にあった仕様フォーマットに反する索引の表、モジュールの README の「話題ごとの設計」の表記である。
- **受け入れ RED の証拠**:
  - **テスト**: `tools/check/src/feature-layout.acceptance.test.ts` の設計領域の索引の 6 件。
  - **要件**: N/A: 文書体系と検査ツールの変更であり、製品の要件を変えない。
  - **観測した失敗**: 索引の見出しを `| 設計領域 | 内容 |` とする期待へ変えた時点で、9 件が `must have a topic index` などで失敗した。
  - **検出できる理由**: 旧い検査は `| 話題 | 記述した場所 |` の表しか索引と認めないので、新しい形の索引を拒否する。リンクと「該当なし：」の両方を書いた行を拒否するテストも加え、ちょうど一方を求める規則を固定した。
- **単体 RED の証拠**:
  - **テスト**: `relocatedSpecPaths` の 3 件（`work-item-references.test.ts`）、配置図の検査（`document-layout-format.test.ts`）、TypeSpec の入口のリンクの解決（`render.test.ts`）、用語検査の 2 件（`terminology.test.ts`）。
  - **要件**: N/A: 検査ツールの変更であり、製品の要件を変えない。
  - **観測した失敗**: `relocatedSpecPaths` は export がなく失敗した。配置図の検査は、`### 配置` の節を見つけられず、配置図にある `docs/README.md` まで欠落として報告して失敗した。描画器は、TypeSpec へのリンクを解決せず、要件文書の索引も到達できないページとして失敗した。用語検査は新しい語を指摘せずに失敗した。
  - **検出できる理由**: 配置図の検査の旧いテストは、配置図を見つけられなくても欠落の報告だけを確かめていたので、見出しの変更を検出できなかった。配置図にある文書を報告しないことも確かめる形へ強めた。
- **変更耐性の結果**:
  `spec-diff.ts` の履歴のパスの読み替えから `docs/domain/` を外すと、`reports nothing when the documents move out of docs/domain` が失敗した。読み替えを戻すと通った。リスクが低い文書と道具の変更なので、体系的な変異は行っていない。
- **検証結果**:
  - `mise run test-tools` - 成功（930 件）
  - `mise run check-links`、`mise run check-terminology`、`mise run check-work-items`、`mise run check-spec` - 成功
  - `mise run render-docs` - 成功。目次が要件文書、全体設計文書、モジュール設計文書、開発文書、運用文書、リファレンス、フォーマットの順に並ぶことを確かめた。全体設計文書の中は設計領域の順に並ぶ。`site/docs/modules/` に各モジュールが出力され、`docs/README.md` のリファレンスの行は `reference/index.html` へ着く
  - `mise run verify` - 成功。最初の実行は `lint-tools` の未使用の import と整形で失敗し、直してから再実行した
