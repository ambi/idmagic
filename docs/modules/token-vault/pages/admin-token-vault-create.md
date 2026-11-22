# 外部サービスの接続定義を登録

この文書は、外部サービスの接続定義を登録の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[TokenVault](../README.md) / [ページ一覧](README.md) / 外部サービスの接続定義を登録

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [外部サービス接続](../../../design/application/pages/admin/connections/README.md)

`admin-token-vault-create`　`/admin/token-vault/new`

OAuth の接続先と利用を許可する要求を登録する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [TokenVault](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | 接続定義の一覧 |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | 認可とトークンの接続先、クライアント認証、接続を使えるアプリケーション、要求の規則と権限 |

利用手順は[TokenVault のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| save | ready | 登録 | 作成が成功 | IdMagic.TokenVault.Operations.CreateTokenVaultConnector | 接続定義の一覧へ戻る | [外部サービスの接続定義](admin-token-vault.md) / ready | なし |
| cancel | ready | キャンセル | 常時 | UI 内の操作 | 移動先を表示する | [外部サービスの接続定義](admin-token-vault.md) / ready | なし |

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

- [TypeSpec の定義](../../../../spec/modules/token-vault/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[TokenVaultのページ一覧へ戻る](README.md)
