---
status: pending
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-11-17
priority: p3
group: federation-protocols
depends_on: [wi-10861-support-poppy-authorization-server-profile, wi-48552-map-cross-device-flows-to-rfc-10027]
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/standards.md, requirement: RFC8628-DEVICE-AUTHORIZATION }
  - { path: docs/modules/oauth2/standards.md, requirement: RFC8628-POLLING }
  - { path: docs/modules/oauth2/features/device.md, requirement: REQ-OAUTH2-027 }
  - { path: docs/modules/oauth2/features/consent.md, requirement: REQ-OAUTH2-008 }
---

# Poppy の Device Sign-In を提供する

## 動機

[[wi-10861-support-poppy-authorization-server-profile]] は Direct Sign-In だけを提供し、Poppy クライアントのほかの sign-in 方式を拒否する。
[Poppy 1.0](https://personalagentprotocol.org/docs/1.0/spec) §4.3 は、ブラウザーへリダイレクトを返せない Personal Agent（音声端末、CLI など）のために RFC 8628 の Device Sign-In を定める。
企業がこれを提供しないと、それらの Agent は Mediated Sign-In（利用者の資格情報を Agent が入力する方式で、2027年3月17日に削除予定）へ流れる。

## 対象範囲

- Poppy プロファイルを有効にした realm での、Poppy クライアントへのデバイス認可グラントの提供と AS メタデータの広告。
- 既存の同意があっても、要求ごとに同意画面を出すこと（§4.3 の SHOULD）。
- 検証 URL を issuer か企業ドメインの配下に置くこと。
- 企業全体の付与と資源の選択を、wi-10861 と同じ規則で扱うこと。

## 対象外

- Mediated Sign-In。資格情報の仲介は提供しない。
- Poppy 以外のデバイスフローの変更。RFC 10027 の対策は [[wi-48552-map-cross-device-flows-to-rfc-10027]] が扱う。

## 設計

既存のデバイスフロー（REQ-OAUTH2-027）を再利用し、Poppy プロファイルの適用を Direct Sign-In と同じ計算（クライアントの許可、スコープ必須、付与の種類）に通す。
`device_code` から得る付与を、認可コードと同じく企業全体の付与か資源に限った付与として記録する。
既存同意による同意画面の省略（REQ-OAUTH2-008）に、Poppy のデバイスフローでは省略しない例外を加える。

## 計画

1. wi-10861 の完了後、プロファイルの計算をデバイスフローへ広げる規範を書く。
2. 受け入れ試験の RED を確認してから実装する。

## タスク

- [ ] T001 [Spec] Poppy の Device Sign-In の規範と既存要件の例外を書く。
- [ ] T002 [Acceptance] 発見から Device Sign-In、資源の取得、失効までの RED を作る。
- [ ] T003 [App] 実装する。
- [ ] T004 [Verify] 検証する。

## 検証

- `mise run test-go-package -- ./backend/oauth2/internal/handlers_http`。
- `mise run test-go-package -- ./backend/shared/http/server_http`：正式な配線での全経路。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>`。

## リスク

デバイスフローは別端末の同意フィッシングに弱い。
wi-48552 の対策を前提にし、同意画面に Personal Agent の名前とドメインを必ず示す。
