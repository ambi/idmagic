---
status: completed
archived_from: 6e0a084c619512e41d524ccd942a8dc9c88372ac
---

# SCIM inbound filter parser のテスト網羅性と fuzz 耐性を強化する

`backend/scim/domain/filter_test.go` に `TestParseFilterNotEqual`
(string/boolean 属性への `ne` の直接検証) と
`TestParseFilterOrderingOperatorsRejectedForNonDateTimeAttributes`
(`UserFilterAttributes`/`GroupFilterAttributes` の非 dateTime 属性すべてに対し
`gt`/`ge`/`lt`/`le` が `*domain.FilterError` になることをテーブル駆動で固定) を
追加した。`backend/scim/domain/filter_fuzz_test.go` を新規追加し、`FuzzParseFilter`
に既存ユニットテストケース(複合式・大文字小文字・文字列 escape・dateTime 比較・
不正 dateTime literal・schema URN プレフィックス・資源上限境界・未終端文字列など
34 件)を seed corpus として登録した。`filter.go` 自体への変更は不要だった
(fuzz 実行で crasher は見つからなかった)。
当初の Scope 記述(「gt/ge/lt/le はどの allowlist 属性でも常に invalidFilter」)は
wi-238 時点の状態を前提にしており、本 work item の起票後に
[[wi-244-scim-filter-grammar-extended-conformance]] が `meta.created`/
`meta.lastModified` への `gt`/`ge`/`lt`/`le` 対応を実装済みだったため、
T002 のテストは「dateTime 属性を除く」に補正して実装した。

全文は `git show 6e0a084c619512e41d524ccd942a8dc9c88372ac:<このファイルのパス>` で読む。
