# WsFederation のページ

この文書は、WsFederation が主担当となるページの設計を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[WsFederation のモジュール設計](../README.md) / [UI 全体の索引](../../../design/application/pages/README.md)

## フェデレーション

| ページ | URL | 目的 |
| --- | --- | --- |
| [Entra フェデレーション](admin-entra-federation.md) | `/admin/federation/entra` | ドメインの信頼状態と署名鍵の適用状況を確認する |
| [Entra ドメインを追加](admin-entra-federation-add.md) | `/admin/federation/entra/new` | ドメインの信頼を登録する |
| [Entra ドメインの詳細](admin-entra-federation-domain.md) | `/admin/federation/entra/$domain` | ドメインの設定と外部で観測した信頼情報を照合する |
| [Entra ドメインを編集](admin-entra-federation-edit.md) | `/admin/federation/entra/$domain/edit` | ドメインの信頼設定を更新する |
