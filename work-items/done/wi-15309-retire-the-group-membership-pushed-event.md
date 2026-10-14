---
status: completed
archived_from: 4ff319b8d34224d088c863525c9e77d94eb3fe61
---

# 発行されない GroupMembershipPushed のイベントと遷移を消す

`mise run spec-diff` は、`ProvisioningTaskLifecycle` の状態遷移の変更、`spec/modules/provisioning/models.tsp:GroupMembershipPushed` の削除、`GroupPushed` の変更を示す。
`GroupMembershipPushed` を、TypeSpec、Go のイベント型と遷移、タスクの状態遷移の表、Provisioning の README のイベント一覧、`tools/check/event-emission-debt.json` から消した。
`GroupPushed` の説明に、メンバーシップの追加が下流に届いたことも含むと明記した。
消した宣言は `affected_spec` から解決できないので、記録の参照先を、メンバーシップの送信の成功を表すようになった `GroupPushed` に改めた。

全文は `git show 4ff319b8d34224d088c863525c9e77d94eb3fe61:<このファイルのパス>` で読む。
