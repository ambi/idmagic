---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-01
priority: p2
depends_on: [wi-78471-transcribe-implicit-specifications-of-tenancy]
change_kind: bugfix
affected_spec:
  - { path: docs/domain/tenancy/lifecycle/README.md, requirement: REQ-TENANCY-026 }
  - { path: docs/domain/tenancy/lifecycle/README.md, requirement: REQ-TENANCY-027 }
  - { path: docs/domain/tenancy/settings/README.md, requirement: REQ-TENANCY-031 }
  - { path: docs/domain/tenancy/branding/README.md, requirement: REQ-TENANCY-032 }
  - { path: docs/domain/tenancy/quota/README.md, requirement: REQ-TENANCY-037 }
  - { path: docs/domain/tenancy/notification-template/README.md, requirement: REQ-TENANCY-039 }
---

# Tenancy の規則に残した要判断を決める

## 動機

Tenancy の既存コードを書き起こしたとき、意図が疑わしいが外部から依存され得る挙動を、現在の挙動のまま規則に書き、要判断の欄に未決定の点を残した。
要判断は、決着したら消す欄である。
残したままでは、規則を読む人は現在の挙動を維持すべきか是正すべきかを判断できない。

| 規則 | 未決定の点 |
| --- | --- |
| REQ-TENANCY-026 | 上限と使用量を読み取れないテナントを、項目を省いて一覧へ返す |
| REQ-TENANCY-027 | すでにその状態にあるテナントの無効化と再開が、`disabled_at` を動かしイベントを重ねて発行する |
| REQ-TENANCY-031 | 値が変わらない項目も `TenantUpdated` の `changed_fields` に載る |
| REQ-TENANCY-032 | フッターリンクのラベルを UTF-8 で 80 バイト以下に制限し、日本語のラベルを 26 文字までにしている |
| REQ-TENANCY-037 | 負の上限を検証せずに保存する |
| REQ-TENANCY-039 | 何も削除しなかったリセットも `NotificationTemplateReset` を発行する |

## 対象範囲

- 表の各点について、現在の挙動を維持するか是正するかを決める。
- 維持する点は、要判断の欄を消し、理由を `decisions.md` に書く。
- 是正する点は、規則文を改め、実装とテストを合わせる。

## 対象外

- 表にない挙動の変更。
- クォータ更新の経路の規約合わせ。`wi-471` が扱う。

## 計画

1. 各点の利用者への影響と、是正した場合の互換性を確かめる。
2. 維持と是正を決め、規則、判断、実装、テストを更新する。

## タスク

- [ ] T001 [Plan] 各点を維持するか是正するかを決める。
- [ ] T002 [Spec] 規則と `decisions.md` を更新し、要判断の欄を消す。
- [ ] T003 [App] 是正する点の実装とテストを更新する。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check`
- `mise run verify`

## リスク

- イベントの発行を減らすと、そのイベントを数える利用者の集計が変わる。
  是正する場合はリリースノートで知らせる。
