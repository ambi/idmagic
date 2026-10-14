---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# リポジトリ全体の依存関係を更新し Promtail を Alloy へ置き換える

`mise run spec-diff` は `no normative specification change against main` を返した。
リポジトリ全体の依存関係を最新安定版へ更新し、EOL 済みの Promtail を Grafana Alloy 1.19.2 へ置き換えた。
ログのラベル、structured metadata、タイムスタンプの意味は維持した。
`mise-config.test.ts` はスキャナーの現在版を複製せず、安定版へ明示固定されていることを検査するようになった。
一時的な変換コマンドは mise タスクに残さず、反復利用しないコマンドの例外を `AGENTS.md` に明記した。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
