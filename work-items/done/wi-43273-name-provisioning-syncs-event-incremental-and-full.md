---
status: completed
archived_from: 9b98297b77932561b0a2ae56ab18fa8bfaf7dcaa
---

# Provisioning の 3 つの同期を、イベント同期、インクリメンタル同期、フル同期と呼ぶ

`mise run spec-diff` が示す変更は `REQ-PLATFORM-003` と、`REQ-PROVISIONING-003`、`-004`、`-005`、`-011`、`-013`、`-016`、`-017`、`-019` のシナリオの文言である。
いずれも「書き込み時の捕捉」「照合」「Full Resync」を「イベント同期」「インクリメンタル同期」「フル同期」へ改めたもので、観測できる振る舞いは変わらない。
用語集は 3 つの同期を Event Sync、Incremental Sync、Full Sync として定義し、コードの識別子（`Capture*`、`Reconcile*`、`FullResync`）との対応を示す。
管理画面の表示名は ja を「フル同期」、en を「Full sync」にした。

全文は `git show 9b98297b77932561b0a2ae56ab18fa8bfaf7dcaa:<このファイルのパス>` で読む。
