---
status: completed
archived_from: b217be43ba27ef4c7f9cb70e555d35d81563ef7c
---

# IdManagement が宣言しているが、どの経路からも起きないエラーコードとイベントを消す

`mise run spec-diff` は、TypeSpec の 6 宣言の削除と 12 操作の応答変更を報告した。
User、Group、GroupMembership のインポートの開始、適用、参照の 9 操作から、依存未配線時の 503 を削除した。
エクスポートの開始の 3 操作から、ルートで固定する対象の `InvalidExportTargetError` を削除した。
未発行の `FederationLinked` と `FederationUnlinked` を宣言、監査の候補、UI の分類、二つの負債リストから削除した。
新しい要件、連携機能、データの移行は追加していない。

全文は `git show b217be43ba27ef4c7f9cb70e555d35d81563ef7c:<このファイルのパス>` で読む。
