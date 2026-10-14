---
status: completed
archived_from: 714d876fe1374bf105ca30f1966df140a802e3d1
---

# 追加した TypeSpec 宣言を `affected_spec` で名乗れないので、無関係な記録へ配られる

追加された TypeSpec 宣言を、work item が `affected_spec` の `{ path, symbol }` で名乗れる
ようになった。`mise run spec-diff` は規範差分なしを返す。検査の突き合わせ方だけを直しており、
標準の行も TypeSpec も製品の振る舞いも変えていないので、これが期待どおりの出力である。
照合は `claimsSpecificationAddition` 1 つになった。素の id はこれまでどおり `requirement` と
`symbol` に完全一致で当たり、`<path>:<name>` の形の要素だけ最後の `:` で割って、`path` の一致と
`symbol` の最後のドット区切りの一致の両方を求める。`documentation-impact.ts` の
`ownsSpecificationAdditions` と `check-workspace.ts` の `specificationAdditionsClaimed` は、
どちらもこの関数を呼ぶだけになり、照合の規則を自分では持たなくなった。
**同じ問いを 2 か所で別々に実装していたことが原因だった。** 参照としては解決する `symbol` の
書き方が、持ち主の判定では 1 件も一致しない。片方だけを読んでいる限り、どちらも正しく見える。
名乗り手がいない追加を全記録へ配る規則は変えていない。変えたのは、名乗ろうとした記録が
名乗れるようにすることだけである。

全文は `git show 714d876fe1374bf105ca30f1966df140a802e3d1:<このファイルのパス>` で読む。
