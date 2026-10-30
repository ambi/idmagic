---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-31
priority: p2
depends_on: [wi-307-datakeys-rotation-lifecycle-operations]
change_kind: bugfix
affected_spec:
  - { path: docs/modules/data-keys/features/lifecycle.md, requirement: REQ-DATAKEYS-003 }
  - { path: docs/modules/data-keys/features/lifecycle.md, requirement: REQ-DATAKEYS-005 }
---

# 無効化したバージョンの行が再暗号化を止めないようにする

## 動機

`MfaFactorReencryptor.ReencryptBatch` は、復号に失敗した行で一括処理全体を止める。
無効化したバージョンの行は復号できないので、そのテナントの再暗号化のジョブは、ほかのバージョンの行を移せないまま失敗を繰り返す。
`data-key-lifecycle` は未移行の行を報告する組に毎日ジョブを投入するので、失敗も毎日続く。
未移行の件数が 0 にならないため、無効化したバージョンの破棄も拒否され続ける。

wi-307 で緊急の無効化の経路を入れたときに、`docs/modules/data-keys/risks.md` に載せた。

## 対象範囲

- 復号できない行を飛ばして残りを移すかを決め、すべての `FieldMigrator` に適用する。
- 読めなくなった行を、未移行の件数と破棄の確認でどう扱うかを決める。

## 対象外

- 破棄を起動する経路そのもの。

## 検証

- 無効化したバージョンの行と退役したバージョンの行を混ぜ、後者が移ることを確かめるテスト

## リスク

読めなくなった行を飛ばすと、破棄の確認が緩む方向に働き得る。
未移行の件数の意味を変える場合は、破棄の前の確認の保証を保つ。
