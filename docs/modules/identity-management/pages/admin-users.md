# ユーザー一覧

この文書は、ユーザー一覧の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[IdManagement](../README.md) / [ページ一覧](README.md) / ユーザー一覧

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [ユーザー管理](../../../design/application/pages/admin/users/README.md)

`admin-users`　`/admin/users`

ユーザーを検索し、対象の確認と管理の入口を示す

| 項目 | 内容 |
| --- | --- |
| 主担当 | [IdManagement](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | 管理コンソールのユーザー一覧または対象ユーザーの詳細 |
| 利用条件 | 対象テナントの管理者として認可を受ける。変更の許可は API が判断する。 |
| 表示内容 | 検索語と状態による絞り込み。条件は URL に保持する、ユーザー名、表示名、メールアドレス、状態とロール、総件数、ページ送りと再取得、選択したユーザーの概要、必須アクションと所属グループ、無効化、削除の予約、完全削除と復元の確認、作成と CSV の入口 |

利用手順は[IdManagement のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| search | ready | 検索または絞り込み | 常時 | IdMagic.IdManagement.Operations.ListAdminUsers | 条件に合う一覧を表示する | [ユーザー一覧](admin-users.md) / ready | query、status、cursor |
| select | ready | ユーザーを選択 | 常時 | UI 内の操作 | 選択したユーザーの概要を表示する | [ユーザー一覧](admin-users.md) / ready | 対象の識別子と操作の入力値 |
| status | ready | 無効化を確認または有効化 | 常時 | IdMagic.IdManagement.Operations.DisableAdminUser、IdMagic.IdManagement.Operations.EnableAdminUser | 状態を変更して一覧を再取得する | [ユーザー一覧](admin-users.md) / ready | 対象の識別子と操作の入力値 |
| lifecycle | ready | 削除・完全削除を確認または復元 | 常時 | IdMagic.IdManagement.Operations.DeleteAdminUser、IdMagic.IdManagement.Operations.RestoreAdminUser | 削除の予約、完全削除または復元を行い一覧を再取得する | [ユーザー一覧](admin-users.md) / ready | 対象の識別子と操作の入力値 |
| edit | ready | 編集 | 常時 | UI 内の操作 | 移動先を表示する | [ユーザーを編集](admin-user-edit.md) / ready | ユーザー ID |
| group | ready | 所属グループを開く | 常時 | UI 内の操作 | 移動先を表示する | [グループ一覧](admin-groups.md) / ready | グループ ID |
| detail | ready | 詳細 | 常時 | UI 内の操作 | 移動先を表示する | [ユーザー詳細](admin-user-detail.md) / ready | ユーザー ID |
| create | ready | ユーザーを追加 | 常時 | UI 内の操作 | 移動先を表示する | [ユーザーを追加](admin-user-create.md) / ready | なし |
| importCsv | ready | CSV インポート | 常時 | UI 内の操作 | 移動先を表示する | [ユーザーの CSV インポート](admin-user-import.md) / ready | なし |
| exportCsv | ready | CSV エクスポート | 常時 | UI 内の操作 | 移動先を表示する | [ユーザーの CSV エクスポート](admin-user-export.md) / ready | なし |

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

- [TypeSpec の定義](../../../../spec/modules/identity-management/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- REQ-FRONTEND-002
- [ナビゲーション](../../../design/application/user-interface.md#ナビゲーション)

[IdManagementのページ一覧へ戻る](README.md)
