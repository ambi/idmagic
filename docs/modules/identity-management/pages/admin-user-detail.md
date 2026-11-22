# ユーザー詳細

この文書は、ユーザー詳細の操作と状態を示す。

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
| 表示内容 | プロフィール、ロールとグループ、認証情報とセッションの一覧、必須アクション、無効化、削除の予約、完全削除と復元の確認。削除はユーザー名の入力で確認する、MFA 登録の一時免除の発行と取消、解除する認証情報を選ぶリセットのダイアログ |

利用手順は[IdManagement のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| edit | ready | 編集 | 常時 | UI 内の操作 | 移動先を表示する | [ユーザーを編集](admin-user-edit.md) / ready | ユーザー ID |
| group | ready | 所属グループを開く | 常時 | UI 内の操作 | 移動先を表示する | [グループ一覧](admin-groups.md) / ready | グループ ID |
| status | ready | 無効化を確認または有効化 | 常時 | IdMagic.IdManagement.Operations.DisableAdminUser、IdMagic.IdManagement.Operations.EnableAdminUser | 状態を変更して詳細を再取得する | [ユーザー詳細](admin-user-detail.md) / ready | 対象の識別子と操作の入力値 |
| delete | ready | 削除を確認または復元 | 確認が成立し API が受理 | IdMagic.IdManagement.Operations.DeleteAdminUser、IdMagic.IdManagement.Operations.RestoreAdminUser | 削除の予約または復元を反映して詳細を再取得する | [ユーザー詳細](admin-user-detail.md) / ready | ユーザー ID |
| purge | ready | 完全削除を確認 | 削除予約中で、確認が成立し API が受理 | IdMagic.IdManagement.Operations.DeleteAdminUser | 一覧へ移る | [ユーザー一覧](admin-users.md) / ready | ユーザー ID |
| bypass | ready | MFA 登録の免除を発行または取消 | 常時 | IdMagic.Authentication.Operations.IssueMfaEnrollmentBypass、IdMagic.Authentication.Operations.RevokeMfaEnrollmentBypass | 免除の状態を更新する | [ユーザー詳細](admin-user-detail.md) / ready | 対象の識別子と操作の入力値 |
| resetAuthenticators | ready | 認証情報のリセットを確認 | 常時 | IdMagic.Authentication.Operations.ResetUserAuthenticators | 選んだ認証情報を解除し結果を表示する | [ユーザー詳細](admin-user-detail.md) / ready | 対象の識別子と操作の入力値 |
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
