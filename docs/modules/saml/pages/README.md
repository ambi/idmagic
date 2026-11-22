# Saml のページ

この文書は、Saml が主担当となるページの設計を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Saml のモジュール設計](../README.md) / [UI 全体の索引](../../../design/application/pages/README.md)

## フェデレーション

| ページ | URL | 目的 |
| --- | --- | --- |
| [SAML IdP プロファイル](admin-saml-idp-profiles.md) | `/admin/settings/saml-idp-profiles` | 共有または専用の IdP 識別子を管理する |
| [SAML IdP プロファイルを作成](admin-saml-idp-profile-create.md) | `/admin/settings/saml-idp-profiles/new` | 名前と共有方式を指定してプロファイルを作る |
| [SAML IdP プロファイルの詳細](admin-saml-idp-profile-detail.md) | `/admin/settings/saml-idp-profiles/$profileId` | 識別子、接続先と利用状態を確認する |
| [SAML IdP プロファイルを編集](admin-saml-idp-profile-edit.md) | `/admin/settings/saml-idp-profiles/$profileId/edit` | プロファイルの名前を変更する |
