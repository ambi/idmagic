---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# OAuth2 のスコープとロールポリシーが宣言する具体例 6 件にテストを対応付け、残る 99 件を規則群ごとに割る

`mise run spec-diff` は `no normative specification change against main` を返す。規範は 1 行も動いていない。
変わったのは 2 つである。`docs/domain/oauth2/scenarios.feature.md` の REQ-OAUTH2-002、REQ-OAUTH2-003、
REQ-OAUTH2-004 が宣言する 6 件のうち 5 件が、その id を名指しするテストから到達されるようになったこと
（台帳は 573 件から 568 件へ減った）。そして、残る 99 件の割り方が測定にもとづいて決まり、
5 つの子 work item として存在するようになったことである。製品コードは 1 行も変えていない。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
