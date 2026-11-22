# ユーザーの CSV エクスポート

この文書は、ユーザーの CSV エクスポートの操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[IdManagement](../README.md) / [ページ一覧](README.md) / ユーザーの CSV エクスポート

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [ユーザー管理](../../../design/application/pages/admin/users/README.md)

`admin-user-export`　`/admin/users/exports`

出力ジョブを開始し、完了した CSV を取得する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [IdManagement](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | 管理コンソールのユーザー一覧または対象ユーザーの詳細 |
| 利用条件 | 対象テナントの管理者として認可を受ける。変更の許可は API が判断する。 |
| 表示内容 | 出力列の選択、ジョブの状態と進捗、再取得、キャンセルと期限内のダウンロード |

利用手順は[IdManagement のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| start | ready | 出力を開始 | 常時 | IdMagic.IdManagement.Operations.StartUserCsvExport | 出力ジョブを一覧へ追加する | [ユーザーの CSV エクスポート](admin-user-export.md) / ready | 対象の識別子と操作の入力値 |
| refresh | ready | 再取得 | 常時 | IdMagic.IdManagement.Operations.ListUserExports | ジョブの状態と進捗を更新する | [ユーザーの CSV エクスポート](admin-user-export.md) / ready | 対象の識別子と操作の入力値 |
| cancel | ready | キャンセル | 停止可能なジョブ | IdMagic.IdManagement.Operations.CancelUserExport | ジョブの停止を要求する | [ユーザーの CSV エクスポート](admin-user-export.md) / ready | 対象の識別子と操作の入力値 |
| download | ready | ダウンロード | 完了し保持期限内 | IdMagic.IdManagement.Operations.DownloadUserExportFile | CSV を取得する | ダウンロード / external | なし |
| back | ready | 一覧へ戻る | 常時 | UI 内の操作 | 移動先を表示する | [ユーザー一覧](admin-users.md) / ready | なし |

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
