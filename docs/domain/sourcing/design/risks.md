# Sourcing のリスク

この文書は、Sourcing の既知の欠陥、技術的負債、受容したリスクを扱う。
脅威は[脅威モデル](../../../design/security/threat-model.md)に置く。

| リスク | 現状 | 影響 |
| --- | --- | --- |
| SCIM の作成と更新が IdManagement の操作を通らない | User の無効化と削除の予約だけが `UserLifecycle` のポートを通り、User と Group の作成と更新、Group のメンバーシップは Repository を直接保存する。この Context はドメインイベントを発行せず、リソース上限も確かめない | SCIM で作った User と Group には、監査の記録、ライフサイクルワークフローの捕捉、下流への通知が伴わず、上限を超えて作れる |
| Group の作成の後始末の失敗を捨てる | 作成の途中で失敗したときの Group と対応の記録の削除は、エラーを無視する | 外部の IdP から見えない孤立した Group が残りうる |
