# グループ一覧

この文書は、グループ一覧の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[IdManagement](../README.md) / [ページ一覧](README.md) / グループ一覧

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [グループ管理](../../../design/application/pages/admin/groups/README.md)

`admin-groups`　`/admin/groups`

グループを選び、ロールと所属人数を確認する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [IdManagement](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | 管理メニュー |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | ページ送りと再取得、選択したグループの詳細と削除の確認、作成、CSV 取り込みと出力への入口 |

利用手順は[IdManagement のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| refresh | ready | 選択または再取得 | 常時 | IdMagic.IdManagement.Operations.ListGroups | 選択対象の詳細と一覧を表示する | [グループ一覧](admin-groups.md) / ready | 対象の識別子と操作の入力値 |
| create | ready | 追加 | 常時 | UI 内の操作 | 移動先を表示する | [グループを作成](admin-group-create.md) / ready | なし |
| detail | ready | 詳細 | 常時 | UI 内の操作 | 移動先を表示する | [グループ詳細](admin-group-detail.md) / ready | なし |
| edit | ready | 編集 | 常時 | UI 内の操作 | 移動先を表示する | [グループを編集](admin-group-edit.md) / ready | なし |
| exportMembers | ready | メンバーを出力する | 常時 | UI 内の操作 | 移動先を表示する | [グループメンバーの CSV エクスポート](admin-group-members-export.md) / ready | なし |
| member | ready | メンバーのユーザーを開く | 常時 | UI 内の操作 | 移動先を表示する | [ユーザー一覧](admin-users.md) / ready | ユーザー名 |
| delete | ready | 削除を確認 | 常時 | IdMagic.IdManagement.Operations.DeleteGroup | 対象を削除して一覧を更新する | [グループ一覧](admin-groups.md) / ready | 対象の識別子と操作の入力値 |
| importCsv | ready | CSV 取り込み | 常時 | UI 内の操作 | 移動先を表示する | [グループの CSV インポート](admin-group-import.md) / ready | なし |
| exportCsv | ready | CSV 出力 | 常時 | UI 内の操作 | 移動先を表示する | [グループの CSV エクスポート](admin-group-export.md) / ready | なし |

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
- [共通の UI 規則](../../../design/application/user-interface.md)

[IdManagementのページ一覧へ戻る](README.md)
