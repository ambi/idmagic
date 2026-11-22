# 本人のアクセスレビュー

この文書は、本人のアクセスレビューの操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[IdGovernance](../README.md) / [ページ一覧](README.md) / 本人のアクセスレビュー

分類：[アカウントポータル](../../../design/application/pages/account/README.md) / [ライフサイクル](../../../design/application/pages/account/lifecycle/README.md)

`account-access-reviews`　`/account/access-reviews`

担当するアクセスの継続または取消を判断する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [IdGovernance](../README.md) |
| 利用者 | 認証済みの本人 |
| 入口 | 本人のポータル |
| 利用条件 | 対象テナントの本人のセッションで参照する。CSRF と再認証は共通規則に従う。 |
| 表示内容 | 対象、期限、未回答時の判断と現在の適用結果、未回答かつ期限前のレビューへの回答、取消の確認ダイアログ、アプリ割り当てへの注意 |

利用手順は[IdGovernance のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| decide | ready | 継続または取消を確認 | レビューが未回答かつ期限前 | IdMagic.IdGovernance.Operations.DecideAccessReview | 判断と反映結果を表示する | [本人のアクセスレビュー](account-access-reviews.md) / ready | 対象の識別子と操作の入力値 |
| refresh | ready | 再取得 | 常時 | IdMagic.IdGovernance.Operations.ListMyAccessReviews | 担当レビューを更新する | [本人のアクセスレビュー](account-access-reviews.md) / ready | 対象の識別子と操作の入力値 |

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
| empty | 対象がないことと、作成または検索条件を変える入口を示す。 |

## 関連する設計

- [TypeSpec の定義](../../../../spec/modules/identity-governance/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[IdGovernanceのページ一覧へ戻る](README.md)
