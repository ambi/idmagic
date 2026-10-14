---
status: completed
archived_from: 06c74e7df76bdaba9bd8e34b1bfb773750037306
---

# work item 実装用の worktree を消えない場所へ移し、タスクごとに checkpoint を残す

`mise run spec-diff` は規範仕様の差分を報告しない。変更はエージェントの作業場所と、リポジトリ root から走査する検査の除外に閉じる。
work item 実装用の worktree は `.worktrees/<work-item-id>` に置く。`.gitignore`、`check-links`、`check-boundaries`、`.betterleaks.toml` がこの場所を除外する。
`parallel-work-items` skill の作成、統合、片付けの手順はすべて `.worktrees/` を使い、`implement-work-item` skill は別の worktree を `/tmp` や `$TMPDIR` に作らないと定める。
`implement-work-item` skill は、タスクが GREEN になるたびに確認なしで checkpoint commit を作り、正式なコミットの前に `git reset --soft <直前の正式なコミット>` で畳むと定める。checkpoint を確認なしで作ることは利用者が選んだ。
readiness で見落とした `betterleaks` の root 走査は、worktree のある状態で走らせた `mise run verify` で見つけて直した。

全文は `git show 06c74e7df76bdaba9bd8e34b1bfb773750037306:<このファイルのパス>` で読む。
