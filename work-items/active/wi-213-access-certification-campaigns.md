---
depends_on: []
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-15
priority: p3
group: admin-governance
change_kind: feature
affected_spec:
  - { path: docs/modules/identity-governance/features/lifecycle-workflow.md, requirement: REQ-IDGOVERNANCE-001 }
  - { path: docs/modules/application/features/assignment.md, requirement: REQ-APPLICATION-011 }
---

# Group の所属、Application の割り当て、Agent を、レビュアーが定期的に見直すキャンペーンを導入する

## 動機

IdMagic は Group の所属と Application の割り当てを持つが、付与したアクセスが今も業務に要るかを定期的に確かめ、要らなければ失効させる仕組みがない。
Microsoft Entra の Access Reviews と Okta の Access Certifications は、レビュアーが対象の所属や割り当てを周期的に承認または却下し、回答がなければデフォルトの処置（失効か維持）を適用する。
これがないと、異動や退職の後に残ったアクセスに気付けず、監査で「誰が何を持ち、それを定期的に見直しているか」を示せない。

Agent も見直しの対象にする。
すべての Agent は所有者を持つので、所有者の異動や退職で放置された Agent は、人のアクセスと同じ理由で見直しが要る。
放置の検出そのものは、Agent の棚卸しの所見（[REQ-IDMANAGEMENT-094](../../docs/modules/identity-management/features/agent.md)）として既にある。

## 対象範囲

- 一回だけ、または周期的に実行するキャンペーンを持つ。対象は Group の所属、Application の割り当て、Agent とする。
- レビュアーは明示的に指定する。Agent のレビュアーは所有者とし、失効の判断は Agent の無効化に対応付ける。
- レビュアーの承認と却下を記録し、期限までに回答のない項目にデフォルトの処置を適用する。
- キャンペーンの結果の報告を管理コンソールで示す。

## 対象外

- 権限の台帳（wi-154）と管理者ロールの資格（wi-152）を対象にすること。対象の種類を後から足せる形にだけしておく。
- 上長をレビュアーに自動で解決すること。SCIM の Enterprise 拡張の `manager` は `User.Attributes` の `manager_sub` に入るが、SCIM 以外の経路では設定されず、管理者が確かめる手段もないので、レビュアーの根拠にしない。
- リスクに基づく見直しの順位付けと、機械学習による推奨。
- 外部のチケットシステムと SIEM への通知。

## 検証

- 却下した項目と、期限までに回答のない項目が、デフォルトの処置で失効することを観測する受け入れのテスト。
- `mise run verify`

## リスク

デフォルトの処置を失効にすると、レビュアーの見落としが業務を止める。
キャンペーンごとにデフォルトの処置を選ばせ、失効を選んだときは期限前にレビュアーへ通知する。
