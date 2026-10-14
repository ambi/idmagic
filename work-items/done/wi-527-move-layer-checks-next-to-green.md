---
status: completed
archived_from: 29840d9aad2215d833581e79eccae035237f9813
---

# 層ごとの検査を、その層が GREEN になった直後へ前倒しする

`mise run spec-diff` は main に対して規範仕様の差分を報告しない。変わったのは実装中の手順だけである。
`implement-work-item` skill の第 8 段は、lint と集約ゲートを最終検証まで遅らせる指示を捨て、触った層と直後に走らせる検査の対応表を持つようになった。
冷状態の `lint-go` を 1 回払う指示は第 1 段の読み取りと並ぶ。
`docs/development/specification-first-workflow.md` の検証のはしごは、作業開始時の `lint-go` を第 1 段、層ごとの検査を第 5 段として持ち、段の数は 6 から 8 になった。

全文は `git show 29840d9aad2215d833581e79eccae035237f9813:<このファイルのパス>` で読む。
