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

# 利用者が Group と Application へのアクセスを申請し、承認されたら自動で付与する

## 動機

Group の所属と Application の割り当ては、管理者が直接行うしかない。
利用者が要るアクセスを自分で申請し、承認されたら自動で付与される流れがないので、付与はすべて管理者への個別の依頼（チケットやチャット）に頼り、遅れと承認の証跡の不足を生む。
Microsoft Entra の access package と Okta の Access Requests は、申請できるリソースのカタログ、承認、自動の付与と期限での失効を提供する。

## 対象範囲

- 申請できる Group と Application をカタログとして公開する。カタログの項目ごとに承認者を指定する。
- 利用者がアカウントポータルから申請し、承認者が承認または却下する。承認で自動で付与し、期限付きの付与は期限で失効させる。
- 申請、承認、却下、失効をドメインイベントとして記録する。同じ対象への重複した申請を防ぐ。

## 対象外

- 承認者の多段の連鎖（直列や並列の合意）。承認は一段で、指定した一人またはいずれか一人に限る。
- 権限の台帳（wi-154）の項目をカタログに含めること。項目の種類を後から足せる形にだけしておく。
- Agent の権限の申請。wi-60346 が、この項目と統合するか分けるかを判断する。
- 外部のチケットシステムとの連携。

## 設計

承認者を範囲付きの管理者ロール（wi-94）のリソースの担当者から解決するかは、wi-94 の状態を見て着手時に決める。
付与の実行は、ライフサイクルワークフローの割り当ての操作と同じ経路を通し、監査と冪等性を共有する。

## 検証

- 承認で付与され、却下では何も変わらず、期限で失効することを観測する受け入れのテスト。
- `mise run verify`

## リスク

申請者が承認者を兼ねると、承認が形だけになる。
申請者本人を承認者から除く規則を、保存の時点と承認の時点の両方で確かめる。
