---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-04
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/domain/signing-keys/lifecycle/README.md, requirement: REQ-SIGNINGKEYS-010 }
---

# 検証用の署名鍵の無効化をドメインイベントと状態遷移で表す

## 動機

管理者による検証用の鍵の無効化（REQ-SIGNINGKEYS-010）は、鍵の `archived_at` を立てて JWKS から外すが、ドメインイベントを発行しない。
`SigningKeyRetired` は宣言されているが、どの処理も発行しない。
そのため、鍵の無効化が監査の記録に残らない。

また、機能仕様の状態遷移の表（`SigningKeyLifecycle`）には、無効化による `Verifying` から `Archived` への直接の遷移がない。
`Retired` も保存した値を持たず、期限の経過だけで表される。

wi-26063 で SigningKeys の内部設計をコードと照合して見つけ、`docs/domain/signing-keys/design/risks.md` に載せた。

## 対象範囲

- 無効化がドメインイベントを発行するようにする。
- 状態遷移の表に無効化の遷移を加え、`Retired` の扱いを表とコードで一致させる。
- REQ-SIGNINGKEYS-010 に、発行するイベントを要件として加える。
- `docs/domain/signing-keys/design/risks.md` の該当の 2 行を消す。

## 対象外

- XML の署名の鍵の自動のローテーション。wi-55565 が扱う。

## 設計

イベントは、宣言だけある `SigningKeyRetired` を使うか、`SigningKeyDisabled` を新しく設けるかを着手時に決める。
`Retired` を状態の表から外し、`Verifying` の期限切れを時刻の判定として書くか、`Retired` を保存した状態にするかも同時に決める。
状態の表を変えるので、規範の変更として `spec-change` から始める。

## 計画

1. 状態遷移の表と REQ-SIGNINGKEYS-010 を改める。
2. 無効化のテストで、イベントの発行の RED を確かめ、実装する。

## タスク

- [ ] T001 [Spec] 状態遷移の表と要件を改める。
- [ ] T002 [App] 無効化でイベントを発行する。
- [ ] T003 [Verify] 検証する。

## 検証

- `mise run check-spec`
- `mise run verify`

## リスク

状態の表の変更は、生成する状態図と状態遷移表の検査に及ぶ。
