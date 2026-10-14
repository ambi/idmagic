---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# HIBP k-anonymity による漏洩パスワード検査 adapter を投入する

ADR-028 の実装側を完成させた。`internal/infrastructure/policy/`
(既存 Noop adapter と同じ package) に HibpBreachedPasswordChecker を新設し、
SHA-1 prefix 5 文字を HIBP Range API に GET する k-anonymity 検査、
Add-Padding ヘッダ、2 秒 timeout、`User-Agent: idmagic/<version>`、
prefix 単位の TTL 付き LRU cache (TTL 10 分 / 上限 1024) を実装した。
HTTP error / timeout / 5xx / 接続失敗は false を返す fail-open とし、
失敗は warn log と OpenTelemetry counter
`breached_password_checker_failures_total{reason}` で可視化する。
bootstrap に `BREACHED_PASSWORD_CHECKER` (`noop` 既定 / `hibp`) を読む
resolveBreachedPasswordChecker を追加し、起動 log を 1 行出して Deps に
注入する。usecase 層 (reset_password_with_token) は無変更で本 adapter を通る。
SCL の objectives.PasswordPolicy に宣言的既定 breached_password_check_enabled:
false を追加し、Go 双子定数と coherence test (ObjectiveBool) で突き合わせた。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
