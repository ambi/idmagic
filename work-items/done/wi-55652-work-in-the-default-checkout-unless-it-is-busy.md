---
status: completed
---

# 作業項目は既定のチェックアウトで進め、使用中のときだけ worktree を作る

`mise run spec-diff` は main に対して規範の差分なしを示す。
`implement-work-item` は、既定のチェックアウトが base を開き変更がなければそこで作業ブランチを切り、使用中のときだけ `.worktrees/` に worktree を作って `mise run setup` で依存を入れる。
`integrate-work-item` は、取り込み先がどこにも開かれていなければ参照だけを fast-forward し（`git fetch . <source>:<target>`）、後片付けは既定のチェックアウトなら取り込み先へ戻し、worktree なら worktree を消す。
`install-ui` と `setup-tools` は `package.json` と `bun.lock` より新しい導入済みの印があれば何もしない。UI の六つの検査と `test-tools`、`typecheck-tools`、`lint-tools` はそれを前提に持ち、`verify-changed` と `verify-integration` は並列の検査の前に一度だけ `setup` を実行する。
依存のない worktree で `verify-ui` と `test-tools` が自分で導入して通り、二回目は導入を飛ばし、`bun.lock` を変えると導入し直すことを確かめた。
2026-10-29 の計測では、worktree の作成は 0.4 秒、依存の導入は合わせて約 2 秒で、新しいディレクトリの初回は `lint-go` が 13.0 秒（二回目 4.1 秒）、`go build ./...` が 11.4 秒（二回目 6.5 秒）だった。この作業項目は既定のチェックアウトで進め、最終検査の `lint-go` は 4.6 秒だった。
