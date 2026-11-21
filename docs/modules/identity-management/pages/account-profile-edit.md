# 本人のプロフィールを編集

この文書は、本人のプロフィールを編集の操作、入力項目と状態を示す。

> 表示確認用の途中プレビュー。全ページの棚卸しと検証は継続中である。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[IdManagement](../README.md) / [ページ一覧](README.md) / 本人のプロフィールを編集

分類：[アカウントポータル](../../../design/application/pages/account/README.md) / [プロフィール](../../../design/application/pages/account/profile/README.md)

`account-profile-edit`　`/account/profile/edit`

本人に編集が許可された基本情報と属性を更新する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [IdManagement](../README.md) |
| 利用者 | 認証済みの本人 |
| 入口 | 本人のプロフィール |
| 利用条件 | 対象テナントの本人のセッションで参照する。CSRF と再認証は共通規則に従う。 |
| 表示内容 | 基本情報の入力、editable_attributes の定義に応じた入力、保存とキャンセル |

利用手順は[IdManagement のページ一覧](README.md)にまとめる。

## フォームと入力項目

項目ごとのバリデーション規則は別の作業で定義する。

### プロフィール

識別子：account-profile-edit/profile。表示条件：常時。確定：保存。

| 項目 | 識別子 | 入力の種類 | 型 | 初期値 | 表示条件 | データとの対応 |
| --- | --- | --- | --- | --- | --- | --- |
| 表示名 | account-profile-edit/profile/name | text | string | profile.name | 常時 | IdMagic.Contract.AccountProfileUpdateRequest.name |
| 名 | account-profile-edit/profile/givenName | text | string | profile.given_name | 常時 | IdMagic.Contract.AccountProfileUpdateRequest.given_name |
| 姓 | account-profile-edit/profile/familyName | text | string | profile.family_name | 常時 | IdMagic.Contract.AccountProfileUpdateRequest.family_name |
| 編集可能なカスタム属性（繰り返し） | account-profile-edit/profile/attributes | 定義の型に応じた text、number、date、checkbox。配列は区切り入力 | Record&lt;unknown&gt; | profile.attributes の同じキー | profile.editable_attributes に含まれる定義ごとに表示 | IdMagic.Contract.AccountProfileUpdateRequest.attributes：attributes[key]。文字列表現を定義の型の AttributeValue へ変換する |

## パラメーター

ページ固有のパラメーターはない。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| save | ready | 保存 | 更新成功 | UI 内の操作 | プロフィールへ戻り success 通知を表示する | [本人のプロフィール](account-profile.md) / ready | 対象の識別子とテナントの文脈 |
| cancel | ready | キャンセル・戻る | リンクを選択したとき | UI 内の操作 | 変更を送信せずプロフィールへ戻る | [本人のプロフィール](account-profile.md) / ready | 対象の識別子とテナントの文脈 |
| failure | ready | 保存 | 更新拒否または通信失敗 | UI 内の操作 | 入力を保持して問題を表示する | [本人のプロフィールを編集](account-profile-edit.md) / error | 対象の識別子と操作の入力値 |

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
