---
status: completed
archived_from: 96d851960015b33264716f958196c7871f81586e
---

# テナントクォータ更新経路を Tenancy 管理 API の規約へそろえる

`spec-diff` は REQ-TENANCY-046 の追加、REQ-TENANCY-012 の変更、`UpdateTenantQuota` の変更を示した。
クォータ更新を realm で解決し、内部 ID へ保存・監査し、不在は本文の妥当性によらず 404 にする。
成功応答へ no-store を付け、保存失敗は共通処理で記録して内部文字列を秘匿する。
管理画面は realm を送信し、独自の呼び出し元は UUID を realm へ差し替える必要がある。
401、403、CSRF、本文の上限と監査発行は既存処理を維持した。

全文は `git show 96d851960015b33264716f958196c7871f81586e:<このファイルのパス>` で読む。
