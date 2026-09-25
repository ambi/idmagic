---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-25
risk_notes: |
  通知が届かないと、隔離で止まった下流の反映に管理者が気付かない。逆に宛先を誤ると、接続名や隔離の理由を第三者へ送る。
priority: p2
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-011 }
---

# 接続の隔離を `notification_email` へメールで通知する

## 動機

`EX-PROVISIONING-011-01` は、誤削除ガードで接続を隔離したら `notification_email` へ通知すると宣言している。
TypeSpec の `ProvisioningConnection.notification_email` も「失敗と隔離を通知する宛先」と説明している。

実装はどの経路でも通知しない。
連続失敗による隔離（`recordConsecutiveFailure`）も、[[wi-61629-quarantine-connections-on-the-accidental-deletion-guard]] が加える誤削除ガードによる隔離も、`ConnectionQuarantined` を発行するだけで、メールは送らない。
隔離はプロビジョニングを止める状態なので、管理者が気付かなければ、下流の反映はいつまでも止まったままになる。

wi-61629 はこの通知を対象外にした。
メールは共有の `Notifier` を通り、テンプレートのキーは Tenancy の TypeSpec `NotificationTemplateKey` と双子定義である。
キーを足すと、Tenancy の公開契約、OpenAPI のベースライン、管理画面の通知テンプレート一覧（ja と en）まで波及し、ガードとは独立に受け入れられる変更になるためである。

## 対象範囲

- 隔離を知らせる通知テンプレートのキーを `NotificationTemplateKey` へ足し、ja と en の組込み既定を用意する。
- 連続失敗と誤削除ガードの双方で、接続を隔離したときに `notification_email` へ送る。宛先が未設定なら送らない。
- `EX-PROVISIONING-011-01` を名指すテストを書き、`tools/check/example-coverage-debt.json` から外す。

## 対象外

- 隔離以外の失敗（`dead_letter` など）の通知。
- 通知の再送や配送の保証。

## 検証

- `mise run check-contract-drift`
- `mise run check-api-compat`
- `mise run verify`
