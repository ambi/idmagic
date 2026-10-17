---
status: in_progress
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-04
priority: p2
depends_on: []
change_kind: refactor
affected_spec:
  - { path: spec/modules/signing-keys/models.tsp, symbol: IdMagic.Contract.SigningKeyRetired }
---

# 発行しない署名鍵の退役イベントの宣言を削除する

## 動機

`tools/check/event-emission-debt.json` は `SigningKeyRetired` の発行経路の欠如を許容している。
しかし、現在のライフサイクル仕様は、重複期間の経過を時刻の判定として扱い、この名前は遷移の契機であってドメインイベントとしては発行しないと明記している。
無効化の遷移もすでに表へ追加されており、起票時の前提が変わっている。
発行しないという現在の規則に合わせて未使用の宣言を除き、監査の新設とは分ける。

## 対象範囲

- `SigningKeyRetired` の TypeSpec と未使用の Go の宣言を削除し、イベント発行の負債から除く。
- すでに一致している状態遷移表のリスクを除き、無効化の監査不足は残す。

## 対象外

- XML の署名の鍵の自動のローテーション。wi-55565 が扱う。
- 無効化の新しい監査イベントとメモリ保存先の結果の変更。wi-83307 が扱う。
  負債を空にするためだけに、新しいイベントや時刻駆動のジョブを追加しない。

## 設計

TypeSpec の差分は `IdMagic.Contract.SigningKeyRetired` の廃止だけである。
状態、HTTP 応答、JWKS の公開期間、鍵素材の保持、既存のイベントは変えない。
コードと仕様の全参照を照合すると、宣言、負債、状態遷移の名前以外に発行側も消費側もない。
規範仕様を実装へ合わせて緩めるのではなく、既存の「発行しない」という規則と矛盾する型を削除する分類 (c) とする。

| 案 | 判断 |
| --- | --- |
| 時刻の経過を永続化してイベントを発行する | 不採用。新しいジョブ、再実行、観測時刻の規則が必要で、現在の時刻判定の約束を変える |
| 無効化を退役イベントとして発行する | 不採用。無効化は Archived への遷移であり、Retired への遷移と意味が違う |
| 発行しないイベントの宣言を除く | 採用。実行時の作用を増やさず、現在の仕様と宣言を一致させる |

## 計画

1. TypeSpec の未使用のイベントを除く。
2. Go の宣言と負債を除き、既存テストとイベント検査で確認する。

## タスク

- [x] T001 [Spec] 現在の仕様に従ってイベントの宣言を廃止する。
- [x] T002 [App] 未使用の Go の宣言と負債を除く。
- [x] T003 [Verify] 検証する。

## 検証

- `mise run check-spec`
- `mise run check-event-contract` と `mise run check`：イベントの宣言、発行、負債と仕様影響を確認する。
- `mise run test-go-package -- ./backend/signingkeys/...`：既存の鍵の操作と保存先の契約を確認する。
- `mise run lint-go`：未使用の型への参照がないことを確認する。
- `mise run test-go-mutation -- backend/signingkeys/domain 2 off /tmp/idmagic-wi96851-mutation.json main`：変更行を対象にし、削除だけで変異を作れない限界を記録する。
- `mise run render-docs` と `mise run check-published-api-compat`：イベントの型の派生表示と公開状態に応じた契約を確認する。

## リスク

無効化の監査不足はこの削除では解消しない。
既存の是正項目 wi-83307 を残す。

## 要約

規範差分は TypeSpec の未使用の SigningKeyRetired の廃止である。
現在のライフサイクル仕様が、退役を時刻の判定とし、このイベントを発行しないと明記するため、Go の未使用の宣言も削除し、イベント発行の負債を0件にした。
無効化の遷移はすでに表にあり、古いリスクを除いた。
無効化の監査イベントは鍵素材の保持と共通契約を要する別の変更として今回は見送り、wi-83307 に理由を残した。
削除だけの変更行には自動 mutation が生成されず、検出能力の証拠には数えない。
