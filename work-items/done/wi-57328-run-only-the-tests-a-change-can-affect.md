---
status: completed
---

# 変更が影響し得るテストだけを実行し、統合では両側の影響が重なる範囲だけを検証する

`mise run spec-diff` は main に対して規範の差分なしを示す。
`mise run verify-changed -- [base]` は作業中の最終検査、`mise run verify-integration -- <検証済みのソースのコミット>` は rebase 後の統合の検証で、どちらも変更が届き得る層だけを並列に実行し、選ばなかった層を理由とともに `.verification/verify-selection.json` に残す。
Go は `go list -test` のテストのバイナリの依存から、変更したパッケージ（`testdata` や埋め込むファイルを含む）を取り込むパッケージだけを lint し、通常のビルドでテストする。`docs/`、`work-items/`、`tools/`、`.agents/`、`frontend/src/`、`frontend/tests/` は Go を選ばず、それ以外のパッケージ外のパスは Go 全体を選ぶ。どの層も持たないパスは全層を選ぶ。
統合は、ソース自身の変更 `merge-base(S, R)..S` と受け入れた変更 `S..R` の両方が届く層と、Go のパッケージの共通部分だけを実行する。
wi-18773 の統合を再現すると、392 パッケージのうち 84 を選び約 43 秒で終わり、同じツリーの `verify` は約 114 秒だった。wi-43903 の統合は `check` だけになる。
`test-go-changed` は同じパッケージの選び方を使う。`integrate-work-item` は `verify-integration` を使い、最終検査の選択は `verify-changed` を基本にした。
race 検査は扱わない。履歴に race 検出器が見つけた data race の修正がなく、race は約 1.5 倍かかるので、手元の既定は wi-52813、main への定期実行は wi-47187 が扱う。
UI と tools はテストファイルの単位では選ばず、層の全体で実行する。Go の結果キャッシュに任せる案は、worktree をまたいで当たらないので採らなかった。
