---
status: completed
---

# 初回公開前は特性化テストを求めない

`mise run spec-diff` は規範の差分なしを報告する。
特性化テストを求めるのは公開状態が `published` の間だけとし、`unpublished` の間は、触れる範囲の保証を表す要件の `//spec:covers` テストが欠けていないかを確かめ、欠けていれば先に足す手順にした。
既存コードからの書き起こしの手順、公開状態の節、`implement-work-item` の手順 5 を同じ条件にそろえた。
置かれた `TestCharacterize*` の扱いと、それを書き換える変更を拒否する検査は変えていない。

全文は `git log -1 -- work-items/active/wi-74867-skip-characterization-tests-before-first-publication.md` が示すコミットの親、またはそのコミットで読む。
