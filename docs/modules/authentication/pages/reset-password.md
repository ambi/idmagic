# パスワードの再設定

この文書は、パスワードの再設定の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Authentication](../README.md) / [ページ一覧](README.md) / パスワードの再設定

分類：[認証](../../../design/application/pages/authentication/README.md) / [アカウント復旧](../../../design/application/pages/authentication/recovery/README.md)

`reset-password`　`/reset_password`

メールのリンクから新しいパスワードを設定する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [Authentication](../README.md) |
| 利用者 | 認証を受ける利用者 |
| 入口 | 認可フロー、メールのリンクまたは直接アクセス |
| 利用条件 | 対象テナントの認可トランザクションとセッションの Cookie。CSRF は表示 API から取得する。 |
| 表示内容 | 新しいパスワード、リンクの有効性、変更の完了とログインへの入口 |

利用手順は[Authentication のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| reset | ready | 変更 | 変更に成功 | IdMagic.Authentication.Operations.ResetPasswordWithToken | 変更完了とログインの入口を表示する | [パスワードの再設定](reset-password.md) / success | token、new_password |
| backAfterReset | success | ログインへ戻る | パスワードの変更後 | UI 内の操作 | ログインの入口を表示する | [ログイン](login.md) / ready | なし |
| back | ready | ログインへ戻る | 常時 | UI 内の操作 | 移動先を表示する | [ログイン](login.md) / ready | なし |
| failure | ready | 変更 | 期限切れ、使用済みまたは変更の拒否 | UI 内の操作 | 問題を表示し、必要なら再設定の依頼へ案内する | [パスワードの再設定](reset-password.md) / error | なし |

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
- [共通のフォーム規則](../../../design/application/user-interface.md#フォームと検証)

[Authenticationのページ一覧へ戻る](README.md)
