# Agent の資格情報

この文書は、Agent の資格情報の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[IdManagement](../README.md) / [ページ一覧](README.md) / Agent の資格情報

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [Agent 管理](../../../design/application/pages/admin/agents/README.md)

`admin-agent-credentials`　`/admin/agents/$agentId/credentials`

認証に使う資格情報を作成または関連付ける

| 項目 | 内容 |
| --- | --- |
| 主担当 | [IdManagement](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | Agent 詳細 |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | 既存の資格情報と新規の公開鍵・SPIFFE の選択、信頼バンドルと公開鍵の案内、作成後の関連付けの部分失敗、解除の確認と関連するリソースサーバー |

利用手順は[IdManagement のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| bind | ready | 作成または関連付け | 常時 | IdMagic.OAuth2.Operations.CreateAdminOAuth2Client、IdMagic.IdManagement.Operations.BindAgentCredential | 資格情報を登録し Agent へ関連付けて再取得する | [Agent の資格情報](admin-agent-credentials.md) / ready | 対象の識別子と操作の入力値 |
| unbind | ready | 解除を確認 | 常時 | IdMagic.IdManagement.Operations.UnbindAgentCredential | 関連付けを解除して資格情報を再取得する | [Agent の資格情報](admin-agent-credentials.md) / ready | 対象の識別子と操作の入力値 |
| trustBundles | ready | 信頼バンドルを管理 | 常時 | UI 内の操作 | 移動先を表示する | [ワークロードの信頼バンドル](../../workloadidentity/pages/admin-workload-identity.md) / ready | なし |
| resourceServer | ready | リソースサーバーを登録 | 常時 | UI 内の操作 | 移動先を表示する | [MCP リソースサーバーを登録](../../oauth2/pages/admin-mcp-resource-server-create.md) / ready | なし |
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
| invalid | 入力不備の表示は共通フォーム規則に従う。項目ごとの規則は別に定義する。 |
| submitting | 実行中の操作を示し、二重送信を防ぐ。 |

## 関連する設計

- [TypeSpec の定義](../../../../spec/modules/identity-management/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[IdManagementのページ一覧へ戻る](README.md)
