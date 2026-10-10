---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-04
priority: p3
depends_on: []
change_kind: refactor
spec_impact:
  kind: none
  reason: "どの操作も遷移しない UserStatus の値を消すだけで、管理 API の応答、保存されている User の状態、発行するイベント、外向きの呼び出しは変えない。"
---

# どの遷移も到達しない UserStatus の値を消す

## 動機

`UserStatus` は、UserLifecycle の状態の表にない `locked`、`staged`、`suspended` を宣言している。
本番コードはどの値へも遷移せず、TypeSpec の契約にも現れない。
状態遷移表（マトリクス形式）から予測するモデルベースのテストは表の状態だけを試すので、これらの値は仕様にもテストにも現れないまま `Valid()` を通る。
未記載の振る舞いの分類では (c) 実装を直す、に当たる。

## 対象範囲

- `backend/idmanagement/domain/enums.go` から三つの値を消し、`Valid()` を状態の表の四つに限る。
- 三つの値を使うテストを、表の状態を使う形に直す。

## 対象外

- 状態の表へ新しい状態を加えること。
  必要になったら、仕様の変更として別に起票する。

## 検証

- `mise run test-go-changed`
- `mise run verify`

## リスク

- 保存済みの User が三つの値のどれかを持っていると、読み込みで拒否される。
  製品は未リリースであり、本番コードはこれらの値を書かないので、移行は要らない。
