# SigningKeys のページ

この文書は、SigningKeys が主担当となるページの設計を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[SigningKeys のモジュール設計](../README.md) / [UI 全体の索引](../../../design/application/pages/README.md)

## 鍵

| ページ | URL | 目的 |
| --- | --- | --- |
| [署名鍵](admin-keys.md) | `/admin/keys` | テナントの JWT 署名鍵と更新を管理する |
| [署名鍵の稼働状態](system-key-health.md) | `/system/keys` | 全テナントの署名鍵の状態を確認する |
| [XML フェデレーションの署名鍵](admin-xml-federation-key.md) | `/admin/keys/xml-federation` | 信頼先の更新を確認して共有 XML 署名鍵を切り替える |
