---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-06
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/authentication/features/trusted-device.md, requirement: REQ-AUTHENTICATION-028 }
---

# User の削除を予約したときも、記憶済みの端末を失効させる

## 動機

[wi-10874](../done/wi-10874-inventory-promises-kept-by-every-entry-path.md) の棚卸しで、User を止める二つの操作のうち、無効化だけが記憶済みの端末を失効させることが分かった。
`SetUserDisabled` は `revokeTrustedDevicesOnDisable` を呼ぶが、`SoftDeleteUser` は呼ばない。
[SCIM の設計](../../docs/modules/sourcing/features/scim.md)と [IdManagement の設計](../../docs/modules/identity-management/README.md)は、`DELETE /Users/{id}` による削除の予約にも「記憶済みの端末の失効」が伴うと書く。
一方、[REQ-AUTHENTICATION-028](../../docs/modules/authentication/features/trusted-device.md) が失効の契機に挙げるのは無効化だけで、削除の予約を挙げていない。

削除を予約した User は `RestoreUser` で `Active` に戻る。
戻ったとき、予約の前の記憶済みの端末が第二要素を省略させる。
無効化で端末を失効させた理由（wi-91：再有効化で古い信頼を生き返らせない）は、予約からの復元にも同じく当たる。

## 対象範囲

- REQ-AUTHENTICATION-028 の契機に、User の削除の予約を加える。
- 管理 API、SCIM、Directory Connector の取り込みのどの経路で削除を予約しても、その User の `Active` の記憶済みの端末を `Revoked` にし、端末ごとに `TrustedDeviceRevoked` を発行する。

## 対象外

- 既に予約中の User に残った端末の失効。
- 完全削除の匿名化。既に `DeleteAllForSub` が物理削除する。

## 設計

### 所有の形

候補 A を採る。
IdManagement の User のユースケースに、User を止める共通の非公開の手順を置き、記憶済みの端末の失効と所有する Agent の無効化をその中に集める。
`SetUserDisabled` と `SoftDeleteUser` は、状態の書き換えの後にこの手順を呼ぶ。
`revokeTrustedDevicesOnDisable` と `disableOwnedAgents` を個別に呼ぶ箇所を残さない。

失効の理由は `TrustedDeviceRevokeReason` に削除の予約を表す値を足すか、`account_disabled` を流用するかを着手時に決める。
値を足す場合は `spec/` の列挙と監査の表示を同時に更新する。

## タスク

- [ ] T001 [Spec] REQ-AUTHENTICATION-028 に削除の予約を加え、例を足す。
- [ ] T002 [Test] 管理 API と SCIM の削除の予約で端末が失効することを観測するテストを RED にする。
- [ ] T003 [App] User を止める共通の手順を置き、二つの操作をそこへ通す。

## 検証

- `mise run test-go-package -- ./backend/idmanagement/internal/user/usecases`
- `mise run check-spec`

## リスク

- 削除の予約の取り消しを想定した運用で、利用者が端末の記憶をやり直す必要が生じる。
  予約からの復元は稀な操作であり、失効を優先する。
