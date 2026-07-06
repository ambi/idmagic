---
name: new-architecture
description: Create or synchronize an ARCHITECTURE.md — the second-layer current-state map (structured contexts and modules in frontmatter; stack, directory structure and dependency direction in prose). Use when core structure changes (a new bounded context, module, adopted technology, or directory convention), or when the user asks to draft/update the architecture map.
---

# ARCHITECTURE.md（構成）の作成・同期

正本書式は `ARCHITECTURE_FORMAT.md`。**既存ファイルを開いて書式を逆算しない**。ARCHITECTURE は
第2層の現状射影（`REGENERATIVE_ARCHITECTURE.md §3.2.1`）であり、ADR が決定の履歴を、これが現在の
構成を持つ。**なぜ**は ADR、**何を一つの変更で**はワークアイテム、**いまどういう構成か**が本ファイル。

## いつ更新するか

コア構造に触れたら同期する。純粋な仕様（SCL）の追加・修正だけなら不要。

- 境界づけられたコンテキストの追加・変更
- モジュール／パッケージの新設・責務変更・実現する SCL 要素の増減
- 採用スタックの変更
- ディレクトリ・命名規約の変更

## 手順

1. **配置先を決める**（`ARCHITECTURE_FORMAT.md §1`）
   - リポジトリ横断はルートの `ARCHITECTURE.md`（`context: repo`）。特定コンテキストはその配下
     （例 `<app>/ARCHITECTURE.md`、`context: <app>`）。接頭辞は `CHANGE_RECORD_FORMAT.md §1.1.1`。
   - 追記型ログではないので版を分けたファイルを増やさない。1 コンテキスト 1 ファイルを更新し続ける。
2. **Frontmatter（機械検証する構造）を現状に合わせる**（`§2`）
   - 必須は `context` と `updated_at`。`contexts` と `modules` を実態に一致させる。frontmatter には
     機械検証する構造だけを置き、採用技術・依存・ディレクトリ構造は本文へ回す（肥大させない）。
   - `modules[].path` は実在するパス、`modules[].realizes` は実在する SCL 要素
     （`interfaces.Xxx` / `models.Yyy` 等）を指す。
3. **本文（叙述）を書く**（`§3`）— 先頭に H1 を 1 つ、各セクションは H2。`## Overview` /
   `## Structure`（ディレクトリツリー＋依存の向き）/ `## Structural Decisions` は必須。構造判断は
   根拠 ADR へリンクし、履歴は再説せず ADR を指す。`## Stack` / `## Cross-cutting Concerns` /
   `## Diagrams` は任意。
4. `updated_at` を更新する。
5. **検証**: `just yaml-check`。スキーマ（Frontmatter）と横断整合検査（modules パス実在・realizes
   の SCL 要素解決・contexts 整合）の両方を通す。落ちたら地図が現実と乖離しているので直す。

## スケルトン

`ARCHITECTURE_FORMAT.md §5` のスケルトンに従う。
