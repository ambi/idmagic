---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 実装が仕様の素朴な期待より狭い場合の開示義務を RA/SCL に組み込む

`tools/yaml-check/schemas/scl-v3.schema.json` の `Standard.requirements[].adoption` に `partial`
を追加し、`reason` 必須条件を `excluded` から `excluded | partial` へ拡張した。
`SPECIFICATION_CORE_LANGUAGE.md` §2.1 を同期し、`partial` の意味と ADR-121 へのリンクを追記した。
`decisions/ADR-121-scope-narrowing-disclosure-obligation.md` を新規作成し、(1) SCL の
`adoption: partial` 機構、(2) ADR 作成トリガーを「非自明だと思ったら」という裁量任せから
具体基準(標準の adoption を partial/excluded にした、または work item の Motivation/Scope/
タイトルが示唆する範囲より狭い実装をした)へ置換、(3) 完了報告への開示義務(「全部グリーン」
だけでの完了報告を禁止)、(4) fuzz/property test はリスクベースの判断基準にとどめ全
untrusted-input parser への一律義務化はしない、の4点を決定として記録した。
`.agents/skills/implement-work-item/SKILL.md` (`.claude/skills` はこれへのシンボリックリンク)
の ADR トリガー・test-first 節・完了処理の3箇所を ADR-121 に合わせて更新した。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
