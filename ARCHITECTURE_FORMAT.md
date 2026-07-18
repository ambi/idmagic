# Architecture Format

Regenerative Architecture が第2層で保つ **Architecture（構成）** ——システムの技術実現と構造の
現状正本——の正本フォーマットを定義する。ここに書かれた書式がマスターであり、機械検証用のスキーマ
（既定では JSON Schema）はこの文書からの派生物として扱う。

`REGENERATIVE_ARCHITECTURE.md` は Architecture を概念として規定し（§3.2 / §3.2.1）、本文書はその
**記法**を定める。SCL に対する `SPECIFICATION_CORE_LANGUAGE.md`、ワークアイテムに対する
`WORK_ITEM_FORMAT.md`、ADR に対する `ADR_FORMAT.md` と同じ役割を、Architecture に対して担う。
別プロジェクトで別の書式を採るときは、本文書だけを差し替える。

新規に Architecture を書くときは、**既存ファイルを開いて書式を確認しない**。本文書の書式に従う。
既存ファイルは「似た題材の中身」を参照したいときだけ開く。

Architecture は ADR とは役割が違う。ADR は決定という出来事の追記ログ、Architecture はその射影で
ある現在の構成である。決定の履歴は ADR に、現在の姿は Architecture に置く。

見出し名・フィールド名は記法上の固定要素である。本文書は日本語の見出しを正本とし、他言語
プロジェクトは構造を保ったまま翻訳してよい。Frontmatter のキーは英語の識別子で固定する。

---

## 1 配置と命名

- ファイル名は `ARCHITECTURE.md`。配置は ADR・ワークアイテムと同じく、対象となる境界づけられた
  コンテキストの近くに置く（`ADR_FORMAT.md` と `WORK_ITEM_FORMAT.md` の配置規約に揃える）。
- リポジトリ全体・複数コンテキストにまたがる横断の構成はルートの `ARCHITECTURE.md`（接頭辞 `repo`）
  に置く。特定コンテキストの構成はそのコンテキスト配下の `ARCHITECTURE.md`（例：`<app>` という
  境界なら `<app>/ARCHITECTURE.md`、接頭辞 `<app>`）に置く。接頭辞とディレクトリの対応は
  実際の ADR・ワークアイテム配置と SCL の context map に従う。
- Architecture は追記型のログではなく現状の射影なので、版を分けたファイルを増やさない。1 コンテキスト
  につき 1 ファイルを更新し続ける。過去の決定経緯は ADR 側に残る。

## 2 フィールド（Frontmatter メタデータ）

Frontmatter には**機械検証できる構造だけ**を置く。読み物としての情報（採用技術やディレクトリ構造）は
本文（§3）に書き、依存の向き、runtime composition、複雑度上限は検証可能な宣言として Frontmatter に置く。

| フィールド | 必須 | 内容 |
| --- | --- | --- |
| `context` | ✓ | このマップが記述するコンテキスト接頭辞。横断はルートで `repo`。§1.1.1 の表と一致する。 |
| `updated_at` | ✓ | 最終更新日。`YYYY-MM-DD` または RFC3339。 |
| `contexts` | △ | 境界づけられたコンテキストの台帳。接頭辞 → オブジェクト（§2.1）。 |
| `modules` | △ | モジュール／パッケージ台帳。id → オブジェクト（§2.2）。 |
| `runtime_units` | △ | 実行単位の台帳。id → オブジェクト（§2.3）。 |
| `complexity` | △ | source の複雑度 budget と期限付き既存 debt（§2.4）。 |

`contexts` と `modules` は横断整合検査の対象になる（§4）。地図として意味を持たせるには、少なくとも
`modules` を持ち、採用技術やディレクトリ構造を本文で述べる。

### 2.1 `contexts` の要素

各コンテキストは接頭辞をキーに次を持つ。

| キー | 必須 | 内容 |
| --- | --- | --- |
| `spec` | ✓ | そのコンテキストを定義する SCL YAML。Architecture からの相対パス。実在しなければならない。 |
| `summary` | ✓ | そのコンテキストの一文説明。 |

### 2.2 `modules` の要素

各モジュールは次を持つ。

| キー | 必須 | 内容 |
| --- | --- | --- |
| `path` | ✓ | モジュールの実体があるディレクトリまたはファイル。`ARCHITECTURE.md` の位置から見た相対パス。**実在しなければならない**（横断検査で確認する）。 |
| `responsibility` | ✓ | そのモジュールが負う責務を一文で。 |
| `context` | ✓ | 所属する `contexts` の接頭辞。 |
| `layer` | ✓ | RA 7層のいずれか。内側から `specification_core`、`decision_record`、`domain`、`use_cases`、`adapters`、`infrastructure`、`deploy_pipeline`。 |
| `role` | ✓ | `implementation`、`published_interface`、`binding`、`technical_shared`、`composition_root` のいずれか。 |
| `realizes` | △ | ADR-114 の context-qualified direct SCL element reference の配列。参照先 context は module の `context` と一致し、要素へ解決できなければならない。 |
| `depends_on` | △ | 宣言依存の配列。各 edge は対象 module の `module` と、境界を通す役割 `via`（`published_interface`、`binding`、`technical_shared`、`composition_root`）を持つ。 |

依存先は同じ layer またはより内側だけに置く。cross-context import は context map の関係と、宣言した
edge の `via` に対応する役割を持つ module を介す。外部 package、生成物、vendor、`node_modules`、
`*_test.go`、`*.test.ts(x)`、`*.spec.ts(x)` は実 import の照合対象外とする。

### 2.3 `runtime_units` の要素

各実行単位は次を持つ。

| キー | 必須 | 内容 |
| --- | --- | --- |
| `kind` | ✓ | `api`、`worker`、`relay`、`batch`、`ui` のいずれか。 |
| `entrypoint` | ✓ | Architecture からの相対パスで示す実在 entrypoint。 |
| `modules` | ✓ | この実行単位で composition する module id の配列。 |

### 2.4 `complexity` の要素

`complexity.budgets` は、`id`、対象 glob の `include`、任意の除外 glob `exclude`、測定する `metric`
（`source_lines` または `react_local_state_hooks`）、正整数の `limit` を持つ。最初に一致した debt ではなく、
budget id と path の組で例外を特定する。

既存違反は `complexity.debts` に限り期限付きで許容する。各 debt は `id`、`budget`、`path`、現在値を
上限として固定する `ceiling`、`owner`、`reason`、解消先 `work_item`、`expires_at`（`YYYY-MM-DD`）を
必須とする。値が ceiling を超えた場合、対応 budget/debt が存在しない場合、または期限切れの場合は
検証を失敗させる。生成物と test source は budget の除外 glob で明示的に対象外とする。

## 3 本文（Markdown セクション）

先頭に `# Architecture: <コンテキスト>` を H1（`#`）として 1 つ置き、各セクションは H2（`##`）で
書く。見出しレベル 1 はファイルに 1 つに保つ。次の見出しをこの順で持つ。

| セクション見出し | 必須 | 内容 |
| --- | --- | --- |
| `## Overview` | ✓ | 構成の全体像を人間の言葉で。何をどう分割し、どこに何があるか。 |
| `## Structure` | ✓ | ディレクトリ構造をコードブロックのツリーで示し、依存の向きを一文で述べる。frontmatter にキー→パスの羅列で無理に押し込まない。 |
| `## Stack` | △ | 採用言語・ランタイム・主要ツール。 |
| `## Structural Decisions` | ✓ | 現在の構成を形づくった主要な構造判断の要点と、根拠となる ADR へのリンク。判断の履歴は再説せず ADR を指す。 |
| `## Cross-cutting Concerns` | △ | 認可・エラー処理・観測・設定など、層やモジュールをまたぐ方針。 |
| `## Diagrams` | △ | コンテキスト図・モジュール依存図など。Mermaid 等のテキスト図を推奨する。 |

## 4 整合規則（SCL・実ツリーとの突き合わせ）

Architecture は現状の射影なので、現実と乖離していないことを検査できる。次を満たす。

1. **コンテキスト整合**: `context` と `contexts` の接頭辞・`spec` は SCL workspace context map と一致する。
2. **実体と参照**: module path と runtime entrypoint が実在し、module、runtime、budget/debt の参照が解決する。
3. **実現参照の解決**: `realizes` は direct SCL reference として解決し、その context は module context と一致する。
4. **依存方向**: module graph は循環せず、RA layer は同層または内向きで、cross-context edge は宣言した役割を通る。
5. **実 import**: Go module path と TypeScript 相対 import を workspace path に正規化し、最長 module path prefix で割り当て、宣言依存と一致させる。
6. **複雑度 ratchet**: budget 超過、debt ceiling 増加、未登録・不完全・期限切れ debt を拒否する。
7. **必須フィールド**: Frontmatter が Architecture schema に適合する。

これらは `ra yaml-check`（`ra verify` 経由）で機械検証する。いずれかに反する `ARCHITECTURE.md` は
検証を落とし、コア構造に触れた変更の完了ゲートを塞ぐ。

## 5 スケルトン

````markdown
---
context: repo               # コンテキスト接頭辞。横断はルートで repo
updated_at: 2026-01-01      # YYYY-MM-DD
contexts:
  example: { spec: "spec/scl.yaml", summary: "例のコンテキスト" }
modules:
  example-module:
    path: tools/example
    responsibility: "一文で表す責務"
    context: example
    layer: adapters
    role: implementation
    realizes:
      - { context: example, kind: interface, element: DoSomething }
    depends_on:
      - { module: example-domain, via: published_interface }
runtime_units:
  example-api:
    kind: api
    entrypoint: cmd/example/main.go
    modules: [example-module]
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
      path: tools/example/legacy.go
      ceiling: 920
      owner: platform
      reason: "段階的な分割が必要"
      work_item: wi-999-split-legacy-file
      expires_at: 2026-10-01
---

# Architecture: repo

## Overview
構成の全体像を人間の言葉で。

## Structure

```text
.
├── tools/example    # 一文の責務
└── ...
```

依存の向きを一文で（例：ra -> 各ツール、SCL -> 派生物、逆流禁止）。

## Stack
- 言語 / ランタイム / 主要ツール。

## Structural Decisions
- 主要な構造判断の要点と根拠 ADR（例: repo-ADR-001）へのリンク。

## Cross-cutting Concerns
- 認可・エラー処理・観測・設定などの横断方針。

## Diagrams
（コンテキスト図・モジュール依存図。Mermaid 等）
````

---

Architecture が **なぜ** その構成なのかは ADR（`ADR_FORMAT.md`）に、**何を** 一つの変更で行うかは
ワークアイテム（`WORK_ITEM_FORMAT.md`）に置く。本文書は **いまどういう構成か** の記法だけを定める。
