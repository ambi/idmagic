---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# decisions/配下の旧ADRを棚卸しして削除し、非work-item箇所のADR参照を除去する

`decisions/`配下の旧ADR 160本と`CONCEPTION.md`/`CONCEPTION_BASELINE.md`を全削除した。棚卸しの結果、大半のADRの
決定内容は既にoauth2・authentication・tenancy・application等各`SPECIFICATION.md`のDesignへ反映
済みだったが、application context のポータルアプリ並び替え・カテゴリ機能(旧ADR-069)だけ未反映
だったため`docs/domain/application/SPECIFICATION.md`へ最小限のDesign節を追記した。CIBA・ReBAC・
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

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
