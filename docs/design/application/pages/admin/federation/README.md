# フェデレーション

この文書は、フェデレーションのページ一覧を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[UI ページ](../../README.md) / [管理コンソール](../README.md)

## ページ一覧

| ページ | URL | 目的 |
| --- | --- | --- |
| [外部 IdP の一覧](../../../../../modules/authentication/pages/admin-identity-providers.md) | `/admin/identity-providers` | 外部認証の接続状態と設定を管理する |
| [外部 IdP の詳細](../../../../../modules/authentication/pages/admin-identity-provider-detail.md) | `/admin/identity-providers/$id` | 認証先の接続、マッピングと状態を確認する |
| [外部 IdP を登録](../../../../../modules/authentication/pages/admin-identity-provider-create.md) | `/admin/identity-providers/new` | 接続方式、認証先と属性マッピングを登録する |
| [外部 IdP を編集](../../../../../modules/authentication/pages/admin-identity-provider-edit.md) | `/admin/identity-providers/$id/edit` | 接続と属性マッピングを変更し試す |
| [SAML IdP プロファイル](../../../../../modules/saml/pages/admin-saml-idp-profiles.md) | `/admin/settings/saml-idp-profiles` | 共有または専用の IdP 識別子を管理する |
| [SAML IdP プロファイルを作成](../../../../../modules/saml/pages/admin-saml-idp-profile-create.md) | `/admin/settings/saml-idp-profiles/new` | 名前と共有方式を指定してプロファイルを作る |
| [SAML IdP プロファイルの詳細](../../../../../modules/saml/pages/admin-saml-idp-profile-detail.md) | `/admin/settings/saml-idp-profiles/$profileId` | 識別子、接続先と利用状態を確認する |
| [SAML IdP プロファイルを編集](../../../../../modules/saml/pages/admin-saml-idp-profile-edit.md) | `/admin/settings/saml-idp-profiles/$profileId/edit` | プロファイルの名前を変更する |
| [Entra フェデレーション](../../../../../modules/ws-federation/pages/admin-entra-federation.md) | `/admin/federation/entra` | ドメインの信頼状態と署名鍵の適用状況を確認する |
| [Entra ドメインを追加](../../../../../modules/ws-federation/pages/admin-entra-federation-add.md) | `/admin/federation/entra/new` | ドメインの信頼を登録する |
| [Entra ドメインの詳細](../../../../../modules/ws-federation/pages/admin-entra-federation-domain.md) | `/admin/federation/entra/$domain` | ドメインの設定と外部で観測した信頼情報を照合する |
| [Entra ドメインを編集](../../../../../modules/ws-federation/pages/admin-entra-federation-edit.md) | `/admin/federation/entra/$domain/edit` | ドメインの信頼設定を更新する |
