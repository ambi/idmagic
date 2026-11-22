# 認可モデルを作成

この文書は、認可モデルを作成の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[AccessControl](../README.md) / [ページ一覧](README.md) / 認可モデルを作成

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [認可](../../../design/application/pages/admin/authorization/README.md)

`admin-authorization-model-create`　`/admin/authorization/model/new`

最初の認可モデルを確認して公開する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [AccessControl](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | モデルがないときの作成案内 |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | フォームと JSON の編集、文書閲覧のテンプレート、公開前の差分確認と競合時の再確認 |

利用手順は[AccessControl のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| review | ready | 公開前に確認 | 常時 | UI 内の操作 | 現行のバージョンと変更差分を表示する | [認可モデルを作成](admin-authorization-model-create.md) / review | 対象の識別子と操作の入力値 |
| publish | review | 公開 | 競合なく公開が成功 | IdMagic.Authorization.Operations.PutAuthorizationModel | 公開したモデルへ進む | [認可モデル](admin-authorization-model.md) / ready | なし |
| edit | review | 編集へ戻る | 常時 | UI 内の操作 | 編集内容を保持して確認を閉じる | [認可モデルを作成](admin-authorization-model-create.md) / ready | 対象の識別子と操作の入力値 |
| cancel | ready | キャンセル | 常時 | UI 内の操作 | 移動先を表示する | [認可モデル](admin-authorization-model.md) / ready | なし |

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
| review | 公開する差分と現行バージョンを確認する。 |

## 関連する設計

- [TypeSpec の定義](../../../../spec/modules/authorization/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[AccessControlのページ一覧へ戻る](README.md)
