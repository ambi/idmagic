---
status: completed
archived_from: 116c6a6966020b948f46b9ce6ff33b184c5fcbcb
---

# worker が発行した Agent と所有者のイベントでも失効エポックを進める

`mise run spec-diff` は main に対する規範仕様の差分を報告しない（`no normative specification change against main`）。REQ-PLATFORM-001 と REQ-SHAREDSIGNALS-007 を変えず、実装を合わせた。
管理 API の外から User を止める経路（`worker` のワークフローの `disable_user` と `api` の SCIM の取り込み）が使う `UserLifecycleCommands` が、管理 API と同じ反応器を発行の後に呼ぶようになり、所有する Agent の失効エポックが進み、SET の配送が作られる。
反応器の組み立ては `sharedsignals/usecases.NewAgentRevocationReactor` の一つにまとめた。
SET の `iss` に使う `ISSUER` を共有の設定へ移し、`worker` も読む。
二つの `design/risks.md` から該当の行を消し、SharedSignals のアーキテクチャの実行時の流れを直した。

全文は `git show 116c6a6966020b948f46b9ce6ff33b184c5fcbcb:<このファイルのパス>` で読む。
