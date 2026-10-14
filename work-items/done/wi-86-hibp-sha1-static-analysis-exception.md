---
status: completed
archived_from: 0fca46fb861aad86ad8b000c8feebf5172954e84
---

# HIBP SHA-1 fingerprint の静的解析例外を明示する

HIBP Range API 用 SHA-1 計算を `hibpRangePrefixSuffix` に閉じ、テストは同じ関数を
使うようにしてテスト側の SHA-1 再実装を削除した。inline `nolint:gosec` は消し、
`.golangci.yml` に HIBP adapter ファイルかつ G401/G505 だけの限定例外を追加した。
ADR-028 には、この SHA-1 が password 保存・照合用 hash ではなく HIBP lookup
fingerprint であること、通常の password hash は Argon2id のままであること、
静的解析例外を HIBP adapter の専用関数 1 箇所に限定することを追記した。

全文は `git show 0fca46fb861aad86ad8b000c8feebf5172954e84:<このファイルのパス>` で読む。
