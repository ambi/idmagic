---
name: new-work-item
description: Create a new Regenerative Architecture work item under work-items/ following the canonical format. Use when starting a new unit of work, a task, or when the user asks to draft/open a work item or wi-NN.
---

# 新規ワークアイテムの作成

正本書式は `WORK_ITEM_FORMAT.md`。**既存ファイルを開いて書式を逆算しない**——
本 Skill と同文書の記法に従う。既存ファイルは「似た題材の中身」を見たいときだけ開く。

## 手順

1. **id を採番する**
   - 既存最大連番を確認（プレフィクスなしの単一名前空間）:
     `ls <app>/work-items <app>/work-items/done 2>/dev/null` または
     `ls work-items work-items/done 2>/dev/null`
   - `<id>` = `wi-<連番>-<kebab-title>`。ファイル名は `work-items/<id>.md`。
   - 連番は正本ではない。並行採番で被っても `check-ids` が検出するので、被ったら採り直す。
2. **未着手・進行中は `work-items/` 直下に置く**。完了・中止になったら `done/` サブディレクトリへ移す（id は変えない）。
3. 機能変更なら **触れる SCL セクションを `## Scope` に列挙する**。判定は SCL-first の網羅表に従う（`scl-change` Skill / `SPECIFICATION_CORE_LANGUAGE.md §3` 冒頭）。
4. 中規模以上なら `## Plan` と `## Tasks` を追加する。複数 scenario、RA 3 層以上、DB migration / 認可 / 外部契約 / 破壊的変更、1 セッションで終わる確信がない作業、複数サブシステム検証は中規模以上として扱う。
5. 下記スケルトンを埋める。
6. **検証**: `just yaml-check-work-items` と `just check-ids` を通す。

## スケルトン

```markdown
---
status: pending  # pending | in_progress | completed | cancelled
authors: [name]
risk: low        # low | medium | high | critical
created_at: 2026-01-01  # YYYY-MM-DD
depends_on: []   # 完了前提の WI ID。依存がなければ空配列
---

# 一文で表す意味変更

## Motivation
なぜこの変更が必要か（What ではなく Why）の背景。

## Scope
- `spec/scl.yaml` の `interfaces.StartTask`
- `src/usecase/` への実装

## Out of Scope
- 明示的にやらないこと。

## Plan
- 採る技術方針、触れる層、却下した代替案、未決定事項。

## Tasks
- [ ] T001 [SCL] 仕様を更新する。
- [ ] T002 [App] 実装する。
- [ ] T003 [Verify] 検証する。

## Verification
- 予定する検証コマンド（例：`go test ./...`）や手動手順。

## Risk Notes
リスクの根拠と軽減方法。

## Completion
- **Completed At**: 2026-01-01
- **Summary**:
  実装した変更の意味上の差分の要約。
- **Verification Results**:
  - `just verify` - passed
```

## 完了時

`status` を `completed` / `cancelled` にする時点で、frontmatter ではなく Markdown 本文末尾に `## Completion` を追記し、`work-items/done/` へ移す。Completion は最低でも `Completed At` / `Summary` / `Verification Results` と `Affected Guarantees State` を持つ。証跡は同節の `Evidence` に手順・実行環境・実行主体・対象ソース版・結果・保存先・要約値を記録し、大容量ログ・バイナリ・機密は埋め込まず、保管先とハッシュだけを記録する。
