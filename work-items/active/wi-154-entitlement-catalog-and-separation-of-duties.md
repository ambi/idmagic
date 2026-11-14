---
depends_on: []
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-10
priority: p3
group: admin-governance
change_kind: feature
affected_spec:
  - { path: docs/modules/application/features/assignment.md, requirement: REQ-APPLICATION-011 }
  - { path: docs/modules/identity-management/features/group.md, requirement: REQ-IDMANAGEMENT-015 }
---

# Application ごとの権限の台帳と、同時に持てない権限の規則を導入する

## 動機

IdMagic は Application の割り当てとクレームの開示を持つが、アプリの中のロールや権限（entitlement）を台帳として管理し、User と Group に割り当てる機能はない。
Okta Entitlement Management と Microsoft Entra の entitlement management は、アプリの中のロールを台帳にし、職務の分離（Separation of Duties）の規則で競合する権限の同時の保有を防ぐ。
台帳がないと、SaaS の中の権限の過剰な付与に気付けず、監査で誰が何を持つかを説明できない。

## 対象範囲

- Application ごとの権限の台帳と、User と Group への権限の割り当てを持つ。Application の割り当てとの関係（権限の割り当てが Application の割り当てを伴うか）を決める。
- 同時に持てない権限の組を規則として持ち、割り当ての保存の時点で競合を拒否する。例外の承認を記録する。
- 権限の割り当てを、クレームの開示と下流へのプロビジョニングに渡す読み出しを用意する。
- 管理コンソールで、台帳、割り当て、規則、競合の事前確認を扱う。

## 対象外

- 細粒度認可（関係タプルによる評価）。AccessControl の既存の機能のまま変えない。
- 外部のアプリから権限を自動で発見するコネクター。
- アクセスの申請とレビューの流れ。wi-214 と wi-213 が扱い、この台帳を対象に加えるかはそれぞれが決める。

## 検証

- 競合する権限の割り当てが拒否され、例外を承認した割り当てだけが保存されることを観測する受け入れのテスト。
- `mise run verify`

## リスク

保存の時点の検査だけでは、Group の所属の変更で間接的に生まれる競合を見落とす。
Group 経由の割り当てを含めて評価する時点を、着手時に決める。
