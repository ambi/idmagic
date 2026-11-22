# 本人のアプリケーションランチャー

この文書は、本人のアプリケーションランチャーの操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Application](../README.md) / [ページ一覧](README.md) / 本人のアプリケーションランチャー

分類：[アカウントポータル](../../../design/application/pages/account/README.md) / [アプリケーション](../../../design/application/pages/account/applications/README.md)

`account-apps`　`/account/apps`

利用可能なアプリケーションを起動し表示順を変更する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [Application](../README.md) |
| 利用者 | 認証済みの本人 |
| 入口 | 本人のポータル |
| 利用条件 | 対象テナントの本人のセッションで参照する。CSRF と再認証は共通規則に従う。 |
| 表示内容 | カテゴリ別のアプリケーションカード、起動の入口、ドラッグ操作またはキーボードによる並べ替え、保存中と保存失敗 |

利用手順は[Application のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| reorder | ready | 表示順を変更 | 並べ替えが確定 | IdMagic.Application.Operations.ReorderMyApplications | 並べ替えた順序を保存する | [本人のアプリケーションランチャー](account-apps.md) / ready | 対象の識別子と操作の入力値 |
| launch | ready | アプリケーションを選択 | launch_url がありドラッグ操作の直後でない | UI 内の操作 | 別タブでアプリケーションを開く | 呼び出し元アプリケーション / external | 対象の識別子とテナントの文脈 |

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

- [TypeSpec の定義](../../../../spec/modules/application/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[Applicationのページ一覧へ戻る](README.md)
