---
status: completed
---

# Directory Connector で AD のコンピューターアカウントを取り込まない

REQ-SOURCING-020 と REQ-DIRECTORYCONNECTOR-007、`SendDirectoryComputerBatch`、`DirectoryComputerBatchRequest`、`DirectoryComputerItem` を取り除き、REQ-SOURCING-017〜019、REQ-DIRECTORYCONNECTOR-002、003、006、`DirectoryItemError`、`IngestionRunCounts`、`IngestionRunLifecycle` を User だけの取り込みに戻した。
Connector の `computers` の設定と読み取り、Sourcing の `directory_computers` と無効にしたコンピューターの数をなくし、完了の項目数は User のバッチの和と照合する。
取り込みは wi-43000 の一つのコミットで入っていたので、作業項目の記録を除いて打ち消し、経路の参照を再生成した。後継のない取り除きであり、どちらの要件も初回公開前に加えたものなので、廃止の見出しを残していない。
取り除いた経路が残れば失敗する受け入れテストを足し、取り除く前の main でコンピューターのバッチが 400 で受け付けられて失敗することを確かめた。Docker が動いていなかったので `check-schema` は実行しておらず、スキーマは wi-43000 の前と同じ内容である。
