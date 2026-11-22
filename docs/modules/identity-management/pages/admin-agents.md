# Agent 一覧

この文書は、Agent 一覧の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[IdManagement](../README.md) / [ページ一覧](README.md) / Agent 一覧

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [Agent 管理](../../../design/application/pages/admin/agents/README.md)

`admin-agents`　`/admin/agents`

Agent を選び、状態と責任者を確認する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [IdManagement](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | 管理メニュー |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | 一覧、ページ送りと再取得、選択対象の詳細と削除の確認、追加と棚卸しへの入口 |

利用手順は[IdManagement のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| refresh | ready | 選択または再取得 | 常時 | IdMagic.IdManagement.Operations.ListAgents | 一覧と選択した Agent を表示する | [Agent 一覧](admin-agents.md) / ready | 対象の識別子と操作の入力値 |
| create | ready | 作成 | 常時 | UI 内の操作 | 移動先を表示する | [Agent を登録](admin-agent-create.md) / ready | なし |
| detail | ready | 詳細 | 常時 | UI 内の操作 | 移動先を表示する | [Agent 詳細](admin-agent-detail.md) / ready | なし |
| edit | ready | 編集 | 緊急停止していない | UI 内の操作 | 移動先を表示する | [Agent を編集](admin-agent-edit.md) / ready | なし |
| access | ready | アクセス概要 | 常時 | UI 内の操作 | 移動先を表示する | [Agent のアクセス概要](admin-agent-access-summary.md) / ready | なし |
| resources | ready | リソース | 常時 | UI 内の操作 | 移動先を表示する | [Agent のリソース](admin-agent-resources.md) / ready | なし |
| credentials | ready | 資格情報 | 常時 | UI 内の操作 | 移動先を表示する | [Agent の資格情報](admin-agent-credentials.md) / ready | なし |
| inventory | ready | 棚卸し | 常時 | UI 内の操作 | 移動先を表示する | [Agent の棚卸し](admin-agent-inventory.md) / ready | なし |
| delete | ready | 削除を確認 | 常時 | IdMagic.IdManagement.Operations.DeleteAgent | 対象を削除し一覧を更新する | [Agent 一覧](admin-agents.md) / ready | 対象の識別子と操作の入力値 |

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
