---
status: pending
authors: [tn]
risk: critical
created_at: 2026-10-24
priority: p3
depends_on: [wi-11530-delegate-password-verification-to-active-directory-through-the-connector]
change_kind: feature
affected_spec:
  - { path: docs/modules/authentication/features/password.md, requirement: REQ-AUTHENTICATION-010 }
  - { path: docs/modules/authentication/features/password.md, requirement: REQ-AUTHENTICATION-016 }
---

# IdMagic でのパスワードの変更と再設定を、Directory Connector で AD へ書き戻す

## 動機

認証を AD へ委ねた User（[[wi-11530-delegate-password-verification-to-active-directory-through-the-connector]]）は、IdMagic でパスワードを変更も再設定もできない。
利用者が AD のパスワードを忘れると、ヘルプデスクへの依頼が要る。

| 製品 | 機能 |
| --- | --- |
| Okta | AD の利用者のパスワードの再設定を Okta で許す設定と、Okta での変更を AD へ同期するパスワードの同期。Agent のサービスアカウントに AD のパスワードの再設定の権限が要る |
| Microsoft Entra | セルフサービスのパスワードの再設定の書き戻し |

## 対象範囲

- 委譲を有効にした取り込み元の User が IdMagic でパスワードを変更または再設定したとき、Connector の経路で AD の `unicodePwd` を書き換える。
- AD のパスワードのポリシーによる拒否を、変更と再設定の拒否として利用者へ返す。
- 書き戻しを使う取り込み元だけ、Connector にパスワードの再設定の権限を持つ別のサービスアカウントを構成する。

## 対象外

- OpenLDAP への書き戻し。需要が出てから足す。
- AD で変えたパスワードの IdMagic への同期。委譲により要らない。

## 設計

### 未解決の問い

- 変更（現在のパスワードを確かめる）と管理者の再設定（確かめない）で、AD の操作を分けるか。
- 書き戻しの権限を持つサービスアカウントを、読み取り用と分けて置く手順。

## 計画

1. 未解決の問いを決め、Authentication と DirectoryConnector の仕様を更新してから実装する。

## タスク

- [ ] T001 [Plan] 未解決の問いを決める。
- [ ] T002 [Spec] 要件を書く。
- [ ] T003 [App] 書き戻しを実装する。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- AD の `unicodePwd` の書き換えは LDAPS を要し、OpenLDAP の fixture では確かめられない。Samba AD のコンテナか実物の AD で確かめる方法を着手時に決める。

## リスク

- Connector に AD のパスワードの再設定の権限を与えると、Connector のホストが侵害されたときの被害が読み取りだけの場合より大きくなる。書き戻しを有効にした取り込み元に限り、権限を分ける。
