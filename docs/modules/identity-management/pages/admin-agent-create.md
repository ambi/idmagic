# Agent を登録

この文書は、Agent を登録の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[IdManagement](../README.md) / [ページ一覧](README.md) / Agent を登録

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [Agent 管理](../../../design/application/pages/admin/agents/README.md)

`admin-agent-create`　`/admin/agents/new`

責任者と初期の資格情報・アクセスを指定する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [IdManagement](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | Agent 一覧 |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | 基本情報と責任者、資格情報は後で設定、公開鍵、既存の資格情報から選択、初期アクセス、段階ごとの進捗と失敗した段階からの再試行 |

利用手順は[IdManagement のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| save | ready | 登録 | 常時 | IdMagic.IdManagement.Operations.RegisterAgent、IdMagic.OAuth2.Operations.CreateAdminOAuth2Client、IdMagic.IdManagement.Operations.BindAgentCredential、IdMagic.Authorization.Operations.WriteRelationTuples | Agent、資格情報と初期アクセスを順に登録し、結果を表示する | [Agent を登録](admin-agent-create.md) / success | 対象の識別子と操作の入力値 |
| retry | ready | 失敗した登録を再試行 | 途中の段階が失敗 | UI 内の操作 | 作成済みの対象を保ち、残りの設定を再試行する | [Agent を登録](admin-agent-create.md) / ready | 対象の識別子と操作の入力値 |
| detail | success | 作成した Agent を開く | 常時 | UI 内の操作 | 移動先を表示する | [Agent 詳細](admin-agent-detail.md) / ready | なし |
| cancel | ready | キャンセルまたは戻る | 常時 | UI 内の操作 | 移動先を表示する | [Agent 一覧](admin-agents.md) / ready | なし |

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
- [共通の UI 規則](../../../design/application/user-interface.md)

[IdManagementのページ一覧へ戻る](README.md)
