# ログイン履歴とセッション

この文書は、ログイン履歴とセッションの操作と状態を示す。

> 表示確認用の途中プレビュー。全ページの棚卸しと検証は継続中である。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Authentication](../README.md) / [ページ一覧](README.md) / ログイン履歴とセッション

分類：[アカウントポータル](../../../design/application/pages/account/README.md) / [セキュリティ](../../../design/application/pages/account/security/README.md)

`account-activity`　`/account/activity`

本人のログイン履歴を確認し不要なセッションを失効させる

| 項目 | 内容 |
| --- | --- |
| 主担当 | [Authentication](../README.md) |
| 利用者 | 認証済みの本人 |
| 入口 | アカウントポータル |
| 利用条件 | 対象テナントの本人のセッションで参照する。CSRF と再認証は共通規則に従う。 |
| 表示内容 | 現在と他のセッション、期限、端末情報、ログインの日時、認証手段、結果 |

利用手順は[Authentication のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| revoke | ready | 対象のセッションを失効 | 再認証と失効が成功 | IdMagic.Authentication.Operations.RevokeMySession | 対象を一覧から除く。現在のセッションなら再ログインへ進む | [ログイン履歴とセッション](account-activity.md) / ready | 対象の識別子と操作の入力値 |
| revokeOthers | ready | 他のセッションをすべて失効 | 再認証が成功 | IdMagic.Authentication.Operations.RevokeMyOtherSessions | 現在以外のセッションを一覧から除く | [ログイン履歴とセッション](account-activity.md) / ready | 対象の識別子と操作の入力値 |

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
