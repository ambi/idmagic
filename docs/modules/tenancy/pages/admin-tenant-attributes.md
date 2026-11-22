# ユーザー属性の定義

この文書は、ユーザー属性の定義の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Tenancy](../README.md) / [ページ一覧](README.md) / ユーザー属性の定義

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [設定](../../../design/application/pages/admin/settings/README.md)

`admin-tenant-attributes`　`/admin/tenant/attributes`

ユーザーのカスタム属性の型と公開範囲を管理する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [Tenancy](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | 管理メニュー |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | 型、表示名、編集・閲覧範囲と並び順、属性を編集するダイアログと削除の確認、新規定義への入口 |

利用手順は[Tenancy のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| create | ready | 追加 | 常時 | UI 内の操作 | 移動先を表示する | [ユーザー属性を追加](admin-tenant-attribute-create.md) / ready | なし |
| save | ready | 編集を保存または削除を確認 | 常時 | IdMagic.Tenancy.Operations.UpdateTenantUserAttributeSchema | 属性の定義を更新する | [ユーザー属性の定義](admin-tenant-attributes.md) / ready | 対象の識別子と操作の入力値 |
| refresh | ready | 再取得 | 常時 | IdMagic.Tenancy.Operations.GetTenantUserAttributeSchema | 現在の定義を表示する | [ユーザー属性の定義](admin-tenant-attributes.md) / ready | 対象の識別子と操作の入力値 |

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

- [TypeSpec の定義](../../../../spec/modules/tenancy/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[Tenancyのページ一覧へ戻る](README.md)
