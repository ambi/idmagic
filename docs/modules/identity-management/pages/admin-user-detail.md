# ユーザー詳細

この文書は、ユーザー詳細の操作、入力項目と状態を示す。

> 表示確認用の途中プレビュー。全ページの棚卸しと検証は継続中である。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[IdManagement](../README.md) / [ページ一覧](README.md) / ユーザー詳細

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [ユーザー管理](../../../design/application/pages/admin/users/README.md)

`admin-user-detail`　`/admin/users/$id`

現在のプロフィール、権限、所属と認証の状態を確認する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [IdManagement](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | 管理コンソールのユーザー一覧または対象ユーザーの詳細 |
| 利用条件 | 対象テナントの管理者として認可を受ける。変更の許可は API が判断する。 |
| 表示内容 | プロフィール、ロールとグループ、認証情報とセッション、必須アクション、無効化と削除の確認 |

利用手順は[IdManagement のページ一覧](README.md)にまとめる。

## フォームと入力項目

項目ごとのバリデーション規則は別の作業で定義する。

### ユーザーの削除確認

識別子：admin-user-detail/deleteConfirmation。表示条件：削除のダイアログを開いた場合。確定：削除。

| 項目 | 識別子 | 入力の種類 | 型 | 初期値 | 表示条件 | データとの対応 |
| --- | --- | --- | --- | --- | --- | --- |
| 確認用ユーザー名 | admin-user-detail/deleteConfirmation/confirmName | text | string | 空 | 削除のダイアログを開いた場合 | UI 内での確認。API へ送信しない |

### 認証情報のリセット

識別子：admin-user-detail/authenticatorReset。表示条件：認証情報のリセットを選択した場合。確定：選択した認証情報を解除。

| 項目 | 識別子 | 入力の種類 | 型 | 初期値 | 表示条件 | データとの対応 |
| --- | --- | --- | --- | --- | --- | --- |
| TOTP | admin-user-detail/authenticatorReset/totp | checkbox | boolean | false | リセットのダイアログを開いた場合 | 解除する認証情報の種類 |
| パスキー | admin-user-detail/authenticatorReset/webauthn | checkbox | boolean | false | リセットのダイアログを開いた場合 | 解除する認証情報の種類 |
| 回復コード | admin-user-detail/authenticatorReset/recoveryCode | checkbox | boolean | false | リセットのダイアログを開いた場合 | 解除する認証情報の種類 |

## パラメーター

| 項目 | 識別子 | 入力の種類 | 型 | 初期値 | 表示条件 | データとの対応 |
| --- | --- | --- | --- | --- | --- | --- |
| ユーザー ID | admin-user-detail/parameters/id | パスパラメーター | string | URL の id | 常時 | 取得と管理操作の対象 ID |

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| edit | ready | 編集 | 常時 | UI 内の操作 | 移動先を表示する | [ユーザーを編集](admin-user-edit.md) / ready | ユーザー ID |
| delete | ready | 削除を確認 | 確認が成立し API が削除を受理 | IdMagic.IdManagement.Operations.DeleteAdminUser | 一覧を再取得する | [ユーザー一覧](admin-users.md) / ready | ユーザー ID |
| back | ready | 一覧へ戻る | 常時 | UI 内の操作 | 移動先を表示する | [ユーザー一覧](admin-users.md) / ready | なし |
| stepUp | ready | 管理操作 | API が再認証を要求 | UI 内の操作 | 再認証後に元の操作を一度だけ再試行する | 再認証 / dialog | 元の操作と再認証の要求 |

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

- [TypeSpec の定義](../../../../spec/modules/identity-management/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- REQ-FRONTEND-002
- [ナビゲーション](../../../design/application/user-interface.md#ナビゲーション)

[IdManagementのページ一覧へ戻る](README.md)
