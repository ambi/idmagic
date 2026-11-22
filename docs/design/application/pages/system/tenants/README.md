# テナント

この文書は、テナントのページ一覧を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[UI ページ](../../README.md) / [システムコンソール](../README.md)

## ページ一覧

| ページ | URL | 目的 |
| --- | --- | --- |
| [テナント一覧](../../../../../modules/tenancy/pages/system-tenants.md) | `/system/tenants` | テナントの状態と管理の入口を確認する |
| [テナントを作成](../../../../../modules/tenancy/pages/system-tenant-create.md) | `/system/new-tenant` | テナントの識別子と初期設定を登録する |
| [テナント詳細](../../../../../modules/tenancy/pages/system-tenant-detail.md) | `/system/tenants/$realm` | 状態、入口の方式とクォータを確認する |
| [テナントの基本情報を編集](../../../../../modules/tenancy/pages/system-tenant-edit.md) | `/system/tenants/$realm/edit` | 表示情報と基本設定を変更する |
| [テナントの入口方式](../../../../../modules/tenancy/pages/system-tenant-endpoint-style.md) | `/system/tenants/$realm/endpoint-style` | テナントの URL 方式を影響を確認して変更する |
| [テナントのクォータ](../../../../../modules/tenancy/pages/system-tenant-quota.md) | `/system/tenants/$realm/quota` | テナントの上限と使用量を確認し上限を変更する |
