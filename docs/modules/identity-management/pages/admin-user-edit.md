# ユーザーを編集

この文書は、ユーザーを編集の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[IdManagement](../README.md) / [ページ一覧](README.md) / ユーザーを編集

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [ユーザー管理](../../../design/application/pages/admin/users/README.md)

`admin-user-edit`　`/admin/users/$id/edit`

プロフィール、ロールとカスタム属性を一回の保存で変更する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [IdManagement](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | 管理コンソールのユーザー一覧または対象ユーザーの詳細 |
| 利用条件 | 対象テナントの管理者として認可を受ける。変更の許可は API が判断する。 |
| 表示内容 | プロフィールの編集、ロールの変更内容の確認、属性定義に応じたカスタム属性、所属グループの追加と除外、必須アクションの設定と解除、セッションの個別失効とすべての失効 |

利用手順は[IdManagement のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| groups | ready | グループへ追加または除外 | 常時 | IdMagic.IdManagement.Operations.AddGroupMember、IdMagic.IdManagement.Operations.RemoveGroupMember | 所属グループを再取得する | [ユーザーを編集](admin-user-edit.md) / ready | 対象の識別子と操作の入力値 |
| group | ready | 所属グループを開く | 常時 | UI 内の操作 | 移動先を表示する | [グループ一覧](admin-groups.md) / ready | グループ ID |
| requiredAction | ready | 必須アクションを切り替える | 常時 | IdMagic.IdManagement.Operations.SetUserRequiredAction、IdMagic.IdManagement.Operations.ClearUserRequiredAction | ユーザーを再取得して切り替えを通知する | [ユーザーを編集](admin-user-edit.md) / ready | 対象の識別子と操作の入力値 |
| sessions | ready | セッションを失効またはすべて失効を確認 | 常時 | IdMagic.Authentication.Operations.RevokeSession、IdMagic.Authentication.Operations.RevokeUserSessions | セッション一覧を更新する | [ユーザーを編集](admin-user-edit.md) / ready | 対象の識別子と操作の入力値 |
| save | ready | 保存 | 変更内容を確認し保存に成功 | IdMagic.IdManagement.Operations.UpdateAdminUser | 更新後の詳細を再取得して一度通知する | [ユーザー詳細](admin-user-detail.md) / ready | ユーザー ID |
| cancel | ready | キャンセル | 常時 | UI 内の操作 | 未保存の入力を破棄して詳細へ戻る | [ユーザー詳細](admin-user-detail.md) / ready | ユーザー ID |
| failure | ready | 保存 | 更新の拒否または競合 | UI 内の操作 | 入力を保持し問題を操作の近くに残す | [ユーザーを編集](admin-user-edit.md) / error | なし |

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
- REQ-FRONTEND-002
- [ナビゲーション](../../../design/application/user-interface.md#ナビゲーション)

[IdManagementのページ一覧へ戻る](README.md)
