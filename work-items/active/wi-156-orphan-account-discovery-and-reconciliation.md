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
  - { path: docs/modules/provisioning/features/connection.md, requirement: REQ-PROVISIONING-013 }
  - { path: docs/modules/provisioning/features/task.md, requirement: REQ-PROVISIONING-008 }
---

# 下流のアプリのアカウントの一覧を取り込み、IdMagic の User と照合して管理外のアカウントを示す

## 動機

下流への SCIM のプロビジョニングがあっても、アプリの側には、IdMagic の外で作られたアカウント、退職者の残ったアカウント、属性の食い違ったアカウントが残り得る。
Microsoft Entra ID Governance の orphan account の発見のように、アプリのアカウントの一覧を取り込み、IdMagic の User と Application の割り当てに照合できないと、管理外のアカウントを見つけて直せない。

## 対象範囲

- プロビジョニングの接続ごとに、下流のアカウントの一覧を取り込む。SCIM の一覧の取得を初めの手段とし、CSV の取り込みも受ける。
- 取り込んだアカウントを IdMagic の User と照合し、一致、不一致、曖昧、状態の食い違い（IdMagic では無効だがアプリでは有効など）に分ける。
- 管理者が照合の結果を確認し、対応付け、無視、下流での無効化の依頼を選べる画面と API を加える。

## 対象外

- 下流のアカウントの自動の削除。
- SCIM 以外のコネクターの実装。
- 人事システムを正とする同期。

## 検証

- 照合の規則ごとに分類が決まることを固定するテスト。
- 模擬の下流から取り込んだ一覧で、管理外のアカウントが示されることを観測する受け入れのテスト。
- `mise run verify`

## リスク

照合の規則が緩いと、別人のアカウントを同じ User に対応付ける。
自動の対応付けは一意に一致した場合に限り、曖昧な場合は管理者の確認を求める。
