---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-11-07
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/frontend/features/localization.md, requirement: REQ-SYSTEM-011 }
---

# API の成功応答の壊れた本文を空のオブジェクトとして扱わない

## 動機

`frontend/src/api/core.ts` の `doFetch` は、`response.json()` の失敗を `{}` に置き換え、`as T` として返す。
成功の状態コードで本文が壊れていると、画面は空の結果を正常に受け取ったものとして描画する。
[ライブラリ選定の見直し](../done/wi-90339-review-library-adoption.md)で、応答の形はスキーマで検証しないが、この置き換えは不具合として直すと決めた。

## 対象範囲

- 成功の状態コードで本文を JSON として読めない場合を、エラーとして呼び出し側へ返す。
- 本文を持たない成功応答（204 など）を明示的に区別する。
- 失敗応答の本文が JSON でない場合に、代わりの文を表示する現在の振る舞い（REQ-SYSTEM-011）を保つ。

## 対象外

- 応答の形の実行時検証。
- エラーの文言の変更。

## 設計

成功応答の扱いを、本文を持たない応答、JSON の本文、読めない本文に分ける。
読めない本文は、画面の[表示状態](../../docs/design/application/user-interface.md#表示状態)の取得失敗として扱い、空の一覧や空の詳細として描画しない。
成功応答の扱いを仕様に書く必要があるかを着手時に確かめ、必要なら `spec-change` で要件を追加する。

## 計画

1. 主要ユースケースの証拠を計画する。
2. 壊れた本文の成功応答で RED を確かめてから直す。

## タスク

- [ ] T001 [Acceptance] 壊れた本文の成功応答を観測する境界で RED を確かめる。
- [ ] T002 [UI] 成功応答の扱いを分ける。
- [ ] T003 [Verify] 204 と失敗応答の振る舞いを保つことを確かめる。

## 検証

- `mise run test-ui-unit-file -- src/api/core.test.ts` などで API 共通処理を確かめる。
- `mise run test-ui-unit` を通す。

## リスク

本文を持たない成功応答を読めない本文と誤って扱うと、削除などの操作が失敗として表示される。
204 と空の本文をテストで固定する。
