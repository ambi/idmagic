# 署名鍵の稼働状態

この文書は、署名鍵の稼働状態の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[SigningKeys](../README.md) / [ページ一覧](README.md) / 署名鍵の稼働状態

分類：[システムコンソール](../../../design/application/pages/system/README.md) / [鍵](../../../design/application/pages/system/keys/README.md)

`system-key-health`　`/system/keys`

全テナントの署名鍵の状態を確認する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [SigningKeys](../README.md) |
| 利用者 | システム管理者 |
| 入口 | システム管理メニュー |
| 利用条件 | 制御プレーンの管理権限と必要な認証保証で参照する。 |
| 表示内容 | テナント別の有効鍵と更新状態、再取得と失敗の表示 |

利用手順は[SigningKeys のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| refresh | ready | 再取得 | 常時 | IdMagic.SigningKeys.Operations.ListTenantKeyHealth | テナントごとの署名鍵の状態を更新する | [署名鍵の稼働状態](system-key-health.md) / ready | 対象の識別子と操作の入力値 |

## 表示状態

| 状態 | 表示 |
| --- | --- |
| ready | 取得した内容を表示する。 |
| loading | ページ移動中は直前の表示を保つ。初回の取得はルーターが扱う。 |
| error | 表示できなければ共通エラーを表示し、操作失敗は操作の近くに残す。 |
| forbidden | 権限不足を表示し、未反映の状態を保つ。 |
| expired | 認証が必要なページでは再認可し、復帰できなければログインへ移る。 |
| success | 成功した操作を一度通知し、表示内容を再取得する。 |

## 関連する設計

- [TypeSpec の定義](../../../../spec/modules/signing-keys/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[SigningKeysのページ一覧へ戻る](README.md)
