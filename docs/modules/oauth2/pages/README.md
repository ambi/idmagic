# OAuth2 のページ

この文書は、OAuth2 が主担当となるページの設計を示す。

> 表示確認用の途中プレビュー。全ページの棚卸しと検証は継続中である。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[OAuth2 のモジュール設計](../README.md) / [UI 全体の索引](../../../design/application/pages/README.md)

## 同意と承認

| ページ | URL | 目的 |
| --- | --- | --- |
| [アプリケーションへの同意](consent.md) | `/consent` | 開示する情報と認可詳細を確認して許可または拒否する |
| [デバイスの認可](device.md) | `/device` | デバイスのコードを指定して認可を許可または拒否する |
| [本人の同意と Agent への委任](account-applications.md) | `/account/applications` | アプリケーションへの同意を取り消し、Agent の委任を管理する |
| [本人宛ての承認依頼](account-approvals.md) | `/account/approvals` | Agent の実行や認可の依頼を確認して承認または拒否する |
