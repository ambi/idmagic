---
depends_on: []
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-03
priority: p3
group: admin-governance
change_kind: feature
affected_spec:
  - { path: docs/modules/identity-management/features/admin-access.md, requirement: REQ-IDMANAGEMENT-014 }
  - { path: docs/modules/identity-management/features/admin-access.md, requirement: REQ-IDMANAGEMENT-025 }
---

# 対象の範囲を限った管理者ロールで、テナントの一部の管理を任せられるようにする

## 動機

管理 API は、実効ロールに `admin` を持つ User の要求を受け付ける（[REQ-IDMANAGEMENT-014](../../docs/modules/identity-management/features/admin-access.md)）。
`admin` はテナント全体に効くので、特定の Group や Application だけを管理する副管理者を表せない。
Microsoft Entra の管理単位、Okta の admin role と resource set、Keycloak の fine-grained admin authorization のように、大きな組織では委任管理が要る。

## 対象範囲

- 「対象のリソースの集合 × 管理の権限」で表す範囲付きの管理者ロールと、その割り当てを持つ。対象の次元は Group、Application、User の属性の集合から選ぶ。
- 管理 API の各操作の認可に範囲を反映し、範囲の外のリソースへの管理の操作を拒否する。デフォルトは拒否とする。
- 割り当ての作成と取り消しをドメインイベントとして記録し、管理コンソールで扱う。

## 対象外

- 利用者向けの細粒度認可（関係タプルによる評価）。範囲付きの管理者ロールは管理の操作の認可に限る。
- 汎用のポリシー言語（Rego、Cedar など）の導入。
- テナントをまたぐ委任。
- 管理者ロールの一時的な有効化。wi-152 が扱う。

## 設計

着手時に、範囲の判定を管理 API の共通の認可で行うか、各操作のユースケースで行うかを決める。
一覧の操作は、範囲の外のリソースを結果から除く必要があり、単一のリソースの操作と判定の形が違う。
判定の方針は[認可の設計](../../docs/design/security/authorization.md)に書く。

## 検証

- 範囲の外のリソースに対する更新が拒否され、一覧が範囲の内側だけを返すことを観測する受け入れのテスト。
- `mise run verify`

## リスク

範囲の判定の漏れは、副管理者に範囲の外のリソースを操作させる。
管理 API の全操作に範囲の判定があることを、操作の一覧と照合する検査で確かめる。
