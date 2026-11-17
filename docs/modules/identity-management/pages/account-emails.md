# 本人のメールアドレス

この文書は、本人のメールアドレスの操作、入力項目と状態を示す。

> 表示確認用の途中プレビュー。全ページの棚卸しと検証は継続中である。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[IdManagement](../README.md) / [ページ一覧](README.md) / 本人のメールアドレス

分類：[アカウントポータル](../../../design/application/pages/account/README.md) / [プロフィール](../../../design/application/pages/account/profile/README.md)

`account-emails`　`/account/emails`

現在の確認状態を確認し、新しいアドレスの確認を依頼する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [IdManagement](../README.md) |
| 利用者 | 認証済みの本人 |
| 入口 | 本人のポータル |
| 利用条件 | 対象テナントの本人のセッションで参照する。CSRF と再認証は共通規則に従う。 |
| 表示内容 | 現在のメールと確認状態、新しいメールアドレス、確認通知の送信結果 |

利用手順は[IdManagement のページ一覧](README.md)にまとめる。

## フォームと入力項目

項目ごとのバリデーション規則は別の作業で定義する。

### メールの変更依頼

識別子：account-emails/change。表示条件：常時。確定：確認メールを送信。

| 項目 | 識別子 | 入力の種類 | 型 | 初期値 | 表示条件 | データとの対応 |
| --- | --- | --- | --- | --- | --- | --- |
| 新しいメールアドレス | account-emails/change/email | email | string | 空 | 対象のフォームを操作するとき | IdMagic.Contract.AccountEmailChangeRequest.new_email：new_email。確認リンクの完了前は現在のアドレスを変えない |

## パラメーター

ページ固有のパラメーターはない。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| send | ready | 確認メールを送信 | 依頼の受け付け成功 | UI 内の操作 | 送信結果を表示し、確認リンクをメールへ届ける | 通知メール / external | 対象の識別子と操作の入力値 |
| failure | ready | 送信 | 入力の拒否または送信失敗 | UI 内の操作 | 入力の近くに問題を表示する | [本人のメールアドレス](account-emails.md) / error | 対象の識別子と操作の入力値 |

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
