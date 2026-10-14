---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# `standards.md` の行を足すと、無関係な in_progress 記録が documentation_impact で落ちる

`mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無い。
差分は `tools/check/src/documentation-impact.ts` の `referenceNames` 1 箇所と、それを固定するテスト 2 件である。
`standards.md` へ足した行を、その行を `affected_spec` で名指した記録に帰属させるようにした。
同関数は追加要素の綴りを 3 通り扱うが、`<path>#<ID>` を扱う分岐だけが無く、標準行の追加は
どの記録も主張できなかった。
**帰属が無いのではなく、逆向きだった。** 追加を主張できない以上、行を足した当人は
`ownsSpecificationAdditions` で「追加を所有しない」と判定されて `documentation_impact` を問われず、
代わりに無関係な in_progress の記録がその追加を被って落ちていた。単体 RED の
`Received: []` はこの逆転を示している。
**同じ欠陥は 2 度目である。** [[wi-523-introspection-token-type-is-not-the-rfc6749-token-type]] の作業で `<path>:<name>` の TypeSpec 宣言に対して
同じ修正が入っている（テスト `gives an added TypeSpec declaration to the record that declares it`
の注記が記録している）。そのとき標準行は見られていない。今回の注記には、3 通りの綴りと、
どれがどの分岐に対応するかを書いた。
`addedScenarios` と `addedDeclarations` に同じ食い違いが無いことを確かめた。scenario は
`spec-diff.ts:178` で id そのものを鍵にするので逐語比較で足り、declaration は
`spec-diff.ts:79,157` で `<path>:<name>` を鍵にし、対応する分岐が既にある。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
