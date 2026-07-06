# Architecture Format

Regenerative Architecture が第2層で保つ **Architecture（構成）** ——システムの技術実現と構造の
現状正本——の正本フォーマットを定義する。ここに書かれた書式がマスターであり、機械検証用のスキーマ
（既定では JSON Schema）はこの文書からの派生物として扱う。

`REGENERATIVE_ARCHITECTURE.md` は Architecture を概念として規定し（§3.2 / §3.2.1）、本文書はその
**記法**を定める。SCL に対する `SPECIFICATION_CORE_LANGUAGE.md`、変更管理レコードに対する
`CHANGE_RECORD_FORMAT.md` と同じ役割を、Architecture に対して担う。別プロジェクトで別の書式を採る
ときは、本文書だけを差し替える。

新規に Architecture を書くときは、**既存ファイルを開いて書式を確認しない**。本文書の書式に従う。
既存ファイルは「似た題材の中身」を参照したいときだけ開く。

Architecture は ADR とは役割が違う。ADR は決定という出来事の追記ログ、Architecture はその射影で
ある現在の構成である。決定の履歴は ADR に、現在の姿は Architecture に置く。

見出し名・フィールド名は記法上の固定要素である。本文書は日本語の見出しを正本とし、他言語
プロジェクトは構造を保ったまま翻訳してよい。Frontmatter のキーは英語の識別子で固定する。

---

## 1 配置と命名

- ファイル名は `ARCHITECTURE.md`。配置は ADR・ワークアイテムと同じく、対象となる境界づけられた
  コンテキストの近くに置く（`CHANGE_RECORD_FORMAT.md` §1.1 と同じ規約）。
- リポジトリ全体・複数コンテキストにまたがる横断の構成はルートの `ARCHITECTURE.md`（接頭辞 `repo`）
  に置く。特定コンテキストの構成はそのコンテキスト配下の `ARCHITECTURE.md`（例：`<app>` という
  境界なら `<app>/ARCHITECTURE.md`、接頭辞 `<app>`）に置く。接頭辞とディレクトリの対応は
  `CHANGE_RECORD_FORMAT.md` §1.1.1 の一元表に従う。
- Architecture は追記型のログではなく現状の射影なので、版を分けたファイルを増やさない。1 コンテキスト
  につき 1 ファイルを更新し続ける。過去の決定経緯は ADR 側に残る。

## 2 フィールド（Frontmatter メタデータ）

Frontmatter には**機械検証できる構造だけ**を置く。読み物としての情報（採用技術・依存の向き・
ディレクトリ構造）は本文（§3）に書く。こうして Frontmatter が肥大せず、人が一目で読める分量に保つ。

| フィールド | 必須 | 内容 |
| --- | --- | --- |
| `context` | ✓ | このマップが記述するコンテキスト接頭辞。横断はルートで `repo`。§1.1.1 の表と一致する。 |
| `updated_at` | ✓ | 最終更新日。`YYYY-MM-DD` または RFC3339。 |
| `contexts` | △ | 境界づけられたコンテキストの台帳。接頭辞 → オブジェクト（§2.2）。 |
| `modules` | △ | モジュール／パッケージ台帳。id → オブジェクト（§2.1）。 |

`contexts` と `modules` は横断整合検査の対象になる（§4）。地図として意味を持たせるには、少なくとも
`modules` を持ち、採用技術やディレクトリ構造を本文で述べる。

### 2.1 `modules` の要素

各モジュールは次を持つ。

| キー | 必須 | 内容 |
| --- | --- | --- |
| `path` | ✓ | モジュールの実体があるディレクトリまたはファイル。`ARCHITECTURE.md` の位置から見た相対パス。**実在しなければならない**（横断検査で確認する）。 |
| `responsibility` | ✓ | そのモジュールが負う責務を一文で。 |
| `realizes` | △ | そのモジュールが実現する SCL 要素の参照配列（例 `interfaces.DiscoverWorkspace`、`models.WorkspaceApp`）。**ワークスペース内の SCL 要素に解決できなければならない**（横断検査で確認する）。 |

### 2.2 `contexts` の要素

各コンテキストは接頭辞をキーに次を持つ。

| キー | 必須 | 内容 |
| --- | --- | --- |
| `root` | ✓ | そのコンテキストのルートディレクトリ。 |
| `summary` | △ | そのコンテキストの一文説明。 |

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

1. **コンテキスト整合**: `context` と `contexts` の接頭辞は `CHANGE_RECORD_FORMAT.md` §1.1.1 の
   一元表、および SCL のコンテキスト宣言（`bounded_contexts` / `context_map`）と矛盾しない。
2. **モジュール実在**: すべての `modules[].path` がワークスペース内に実在する。
3. **実現参照の解決**: すべての `modules[].realizes` がワークスペース内の SCL 要素に解決する。
4. **必須フィールド**: `context` と `updated_at` を持ち、Frontmatter がスキーマに適合する。

これらは `ra yaml-check`（`ra verify` 経由）で機械検証する。いずれかに反する `ARCHITECTURE.md` は
検証を落とし、コア構造に触れた変更の完了ゲートを塞ぐ。

## 5 スケルトン

````markdown
---
context: repo               # コンテキスト接頭辞。横断はルートで repo
updated_at: 2026-01-01      # YYYY-MM-DD
contexts:
  repo: { root: ".", summary: "手法・横断ツール" }
modules:
  example-module:
    path: tools/example
    responsibility: "一文で表す責務"
    realizes: [interfaces.DoSomething, models.Thing]
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

Architecture が **なぜ** その構成なのかは ADR（`CHANGE_RECORD_FORMAT.md` §2）に、**何を** 一つの
変更で行うかはワークアイテム（同 §1）に置く。本文書は **いまどういう構成か** の記法だけを定める。
