---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-04
priority: p3
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/domain/sharedsignals/transmitter/README.md, requirement: REQ-SHAREDSIGNALS-006 }
---

# 配信不能になった SET の配送をやり直せるようにする

## 動機

`SecurityEventDelivery` は `max_delivery_attempts` を使い切ると `dead_letter` になり、以後は送らない。
`dead_letter` の配送を `pending` へ戻す操作も、作り直す操作もない。
受信側の長い障害の後、その間の失効は外部へ伝わらないまま残る。

wi-26063 で SharedSignals の内部設計をコードと照合して見つけ、`docs/domain/sharedsignals/design/risks.md` に載せた。

## 対象範囲

- 管理者が、ストリームの `dead_letter` の配送をやり直せる操作を加える。
- 状態遷移の表に、やり直しの遷移を加える。
- `docs/domain/sharedsignals/design/risks.md` の該当の行を消す。

## 対象外

- 配送の再試行の間隔と上限の変更。

## 設計

やり直しは、Provisioning のタスクの再試行（`dead_letter` から新しい `pending` のレコードを作る）にそろえる案を第一の候補とする。
同じレコードを戻すと、`dead_letter` を終端とする状態の表と食い違うからである。
古い失効の SET を、受信側の復旧の後にまとめて送ることの意味（受信側が古い事象をどう扱うか）を、着手時に CAEP の規定と照らして確かめる。

## 計画

1. 要件と状態の表を改める。
2. 管理 API を TypeSpec に加え、実装する。

## タスク

- [ ] T001 [Spec] 要件、状態の表、TypeSpec を改める。
- [ ] T002 [App] やり直しを実装する。
- [ ] T003 [Verify] 検証する。

## 検証

- `mise run check-spec`
- `mise run verify`

## リスク

古い SET の再送で、受信側が失効を二重に扱う。
SET の `jti` を作り直すか保つかで、受信側の重複排除の挙動が変わる。
