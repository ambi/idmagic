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
  - { path: docs/modules/identity-management/features/admin-access.md, requirement: REQ-IDMANAGEMENT-014 }
  - { path: docs/modules/identity-management/features/admin-access.md, requirement: REQ-IDMANAGEMENT-025 }
---

# 管理者ロールを、理由と期限とステップアップを満たしたときだけ一時的に有効にする

## 動機

管理 API は、要求先のテナントで実効ロールに `admin` を持つ User の要求を受け付ける（[REQ-IDMANAGEMENT-014](../../docs/modules/identity-management/features/admin-access.md)）。
`admin` は User のロールとして常に有効であり、管理者は普段から強い権限を持ち続ける。
資格情報が盗まれたときの影響が大きい。
Microsoft Entra Privileged Identity Management は、対象者が理由、期限、MFA、必要なら承認を満たしたときだけロールを一時的に有効にする。

## 対象範囲

- 管理者ロールの資格（有効にできる権利）と、有効化（期限付きで有効な状態）を分ける。実効ロールの評価は、資格だけでは `admin` を与えない。
- 有効化の申請に、理由、期限、直近のステップアップを求める。テナントの設定で承認を必須にできる。
- 申請、承認、拒否、期限切れ、取り消しをドメインイベントとして記録する。
- 本人が資格を一覧して有効化を申請し、承認者が判断できる画面と API を加える。

## 対象外

- 範囲を限った管理者ロール（委任管理）。wi-94 が扱う。両方を採るときの関係は、先に着手したほうが決める。
- 外部の PAM とチケットシステムとの連携。
- 緊急用アカウント（break-glass）の運用。
- テナントをまたぐ特権のアクセス。制御面の運用者の権限は System が扱う。

## 設計

着手時に、既存の `admin` を持つ User を資格へ移す移行の方法と、テナントの最後の管理者が期限切れでいなくなる状態を防ぐ規則を決める。
後者を決めずに有効化を必須にすると、テナントの管理が誰にもできなくなる。

## 検証

- 資格だけの User の管理 API の要求が拒否され、有効化の期限が切れた後も拒否されることを観測する受け入れのテスト。
- `mise run verify`

## リスク

既存のテナントの管理者を資格へ移すと、利用者の操作の手順が変わる。
移行はテナントの設定で選べるようにし、デフォルトでは常に有効なロールを残す。
