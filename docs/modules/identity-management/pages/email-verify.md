# メールアドレスの確認

この文書は、メールアドレスの確認の操作、入力項目と状態を示す。

> 表示確認用の途中プレビュー。全ページの棚卸しと検証は継続中である。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[IdManagement](../README.md) / [ページ一覧](README.md) / メールアドレスの確認

分類：[認証](../../../design/application/pages/authentication/README.md) / [プロフィール](../../../design/application/pages/authentication/profile/README.md)

`email-verify`　`/account/email/verify`

メールに含まれるトークンで変更を確認する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [IdManagement](../README.md) |
| 利用者 | 認可フローの利用者 |
| 入口 | 確認メールのリンク |
| 利用条件 | 対象テナントのブラウザートランザクションを使う。CSRF は表示 API から得る。 |
| 表示内容 | 確認の実行ボタン、確認成功、無効・期限切れなどの結果 |

利用手順は[IdManagement のページ一覧](README.md)にまとめる。

## フォームと入力項目

項目ごとのバリデーション規則は別の作業で定義する。

入力フォームはない。

## パラメーター

| 項目 | 識別子 | 入力の種類 | 型 | 初期値 | 表示条件 | データとの対応 |
| --- | --- | --- | --- | --- | --- | --- |
| 確認トークン | email-verify/parameters/token | URL | string | token | メールのリンクから受け取る | IdMagic.Contract.AccountEmailChangeConfirm.token：AccountEmailChangeConfirm.token。利用者は編集しない |

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| confirm | ready | メールアドレスを確認 | トークンが有効で確認成功 | UI 内の操作 | 確認完了を表示する | [メールアドレスの確認](email-verify.md) / success | 対象の識別子と操作の入力値 |
| failure | ready | 確認 | トークンがない、期限切れ、使用済みまたは拒否 | UI 内の操作 | 確認できない理由を表示する | [メールアドレスの確認](email-verify.md) / error | 対象の識別子と操作の入力値 |

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
