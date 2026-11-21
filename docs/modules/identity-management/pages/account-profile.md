# 本人のプロフィール

この文書は、本人のプロフィールの操作、入力項目と状態を示す。

> 表示確認用の途中プレビュー。全ページの棚卸しと検証は継続中である。

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

## フォームと入力項目

項目ごとのバリデーション規則は別の作業で定義する。

入力フォームはない。

## パラメーター

| 項目 | 識別子 | 入力の種類 | 型 | 初期値 | 表示条件 | データとの対応 |
| --- | --- | --- | --- | --- | --- | --- |
| 更新通知 | account-profile/parameters/notice | URL | string | notice | 編集完了から来た場合 | success のとき更新完了を通知する。永続化しない |

## 実装と照合していない入力

次のフォーム部品は静的な `name` 属性を持たないため、上の入力項目との対応を機械的に確かめていない。ページを変更するときは、これらの部品を入力項目と突き合わせる。

| ファイル | 部品の数 |
| --- | --- |
| [`features/account/AccountProfileAttributes.tsx`](../../../../frontend/src/features/account/AccountProfileAttributes.tsx) | 2 |
| [`features/account/AccountProfilePage.tsx`](../../../../frontend/src/features/account/AccountProfilePage.tsx) | 3 |

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| edit | ready | プロフィールを編集 | リンクを選択したとき | UI 内の操作 | 編集可能な項目を表示する | [本人のプロフィールを編集](account-profile-edit.md) / ready | 対象の識別子とテナントの文脈 |

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
