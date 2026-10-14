---
status: cancelled
archived_from: 820e03c6f7f454e55dd4a8fd419bc9d5081061a8
---

# 完了済み記録の証拠を当時のリビジョンで解決し、現在の仕様への追従を不要にする

取り消す。動機の「完了済みの記録も現在まで解決する」要求は、2026-10-10 のコミット `1f47b7ea2`（refactor(tools): verify completed work items only when they change）で既になくなっていた。
完了した記録は完了させる変更の中で一度だけ検証され、仕様の移動に追従させる必要も、旧パスの対応表もない。
さらに wi-29555 で、main に取り込んだ完了記録は `affected_spec` を持たない要約へ縮み、全文は `archived_from` のコミットで読む。
当時の revision で証拠を解決する仕組みを作る理由は残らないので、製品と仕様には変更を加えずに終了する。

全文は `git show 820e03c6f7f454e55dd4a8fd419bc9d5081061a8:<このファイルのパス>` で読む。
