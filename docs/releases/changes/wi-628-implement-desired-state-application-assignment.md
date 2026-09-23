# wi-628-implement-desired-state-application-assignment

ライフサイクルワークフローの `assign_application` と `unassign_application` は、User への直接割り当てを指定どおりの状態にそろえる操作になった（`REQ-APPLICATION-014`）。

- 割り当てを実際に作成、更新、削除したときだけ、監査へ `ApplicationAssigned` または `ApplicationUnassigned` を記録する。actor は `lifecycle-workflow` になる。
- 直接割り当てを作成または削除したときは、管理 API による割り当てと同じく Provisioning へ通知する。
- 直接割り当てが指定と異なる `visibility` で存在する場合は、指定どおりに更新してステップを `changed` とする。これまでは `no_op` として変更しなかった。
- 指定どおりの直接割り当てがすでに存在する場合は、何も変えずステップを `no_op` とする。
- グループ割り当ては変更しない。直接割り当てを解除しても、グループを介した割り当てによるフェデレーションは引き続き許可される。
