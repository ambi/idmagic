---
status: completed
authors: [tn]
risk: low
created_at: 2026-08-13
depends_on: []
change_kind: docs
spec_impact:
  kind: none
  reason: 製品挙動を変えず、仕様ファースト開発方式の記述と work item スキーマの不整合を直すだけである。
---

# 仕様ファースト方式ドキュメントを工程・ゲート中心に再編し、work item スキーマとの不整合を解消する

## Motivation

`DEVELOPMENT.md`・`SPECIFICATION_FORMAT.md`・`WORK_ITEM_FORMAT.md` が方式の正本だが、次の問題がある。

- 3 文書のうち `WORK_ITEM_FORMAT.md` が `AGENTS.md` に登録されておらず、skill 経由でしか到達できない。
- 記述の多くが過去の失敗に対する散文の禁止事項で、機械検査されるものと人手レビュー観点が混在している。
  どの規則が `just check` で落ちるのかが読み取れず、毎タスクの固定コンテキストコストだけが増える。
- `WORK_ITEM_FORMAT.md` のテンプレートが `initial_context.specification` を使う一方、
  `tools/check/schemas/work-item.schema.json` の `initial_context` は `additionalProperties: false` で
  `specification` を許可していない。テンプレートどおりに書くと `just check-work-items` が落ちる実バグ。
- スキーマに廃止済み Regenerative Architecture の残骸（`$id` の `regenerative-architecture`、`scl`、
  `sclElementReference`、`affected_guarantees`、`evidence`、`target_state` 等）が残り、実際には
  pending/新規では使わないフィールドが required 判定や補完のノイズになっている。
- `initial_context` を起票時に必須としているため、backlog に積まれた 90 件のうち 18 件が廃止済み `scl:`、
  13 件が削除済み `decisions/ADR-*.md` を参照したまま腐っている。AI エージェントが最初に読む導線が
  実在しないファイルを指している。
- `.agents/skills/scl-change`・`scl-render` が空ディレクトリとして残っている。

## Scope

- `DEVELOPMENT.md` を「工程 → 所有 skill → 通すゲート」を中心とした構成に書き換える。検証ラダー
  （`just check-spec` → 層局所テスト → `just verify`）と、索引を初手にする context economy を明記する。
- `SPECIFICATION_FORMAT.md` の各規則に、機械検査されるか人手レビュー観点かの区別を与える。文法の正本は
  `tools/check/src/specification-doc.ts` の検査であることを明記する。
- `SPECIFICATION_FORMAT.md` に規約を 2 つ追加する。
  - 廃止された正規シナリオの表記 `(superseded by REQ-X-NNN)`。
  - 複数 context にまたがる挙動を root `SPECIFICATION.md` の `REQ-SYSTEM-*` に置く規約。
- `WORK_ITEM_FORMAT.md` で `initial_context` を `in_progress` 昇格時点の必須事項に変更する。
- `tools/check/schemas/work-item.schema.json` を上記に合わせる。`specification` キーの追加、`$id` の更新、
  legacy フィールドの完了済み記録専用扱いの明示。
- `AGENTS.md` に 3 文書を役割 1 行付きで登録する。
- 空の skill ディレクトリ `scl-change`・`scl-render` を削除する。

## Out of Scope

- 新しい checker の実装（wi-360 / wi-361 / wi-362 で行う）。
- 既存 pending work item 90 件・done 267 件の本文一括修正。着手時更新のラチェット方式を採る。
- `spec/**/SPECIFICATION.md` 本体の内容変更。superseded 表記は規約を追加するだけで、既存 REQ は触らない。

## Design

- 規則は「機械検査されるか、消すか」を原則とする。検査済みの規則は文書側では意図と例だけを述べ、正確な
  条件は checker のエラーメッセージに委ねる。二重記述はドリフト源であり、そこが今回の主要な削減対象。
- `initial_context` の腐敗は「起票時に書く」構造が原因なので、必須化のタイミングを着手時へ移す。
  backlog は Motivation / Scope / Out of Scope だけで成立させる。
- スキーマの legacy フィールドは、実際に使われているもの（`initial_context.scl` 76 件、`decisions` 26 件、
  `affected_spec` の SCL 参照 26 件）だけを互換のため残し、`description` で完了済み記録専用と明示する。
  全 361 件で使用実績が 0 のもの（`evidence`・`affected_guarantees`・`target_state` ほか）は削除する。
  使われていない選択肢はエージェントにとって「埋めるべき項目」に見えるだけで、方式を重く見せる。

## Plan

1. スキーマを先に直す（テンプレートとの不整合が実バグのため）。
2. `WORK_ITEM_FORMAT.md` → `SPECIFICATION_FORMAT.md` → `DEVELOPMENT.md` の順に、細かい正本から書き換える。
3. `AGENTS.md` と skill ディレクトリを同期する。
4. `just check` と `just verify` を通す。

## Tasks

- [x] T001 [Spec] `work-item.schema.json` に `specification` を追加し、`$id` と legacy フィールドの
      説明を現行方式へ同期する。
- [x] T002 [Docs] `WORK_ITEM_FORMAT.md` を `initial_context` 着手時必須へ改訂する。
- [x] T003 [Docs] `SPECIFICATION_FORMAT.md` に enforced/review 区別、superseded 表記、cross-context
      配置規約を反映する。
- [x] T004 [Docs] `DEVELOPMENT.md` を工程・ゲート・検証ラダー中心に書き換える。
- [x] T005 [Docs] `AGENTS.md` に 3 文書を登録し、空 skill ディレクトリを削除する。
- [x] T006 [Verify] `just check` と `just verify` を通す。
- [x] T007 [Completion] 完了記録を追加して `work-items/done/` へ移動する。

## Verification

- `just check`
- `just verify`

## Risk Notes

方式ドキュメントの改訂は、既存 work item の記述形式と矛盾すると全件が赤くなる。使用実績のある legacy
フィールドを互換扱いで残すことでこれを避ける。`initial_context` の必須タイミング変更はスキーマ上は
緩和方向のため、既存記録を壊さない。

## Completion

- **Completed At**: 2026-08-13
- **Summary**:
  方式ドキュメント 3 点を役割で分離し、`AGENTS.md` から 3 点すべてに到達できるようにした。
  `DEVELOPMENT.md` は「工程 → skill → ゲート」の表と検証ラダー（`just check-spec` → 層局所テスト →
  `just verify`）を中心に再構成した。`SPECIFICATION_FORMAT.md` には、文法の正本が `just check-spec` の
  診断であることの宣言、規則ごとの *(checked)* 表記、cross-context 挙動を root の `REQ-SYSTEM-*` に置く
  規約、正規シナリオの廃止表記 `(superseded by REQ-X-NNN)` を追加した。廃止表記は checker 側にも実装し、
  ステップを持たない廃止シナリオを許容しつつ、後継 ID の実在を検査する。
  work item 側は `initial_context` を起票時必須から着手時（`in_progress`）必須へ移し、テンプレートと
  スキーマの不整合（`specification` キーが `additionalProperties: false` で拒否されていた）を解消した。
  スキーマからは全 361 記録で使用実績 0 の Regenerative Architecture 残存フィールドを削除した。
- **Verification Results**:
  - `just check` - passed
  - `just verify` - passed（12s、全 10 チェック ok）
