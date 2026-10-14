---
status: completed
archived_from: 3ba6a799b8be5741b759ec1739c307517d571cec
---

# audit-only event log を best-effort 記録と reconciliation に切り替える

HTTP adapter まで transaction を伝播する command envelope を撤去し、
audit-only event は既存の best-effort 経路で記録する形に戻した。event sink、audit record
変換、audit record append の各失敗を構造化ログに残すようにした。

全文は `git show 3ba6a799b8be5741b759ec1739c307517d571cec:<このファイルのパス>` で読む。
