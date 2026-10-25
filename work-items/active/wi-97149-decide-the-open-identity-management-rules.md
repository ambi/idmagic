---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-03
priority: p2
depends_on: [wi-96676-transcribe-implicit-specifications-of-identity-management]
change_kind: bugfix
affected_spec:
  - { path: docs/modules/identity-management/features/data-export.md, requirement: REQ-IDMANAGEMENT-040 }
  - { path: docs/modules/identity-management/features/account.md, requirement: REQ-IDMANAGEMENT-051 }
  - { path: docs/modules/identity-management/features/account.md, requirement: REQ-IDMANAGEMENT-052 }
  - { path: docs/modules/identity-management/features/account.md, requirement: REQ-IDMANAGEMENT-053 }
  - { path: docs/modules/identity-management/features/group.md, requirement: REQ-IDMANAGEMENT-060 }
  - { path: docs/modules/identity-management/features/group.md, requirement: REQ-IDMANAGEMENT-064 }
  - { path: docs/modules/identity-management/features/dynamic-group.md, requirement: REQ-IDMANAGEMENT-070 }
  - { path: docs/modules/identity-management/features/agent.md, requirement: REQ-IDMANAGEMENT-076 }
  - { path: docs/modules/identity-management/features/agent.md, requirement: REQ-IDMANAGEMENT-078 }
---

# IdManagement の規則に残した要判断を決める

## 動機

IdManagement の既存コードを書き起こしたとき、意図が疑わしいが外部から依存され得る挙動を、現在の挙動のまま規則に書き、要判断の欄に未決定の点を残した。
要判断は、決着したら消す欄である。
残したままでは、規則を読む人は現在の挙動を維持すべきか是正すべきかを判断できない。

| 規則 | 未決定の点 |
| --- | --- |
| REQ-IDMANAGEMENT-040 | エクスポートの一覧が種類で絞る前に直近 200 件で切るため、古いエクスポートが一覧から消える |
| REQ-IDMANAGEMENT-051 | 本人の更新が、値の変わらない属性も `changed_fields` に載せる |
| REQ-IDMANAGEMENT-052 | 本人のデータのエクスポートが、`admin_readable` と `private` の属性を含めない |
| REQ-IDMANAGEMENT-053 | 確認のメールの送信の失敗を、起票した本人へ伝えない |
| REQ-IDMANAGEMENT-060 | 管理 API と CSV で、表示名付きのメールアドレスの扱いが異なる |
| REQ-IDMANAGEMENT-064 | Group の変更が、下流への通知の失敗で確定したままエラーを返す |
| REQ-IDMANAGEMENT-070 | 動的な所属の変化が、メンバーごとのイベントを発行しない |
| REQ-IDMANAGEMENT-076 | Agent の無効化と再有効化が、すでにその状態でも時刻を進めてイベントを重ねる |
| REQ-IDMANAGEMENT-078 | 停止した Agent を削除できない |

2026-10-25 の棚卸しで、次の五点は現在の仕様で方針が確定していたため、未決定の対象から外した。
この整理は製品の振る舞いを変更せず、残る九点だけを本項目で扱う。

| 除外した規則 | 現在の判断を持つ一次情報 |
| --- | --- |
| REQ-IDMANAGEMENT-042 | [値オブジェクト](../../docs/modules/identity-management/data.md#値オブジェクト) が名前を case folding で比較し、REQ-IDMANAGEMENT-042 と REQ-IDMANAGEMENT-089 がメールアドレスの重複を拒否する |
| REQ-IDMANAGEMENT-043 | [作成の経路に共通する規則](../../docs/modules/identity-management/features/user.md#作成の経路に共通する規則) の REQ-IDMANAGEMENT-089 が JIT で作った User も動的グループの規則で評価する |
| REQ-IDMANAGEMENT-044 | [保持期限の削除](../../docs/modules/identity-management/features/user.md#保持期限の削除) と wi-92312 が、一覧の取得から Batch へ完全削除を移した |
| REQ-IDMANAGEMENT-045 | [ユーザーの仕様](../../docs/modules/identity-management/features/user.md) の REQ-IDMANAGEMENT-045 が、明示的な指定を伴わないメールアドレスの変更で `email_verified` を `false` にする |
| REQ-IDMANAGEMENT-049 | [復元](../../docs/modules/identity-management/features/user.lifecycle.md#復元) が、User の再有効化として下流へ通知する |

## 対象範囲

- 表の各点について、現在の挙動を維持するか是正するかを決める。
- 維持する点は、理由をその要件の **判断** の欄に書く。要判断の欄は、wi-17076 と wi-83002 の書き直しで消し、未決定の点はこの表だけに残っている。代替案を比べた判断であれば、`design/decisions.md` に書いてリンクする（wi-21670 の移行より前に着手する場合は `decisions.md` に書く）。
- 是正する点は、規則文を改め、実装とテストを合わせる。

## 対象外

- 表にない挙動の変更。
- 状態遷移表との食い違い。`wi-18703` が扱う。

## 計画

1. 各点の利用者への影響と、是正した場合の互換性を確かめる。
2. 維持と是正を決め、規則、判断、実装、テストを更新する。

## タスク

- [ ] T001 [Plan] 各点を維持するか是正するかを決める。
- [ ] T002 [Spec] 要件と判断の記述を更新する。
- [ ] T003 [App] 是正する点の実装とテストを更新する。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check`
- `mise run verify`

## リスク

- イベントの発行や通知を増やすと、それを数える利用者の集計が変わる。
  是正する場合はリリースノートで知らせる。
