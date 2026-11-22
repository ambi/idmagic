# Agent のアクセス概要

この文書は、Agent のアクセス概要の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[IdManagement](../README.md) / [ページ一覧](README.md) / Agent のアクセス概要

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [Agent 管理](../../../design/application/pages/admin/agents/README.md)

`admin-agent-access-summary`　`/admin/agents/$agentId/access`

Agent に関連する認証、委任、リソースへの権限を確認する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [IdManagement](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | Agent 詳細 |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | 資格情報、本人からの委任と認可詳細、リソースサーバーの設定と各リソースへのアクセスの説明 |

利用手順は[IdManagement のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| credentials | ready | 資格情報 | 常時 | UI 内の操作 | 移動先を表示する | [Agent の資格情報](admin-agent-credentials.md) / ready | なし |
| server | ready | 対象リソースサーバーを編集 | 常時 | UI 内の操作 | 移動先を表示する | [MCP リソースサーバーを編集](../../oauth2/pages/admin-mcp-resource-server-edit.md) / ready | なし |
| user | ready | 委任したユーザーを開く | 常時 | UI 内の操作 | 移動先を表示する | [ユーザー詳細](admin-user-detail.md) / ready | ユーザー ID |
| back | ready | 戻る | 常時 | UI 内の操作 | 移動先を表示する | [Agent 詳細](admin-agent-detail.md) / ready | なし |

## 表示状態

| 状態 | 表示 |
| --- | --- |
| ready | 取得した内容を表示する。 |
| loading | ページ移動中は直前の表示を保つ。初回の取得はルーターが扱う。 |
| error | 表示できなければ共通エラーを表示し、操作失敗は操作の近くに残す。 |
| forbidden | 権限不足を表示し、未反映の状態を保つ。 |
| expired | 認証が必要なページでは再認可し、復帰できなければログインへ移る。 |
| success | 成功した操作を一度通知し、表示内容を再取得する。 |

## 関連する設計

- [TypeSpec の定義](../../../../spec/modules/identity-management/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[IdManagementのページ一覧へ戻る](README.md)
