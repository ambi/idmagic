# サインインポリシー

この文書は、サインインポリシーの操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Application](../README.md) / [ページ一覧](README.md) / サインインポリシー

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [セキュリティ](../../../design/application/pages/admin/security/README.md)

`admin-sign-in-policy`　`/admin/sign-in-policy`

テナントのデフォルトの規則とアプリごとの上書きを確認する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [Application](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | 管理メニュー |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | デフォルトの規則の閲覧とページ内での編集、条件と要求する認証の強さ、アプリケーション別の規則と編集への入口 |

利用手順は[Application のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| save | ready | 編集して保存 | 常時 | IdMagic.Application.Operations.UpdateTenantDefaultSignInPolicy | デフォルトのサインイン規則を更新し閲覧へ戻す | [サインインポリシー](admin-sign-in-policy.md) / ready | 対象の識別子と操作の入力値 |
| cancel | ready | キャンセル | 常時 | UI 内の操作 | 編集を破棄して現在の規則を表示する | [サインインポリシー](admin-sign-in-policy.md) / ready | 対象の識別子と操作の入力値 |
| application | ready | アプリの規則を編集 | 常時 | UI 内の操作 | 移動先を表示する | [アプリケーションを編集](admin-application-edit.md) / ready | アプリケーション ID |

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

- [TypeSpec の定義](../../../../spec/modules/application/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[Applicationのページ一覧へ戻る](README.md)
