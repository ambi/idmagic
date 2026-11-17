# ユーザーを編集

この文書は、ユーザーを編集の操作、入力項目と状態を示す。

> 表示確認用の途中プレビュー。全ページの棚卸しと検証は継続中である。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[IdManagement](../README.md) / [ページ一覧](README.md) / ユーザーを編集

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [ユーザー管理](../../../design/application/pages/admin/users/README.md)

`admin-user-edit`　`/admin/users/$id/edit`

プロフィール、ロールとカスタム属性を一回の保存で変更する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [IdManagement](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | 管理コンソールのユーザー一覧または対象ユーザーの詳細 |
| 利用条件 | 対象テナントの管理者として認可を受ける。変更の許可は API が判断する。 |
| 表示内容 | プロフィールの編集、ロールの変更内容の確認、属性定義に応じたカスタム属性 |

利用手順は[IdManagement のページ一覧](README.md)にまとめる。

## フォームと入力項目

項目ごとのバリデーション規則は別の作業で定義する。

### ユーザーの変更

識別子：admin-user-edit/profile。表示条件：常時。確定：変更内容を確認して保存。

| 項目 | 識別子 | 入力の種類 | 型 | 初期値 | 表示条件 | データとの対応 |
| --- | --- | --- | --- | --- | --- | --- |
| ユーザー名 | admin-user-edit/profile/preferred_username | text | string | 現在のユーザーの値 | 常時 | IdMagic.Contract.AdminUserUpdateRequest.preferred_username：preferred_username |
| 表示名 | admin-user-edit/profile/name | text | string | 現在のユーザーの値 | 常時 | IdMagic.Contract.AdminUserUpdateRequest.name：name |
| 名 | admin-user-edit/profile/given_name | text | string | 現在のユーザーの値 | 常時 | IdMagic.Contract.AdminUserUpdateRequest.given_name：given_name |
| 姓 | admin-user-edit/profile/family_name | text | string | 現在のユーザーの値 | 常時 | IdMagic.Contract.AdminUserUpdateRequest.family_name：family_name |
| メールアドレス | admin-user-edit/profile/email | email | string | 現在のユーザーの値 | 常時 | IdMagic.Contract.AdminUserUpdateRequest.email：email |
| メールの確認状態 | admin-user-edit/profile/email_verified | checkbox | boolean | 現在のユーザーの値 | 常時 | IdMagic.Contract.AdminUserUpdateRequest.email_verified：email_verified |
| ロール | admin-user-edit/profile/roles | text | string | 現在のユーザーの値 | 常時 | IdMagic.Contract.AdminUserUpdateRequest.roles：カンマ区切りを roles 配列へ変換する |
| カスタム属性 | admin-user-edit/profile/attributes | 属性の型に応じた入力 | Record&lt;unknown&gt; | 現在のユーザーの値 | テナントに属性定義がある場合 | IdMagic.Contract.AdminUserUpdateRequest.attributes：定義ごとの入力を属性マップへ変換する |

## パラメーター

| 項目 | 識別子 | 入力の種類 | 型 | 初期値 | 表示条件 | データとの対応 |
| --- | --- | --- | --- | --- | --- | --- |
| ユーザー ID | admin-user-edit/parameters/id | パスパラメーター | string | URL の id | 常時 | 更新するユーザー ID |

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| save | ready | 保存 | 変更内容を確認し保存に成功 | IdMagic.IdManagement.Operations.UpdateAdminUser | 更新後の詳細を再取得して一度通知する | [ユーザー詳細](admin-user-detail.md) / ready | ユーザー ID |
| cancel | ready | キャンセル | 常時 | UI 内の操作 | 未保存の入力を破棄して詳細へ戻る | [ユーザー詳細](admin-user-detail.md) / ready | ユーザー ID |
| failure | ready | 保存 | 更新の拒否または競合 | UI 内の操作 | 入力を保持し問題を操作の近くに残す | [ユーザーを編集](admin-user-edit.md) / error | なし |

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
