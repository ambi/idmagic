---
name: new-work-item
description: Create a new Regenerative Architecture work item under work-items/ following the canonical format. Use when starting a new unit of work, a task, or when the user asks to draft/open a work item or wi-NN.
---

# 新規ワークアイテムの作成

正本書式は `CHANGE_RECORD_FORMAT.md §1`。**既存ファイルを開いて書式を逆算しない**——
本 Skill と §1 の記法に従う。既存ファイルは「似た題材の中身」を見たいときだけ開く。

## 手順

1. **配置先を決め、接頭辞を選ぶ**（§1.1 / §1.1.1）
   - 特定アプリ / コンテキストの作業 → その近くの `work-items/`（例 `apps/foo/work-items/`、接頭辞 `foo`）。
   - リポジトリ全体の規約・横断ツール・複数コンテキスト跨ぎ → ルートの `work-items/`（接頭辞 `repo`）。
2. **id を採番する**
   - そのコンテキストの既存最大連番を確認（接頭辞ごとに別空間）:
     `ls <app>/work-items <app>/work-items/done 2>/dev/null` または
     `ls work-items work-items/done 2>/dev/null`
   - `<id>` = `<接頭辞>-wi-<連番>-<kebab-title>`。ファイル名は `work-items/<id>.md`。
   - 連番は正本ではない。並行採番で被っても `check-ids` が検出するので、被ったら採り直す（§1.1.2）。
3. **未着手・進行中は `work-items/` 直下に置く**。完了・中止になったら `done/` サブディレクトリへ移す（id は変えない）。
4. 機能変更なら **触れる SCL セクションを `## Scope` に列挙する**。判定は SCL-first の網羅表に従う（`scl-change` Skill / `SPECIFICATION_CORE_LANGUAGE.md §3` 冒頭）。
5. 中規模以上なら `## Plan` と `## Tasks` を追加する。複数 scenario、RA 3 層以上、DB migration / 認可 / 外部契約 / 破壊的変更、1 セッションで終わる確信がない作業、複数サブシステム検証は中規模以上として扱う。
6. 下記スケルトンを埋める。見出しレベル 1 は `title` の 1 つだけにし、各セクションは H2 にする。`## Motivation` は **Why を書く（What ではない）**。
7. **検証**: `just yaml-check-work-items` と `just check-ids` を通す。

## スケルトン

```markdown
---
id: repo-wi-NN-kebab-title
title: "一文で表す意味変更"
created_at: YYYY-MM-DD
authors: [name]
status: pending
risk: low
---

# 一文で表す意味変更

## Motivation
なぜこの変更が必要か（Why）。

## Scope
- 対象範囲。機能変更なら触れる SCL セクションを含める。

## Out of Scope
- 明示的にやらないこと。

## Plan
- 中規模以上の場合だけ追加。採る技術方針、触れる層、却下した代替案、未決定事項を書く。

## Tasks
- [ ] T001 [SCL] 仕様を更新する。
- [ ] T002 [App] 実装する。
- [ ] T003 [Verify] 検証する。

## Verification
- 予定する検証コマンド / 手動手順。

## Risk Notes
リスクの根拠と軽減方法。
```

## 完了時（§1.3）

`status` を `completed` / `cancelled` にする時点で同じファイルに `completion` を追記し、
`work-items/done/` へ移す。`completion` は最低でも `completed_at` / `summary` / `verification` と
`affected_guarantees_state` を持つ。証跡は `completion.evidence` に手順・実行環境・実行主体・
対象ソース版・結果・保存先・要約値を記録し、大容量ログ・バイナリ・機密は埋め込まず
`evidence[].artifacts` に保管先とハッシュだけ残す。
