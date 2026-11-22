# Tenancy のページ

この文書は、Tenancy が主担当となるページの設計を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Tenancy のモジュール設計](../README.md) / [UI 全体の索引](../../../design/application/pages/README.md)

## テナント

| ページ | URL | 目的 |
| --- | --- | --- |
| [テナント一覧](system-tenants.md) | `/system/tenants` | テナントの状態と管理の入口を確認する |
| [テナントを作成](system-tenant-create.md) | `/system/new-tenant` | テナントの識別子と初期設定を登録する |
| [テナント詳細](system-tenant-detail.md) | `/system/tenants/$realm` | 状態、入口の方式とクォータを確認する |
| [テナントの基本情報を編集](system-tenant-edit.md) | `/system/tenants/$realm/edit` | 表示情報と基本設定を変更する |
| [テナントの入口方式](system-tenant-endpoint-style.md) | `/system/tenants/$realm/endpoint-style` | テナントの URL 方式を影響を確認して変更する |
| [テナントのクォータ](system-tenant-quota.md) | `/system/tenants/$realm/quota` | テナントの上限と使用量を確認し上限を変更する |

## 設定

| ページ | URL | 目的 |
| --- | --- | --- |
| [テナント設定](admin-settings.md) | `/admin/settings` | テナントの共通設定と通知・表示を管理する |
| [ユーザー属性の定義](admin-tenant-attributes.md) | `/admin/tenant/attributes` | ユーザーのカスタム属性の型と公開範囲を管理する |
| [ユーザー属性を追加](admin-tenant-attribute-create.md) | `/admin/tenant/attributes/new` | 新しいカスタム属性を登録する |
| [グループ属性の定義](admin-tenant-group-attributes.md) | `/admin/tenant/group_attributes` | グループのカスタム属性を管理する |
| [グループ属性を追加](admin-group-attribute-create.md) | `/admin/tenant/group_attributes/new` | 新しいグループ属性を登録する |
