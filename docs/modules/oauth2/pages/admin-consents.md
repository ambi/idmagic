# 同意と Agent 委任の管理

この文書は、同意と Agent 委任の管理の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[OAuth2](../README.md) / [ページ一覧](README.md) / 同意と Agent 委任の管理

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [同意と承認](../../../design/application/pages/admin/consent/README.md)

`admin-consents`　`/admin/consents`

利用者の同意を確認して取り消し、Agent への委任を確認する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [OAuth2](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | 管理メニュー |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | ユーザー、アプリケーション、スコープと期限、検索と再取得、同意の取消確認、Agent 委任の一覧 |

利用手順は[OAuth2 のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| refresh | ready | 検索または再取得 | 常時 | IdMagic.OAuth2.Operations.ListAdminConsents、IdMagic.OAuth2.Operations.ListAdminAgentDelegations | 対象の同意と委任を表示する | [同意と Agent 委任の管理](admin-consents.md) / ready | 対象の識別子と操作の入力値 |
| revoke | ready | 取消を確認 | 常時 | IdMagic.OAuth2.Operations.RevokeAdminConsent | 対象の同意を取り消し一覧を更新する | [同意と Agent 委任の管理](admin-consents.md) / ready | 対象の識別子と操作の入力値 |

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
| empty | 対象がないことと、作成または検索条件を変える入口を示す。 |

## 関連する設計

- [TypeSpec の定義](../../../../spec/modules/oauth2/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[OAuth2のページ一覧へ戻る](README.md)
