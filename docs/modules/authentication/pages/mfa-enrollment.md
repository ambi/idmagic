# 認可フロー内の MFA 登録

この文書は、認可フロー内の MFA 登録の操作、入力項目と状態を示す。

> 表示確認用の途中プレビュー。全ページの棚卸しと検証は継続中である。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Authentication](../README.md) / [ページ一覧](README.md) / 認可フロー内の MFA 登録

分類：[認証](../../../design/application/pages/authentication/README.md) / [サインイン](../../../design/application/pages/authentication/sign-in/README.md)

`mfa-enrollment`　`/mfa-enrollment`

ログインを継続する前に TOTP を登録する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [Authentication](../README.md) |
| 利用者 | 認可フローの利用者 |
| 入口 | MFA 登録を要求する認可フロー |
| 利用条件 | 対象テナントのブラウザートランザクションを使う。CSRF は表示 API から得る。 |
| 表示内容 | 登録用 QR と設定キー、確認コード、登録結果 |

利用手順は[Authentication のページ一覧](README.md)にまとめる。

## フォームと入力項目

項目ごとのバリデーション規則は別の作業で定義する。

### TOTP 登録の確認

識別子：mfa-enrollment/confirm。表示条件：登録開始 API の成功後。確定：登録を完了。

| 項目 | 識別子 | 入力の種類 | 型 | 初期値 | 表示条件 | データとの対応 |
| --- | --- | --- | --- | --- | --- | --- |
| 確認コード | mfa-enrollment/confirm/code | text | string | 空 | 対象のフォームを操作するとき | IdMagic.Contract.BrowserMfaEnrollmentConfirmRequest.code |

## パラメーター

ページ固有のパラメーターはない。

## 実装と照合していない入力

次のフォーム部品は静的な `name` 属性を持たないため、上の入力項目との対応を機械的に確かめていない。ページを変更するときは、これらの部品を入力項目と突き合わせる。

| ファイル | 部品の数 |
| --- | --- |
| [`features/auth-flow/MfaEnrollmentPage.tsx`](../../../../frontend/src/features/auth-flow/MfaEnrollmentPage.tsx) | 1 |

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| start | ready | 登録を開始 | 開始要求が成功 | IdMagic.Authentication.Operations.StartBrowserMfaEnrollment | QR と設定キーを表示する | [認可フロー内の MFA 登録](mfa-enrollment.md) / ready | 対象の識別子と操作の入力値 |
| confirm | ready | 登録を完了 | コードの確認成功 | IdMagic.Authentication.Operations.ConfirmBrowserMfaEnrollment | 認可トランザクションの次の表示へ進む | 呼び出し元アプリケーション / external | 対象の識別子と操作の入力値 |
| failure | ready | 確認 | コードや登録要求の拒否 | UI 内の操作 | 登録せずエラーを表示する | [認可フロー内の MFA 登録](mfa-enrollment.md) / error | 対象の識別子と操作の入力値 |

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
- [共通の UI 規則](../../../design/application/user-interface.md)

[Authenticationのページ一覧へ戻る](README.md)
