# 認証と承認の結果

この文書は、認証と承認の結果の操作と状態を示す。

> 表示確認用の途中プレビュー。全ページの棚卸しと検証は継続中である。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Frontend](../README.md) / [ページ一覧](README.md) / 認証と承認の結果

分類：[認証](../../../design/application/pages/authentication/README.md) / [入口と共通表示](../../../design/application/pages/authentication/navigation/README.md)

`status`　`/status`

操作が終わったことと次の入口を案内する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [Frontend](../README.md) |
| 利用者 | 認可フローの利用者 |
| 入口 | 承認・拒否・ログアウトの結果 |
| 利用条件 | 対象テナントのブラウザートランザクションを使う。CSRF は表示 API から得る。 |
| 表示内容 | approved、denied、signed-out、authentication-required に応じた説明、ログアウト後のアカウントと管理コンソールへの入口 |

利用手順は[Frontend のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| account | ready | アカウントへログイン | signed-out | UI 内の操作 | 本人のポータルの認可を開始する | [アカウントの概要](account-home.md) / ready | 対象の識別子とテナントの文脈 |
| admin | ready | 管理コンソールへログイン | signed-out | UI 内の操作 | 管理コンソールの認可を開始する | [管理ダッシュボード](admin-dashboard.md) / ready | 対象の識別子とテナントの文脈 |

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

- [TypeSpec の定義](../../../../spec/modules/frontend/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[Frontendのページ一覧へ戻る](README.md)
