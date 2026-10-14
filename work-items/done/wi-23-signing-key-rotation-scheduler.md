---
status: completed
archived_from: c17777b81c2a45156bab0d84ddf46e0460b675e7
---

# 署名鍵 rotation を運用で自動化する (scheduler + grace 期間中の旧鍵保持)

署名鍵 lifecycle を `retired_at`、`expires_at`、`archived_at` の
3 時刻に正規化し、旧鍵を grace 期間だけ JWKS へ残して archive する経路を
memory/PostgreSQL/Vault adapter、HTTP JWKS、監査イベント、one-shot batch、CronJob に通した。
retention sweep は通常 worker から同じ batch executable の別サブコマンドへ移した。
`rotated_at` は `retired_at` と重複するため削除した。

全文は `git show c17777b81c2a45156bab0d84ddf46e0460b675e7:<このファイルのパス>` で読む。
