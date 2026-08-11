---
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-11
depends_on: [wi-355-replace-scl-architecture-ledgers-and-adrs]
change_kind: tooling
spec_impact:
  kind: none
  reason: 製品挙動を変えず、判断履歴の保管場所と現在設計の記述先を整理するだけである。
---

# decisions/配下の旧ADRを棚卸しして削除し、非work-item箇所のADR参照を除去する

## Motivation

wi-355でADRの新規作成・必須索引・supersession運用を終了したが、`decisions/`配下160本の既存ADR本文の
削除自体は明示的にOut of Scopeとした。結果として過去の判断履歴が`decisions/`に残ったままになり、
`backend/`・`frontend/`・`tools/`配下の566ファイル（出現数700件超）にADR番号への参照コメントが残存
している。判断履歴の正本をwork itemとcurrent-state docs（SPECIFICATION.md）に一本化する方針を、
実体のクリーンアップでも完了させる。

## Scope

- `decisions/`配下の全ADRを棚卸しし、原則削除する。git履歴に残るため、削除自体に高い精査コストは
  かけない。
- 各ADRの決定内容が、対応するroot/context `SPECIFICATION.md`の設計記述として読み取れない場合は、
  現在設計としてなお有効な内容だけを該当`SPECIFICATION.md`のDesign/Architecture相当の節へ最小限
  追記してから削除する。コストが高すぎる場合は追記を諦めて削除してよい（判断はwork itemに残らないが、
  git履歴とこのwork itemの完了記録が経緯を保持する）。
- `backend/`・`frontend/`・`tools/`・`infra/`・`README.md`・`justfile`など、`work-items/`以外で
  `ADR-NNN`を参照している箇所からその参照表記を除去する。周辺の説明文はADR参照を除いても意味が通る
  よう最小限整える（丁寧な文章推敲はしない）。
- `tools/check/src/adr-links.ts`・`adr-supersession.ts`とそのテストなど、ADR運用専用のcheckerを
  合わせて削除し、`just check`から呼び出しを外す。
- `ADR_FORMAT.md`など、ADRの新規作成・運用を前提にした参照文書があれば、既に廃止された運用である旨に
  同期する（本文の大規模な書き直しはしない）。

## Out of Scope

- `work-items/`配下（`done/`含む）のADR参照の変更。過去の実装記録としてそのまま残す。
- 製品API・永続化・認証認可挙動の変更。
- 新しいADR運用の導入。

## Design

- 判断履歴の正本は「完了したwork item」と「各`SPECIFICATION.md`の現在設計記述」の2つに一本化する。
  `decisions/`は廃止し、過去のADR本文を読みたい場合はgit履歴（`git log -- decisions/ADR-NNN-*.md`）を
  参照する運用にする。
- ADR参照の除去は、ADR番号ごとに`grep`で出現箇所を洗い出し、コメント中の該当断片を削るスクリプト的な
  一括編集で行う。個別の文章磨き込みより除去の網羅性を優先する。
- checkerの削除により`decisions/`不在を前提にした検証系へ揃える。`check-work-items`・`check-ids`など
  work item側の検証には影響しない。

## Plan

- `decisions/`を全走査し、SPECIFICATION.mdへの追記要否を context 単位でまとめて判断してから、
  削除を実行する。
- ADR参照除去はディレクトリ単位（`backend/<context>`ごと、`frontend/`、`tools/`、root直下）に
  バッチで進める。
- 最後にcheckerとformat文書を同期し、`just check`・`just verify`を通す。

## Tasks

- [x] T001 [Spec] `decisions/`配下ADRを棚卸しし、必要な現在設計情報を各`SPECIFICATION.md`へ最小限
      追記する。
- [x] T002 [Cleanup] `decisions/`配下の全ADRファイルを削除する。
- [x] T003 [Cleanup] `work-items/`以外のADR参照（コメント・ドキュメント・justfile）を除去する。
- [x] T004 [Tooling] `adr-links`・`adr-supersession` checkerとテストを削除し、`just check`から外す。
- [x] T005 [Docs] `ADR_FORMAT.md`等の参照文書をADR廃止後の実態に同期する。
- [x] T006 [Verify] `just check`・`just verify`を通す。
- [x] T007 [Completion] 完了記録を追加してdoneへ移動する。

## Verification

- `just check`
- `just verify`

## Risk Notes

削除量が大きいスコープの広い変更である。ADR番号への参照除去がコード側コメントの意味を壊さないよう、
除去後もコメントが構文的・意味的に成立する範囲でのみ編集する。checker削除後に`decisions/`参照が
tooling側に残らないことを`just check`で確認する。

## Completion

- **Completed At**: 2026-08-11
- **Summary**:
  `decisions/`配下の旧ADR 160本と`CONCEPTION.md`/`CONCEPTION_BASELINE.md`を全削除した。棚卸しの結果、大半のADRの
  決定内容は既にoauth2・authentication・tenancy・application等各`SPECIFICATION.md`のDesignへ反映
  済みだったが、application context のポータルアプリ並び替え・カテゴリ機能(旧ADR-069)だけ未反映
  だったため`spec/contexts/application/SPECIFICATION.md`へ最小限のDesign節を追記した。CIBA・ReBAC・
  Token Vault・Cross-App Access・Agent governance guardrails など「suggested」止まりで未実装のADRは、
  対応するwork item（wi-52・wi-53・wi-55・wi-57・wi-59等）が既に設計の続きを引き継いでいるため追記
  せず削除した。
  `work-items/`以外でADR番号を参照していた約590ファイル（コード コメント、SQL、YAML、JSON、Caddyfile、
  CSS、shell script、Markdown）から参照表記を除去した。除去はスクリプトによる機械的な一括編集で行い、
  コメント構文（`//`/`--`/`#`のマーカーと空白）と隣接する`wi-NNN`引用は壊さないよう注意した。
  `tools/check/src/adr-links.ts`・`adr-supersession.ts`とそれらのテスト、および`record-ids.ts`・
  `check-ids.ts`・`workspace.ts`のADR固有ロジック（`--decisions`/`--links`引数、ADR ID抽出、
  supersession検証）を削除した。`ADR_FORMAT.md`は削除し、`AGENTS.md`/`CLAUDE.md`・`DEVELOPMENT.md`・
  `SPECIFICATION_FORMAT.md`・`WORK_ITEM_FORMAT.md`・`README.md`・`new-adr` skillをADR完全廃止後の
  実態に同期した。
- **Verification Results**:
  - `just check` - passed
  - `just check-api-compat` - passed (no breaking changes)
  - `just verify` - passed (lint-go, test-go, test-tools, typecheck-tools, lint-ui, format-check-ui,
    test-ui-unit, build-ui, check, check-api-compat all green)
