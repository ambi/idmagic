---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# HTTP request cancellation policy を定義し、mutation の整合性とログ分類を整理する

ADR-074 で HTTP request cancellation policy を採用し、SCL の system context に
CancellationConsistency と ClientAbortLogClassification を追加した。HTTP support
layer には OperationContext、DetachedCompletionContext、cancel classification、
abort metrics hook を追加し、admin user / key / authorize / par / token の代表
mutation handler を request cancel から切り離した operation context へ寄せた。
client abort は server error ではなく client_aborted として分類する。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
