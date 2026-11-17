# UI ページ

この文書は、利用者の作業で分類したページ設計の入口を示す。

> 表示確認用の途中プレビュー。全ページの棚卸しと検証は継続中である。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

## 主担当のモジュール

| モジュール | ページ数 |
| --- | --- |
| [Application](../../../modules/application/pages/README.md) | 1 |
| [Authentication](../../../modules/authentication/pages/README.md) | 8 |
| [Frontend](../../../modules/frontend/pages/README.md) | 5 |
| [IdManagement](../../../modules/identity-management/pages/README.md) | 10 |
| [OAuth2](../../../modules/oauth2/pages/README.md) | 4 |
| [TokenVault](../../../modules/token-vault/pages/README.md) | 1 |

## 領域

| 領域 | カテゴリ | ページ数 |
| --- | --- | --- |
| [認証](authentication/README.md) | サインイン、アカウント復旧、入口と共通表示、プロフィール、同意と承認 | 11 |
| [アカウントポータル](account/README.md) | アプリケーション、セキュリティ、入口と共通表示、プロフィール、同意と承認、外部サービス接続 | 12 |
| [管理コンソール](admin/README.md) | 入口と共通表示、ユーザー管理 | 6 |

## 共通の設計

- [UI の共通規則](../user-interface.md)
- [フロントエンド設計](../frontend.md)
- [共通表示と外部の到達先](../../../modules/frontend/pages/shared.md)
