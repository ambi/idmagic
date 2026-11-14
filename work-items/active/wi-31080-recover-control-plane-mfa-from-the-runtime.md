---
status: pending
authors: [tn]
risk: high
created_at: 2026-11-14
change_kind: feature
priority: p3
depends_on: []
affected_spec:
  - { path: docs/modules/authentication/features/mfa.md, requirement: REQ-AUTHENTICATION-022 }
---

# リセットできる管理者がいなくても制御面の運用者の MFA を復旧できるようにする

## 動機

[[wi-93893-verify-control-plane-mfa-recovery]] で、MFA を失った制御面の運用者の復旧は、デフォルトのテナントの `admin` による認証器のリセットに頼ると確かめた。
次の構成では、リセットできる人がいないので復旧できない。

- `admin` を持つのが MFA を失った運用者本人だけで、デフォルトのサインインポリシーが IdMagic 自身の画面に MFA を求めている。
- デフォルトのテナントに `admin` を持つ利用者がいない。

今は運用の予防（`admin` を二人以上置く、復旧コードを保管する）で避け、runbook に書いた。
予防を外れた構成では、データベースへの直接の操作しか手段がない。

## 対象範囲

- 実行環境の権限で、制御面の運用者の認証器をリセットする手段を設ける。Seeding と同じく HTTP の接点を持たず、プロセスを起動できる実行環境を境界にする案から検討する。
- 実行を監査に残す。

## 対象外

- break-glass の資格情報の自動発行。
- 外部 PAM との連携。

## 計画

1. 手段の候補（専用のコマンド、Seeding のマニフェストの拡張など）を比べて決める。
2. 規範と runbook を更新して実装する。

## タスク

- [ ] T001 [Decision] 手段を比べて決める。
- [ ] T002 [Spec] 規範を書く。
- [ ] T003 [App] 実装し、runbook の「復旧できない構成」を更新する。

## 検証

- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

実行環境の権限で認証器を消せる経路は、それ自体が強い権限になる。実行環境の境界と監査で守る。
