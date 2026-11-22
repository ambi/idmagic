# グループを編集

この文書は、グループを編集の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[IdManagement](../README.md) / [ページ一覧](README.md) / グループを編集

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [グループ管理](../../../design/application/pages/admin/groups/README.md)

`admin-group-edit`　`/admin/groups/$groupId/edit`

属性、ロール、所属と動的ルールを管理する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [IdManagement](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | 一覧または詳細 |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | 基本情報と属性の保存、SCIM 管理元の注意、メンバーの追加と除外、メンバーの CSV 取り込みと出力の入口、動的ルールのビルダーと CEL 編集、プレビュー、有効化、ロール委任の申請と取消 |

利用手順は[IdManagement のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| save | ready | 保存 | 変更があり保存が成功 | IdMagic.IdManagement.Operations.UpdateGroup | 詳細へ戻る | [グループ詳細](admin-group-detail.md) / ready | なし |
| rule | ready | ルールを保存・試行・有効化または無効化 | 常時 | IdMagic.IdManagement.Operations.UpdateDynamicGroupRule、IdMagic.IdManagement.Operations.PreviewDynamicGroupRule、IdMagic.IdManagement.Operations.EnableDynamicGroupRule、IdMagic.IdManagement.Operations.DisableDynamicGroupRule | ルールまたは試行結果を表示する | [グループを編集](admin-group-edit.md) / ready | 対象の識別子と操作の入力値 |
| delegation | ready | 委任を申請または取消 | 常時 | IdMagic.IdManagement.Operations.SetGroupRoleDelegation | 委任の状態を更新する | [グループを編集](admin-group-edit.md) / ready | 対象の識別子と操作の入力値 |
| members | ready | メンバーを追加または除外 | 常時 | IdMagic.IdManagement.Operations.AddGroupMember、IdMagic.IdManagement.Operations.RemoveGroupMember | メンバーを再取得する | [グループを編集](admin-group-edit.md) / ready | 対象の識別子と操作の入力値 |
| importMembers | ready | メンバーを取り込む | 静的なメンバーで外部の管理元がない | UI 内の操作 | 移動先を表示する | [グループメンバーの CSV インポート](admin-group-member-import.md) / ready | なし |
| exportMembers | ready | メンバーを出力する | 常時 | UI 内の操作 | 移動先を表示する | [グループメンバーの CSV エクスポート](admin-group-members-export.md) / ready | なし |
| member | ready | メンバーのユーザーを開く | 常時 | UI 内の操作 | 移動先を表示する | [ユーザー一覧](admin-users.md) / ready | ユーザー名 |
| cancel | ready | キャンセルまたは戻る | 常時 | UI 内の操作 | 移動先を表示する | [グループ詳細](admin-group-detail.md) / ready | なし |

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
