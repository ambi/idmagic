# パスワード変更

この文書は、パスワード変更の操作、入力項目と状態を示す。

> 表示確認用の途中プレビュー。全ページの棚卸しと検証は継続中である。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Authentication](../README.md) / [ページ一覧](README.md) / パスワード変更

分類：[アカウントポータル](../../../design/application/pages/account/README.md) / [セキュリティ](../../../design/application/pages/account/security/README.md)

`change-password`　`/account/password`

現在のパスワードを確認して新しいパスワードへ変更する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [Authentication](../README.md) |
| 利用者 | 認証済みの本人 |
| 入口 | 本人のセキュリティ設定、認可フローの必須アクション |
| 利用条件 | 対象テナントの本人のセッションで参照する。CSRF と再認証は共通規則に従う。 |
| 表示内容 | 現在と新しいパスワード、パスワードの表示切り替え、ポリシー違反と変更結果 |

利用手順は[Authentication のページ一覧](README.md)にまとめる。

## フォームと入力項目

項目ごとのバリデーション規則は別の作業で定義する。

### パスワード変更

識別子：change-password/change。表示条件：常時。確定：変更。

| 項目 | 識別子 | 入力の種類 | 型 | 初期値 | 表示条件 | データとの対応 |
| --- | --- | --- | --- | --- | --- | --- |
| 現在のパスワード | change-password/change/currentPassword | password | string | 空 | 対象のフォームを操作するとき | IdMagic.Contract.BrowserChangePasswordRequest.current_password |
| 新しいパスワード | change-password/change/newPassword | password | string | 空 | 対象のフォームを操作するとき | IdMagic.Contract.BrowserChangePasswordRequest.new_password |

## パラメーター

| 項目 | 識別子 | 入力の種類 | 型 | 初期値 | 表示条件 | データとの対応 |
| --- | --- | --- | --- | --- | --- | --- |
| 認証後の戻り先 | change-password/parameters/returnTo | URL | string | return_to | 認可フローから来た場合 | 同じテナントの許可された戻り先 |

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| save | ready | 変更 | 更新が成功 | IdMagic.Authentication.Operations.ChangePassword | 必須アクションなら認可フローを継続し、それ以外は完了を表示する | [パスワード変更](change-password.md) / ready | 対象の識別子と操作の入力値 |
| failure | ready | 変更 | 現在のパスワードが違う、ポリシー違反または更新失敗 | UI 内の操作 | 変更せず理由を表示する | [パスワード変更](change-password.md) / error | 対象の識別子と操作の入力値 |

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

- [TypeSpec の定義](../../../../spec/modules/authentication/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[Authenticationのページ一覧へ戻る](README.md)
