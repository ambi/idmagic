# Architecture Format

Regenerative Architecture が第2層で保つ **Architecture（設計）** ——システムの技術実現と構造の
現状正本——の正本フォーマットを定義する。ここに書かれた書式がマスターであり、機械検証用のスキーマ
（既定では JSON Schema）はこの文書からの派生物として扱う。

`REGENERATIVE_ARCHITECTURE.md` は Architecture を概念として規定し（§3.2）、本文書はその**記法**を
定める。SCL に対する `SPECIFICATION_CORE_LANGUAGE.md`、ワークアイテムに対する
`WORK_ITEM_FORMAT.md`、ADR に対する `ADR_FORMAT.md` と同じ役割を、Architecture に対して担う。
別プロジェクトで別の書式を採るときは、本文書だけを差し替える。

新規に Architecture を書くときは、**既存ファイルを開いて書式を確認しない**。本文書の書式に従う。
既存ファイルは「似た題材の中身」を参照したいときだけ開く。

Architecture は ADR とは役割が違う。ADR は決定という出来事の追記ログ、Architecture はその射影で
ある現在の設計である。決定の履歴は ADR に、現在の姿は Architecture に置く。

---

## 1 二つの成果物

Architecture は、答える相手が違う二つのファイルに分かれる（ADR-143）。

| ファイル | 中身 | 読み手 |
| --- | --- | --- |
| `ARCHITECTURE.md` | いまどうなっているか / なぜそうなっているか。散文の設計正本。 | 人間 |
| `architecture.yaml` | 構成を機械が検査できる宣言。module 台帳、依存、実行単位、複雑度 budget。 | `ra check` |

同じファイルに混ぜない。台帳は module 数に比例して増えるので、文書に同居させると設計として読めなく
なる。逆に散文を台帳へ入れると機械検証できない。

### 1.1 配置と命名

- ファイル名は `ARCHITECTURE.md` と `architecture.yaml`。配置は ADR・ワークアイテムと同じく、対象と
  なる境界づけられたコンテキストの近くに置く。
- リポジトリ全体・複数コンテキストにまたがる横断のものはワークスペースルート（接頭辞 `repo`）に置く。
- 特定コンテキストのものはそのコンテキストの実装ディレクトリ直下に置く
  （例: `backend/jobs/architecture.yaml`、`backend/jobs/ARCHITECTURE.md`、接頭辞 `jobs`）。
- **`architecture.yaml` は台帳なので、module を持つ全コンテキストに置く。**
  **`ARCHITECTURE.md` は必要になったコンテキストにだけ置く。** SCL に載らない設計が実在しない
  コンテキストに空の文書を置かない。ディレクトリが構造を叫ばなくなる。
- `ARCHITECTURE.md` を置くディレクトリには `architecture.yaml` も要る（`context` が一致すること）。
- Architecture は追記型のログではなく現状の射影なので、版を分けたファイルを増やさない。1 コンテキスト
  につき 1 組を更新し続ける。過去の決定経緯は ADR 側に残る。

### 1.2 言語

`ARCHITECTURE.md` の散文は **English で書く**（`README.md` と同じ扱い）。見出し名・フィールド名は
記法上の固定要素であり、本文書の英語見出しをそのまま使う。Frontmatter のキーは英語の識別子で固定する。

---

## 2 台帳（`architecture.yaml`）

**機械検証できる構造だけ**を置く。読み物としての情報（採用技術やディレクトリ構造の説明）は
`ARCHITECTURE.md` に書く。

| フィールド | 必須 | 内容 |
| --- | --- | --- |
| `context` | ✓ | この台帳が記述するコンテキスト接頭辞。横断はルートで `repo`。 |
| `updated_at` | ✓ | 最終更新日。`YYYY-MM-DD` または RFC3339。 |
| `contexts` | root のみ | 境界づけられたコンテキストの台帳。接頭辞 → オブジェクト（§2.2）。 |
| `modules` | △ | モジュール／パッケージ台帳。id → オブジェクト（§2.3）。 |
| `runtime_units` | root のみ | 実行単位の台帳。id → オブジェクト（§2.4）。 |
| `complexity` | root のみ | source の複雑度 budget と期限付き既存 debt（§2.5）。 |

### 2.1 台帳の分割規則

台帳は複数ファイルに分かれるが、検査は合成された単一のグラフに対して行う。

- **ある module は、その `path` を含む最も近い祖先の `architecture.yaml` が宣言する。** ルートの台帳が
  fallback になる。
- `modules[].path` は**その台帳ファイルからの相対パス**であり、その台帳のディレクトリ配下になければ
  ならない。
- `contexts` / `runtime_units` / `complexity` は横断整合検査の対象なので、**ルートの台帳だけ**が持つ。
- module id はワークスペース全体で一意。`depends_on` は台帳をまたいで解決する。

### 2.2 `contexts` の要素

各コンテキストは接頭辞をキーに次を持つ。

| キー | 必須 | 内容 |
| --- | --- | --- |
| `spec` | ✓ | そのコンテキストを定義する SCL YAML。root 台帳からの相対パス。実在しなければならない。 |
| `summary` | ✓ | そのコンテキストの一文説明。 |

### 2.3 `modules` の要素

各モジュールは次を持つ。

| キー | 必須 | 内容 |
| --- | --- | --- |
| `path` | ✓ | モジュールの実体があるディレクトリまたはファイル。宣言元 `architecture.yaml` からの相対パス。**実在し、宣言元ディレクトリ配下でなければならない**。 |
| `responsibility` | ✓ | そのモジュールが負う責務を一文で。 |
| `context` | ✓ | 所属する `contexts` の接頭辞。 |
| `layer` | ✓ | RA 7層のいずれか。内側から `specification_core`、`decision_record`、`domain`、`use_cases`、`adapters`、`infrastructure`、`deploy_pipeline`。 |
| `role` | ✓ | `implementation`、`published_interface`、`binding`、`technical_shared`、`composition_root` のいずれか。 |
| `realizes` | △ | ADR-114 の context-qualified direct SCL element reference の配列。参照先 context は module の `context` と一致し、要素へ解決できなければならない。 |
| `depends_on` | △ | 宣言依存の配列。各 edge は対象 module の `module` と、境界を通す役割 `via`（`published_interface`、`binding`、`technical_shared`、`composition_root`）を持つ。 |

依存先は同じ layer またはより内側だけに置く。cross-context import は context map の関係と、宣言した
edge の `via` に対応する役割を持つ module を介す。外部 package、生成物、vendor、`node_modules`、
`*_test.go`、`*.test.ts(x)`、`*.spec.ts(x)` は実 import の照合対象外とする。

### 2.4 `runtime_units` の要素

各実行単位は次を持つ。

| キー | 必須 | 内容 |
| --- | --- | --- |
| `kind` | ✓ | `api`、`worker`、`relay`、`batch`、`ui` のいずれか。 |
| `entrypoint` | ✓ | root 台帳からの相対パスで示す実在 entrypoint。 |
| `modules` | ✓ | この実行単位で composition する module id の配列。 |

### 2.5 `complexity` の要素

`complexity.budgets` は、`id`、対象 glob の `include`、任意の除外 glob `exclude`、測定する `metric`
（`source_lines` または `react_local_state_hooks`）、正整数の `limit` を持つ。最初に一致した debt では
なく、budget id と path の組で例外を特定する。

既存違反は `complexity.debts` に限り期限付きで許容する。各 debt は `id`、`budget`、`path`、現在値を
上限として固定する `ceiling`、`owner`、`reason`、解消先 `work_item`、`expires_at`（`YYYY-MM-DD`）を
必須とする。値が ceiling を超えた場合、対応 budget/debt が存在しない場合、または期限切れの場合は
検証を失敗させる。生成物と test source は budget の除外 glob で明示的に対象外とする。

---

## 3 設計正本（`ARCHITECTURE.md`）

Frontmatter は `context` と `updated_at` **だけ**を持つ。台帳は隣の `architecture.yaml` にある。

先頭に `# Architecture: <コンテキスト>` を H1（`#`）として 1 つ置き、各セクションは H2（`##`）で書く。
見出しレベル 1 はファイルに 1 つに保つ。

### 3.1 横断（`context: repo`）

次の 9 セクションをこの順で持つ。すべて必須。

| セクション見出し | 内容 |
| --- | --- |
| `## Overview` | 役割宣言と読む順序。この文書が何を持ち、何を持たないか。 |
| `## Structure` | ディレクトリ構造のツリー、依存の向き、RA レイヤ対応。 |
| `## Stack` | 採用言語・ランタイム・主要ツール。 |
| `## Context Map` | SCL context と実装 package の対応。 |
| `## Conventions` | package 規約、Adapter 命名、feature 垂直スライスなどの構造規約。 |
| `## Cross-cutting Concerns` | 認可・エラー処理・観測・routing・セキュリティ・永続化方針など層やモジュールをまたぐ方針。 |
| `## Runtime Composition` | bootstrap、DI、実行単位、可用性と共有状態。 |
| `## Structural Decisions` | 現在の構成を形づくった主要な構造判断の要点と、根拠となる ADR へのリンク。 |
| `## Documentation Policy` | どこに何を書くかの判断表。 |

`## Diagrams`（Mermaid 等のテキスト図）は任意で追加してよい。

### 3.2 コンテキスト単位

`## Overview` だけが必須で、そのコンテキストが何を所有しどう分割されているかを述べる。以降は
**話題別の H2 を自由に置ける**。台帳を持たない読み物としての設計文書である。

推奨の骨組み（必須ではない）:

```text
## Overview          このコンテキストが何を所有し、どう分割されているか
## <メカニズム名>     主要フロー・プロトコルの動作（複数可）
## Conventions       このコンテキスト固有の規約
## Design Decisions  主要判断の要点と根拠 ADR へのリンク
```

### 3.3 根拠のインライン化

設計記述には、**その形になった理由を1〜2文で添える**。読み手が設計を理解するために ADR を開かなくて
よい状態を保つのが目的である。却下した選択肢と当時の前提の全文は ADR に置き、そこへリンクする。

**ADR 本文を転記しない。** 複製は必ず drift する（`ARCHITECTURE.md` が ADR-082 の tenant_id ルールを
複製し、ADR-083 が置き換えた後も古い方を載せ続けた実例がある）。移送するなら移送し、ADR 側には
移送先へのポインタと「なぜ」だけを残す（`ADR_FORMAT.md`「役割の境界」）。

---

## 4 整合規則（SCL・実ツリーとの突き合わせ）

Architecture は現状の射影なので、現実と乖離していないことを検査できる。次を満たす。

1. **台帳の合成**: 全 `architecture.yaml` の `modules` を 1 つのグラフへ合成する。module id は一意で、
   `path` は宣言元ディレクトリ配下であり、`contexts`/`runtime_units`/`complexity` は root だけが持つ。
2. **コンテキスト整合**: `context` と `contexts` の接頭辞・`spec` は SCL workspace context map と一致する。
3. **実体と参照**: module path と runtime entrypoint が実在し、module、runtime、budget/debt の参照が解決する。
4. **実現参照の解決**: `realizes` は direct SCL reference として解決し、その context は module context と一致する。
5. **依存方向**: module graph は循環せず、RA layer は同層または内向きで、cross-context edge は宣言した役割を通る。
6. **実 import**: Go module path と TypeScript 相対 import を workspace path に正規化し、最長 module path prefix で割り当て、宣言依存と一致させる。
7. **複雑度 ratchet**: budget 超過、debt ceiling 増加、未登録・不完全・期限切れ debt を拒否する。
8. **必須フィールド**: 台帳が Architecture map schema に、設計正本が Architecture doc schema に適合する。
9. **文書と台帳の対応**: `ARCHITECTURE.md` の `context` は同じディレクトリの `architecture.yaml` と一致する。
10. **ADR リンクの実在**: 設計正本とワークアイテムが参照する `decisions/ADR-*.md` が実在する。

これらは `ra check`（`ra verify` 経由）で機械検証する。いずれかに反する Architecture は検証を落とし、
コア構造に触れた変更の完了ゲートを塞ぐ。

---

## 5 スケルトン

### 5.1 `architecture.yaml`（root）

```yaml
context: repo               # コンテキスト接頭辞。横断はルートで repo
updated_at: 2026-01-01      # YYYY-MM-DD
contexts:
  Example: { spec: "spec/contexts/example.yaml", summary: "例のコンテキスト" }
modules:
  shared-thing:
    path: backend/shared/thing
    responsibility: "一文で表す責務"
    context: Example
    layer: adapters
    role: technical_shared
runtime_units:
  example-api:
    kind: api
    entrypoint: backend/cmd/example/main.go
    modules: [shared-thing]
complexity:
  budgets:
    - id: go-source-lines
      include: ["**/*.go"]
      exclude: ["**/*_test.go", "**/generated/**"]
      metric: source_lines
      limit: 800
  debts:
    - id: legacy-large-file
      budget: go-source-lines
      path: backend/example/legacy.go
      ceiling: 920
      owner: platform
      reason: "段階的な分割が必要"
      work_item: wi-999-split-legacy-file
      expires_at: 2026-10-01
```

### 5.2 `architecture.yaml`（コンテキスト単位）

```yaml
context: example
updated_at: 2026-01-01
modules:
  example-domain:
    path: domain              # この architecture.yaml からの相対
    responsibility: "一文で表す責務"
    context: Example
    layer: domain
    role: published_interface
  example-usecases:
    path: usecases
    responsibility: "一文で表す責務"
    context: Example
    layer: use_cases
    role: implementation
    realizes:
      - { context: Example, kind: interface, element: DoSomething }
    depends_on:
      - { module: example-domain, via: published_interface }
```

### 5.3 `ARCHITECTURE.md`

````markdown
---
context: repo               # コンテキスト接頭辞。横断はルートで repo
updated_at: 2026-01-01      # YYYY-MM-DD
---

# Architecture: repo

## Overview
What this document holds, what it does not, and the order to read things in.

## Structure

```text
.
├── backend/    # one-line responsibility
└── ...
```

The direction of dependencies, in a sentence.

## Stack
Languages, runtimes, principal tools.

## Context Map
SCL contexts against implementation packages.

## Conventions
Package layout, adapter naming, feature slices.

## Cross-cutting Concerns
Authorization, error handling, observability, routing, persistence policy.

## Runtime Composition
Bootstrap, DI, runtime units, availability and shared state.

## Structural Decisions
The main structural judgements, each with a short reason and a link to its ADR.

## Documentation Policy
The table that decides where a given piece of writing goes.
````

---

Architecture が **なぜ** その構成なのかの全文は ADR（`ADR_FORMAT.md`）に、**何を** 一つの変更で行うかは
ワークアイテム（`WORK_ITEM_FORMAT.md`）に置く。本文書は **いまどういう設計か** の記法だけを定める。
