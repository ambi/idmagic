# アプリケーションへの同意

この文書は、アプリケーションへの同意の操作、入力項目と状態を示す。

> 表示確認用の途中プレビュー。全ページの棚卸しと検証は継続中である。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[OAuth2](../README.md) / [ページ一覧](README.md) / アプリケーションへの同意

分類：[認証](../../../design/application/pages/authentication/README.md) / [同意と承認](../../../design/application/pages/authentication/consent/README.md)

`consent`　`/consent`

開示する情報と認可詳細を確認して許可または拒否する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [OAuth2](../README.md) |
| 利用者 | 認可フローの利用者 |
| 入口 | 認可トランザクション |
| 利用条件 | 対象テナントのブラウザートランザクションを使う。CSRF は表示 API から得る。 |
| 表示内容 | 要求元アプリケーション、スコープごとの説明、認可詳細、許可と拒否 |

利用手順は[OAuth2 のページ一覧](README.md)にまとめる。

## フォームと入力項目

項目ごとのバリデーション規則は別の作業で定義する。

### 同意の判断

識別子：consent/decision。表示条件：トランザクションを表示できる場合。確定：許可または拒否。

| 項目 | 識別子 | 入力の種類 | 型 | 初期値 | 表示条件 | データとの対応 |
| --- | --- | --- | --- | --- | --- | --- |
| 同意の判断 | consent/decision/action | 許可・拒否ボタン | "allow" &#124; "deny" | 未選択 | 送信中以外 | IdMagic.Contract.BrowserConsentRequest.action：BrowserConsentRequest.action |

## パラメーター

ページ固有のパラメーターはない。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| decide | ready | 許可・拒否 | 判断の記録成功 | IdMagic.OAuth2.Operations.SubmitBrowserConsent | 認可トランザクションの結果を呼び出し元へ返す | 呼び出し元アプリケーション / external | 対象の識別子と操作の入力値 |
| failure | ready | 送信 | 拒否または通信失敗 | UI 内の操作 | 判断が完了していないことを表示し再試行できる | [アプリケーションへの同意](consent.md) / error | 対象の識別子と操作の入力値 |

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

- [TypeSpec の定義](../../../../spec/modules/oauth2/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[OAuth2のページ一覧へ戻る](README.md)
