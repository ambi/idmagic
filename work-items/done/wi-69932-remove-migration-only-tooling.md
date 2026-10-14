---
status: completed
archived_from: 1f47b7ea25db560821cdd677cf7ee8d0d0e761b7
---

# 完了済みの作業項目を完了の時点でだけ検証し、旧い配置と旧い形式のための移行用の処理を検査ツールから外す

`mise run spec-diff` は main に対して規範仕様の差分がないことを示した。
変わったのは検査ツール、作業項目のスキーマ、開発文書である。
`check-work-items` と `check-links` は、`work-items/done/` の記録を、main とのマージベースから、または作業ツリーで変わったときだけ検証する。
変わった記録の判定は `tools/check/src/work-item-changes.ts` にまとめ、完了の変更で新しく作った `work-items/done/` の中の記録も拾えるよう、未追跡のファイルを一つずつ列挙する。
依存と識別番号の検査は、記録の間の現在の関係なので、すべての記録へ続けて適用する。
旧パスの対応表（`tools/check/relocated-spec-paths.json`）とその解決、主要ユースケースの証拠の `internal/` への読み替え、`spec-diff` と `documentKind` の旧い配置と旧形式（`SPECIFICATION.md`、`scenarios.md`、`states.md`、`examples.feature.md`）の読み取り、描画器の旧い出力先の削除を外した。
スキーマは `risk-based-v4` だけを受け付け、`risk-based-v1` から `v3` の分岐、作業項目の番号による除外、旧い SCL 参照、`docs/domain/` のパスを外した。
完了した記録には `evidence_policy` と `documentation_impact` を求める。
旧形式への逆戻りの拒否は、形式の検査が同じ文書を拒否しないことを確かめたので、現在の規則として残した。

全文は `git show 1f47b7ea25db560821cdd677cf7ee8d0d0e761b7:<このファイルのパス>` で読む。
