# Agent のガードレールを編集

この文書は、Agent のガードレールを編集の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[OAuth2](../README.md) / [ページ一覧](README.md) / Agent のガードレールを編集

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [Agent 管理](../../../design/application/pages/admin/agents/README.md)

`admin-agent-guardrail-edit`　`/admin/agents/$agentId/guardrail/edit`

Agent 固有の制約を保存し、継承するデフォルト値と区別する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [OAuth2](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | Agent 詳細 |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | デフォルトのポリシーと個別の上書き、許可する操作、制約と上限、変更の保存と個別設定の削除確認 |

利用手順は[OAuth2 のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| save | ready | 保存 | 保存が成功 | IdMagic.OAuth2.Operations.PutAdminAgentGuardrailPolicy | Agent の詳細へ戻る | [Agent 詳細](../../identity-management/pages/admin-agent-detail.md) / ready | なし |
| delete | ready | 個別設定の削除を確認 | 削除が成功 | IdMagic.OAuth2.Operations.DeleteAdminAgentGuardrailPolicy | デフォルト値を使う詳細へ戻る | [Agent 詳細](../../identity-management/pages/admin-agent-detail.md) / ready | なし |
| cancel | ready | キャンセル | 常時 | UI 内の操作 | 移動先を表示する | [Agent 詳細](../../identity-management/pages/admin-agent-detail.md) / ready | なし |

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
