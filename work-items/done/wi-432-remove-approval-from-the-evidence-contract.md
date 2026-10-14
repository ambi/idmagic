---
status: completed
archived_from: 67100f2b2f3cd79c718e25e9517d4b67e059912a
---

# 実装前の承認を証拠契約から外し、リスクに応じた証拠だけを残す

`mise run spec-diff` は `no normative specification change against main` を返す。規範仕様は動いていない。変わったのは証拠契約であり、`medium` 以上の work item が実装開始前に `approval: {by, at, scope, baseline}` を記録する義務が消え、完了記録の `Post-Approval Changes` も消えた。残ったのは、リスク表そのもの、`medium` 以上の独立検証と変化耐性の証拠、Acceptance RED / Unit RED である。`DEVELOPMENT.md` §4 の見出しは `Evidence contract and approval` から `Evidence contract` になり、`medium` の `Before implementation` は「`low` の要求を適用する」— すなわち `initial_context` の確定、作るものを変えうる疑問の解決、両 RED の指名 — になった。完了時の Summary は `mise run spec-diff` から読み取ることが明示され、`Post-Approval Changes` が担っていた観測はそこに吸収された。

全文は `git show 67100f2b2f3cd79c718e25e9517d4b67e059912a:<このファイルのパス>` で読む。
