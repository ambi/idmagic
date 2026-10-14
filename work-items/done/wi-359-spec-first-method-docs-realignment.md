---
status: completed
archived_from: b37fc2a653cacde986350082ff18bbd5570555e5
---

# 仕様ファースト方式ドキュメントを工程・ゲート中心に再編し、work item スキーマとの不整合を解消する

方式ドキュメント 3 点を役割で分離し、`AGENTS.md` から 3 点すべてに到達できるようにした。
`DEVELOPMENT.md` は「工程 → skill → ゲート」の表と検証ラダー（`just check-spec` → 層局所テスト →
`just verify`）を中心に再構成した。`SPECIFICATION_FORMAT.md` には、文法の正本が `just check-spec` の
診断であることの宣言、規則ごとの *(checked)* 表記、cross-context 挙動を root の `REQ-SYSTEM-*` に置く
規約、正規シナリオの廃止表記 `(superseded by REQ-X-NNN)` を追加した。廃止表記は checker 側にも実装し、
ステップを持たない廃止シナリオを許容しつつ、後継 ID の実在を検査する。
work item 側は `initial_context` を起票時必須から着手時（`in_progress`）必須へ移し、テンプレートと
スキーマの不整合（`specification` キーが `additionalProperties: false` で拒否されていた）を解消した。
スキーマからは全 361 記録で使用実績 0 の Regenerative Architecture 残存フィールドを削除した。

全文は `git show b37fc2a653cacde986350082ff18bbd5570555e5:<このファイルのパス>` で読む。
