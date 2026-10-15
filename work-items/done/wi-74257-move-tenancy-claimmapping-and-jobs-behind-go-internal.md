---
status: completed
---

# Tenancy、ClaimMapping、Jobs を Go の internal/ へ移す

`mise run spec-diff` は規範の変更なしと報告した。
Tenancy、ClaimMapping、Jobs の責務表を `internal` にし、実在する公開契約だけを公開パッケージに列挙した。
Tenancy と Jobs は実装を `internal/` へ移し、保存、HTTP 登録、worker の構築をルートの操作へ集めた。
Tenancy のテスト補助 `testing_tenant` は公開ポートだけで書けるため公開した。
非公開 import の負債数は変わらず、残る 5 モジュールと `legacy` の撤去は wi-75383 が受け持つ。

全文は `git log -1 -- work-items/active/wi-74257-move-tenancy-claimmapping-and-jobs-behind-go-internal.md` が示すコミットの親、またはそのコミットで読む。
