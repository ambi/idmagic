# 二要素認証

この文書は、二要素認証の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Authentication](../README.md) / [ページ一覧](README.md) / 二要素認証

分類：[認証](../../../design/application/pages/authentication/README.md) / [サインイン](../../../design/application/pages/authentication/sign-in/README.md)

`totp`　`/totp`

利用可能な認証手段で第二要素を確認する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [Authentication](../README.md) |
| 利用者 | 認証を受ける利用者 |
| 入口 | 認可フロー、メールのリンクまたは直接アクセス |
| 利用条件 | 対象テナントの認可トランザクションとセッションの Cookie。CSRF は表示 API から取得する。 |
| 表示内容 | TOTP、パスキーと回復コードの切り替え、認証コードの入力、デバイスを記憶する選択 |

利用手順は[Authentication のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| verify | ready | 認証 | 第二要素が成立し認可が完了 | IdMagic.Authentication.Operations.SubmitBrowserTotp、IdMagic.Authentication.Operations.SubmitBrowserRecoveryCode、IdMagic.Authentication.Operations.SubmitBrowserWebAuthn | 認可の結果を返す | 呼び出し元アプリケーション / external | 認可トランザクション |
| failure | ready | 認証 | コードまたはパスキーの確認に失敗 | UI 内の操作 | 選択した手段と問題を表示する | [二要素認証](totp.md) / error | なし |

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
