# MCP リソースサーバーを登録

この文書は、MCP リソースサーバーを登録の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[OAuth2](../README.md) / [ページ一覧](README.md) / MCP リソースサーバーを登録

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [Agent 管理](../../../design/application/pages/admin/agents/README.md)

`admin-mcp-resource-server-create`　`/admin/mcp-resource-servers/new`

接続先と認証・アクセス規則を登録する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [OAuth2](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | リソースサーバー一覧 |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | 既存アプリケーションの利用または新規作成、接続 URL、リソース識別子と認証、ツールとアクセスの規則、スコープとツールの変更度、個人データの扱いと申告の説明、登録結果と初期資格情報の表示、マネージド型の対応条件と準備手順、作成途中の失敗時は作成済みの対象を保持 |

利用手順は[OAuth2 のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| save | ready | 登録 | 常時 | IdMagic.Application.Operations.CreateAdminApplication、IdMagic.Application.Operations.UpdateApplicationOidcConfig、IdMagic.OAuth2.Operations.CreateAdminMcpResourceServer | 必要なアプリケーションと接続先を登録し結果を表示する | [MCP リソースサーバーを登録](admin-mcp-resource-server-create.md) / success | 対象の識別子と操作の入力値 |
| edit | success | ツールを設定 | 常時 | UI 内の操作 | 移動先を表示する | [MCP リソースサーバーを編集](admin-mcp-resource-server-edit.md) / ready | なし |
| cancel | ready | キャンセルまたは一覧へ | 常時 | UI 内の操作 | 移動先を表示する | [MCP リソースサーバー](admin-mcp-resource-servers.md) / ready | なし |

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

- [TypeSpec の定義](../../../../spec/modules/oauth2/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[OAuth2のページ一覧へ戻る](README.md)
