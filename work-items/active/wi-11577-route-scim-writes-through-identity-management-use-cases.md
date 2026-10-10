---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/sourcing/scim/README.md, requirement: REQ-SOURCING-002, impact: conforms }
  - { path: docs/modules/sourcing/scim/README.md, requirement: REQ-SOURCING-005, impact: conforms }
---

# SCIM による User と Group の作成と更新を IdManagement のユースケースへ通す

## 動機

SCIM の取り込みのうち、User の無効化と削除の予約だけは `UserLifecycle` のポートを通り、管理 API で同じ変更をしたときと同じイベントと通知が伴う。
一方、User と Group の作成と更新、Group のメンバーシップは、IdManagement の Repository を直接保存する。
Sourcing はドメインイベントを発行せず、リソース上限も確かめない。
SCIM で作った User と Group には、監査の記録、ライフサイクルワークフローの捕捉、下流への通知が伴わず、上限を超えて作れる。

また、Group の作成の途中で失敗したときの後始末（作った Group と対応の記録の削除）はエラーを捨てるので、外部の IdP から見えない孤立した Group が残りうる。

wi-26063 で Sourcing の内部設計をコードと照合して見つけ、`docs/modules/sourcing/design/risks.md` に載せた。

## 対象範囲

- SCIM の User と Group の作成、更新、メンバーシップの変更を、IdManagement のユースケースまたは公開する操作へ通す。
- 作成と対応の記録の保存を、片方だけが残らないように確定する。
- `docs/modules/sourcing/design/risks.md` の該当の行を消し、`scim/design.md` を直す。

## 対象外

- SCIM の属性の対応付けの変更。

## 設計

ユースケースへ通すと、SCIM の取り込みにもリソース上限が効くようになる。
上限を超えた SCIM の作成に返す SCIM のエラーの形を、RFC 7644 と照らして着手時に決める。
作成の後始末は、対応の記録の保存を User と Group の保存と同じトランザクションにする案と、照合で回収する案を比べて決める。

## 計画

1. SCIM の作成で、イベントが発行されないことと上限が効かないことを RED として確かめる。
2. 書き込みをユースケースへ移す。
3. 作成の原子性を直す。

## タスク

- [ ] T001 [App] SCIM の書き込みをユースケースへ通す。
- [ ] T002 [App] 作成の原子性を直す。
- [ ] T003 [Verify] 検証する。

## 検証

- `mise run test-go-changed`
- `mise run verify`

## リスク

上限が効くようになると、既存の外部の IdP の同期が上限で止まりうる。
リリース文書で知らせる。
