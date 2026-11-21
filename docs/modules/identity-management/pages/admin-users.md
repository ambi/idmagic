# ユーザー一覧

この文書は、ユーザー一覧の操作、入力項目と状態を示す。

> 表示確認用の途中プレビュー。全ページの棚卸しと検証は継続中である。

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
| 表示内容 | ユーザー名、表示名、メールアドレス、状態とロール、総件数と再取得、作成と CSV の入口 |

利用手順は[IdManagement のページ一覧](README.md)にまとめる。

## フォームと入力項目

項目ごとのバリデーション規則は別の作業で定義する。

### 検索と絞り込み

識別子：admin-users/search。表示条件：常時。確定：URL の条件を更新して再取得。

| 項目 | 識別子 | 入力の種類 | 型 | 初期値 | 表示条件 | データとの対応 |
| --- | --- | --- | --- | --- | --- | --- |
| 検索語 | admin-users/search/query | text | string | URL の query | 常時 | URL と一覧の query |
| 状態 | admin-users/search/status | 選択 | string | URL の status または active | 常時 | URL と一覧の status |

## パラメーター

| 項目 | 識別子 | 入力の種類 | 型 | 初期値 | 表示条件 | データとの対応 |
| --- | --- | --- | --- | --- | --- | --- |
| 次ページのカーソル | admin-users/parameters/cursor | URL パラメーター | string | URL の cursor | ページ送りの場合 | API の cursor |

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| search | ready | 検索または絞り込み | 常時 | IdMagic.IdManagement.Operations.ListAdminUsers | 条件に合う一覧を表示する | [ユーザー一覧](admin-users.md) / ready | query、status、cursor |
| detail | ready | 詳細 | 常時 | UI 内の操作 | 移動先を表示する | [ユーザー詳細](admin-user-detail.md) / ready | ユーザー ID |
| create | ready | ユーザーを追加 | 常時 | UI 内の操作 | 移動先を表示する | [ユーザーを追加](admin-user-create.md) / ready | なし |
| importCsv | ready | CSV インポート | 常時 | UI 内の操作 | 移動先を表示する | [ユーザーの CSV インポート](admin-user-import.md) / ready | なし |

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
