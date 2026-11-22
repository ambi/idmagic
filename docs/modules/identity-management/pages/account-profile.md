# 本人のプロフィール

この文書は、本人のプロフィールの操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[IdManagement](../README.md) / [ページ一覧](README.md) / 本人のプロフィール

分類：[アカウントポータル](../../../design/application/pages/account/README.md) / [プロフィール](../../../design/application/pages/account/profile/README.md)

`account-profile`　`/account/profile`

本人の基本情報と公開された属性を確認する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [IdManagement](../README.md) |
| 利用者 | 認証済みの本人 |
| 入口 | 本人のポータル、編集完了 |
| 利用条件 | 対象テナントの本人のセッションで参照する。CSRF と再認証は共通規則に従う。 |
| 表示内容 | ユーザー名、表示名、氏名、メール、属性グループごとの登録内容、編集の入口と更新通知 |

利用手順は[IdManagement のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| edit | ready | プロフィールを編集 | リンクを選択したとき | UI 内の操作 | 編集可能な項目を表示する | [本人のプロフィールを編集](account-profile-edit.md) / ready | 対象の識別子とテナントの文脈 |
| email | ready | メールアドレスを変更 | リンクを選択したとき | UI 内の操作 | 確認の依頼へ進む | [本人のメールアドレス](account-emails.md) / ready | 対象の識別子とテナントの文脈 |

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
