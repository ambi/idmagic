# サインイン

この文書は、サインインのページ一覧を示す。

> 表示確認用の途中プレビュー。全ページの棚卸しと検証は継続中である。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[UI ページ](../../README.md) / [認証](../README.md)

## ページ一覧

| ページ | URL | 目的 |
| --- | --- | --- |
| [ログイン](../../../../../modules/authentication/pages/login.md) | `/login` | 資格情報または既存の認証手段で認証を開始する |
| [二要素認証](../../../../../modules/authentication/pages/totp.md) | `/totp` | 利用可能な認証手段で第二要素を確認する |
| [認可フロー内の MFA 登録](../../../../../modules/authentication/pages/mfa-enrollment.md) | `/mfa-enrollment` | ログインを継続する前に TOTP を登録する |
