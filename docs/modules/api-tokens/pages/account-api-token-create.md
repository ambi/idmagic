# 本人の API トークンを発行

この文書は、本人の API トークンを発行の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[ApiTokens](../README.md) / [ページ一覧](README.md) / 本人の API トークンを発行

分類：[アカウントポータル](../../../design/application/pages/account/README.md) / [API トークン](../../../design/application/pages/account/api-tokens/README.md)

`account-api-token-create`　`/account/api-tokens/new`

許可された権限の範囲でトークンを発行する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [ApiTokens](../README.md) |
| 利用者 | 認証済みの本人 |
| 入口 | 本人の API トークン一覧 |
| 利用条件 | 対象テナントの本人のセッションで参照する。CSRF と再認証は共通規則に従う。 |
| 表示内容 | 説明、スコープと有効日数、必要な再認証、発行したトークンの一度だけの表示とコピー |

利用手順は[ApiTokens のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| issue | ready | 発行 | 必要な再認証と発行が成功 | IdMagic.ApiTokens.Operations.IssuePersonalApiToken | 生成したトークンを表示する | [本人の API トークンを発行](account-api-token-create.md) / success | 対象の識別子と操作の入力値 |
| back | ready | キャンセルまたは一覧へ | 常時 | UI 内の操作 | 移動先を表示する | [本人の API トークン](account-api-tokens.md) / ready | なし |

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

- [TypeSpec の定義](../../../../spec/modules/api-tokens/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[ApiTokensのページ一覧へ戻る](README.md)
