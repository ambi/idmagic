---
status: completed
---

# SCIM による User と Group の作成と更新を IdManagement のユースケースへ通す

REQ-SOURCING-002、005 に、SCIM の書き込みを IdManagement の操作へ通し、使用量、イベント、ワークフロー捕捉と下流通知を共有することを定めた。
User と対応表、または Group と初期所属と対応表を同じ PostgreSQL トランザクションで確定し、イベントと通知を確定後へ遅らせる。
User と Group の作成上限は、TypeSpec と要件に定めた 422 の SCIM エラーで拒否する。
既存の `active=false` による Disabled の作成は要件へ書き起こして維持した。
既存 Group の複数所属の更新は原子性の対象外とし、補償の限界を現在のリスクへ残した。

全文は `git log -1 -- work-items/active/wi-11577-route-scim-writes-through-identity-management-use-cases.md` が示すコミットの親、またはそのコミットで読む。
