# ディレクトリ接続の詳細

この文書は、ディレクトリ接続の詳細の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Sourcing](../README.md) / [ページ一覧](README.md) / ディレクトリ接続の詳細

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [ディレクトリ連携](../../../design/application/pages/admin/directories/README.md)

`admin-directory-connector-detail`　`/admin/directory-connectors/$connectorId`

接続の登録状態と取り込み結果を確認する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [Sourcing](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | 接続の一覧または登録後 |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | 接続日時、鍵と登録期限、登録トークン再発行と無効化の確認、取り込み履歴と展開した失敗の詳細、委任認証への入口 |

利用手順は[Sourcing のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| reissue | ready | 登録トークンの再発行を確認 | 常時 | IdMagic.Sourcing.Operations.ReissueDirectoryConnectorEnrollment | 新しいトークンを一度だけ表示する | [ディレクトリ接続の詳細](admin-directory-connector-detail.md) / ready | 対象の識別子と操作の入力値 |
| disable | ready | 無効化を確認 | 常時 | IdMagic.Sourcing.Operations.DisableDirectoryConnector | 接続の無効状態を表示する | [ディレクトリ接続の詳細](admin-directory-connector-detail.md) / ready | 対象の識別子と操作の入力値 |
| delegation | ready | 委任認証 | 常時 | UI 内の操作 | 移動先を表示する | [ディレクトリの委任認証](admin-directory-delegated-authentication.md) / ready | なし |
| back | ready | 戻る | 常時 | UI 内の操作 | 移動先を表示する | [ディレクトリ接続](admin-directory-connectors.md) / ready | なし |

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

- [TypeSpec の定義](../../../../spec/modules/sourcing/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[Sourcingのページ一覧へ戻る](README.md)
