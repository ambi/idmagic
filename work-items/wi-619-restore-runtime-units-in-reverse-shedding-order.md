---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-18
priority: p3
depends_on: []
change_kind: docs
spec_impact:
  kind: none
  reason: "ランブックの手順を設計に合わせるだけで、規範要素は変えない。"
---

# 復元後の実行単位の起動を、ロードシェディング順序の逆に並べる

## Motivation

[リカバリ設計](../docs/design/reliability/recovery.md#復元の順序)は、復元後に `idmagic-api`、ワーカーの `latency_sensitive`、`default`、`bulk`、`idmagic-batch` の順に起動すると定めている。
先に止めるものから戻すと、戻した処理が認証の中核とデータベースの接続を奪い合い、復旧の途中でロードシェディングがまた始まるためである。
現在の[バックアップ、復元、災害復旧のランブック](../docs/runbooks/backup-restore-dr.md)は、API とワーカーを同時に起動する。

## Scope

- ランブックの「復元時の整合性を保つ順序」を、設計の手順 6 から 8 に合わせる。
- 各手順で次へ進む前に確かめることを書く。

## Out of Scope

- 起動順序を自動化する仕組み。

## Design

ワーカーはレーンごとの Deployment のレプリカ数を 0 から戻す。
`bulk` を戻す前に、PostgreSQL の接続の使用率が 70% を下回っていることを確かめる。

## Plan

1. ランブックを書き直す。

## Tasks

- [ ] T001 [Docs] ランブックの起動順序を書き直す。
- [ ] T002 [Verify] `mise run check-links` と `mise run verify` を通す。

## Verification

- `mise run check-links`
- `mise run verify`

## Risk Notes

ランブックの手順が長くなると、障害対応中に読み飛ばされる。
確認は数値で判断できる形にする。
