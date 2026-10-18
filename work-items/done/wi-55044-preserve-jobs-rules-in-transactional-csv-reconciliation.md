---
status: completed
---

# CSV の再評価ジョブを原子性と Jobs の規則を保って投入する

spec-diff は規範仕様の差分なしだった。
CSV の再評価ジョブを、呼び出し側の Tx に結んだ Jobs の正規の投入操作へ切り替え、レーン、デフォルト、重複排除、active_jobs の判定を同じ処理に集めた。
クォータ拒否では行、規則、監査、ジョブと使用量を残さず、ロールバック後に QuotaExceeded を発行する。
新規の JobEnqueued は確定後だけ発行し、重複とコミット失敗では発行しない。
IdManagement の jobs への越境 SQL を削除し、越境書き込みは4件から3件へ減った。
変更行の変異8件と、確定前の通知と通知先の切断の手動故障2件を検出した。
check、lint-go、文書生成と全 Go race 検査（6444件成功、3件スキップ）が成功した。
保存と通知の間の停止への保証の追加とメモリの既存の不足は対象外とした。

全文は `git log -1 -- work-items/active/wi-55044-preserve-jobs-rules-in-transactional-csv-reconciliation.md` が示すコミットの親、またはそのコミットで読む。
