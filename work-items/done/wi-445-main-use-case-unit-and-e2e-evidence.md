---
status: completed
archived_from: 2cfbbc6db5991ac837de94674e62d2f39280529e
---

# 機能と標準対応の主要ユースケースに単体テストと E2E テストを要求する

`mise run spec-diff` は `no normative specification change against main` を返した。製品の規範仕様は変更せず、機能、欠陥修正、`standards.md` の規範対応を行う新規作業項目に `risk-based-v3` を適用した。主要ユースケースごとに要求参照、観測結果、単体テストと E2E テスト、異なる故障モデルを実装前に宣言し、完了時に両方の RED と故障注入結果を残す契約を追加した。作業項目検査は、完了時のテスト実在性、識別子、要求 ID、標準検証または CI から到達可能な `mise` タスクまで検査する。完了済みの `risk-based-v1` / `risk-based-v2` 記録は履歴として再解釈しない。

全文は `git show 2cfbbc6db5991ac837de94674e62d2f39280529e:<このファイルのパス>` で読む。
