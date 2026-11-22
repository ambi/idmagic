# テナント詳細

この文書は、テナント詳細の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Tenancy](../README.md) / [ページ一覧](README.md) / テナント詳細

分類：[システムコンソール](../../../design/application/pages/system/README.md) / [テナント](../../../design/application/pages/system/tenants/README.md)

`system-tenant-detail`　`/system/tenants/$realm`

状態、入口の方式とクォータを確認する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [Tenancy](../README.md) |
| 利用者 | システム管理者 |
| 入口 | テナント一覧または保存後 |
| 利用条件 | 制御プレーンの管理権限と必要な認証保証で参照する。 |
| 表示内容 | 基本情報と状態、有効化と無効化の確認、基本情報、入口方式とクォータへの編集の入口 |

利用手順は[Tenancy のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| edit | ready | 基本情報を編集 | 常時 | UI 内の操作 | 移動先を表示する | [テナントの基本情報を編集](system-tenant-edit.md) / ready | なし |
| endpoint | ready | 入口方式を変更 | 常時 | UI 内の操作 | 移動先を表示する | [テナントの入口方式](system-tenant-endpoint-style.md) / ready | なし |
| quota | ready | クォータを編集 | 常時 | UI 内の操作 | 移動先を表示する | [テナントのクォータ](system-tenant-quota.md) / ready | なし |
| status | ready | 有効化または無効化を確認 | 常時 | IdMagic.Tenancy.Operations.EnableTenant、IdMagic.Tenancy.Operations.DisableTenant | 状態を変更し詳細を再取得する | [テナント詳細](system-tenant-detail.md) / ready | 対象の識別子と操作の入力値 |
| back | ready | 戻る | 常時 | UI 内の操作 | 移動先を表示する | [テナント一覧](system-tenants.md) / ready | なし |

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

- [TypeSpec の定義](../../../../spec/modules/tenancy/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[Tenancyのページ一覧へ戻る](README.md)
