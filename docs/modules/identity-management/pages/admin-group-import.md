# グループの CSV インポート

この文書は、グループの CSV インポートの操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[IdManagement](../README.md) / [ページ一覧](README.md) / グループの CSV インポート

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [グループ管理](../../../design/application/pages/admin/groups/README.md)

`admin-group-import`　`/admin/groups/import`

CSV の内容を確認してグループを取り込む

| 項目 | 内容 |
| --- | --- |
| 主担当 | [IdManagement](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | グループ一覧 |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | テンプレート、ファイル選択とプレビュー、適用確認と行ごとの結果、エラーのページ送りと部分失敗 |

利用手順は[IdManagement のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| preview | ready | プレビュー | 常時 | IdMagic.IdManagement.Operations.ImportAdminGroups、IdMagic.IdManagement.Operations.GetAdminGroupImport | 完了まで状態を取得して行ごとの問題を表示する | [グループの CSV インポート](admin-group-import.md) / preview | 対象の識別子と操作の入力値 |
| confirm | preview | 適用を確認 | 常時 | UI 内の操作 | 確認ダイアログを表示する | [グループの CSV インポート](admin-group-import.md) / confirm | 対象の識別子と操作の入力値 |
| cancel | confirm | キャンセル | 常時 | UI 内の操作 | プレビューへ戻る | [グループの CSV インポート](admin-group-import.md) / preview | 対象の識別子と操作の入力値 |
| apply | confirm | 確認して適用 | 常時 | IdMagic.IdManagement.Operations.ApplyAdminGroupImport、IdMagic.IdManagement.Operations.GetAdminGroupImport | 適用結果と失敗した行を表示する | [グループの CSV インポート](admin-group-import.md) / success | 対象の識別子と操作の入力値 |
| back | ready | 戻る | 常時 | UI 内の操作 | 移動先を表示する | [グループ一覧](admin-groups.md) / ready | なし |

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
| preview | 取り込み前の内容と行ごとの問題を表示する。 |
| confirm | 取り込みの適用対象と影響を確認するダイアログを表示する。 |
| partial | 成功と失敗を区別し、失敗した行と項目を示す。 |

## 関連する設計

- [TypeSpec の定義](../../../../spec/modules/identity-management/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[IdManagementのページ一覧へ戻る](README.md)
