# Agent の棚卸し

この文書は、Agent の棚卸しの操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[IdManagement](../README.md) / [ページ一覧](README.md) / Agent の棚卸し

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [Agent 管理](../../../design/application/pages/admin/agents/README.md)

`admin-agent-inventory`　`/admin/agents/inventory`

責任者、資格情報、利用状態の問題を見つける

| 項目 | 内容 |
| --- | --- |
| 主担当 | [IdManagement](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | Agent 一覧 |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | 検出事項と休眠日数による絞り込み、責任者、最終利用と問題の一覧、対象 Agent の詳細、観測された Agent の候補、確認・除外・再開 |

利用手順は[IdManagement のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| filter | ready | 絞り込みまたは再取得 | 常時 | IdMagic.IdManagement.Operations.ListAgentInventory | 条件に合う棚卸し結果を表示する | [Agent の棚卸し](admin-agent-inventory.md) / ready | 対象の識別子と操作の入力値 |
| observations | ready | 観測を絞り込み・確認・除外・再開 | 常時 | IdMagic.IdManagement.Operations.ListAgentObservations、IdMagic.IdManagement.Operations.ConfirmAgentObservation、IdMagic.IdManagement.Operations.ExcludeAgentObservation、IdMagic.IdManagement.Operations.ReopenAgentObservation | 観測の判断と対応する Agent を更新する | [Agent の棚卸し](admin-agent-inventory.md) / ready | 対象の識別子と操作の入力値 |
| detail | ready | 詳細 | 常時 | UI 内の操作 | 移動先を表示する | [Agent 詳細](admin-agent-detail.md) / ready | なし |
| back | ready | 戻る | 常時 | UI 内の操作 | 移動先を表示する | [Agent 一覧](admin-agents.md) / ready | なし |

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
