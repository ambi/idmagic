---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-04
priority: p2
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/authentication/security-notification/README.md, requirement: REQ-AUTHENTICATION-031 }
---

# 管理者が User になりすますセッションを始め、終えられるようにする

## 動機

`SessionImpersonationStarted` と `SessionImpersonationEnded` のイベント、通知の種別 `impersonation`、REQ-AUTHENTICATION-031 のなりすましの条項、受信設定の画面の項目は宣言されている。
しかし、管理者が User になりすますセッションを始める機能そのものがバックエンドにも UI にもないため、どの経路もこれらのイベントを発行せず、`impersonation` の通知も届かない。
Token Exchange のなりすまし（`act` を省き `sub` を差し替える形）も対象外として拒否している。

## 対象範囲

- 管理者が User になりすますセッションを開始し、終了する操作の要件、管理 API、UI。
- 開始で `SessionImpersonationStarted`、終了で `SessionImpersonationEnded` を発行し、なりすまされた User へ `impersonation` の通知を送る。
- 語彙の検査の許容リストから二つのイベントを外す。

## 対象外

- Token Exchange によるなりすまし。

## 設計

なりすましを許す管理者の権限、セッションの有効期間、なりすまし中に禁じる操作（資格情報の変更など）、監査での行為者の表し方を、着手時に決める。

## タスク

- [ ] T001 [Spec] なりすましの開始と終了の要件と管理 API を書く。
- [ ] T002 [App] セッション、イベントの発行、通知を実装する。
- [ ] T003 [UI] 管理画面に開始と終了の操作を置く。
- [ ] T004 [Verify] `mise run verify` を通す。

## 検証

- `mise run verify`

## リスク

- なりすましは管理者の権限を User の権限へ広げる操作なので、認可の漏れがそのまま本人の資格情報の変更につながる。
