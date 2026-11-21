# ユーザーを追加

この文書は、ユーザーを追加の操作、入力項目と状態を示す。

> 表示確認用の途中プレビュー。全ページの棚卸しと検証は継続中である。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[IdManagement](../README.md) / [ページ一覧](README.md) / ユーザーを追加

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [ユーザー管理](../../../design/application/pages/admin/users/README.md)

`admin-user-create`　`/admin/users/new`

初期資格情報とプロフィールを指定してユーザーを作成する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [IdManagement](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | 管理コンソールのユーザー一覧または対象ユーザーの詳細 |
| 利用条件 | 対象テナントの管理者として認可を受ける。変更の許可は API が判断する。 |
| 表示内容 | ユーザー名と表示名、メールと確認済みの指定、初期パスワード、初期ロール |

利用手順は[IdManagement のページ一覧](README.md)にまとめる。

## フォームと入力項目

項目ごとのバリデーション規則は別の作業で定義する。

### ユーザーの作成

識別子：admin-user-create/create。表示条件：常時。確定：作成。

| 項目 | 識別子 | 入力の種類 | 型 | 初期値 | 表示条件 | データとの対応 |
| --- | --- | --- | --- | --- | --- | --- |
| ユーザー名 | admin-user-create/create/preferredUsername | text | string | 空 | 常時 | IdMagic.Contract.AdminUserCreateRequest.preferred_username |
| 表示名 | admin-user-create/create/name | text | string | 空 | 常時 | IdMagic.Contract.AdminUserCreateRequest.name |
| メールアドレス | admin-user-create/create/email | email | string | 空 | 常時 | IdMagic.Contract.AdminUserCreateRequest.email |
| メールを確認済みにする | admin-user-create/create/emailVerified | checkbox | boolean | false | 常時 | IdMagic.Contract.AdminUserCreateRequest.email_verified |
| 初期パスワード | admin-user-create/create/password | password | string | 空 | 常時 | IdMagic.Contract.AdminUserCreateRequest.password |
| 初期ロール | admin-user-create/create/roles | text | string | 空 | 常時 | IdMagic.Contract.AdminUserCreateRequest.roles：カンマ区切りを roles 配列へ変換 |

## パラメーター

ページ固有のパラメーターはない。

## 実装と照合していない入力

次のフォーム部品は静的な `name` 属性を持たないため、上の入力項目との対応を機械的に確かめていない。ページを変更するときは、これらの部品を入力項目と突き合わせる。

| ファイル | 部品の数 |
| --- | --- |
| [`features/admin-users/AdminUserCreatePage.tsx`](../../../../frontend/src/features/admin-users/AdminUserCreatePage.tsx) | 5 |
| [`features/admin-users/AdminUsersPrimitives.tsx`](../../../../frontend/src/features/admin-users/AdminUsersPrimitives.tsx) | 1 |

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| create | ready | 作成 | 作成に成功 | IdMagic.IdManagement.Operations.CreateAdminUser | 作成したユーザーの詳細を取得する | [ユーザー詳細](admin-user-detail.md) / ready | 作成したユーザー ID |
| cancel | ready | キャンセル | 常時 | UI 内の操作 | 移動先を表示する | [ユーザー一覧](admin-users.md) / ready | なし |
| failure | ready | 作成 | 作成を拒否または通信失敗 | UI 内の操作 | 入力を保持して問題を表示する | [ユーザーを追加](admin-user-create.md) / error | なし |

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
