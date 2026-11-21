# Frontend のページ

この文書は、Frontend が主担当となるページの設計を示す。

> 表示確認用の途中プレビュー。全ページの棚卸しと検証は継続中である。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Frontend のモジュール設計](../README.md) / [UI 全体の索引](../../../design/application/pages/README.md)

## 入口と共通表示

| ページ | URL | 目的 |
| --- | --- | --- |
| [サービスの入口](home.md) | `/` | アカウントと管理コンソールの入口を示す |
| [アカウントの概要](account-home.md) | `/account` | 本人の登録状態と各セルフサービスの入口を示す |
| [管理ダッシュボード](admin-dashboard.md) | `/admin` | テナントの概要と管理操作への入口を示す |
| [認証コールバック](callback.md) | `/callback` | ポータルの認証応答を処理し、処理できない応答の結果を表示する |
| [認証と承認の結果](status.md) | `/status` | 操作が終わったことと次の入口を案内する |

## 共通表示

[共通表示と外部の到達先](shared.md)
