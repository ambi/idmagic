---
status: completed
archived_from: b37fc2a653cacde986350082ff18bbd5570555e5
---

# 着手中の work item の initial_context が実在する対象を指すことを検査する

`check-workspace.ts` に散在していた work item の参照検査を `tools/check/src/work-item-references.ts`
の純関数へ切り出し、単体テストを付けた。検査内容は 2 点変えた。`affected_spec[].requirement` は
`REQ-` 始まりのとき `### <ID>:` 見出しの宣言を要求する（standard ID は従来どおり出現照合）。
`status: in_progress` の記録に限り `initial_context` を解決し、`source` / `tests` /
`stop_before_reading` のパス実在と、`specification` の `path#REQ-ID` 形式の解決を検査する。
新しいコマンドや設定は追加しておらず、`just check-work-items` の内側で完結する。
本 work item 自身を `in_progress` にして新検査を通し、運用として成立することを確認した。

全文は `git show b37fc2a653cacde986350082ff18bbd5570555e5:<このファイルのパス>` で読む。
