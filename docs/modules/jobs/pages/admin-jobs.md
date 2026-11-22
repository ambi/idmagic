# テナントのジョブ

この文書は、テナントのジョブの操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Jobs](../README.md) / [ページ一覧](README.md) / テナントのジョブ

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [ジョブ](../../../design/application/pages/admin/jobs/README.md)

`admin-jobs`　`/admin/jobs`

非同期処理の進捗と失敗を調べ、停止を要求する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [Jobs](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | 管理メニュー |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | 種類、状態と期間による絞り込み、追加取得と選択したジョブの詳細、進捗、失敗とキャンセルの確認 |

利用手順は[Jobs のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| search | ready | 検索・再取得・追加取得 | 常時 | IdMagic.Jobs.Operations.ListJobs | 対象ジョブと進捗を表示する | [テナントのジョブ](admin-jobs.md) / ready | 対象の識別子と操作の入力値 |
| detail | ready | ジョブを選択 | 常時 | UI 内の操作 | 選択したジョブの内容を表示する | [テナントのジョブ](admin-jobs.md) / ready | 対象の識別子と操作の入力値 |
| cancel | ready | キャンセルを確認 | 停止可能なジョブ | IdMagic.Jobs.Operations.CancelJob | 停止を要求し状態を更新する | [テナントのジョブ](admin-jobs.md) / ready | 対象の識別子と操作の入力値 |

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

- [TypeSpec の定義](../../../../spec/modules/jobs/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[Jobsのページ一覧へ戻る](README.md)
