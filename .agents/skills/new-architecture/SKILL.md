---
name: new-architecture
description: Create or synchronize the second-layer Architecture — the prose design record (ARCHITECTURE.md, English) and the machine-checked ledger beside it (architecture.yaml). Use when core structure changes (a new bounded context, module, adopted technology, or directory convention), when design currently trapped in an ADR or README should be moved, or when the user asks to draft/update the architecture.
---

# Architecture（設計）の作成・同期

正本書式は `ARCHITECTURE_FORMAT.md`。**既存ファイルを開いて書式を逆算しない**。Architecture は
第2層の現状射影（`REGENERATIVE_ARCHITECTURE.md §3.2`）であり、ADR が決定の履歴を、これが現在の
設計を持つ。**なぜ**の全文は ADR、**何を一つの変更で**はワークアイテム、**いまどういう設計か**が
本ファイル。

## 二つの成果物のどちらを触るか

| 変更 | 更新先 |
| --- | --- |
| module の新設・移動・責務変更・依存の増減、実行単位、複雑度 budget/debt | `architecture.yaml`（台帳） |
| 構造の説明、規約、横断方針、メカニズムの動作、その形にした理由 | `ARCHITECTURE.md`（設計正本） |

台帳と散文を同じファイルに混ぜない（ADR-143）。

## いつ更新するか

コア構造に触れたら同期する。純粋な仕様（SCL）の追加・修正だけなら不要。

- 境界づけられたコンテキストの追加・変更
- モジュール／パッケージの新設・責務変更・実現する SCL 要素の増減
- 採用スタックの変更
- ディレクトリ・命名規約の変更
- 主要フロー・プロトコルの動作、データモデル設計、横断方針の変更
- ADR や README に設計本文が残っているのを見つけたとき（移送する）

## 手順

1. **配置先を決める**（`ARCHITECTURE_FORMAT.md §1.1`）
   - リポジトリ横断はワークスペースルート（`context: repo`）。特定コンテキストはその実装ディレクトリ
     直下（例 `backend/jobs/`、`context: jobs`）。
   - `architecture.yaml` は module を持つ全コンテキストに置く。`ARCHITECTURE.md` は SCL に載らない
     設計が実在するコンテキストにだけ置く。空の文書を作らない。
   - 追記型ログではないので版を分けたファイルを増やさない。1 コンテキスト 1 組を更新し続ける。
2. **台帳を現状に合わせる**（`§2`）
   - module は `path` を含む最も近い祖先の `architecture.yaml` が宣言する。`path` はその台帳からの
     相対で、台帳のディレクトリ配下でなければならない。
   - `contexts` / `runtime_units` / `complexity` は root の台帳だけが持つ。
   - `modules[].realizes` は実在する SCL 要素を指す。
3. **設計正本を書く**（`§3`）— **English で書く**。先頭に H1 を 1 つ、各セクションは H2。
   - 横断（`context: repo`）は 9 セクション必須: Overview / Structure / Stack / Context Map /
     Conventions / Cross-cutting Concerns / Runtime Composition / Structural Decisions /
     Documentation Policy。
   - コンテキスト単位は `## Overview` だけ必須。以降は話題別の H2 を自由に置く。
   - **根拠をインラインで添える**（`§3.3`）: 設計記述には、その形になった理由を1〜2文書き、ADR へ
     リンクする。**ADR 本文を転記しない**——複製は必ず drift する。
4. 両ファイルの `updated_at` を更新する。
5. **検証**: `just check`。台帳スキーマ・設計正本スキーマ・横断整合検査（module path 実在、
   realizes の SCL 要素解決、contexts 整合、依存方向、実 import、複雑度 ratchet、ADR リンク実在）を
   すべて通す。落ちたら地図が現実と乖離しているので直す。

## 設計を書くのか ADR を書くのか

設計を記述したいだけなら **ADR を起こさない**。ADR は実際に分岐があり、却下した選択肢が実在する
ときだけ書く（`ADR_FORMAT.md`「ADR を書く条件」）。既存 ADR に設計本文が残っていたら、この Skill で
設計正本へ移し、ADR にはポインタと「なぜ」だけを残す。

## スケルトン

`ARCHITECTURE_FORMAT.md §5` のスケルトンに従う。
