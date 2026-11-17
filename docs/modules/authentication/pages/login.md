# ログイン

この文書は、ログインの操作、入力項目と状態を示す。

> 表示確認用の途中プレビュー。全ページの棚卸しと検証は継続中である。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Authentication](../README.md) / [ページ一覧](README.md) / ログイン

分類：[認証](../../../design/application/pages/authentication/README.md) / [サインイン](../../../design/application/pages/authentication/sign-in/README.md)

`login`　`/login`

資格情報または既存の認証手段で認証を開始する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [Authentication](../README.md) |
| 利用者 | 認証を受ける利用者 |
| 入口 | 認可フロー、メールのリンクまたは直接アクセス |
| 利用条件 | 対象テナントの認可トランザクションとセッションの Cookie。CSRF は表示 API から取得する。 |
| 表示内容 | ユーザー名とパスワード、Windows 統合認証の入口、外部 IdP の入口、パスワード再設定の入口 |

利用手順は[Authentication のページ一覧](README.md)にまとめる。

## フォームと入力項目

項目ごとのバリデーション規則は別の作業で定義する。

### 資格情報

識別子：login/credentials。表示条件：常時。確定：ログイン。

| 項目 | 識別子 | 入力の種類 | 型 | 初期値 | 表示条件 | データとの対応 |
| --- | --- | --- | --- | --- | --- | --- |
| ユーザー名 | login/credentials/username | text | string | 空 | 資格情報を使う場合 | IdMagic.Authentication.Operations.SubmitBrowserLoginHttpRequest.username：username |
| パスワード | login/credentials/password | password | string | 空 | 資格情報を使う場合 | IdMagic.Authentication.Operations.SubmitBrowserLoginHttpRequest.password：password |

## パラメーター

| 項目 | 識別子 | 入力の種類 | 型 | 初期値 | 表示条件 | データとの対応 |
| --- | --- | --- | --- | --- | --- | --- |
| 認証後の戻り先 | login/parameters/returnTo | URL パラメーター | string | URL の return_to | 指定された場合 | 同じテナントの許可された戻り先 |
| Windows 統合認証の抑止 | login/parameters/negotiate | URL パラメーター | string | URL の negotiate | Windows 統合認証を提供する場合 | skip なら統合認証を自動実行しない |

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| signIn | ready | ログイン | 第二要素が必要 | IdMagic.Authentication.Operations.SubmitBrowserLogin | 第二要素の選択と入力を表示する | [二要素認証](totp.md) / ready | 認可トランザクション、return_to |
| complete | ready | ログイン | 認証が成立し追加の対話が不要 | IdMagic.Authentication.Operations.SubmitBrowserLogin | 認可の結果を返す | 呼び出し元アプリケーション / external | 認可コードと state |
| federate | ready | 外部 IdP を選択 | 常時 | UI 内の操作 | 選択した IdP の認証へ進む | 外部 IdP / external | IdP の識別子、return_to |
| reset | ready | パスワードを忘れた | 常時 | UI 内の操作 | 移動先を表示する | [パスワード再設定の依頼](forgot-password.md) / ready | なし |
| failure | ready | ログイン | 認証の失敗 | UI 内の操作 | 失敗を表示して入力を保持する | [ログイン](login.md) / error | なし |

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

- [TypeSpec の定義](../../../../spec/modules/authentication/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通のフォーム規則](../../../design/application/user-interface.md#フォームと検証)

[Authenticationのページ一覧へ戻る](README.md)
