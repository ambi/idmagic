# プロビジョニングの概要

この文書は、プロビジョニングの概要の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Provisioning](../README.md) / [ページ一覧](README.md) / プロビジョニングの概要

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [プロビジョニング](../../../design/application/pages/admin/provisioning/README.md)

`admin-provisioning`　`/admin/provisioning`

テナント内の接続状態を一覧で確認する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [Provisioning](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | 管理メニュー |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | 接続先、状態とエラー、対象アプリケーションの同期管理への入口 |

利用手順は[Provisioning のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| detail | ready | 接続先を開く | 常時 | UI 内の操作 | 移動先を表示する | [アプリケーションのプロビジョニング](admin-application-provisioning.md) / ready | アプリケーション ID |

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

- [TypeSpec の定義](../../../../spec/modules/provisioning/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[Provisioningのページ一覧へ戻る](README.md)
