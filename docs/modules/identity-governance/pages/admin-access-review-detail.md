# レビュー計画の詳細と結果

この文書は、レビュー計画の詳細と結果の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[IdGovernance](../README.md) / [ページ一覧](README.md) / レビュー計画の詳細と結果

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [ライフサイクル](../../../design/application/pages/admin/lifecycle/README.md)

`admin-access-review-detail`　`/admin/access-reviews/$planId`

計画の稼働状態と各レビューの判断・適用結果を確認する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [IdGovernance](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | 計画一覧または保存後 |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | 対象、担当、期間とデフォルトの判断、次回開始、開始失敗と通知失敗、回答、デフォルトの判断の適用、取消と適用エラー、計画の無効化を確認するダイアログ |

利用手順は[IdGovernance のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| edit | ready | 編集 | 常時 | UI 内の操作 | 移動先を表示する | [レビュー計画を編集](admin-access-review-edit.md) / ready | なし |
| toggle | ready | 有効化または無効化を確認 | 常時 | IdMagic.IdGovernance.Operations.UpdateAccessReviewPlan | 計画の有効状態を更新する | [レビュー計画の詳細と結果](admin-access-review-detail.md) / ready | 対象の識別子と操作の入力値 |
| refresh | ready | 再取得 | 常時 | IdMagic.IdGovernance.Operations.ListAccessReviewPlans、IdMagic.IdGovernance.Operations.ListAccessReviews | 計画とレビュー結果を更新する | [レビュー計画の詳細と結果](admin-access-review-detail.md) / ready | 対象の識別子と操作の入力値 |
| back | ready | 戻る | 常時 | UI 内の操作 | 移動先を表示する | [アクセスレビュー計画](admin-access-reviews.md) / ready | なし |

## 表示状態

| 状態 | 表示 |
| --- | --- |
| ready | 取得した内容を表示する。 |
| loading | ページ移動中は直前の表示を保つ。初回の取得はルーターが扱う。 |
| error | 表示できなければ共通エラーを表示し、操作失敗は操作の近くに残す。 |
| forbidden | 権限不足を表示し、未反映の状態を保つ。 |
| expired | 認証が必要なページでは再認可し、復帰できなければログインへ移る。 |
| success | 成功した操作を一度通知し、表示内容を再取得する。 |
| invalid | 入力不備の表示は共通フォーム規則に従う。項目ごとの規則は別に定義する。 |
| submitting | 実行中の操作を示し、二重送信を防ぐ。 |

## 関連する設計

- [TypeSpec の定義](../../../../spec/modules/identity-governance/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[IdGovernanceのページ一覧へ戻る](README.md)
