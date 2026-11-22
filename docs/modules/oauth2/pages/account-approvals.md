# 本人宛ての承認依頼

この文書は、本人宛ての承認依頼の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[OAuth2](../README.md) / [ページ一覧](README.md) / 本人宛ての承認依頼

分類：[アカウントポータル](../../../design/application/pages/account/README.md) / [同意と承認](../../../design/application/pages/account/consent/README.md)

`account-approvals`　`/account/approvals`

Agent の実行や認可の依頼を確認して承認または拒否する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [OAuth2](../README.md) |
| 利用者 | 認証済みの本人 |
| 入口 | 本人のポータル |
| 利用条件 | 対象テナントの本人のセッションで参照する。CSRF と再認証は共通規則に従う。 |
| 表示内容 | 要求元、対象、要求内容と期限、承認と拒否、判断の完了通知 |

利用手順は[OAuth2 のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| decide | ready | 承認・拒否 | 再認証と判断の記録成功 | IdMagic.OAuth2.Operations.DecideMyApprovalRequest | 対象を一覧から除き、判断を通知する | [本人宛ての承認依頼](account-approvals.md) / ready | 対象の識別子と操作の入力値 |
| failure | ready | 判断 | 再認証の中止、期限切れまたは API の拒否 | UI 内の操作 | 判断を確定せず理由を表示する | [本人宛ての承認依頼](account-approvals.md) / error | 対象の識別子と操作の入力値 |

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
