# ユーザーの CSV インポート

この文書は、ユーザーの CSV インポートの操作、入力項目と状態を示す。

> 表示確認用の途中プレビュー。全ページの棚卸しと検証は継続中である。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[IdManagement](../README.md) / [ページ一覧](README.md) / ユーザーの CSV インポート

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [ユーザー管理](../../../design/application/pages/admin/users/README.md)

`admin-user-import`　`/admin/users/import`

CSV をプレビューしてから確認を経て適用し、行ごとの結果を確認する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [IdManagement](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | 管理コンソールのユーザー一覧または対象ユーザーの詳細 |
| 利用条件 | 対象テナントの管理者として認可を受ける。変更の許可は API が判断する。 |
| 表示内容 | テンプレートのダウンロード、ファイル選択、プレビュー、適用の確認ダイアログ、行と列ごとの結果と問題 |

利用手順は[IdManagement のページ一覧](README.md)にまとめる。

## フォームと入力項目

項目ごとのバリデーション規則は別の作業で定義する。

### CSV ファイルの取り込み

識別子：admin-user-import/upload。表示条件：常時。確定：プレビュー。

| 項目 | 識別子 | 入力の種類 | 型 | 初期値 | 表示条件 | データとの対応 |
| --- | --- | --- | --- | --- | --- | --- |
| CSV ファイル | admin-user-import/upload/file | file | string | 未選択 | 取り込みの開始時 | 選択した CSV の内容をプレビューの要求へ渡す |

## パラメーター

ページ固有のパラメーターはない。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| preview | ready | プレビュー | プレビューのジョブが完了 | UI 内の操作 | 適用前の内容と行ごとの問題を表示する | [ユーザーの CSV インポート](admin-user-import.md) / preview | CSV 内容 |
| confirm | preview | 取り込みを適用 | プレビューが完了 | UI 内の操作 | 適用前の確認ダイアログを表示する | [ユーザーの CSV インポート](admin-user-import.md) / confirm | プレビューのジョブ ID |
| cancelApply | confirm | キャンセル | 適用を取りやめる | UI 内の操作 | 確認を閉じてプレビューへ戻る | [ユーザーの CSV インポート](admin-user-import.md) / preview | なし |
| apply | confirm | 取り込みの適用を確認 | 確認が成立し適用が完了 | UI 内の操作 | 処理件数と失敗した行を表示する | [ユーザーの CSV インポート](admin-user-import.md) / success | プレビューのジョブ ID |
| partial | confirm | 取り込みの適用を確認 | 一部の行の処理が失敗 | UI 内の操作 | 成功と失敗を分けて表示する | [ユーザーの CSV インポート](admin-user-import.md) / partial | なし |
| back | ready | 一覧へ戻る | 常時 | UI 内の操作 | 移動先を表示する | [ユーザー一覧](admin-users.md) / ready | なし |

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
| preview | 取り込み前の内容と行ごとの問題を表示する。 |
| confirm | 取り込みの適用対象と影響を確認するダイアログを表示する。 |
| partial | 成功と失敗を区別し、失敗した行と項目を示す。 |

## 関連する設計

- [TypeSpec の定義](../../../../spec/modules/identity-management/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- REQ-FRONTEND-002
- [ナビゲーション](../../../design/application/user-interface.md#ナビゲーション)

[IdManagementのページ一覧へ戻る](README.md)
