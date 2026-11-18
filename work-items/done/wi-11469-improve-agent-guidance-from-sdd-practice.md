---
status: completed
---

# エージェント向けの指示の置き場所と書き方を定め、skill と起票の手順を整える

`spec-diff` は規範仕様の差分なしを示す。
エージェント向けの指示の置き場所、書くものと書かないもの、skill の書き方を `docs/development/agent-guidance.md` に定め、AGENTS.md の参照表から `brief` が返すようにした。
書く基準は「このリポジトリに固有で、どのエージェントの実行環境でも成り立つ規約だけ」であり、着手時に個人メモリの指摘をどれも移さないと決めた判断を一般化したものである（個人メモリは利用者の指示で削除した）。
`new-work-item`、`integrate-work-item`、`spec-change` を日本語にし、全 skill の description を「何をするか。いつ使うか。」にそろえた。
`new-work-item` に重複の探索、動機の事実の確認、出典と調査日、`feature` での既存製品との比較（形式は `work-item-format.md`）、新しい要件を `affected_spec` に挙げられないことを足し、`implement-work-item` に起票の後に変わったパスがあるときだけ前提を確かめ直す手順を足した。
変更前後の skill をサブエージェントで比べ、新しい手順が効くことと訳で意味が変わらないことを確かめた。
比較で見つかった章のページの見出しの深さを形式と `spec-change` に明記した。
評価シナリオ（`evals.json`）と陳腐化を探す skill は、一度実装または設計したうえで採らなかった。
AGENTS.md から Codex 固有の `apply_patch` を除き、README に mise の導入（`curl https://mise.run | sh` に統一）を足し、ルートの生成文書を `docs/operations/configuration.md` と `docs/operations/route-priority.md` へ移した。
