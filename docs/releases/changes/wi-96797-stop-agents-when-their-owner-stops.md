# WI-96797: Stop agents when their owner stops

作業項目は `wi-96797-stop-agents-when-their-owner-stops` である。

WI-96797 は、所有者の User が止まると、その User が所有する Agent も止める。

所有者の User を無効化する、削除を予約する、完全削除すると、その User が所有する `Active` の Agent が `Disabled` になり、Agent ごとに `AgentDisabled` が発行される。
所有者の User を再有効化または復元しても、Agent は `Disabled` のまま残る。
すでに止まっている User をもう一度無効化する、または削除をもう一度予約すると、残っている `Active` の Agent だけを無効化する。
規範上の条件は[エージェント](../../domain/identity-management/agent/README.md)の REQ-IDMANAGEMENT-081 が定める。

所有者の User が `Active` でない Agent の再有効化は、`POST /api/admin/v1/agents/{agent_id}/enable` が 409 と `agent_owner_inactive` で拒否し、Agent を変えない。
所有者を `Active` に戻すか、`Active` の別の User へ所有者を変えてから再有効化する。
規範上の条件は同じ文書の REQ-IDMANAGEMENT-082 が定める。

ライフサイクルワークフローの `disable_user` と `enable_user`、SCIM の `active` の切り替えと `DELETE` は、管理 API と同じ IdManagement の User の操作を通るようになる。
これらの経路でも、`UserDisabled`、`UserEnabled`、`UserSoftDeleted` が発行され、記憶済みの端末の失効、下流のアプリケーションへの通知、所有する Agent の無効化が伴う。
監査イベントの操作者は、ワークフローでは `lifecycle-workflow`、SCIM では `scim` になる。
ワークフローの手順が削除予約中の User を無効化または再有効化しようとしたときは、User を変えず、ステップは変更なしになる。

Agent の所有者は、同じテナントの User だけである。
用語集が Group も所有者になれるとしていた記述を直した。

Agent の失効エポックは、すでに同じ時刻のエポックがあれば進めない。
所有者の停止とそれに伴う Agent の無効化が重なっても、`AgentAccessRevoked` と SSF の送信は Agent ごとに一度だけになる。
