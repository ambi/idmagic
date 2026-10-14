---
status: completed
archived_from: 64bb562848a58809bea036ae746d629afc99f760
---

# 所有者の User が止まると、その User が所有する Agent を止める

`mise run spec-diff -- work-item/wi-18703` は、追加した規則 REQ-IDMANAGEMENT-081 と 082、変更した規則 REQ-IDGOVERNANCE-008、追加した TypeSpec の `AgentOwnerInactiveError`、変更した `EnableAgent` を報告する。
所有者の User を無効化する、削除を予約する、完全削除すると、その User が所有する `Active` の Agent を `Disabled` にする。所有者が `Active` でない Agent の再有効化は 409 と `agent_owner_inactive` で拒否する。
ライフサイクルワークフローと SCIM の User の停止は、IdManagement の `UserLifecycleCommands` を通り、管理 API と同じイベントと副作用を伴う。
失効エポックは同じ時刻へは進めず、所有者の停止と Agent の無効化が重なっても Agent ごとの失効は一度だけになる。

全文は `git show 64bb562848a58809bea036ae746d629afc99f760:<このファイルのパス>` で読む。
