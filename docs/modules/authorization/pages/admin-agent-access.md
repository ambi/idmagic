# Agent のアクセス割り当て

この文書は、Agent のアクセス割り当ての操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[AccessControl](../README.md) / [ページ一覧](README.md) / Agent のアクセス割り当て

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [認可](../../../design/application/pages/admin/authorization/README.md)

`admin-agent-access`　`/admin/authorization/access`

Agent とリソースサーバーの関係を管理する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [AccessControl](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | 管理メニュー、Agent のリソース |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | Agent とサーバーの絞り込み、リソースと関係の割り当て、解除の確認、認可モデル、関係タプルと試行の入口 |

利用手順は[AccessControl のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| create | ready | 追加 | 常時 | UI 内の操作 | 移動先を表示する | [アクセス割り当てを追加](admin-agent-access-create.md) / ready | なし |
| delete | ready | 解除を確認 | 常時 | IdMagic.Authorization.Operations.WriteRelationTuples | 関係を削除して一覧を再取得する | [Agent のアクセス割り当て](admin-agent-access.md) / ready | 対象の識別子と操作の入力値 |
| filter | ready | 対象を絞り込む | 常時 | UI 内の操作 | 選択した Agent とサーバーの関係を表示する | [Agent のアクセス割り当て](admin-agent-access.md) / ready | 対象の識別子と操作の入力値 |
| authorizationModel | ready | 認可モデル | 常時 | UI 内の操作 | 移動先を表示する | [認可モデル](admin-authorization-model.md) / ready | なし |
| tuples | ready | 関係タプル | 常時 | UI 内の操作 | 移動先を表示する | [関係タプル](admin-authorization-tuples.md) / ready | なし |
| check | ready | アクセスを試す | 常時 | UI 内の操作 | 移動先を表示する | [アクセス判定の試行](admin-authorization-check.md) / ready | なし |

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

- [TypeSpec の定義](../../../../spec/modules/authorization/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[AccessControlのページ一覧へ戻る](README.md)
