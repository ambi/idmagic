import { defineDictionary } from '../../lib/i18n'

export const adminEntraFederationDictionary = defineDictionary(
  {
    federationMetadataLabel: 'Federation metadata',
    wsTrustMexLabel: 'WS-Trust MEX',
    wsTrustUsernamemixedLabel: 'WS-Trust usernamemixed',
    federationDeletedNotice: '{domain} のフェデレーションを削除しました。',
    deleteFailedError: '削除に失敗しました。',
    pageTitle: 'Entra ドメインフェデレーション',
    pageDescription:
      '検証済みドメインを Microsoft Entra から本 IdP へフェデレーションします。ドメインごとに UPN / ImmutableID / persistent NameID の claim preset を持つ relying party を作成します。',
    addDomainFederation: 'ドメインフェデレーションを追加',
    noFederationsNotice:
      'まだフェデレーション済みのドメインがありません。「ドメインフェデレーションを追加」から作成してください。',
    sourceAnchorPrefix: 'sourceAnchor: ',
    deleteAriaLabel: '{wtrealm} を削除',
    federationSavedNotice: '{domain} のドメインフェデレーションを保存しました。',
    federationSaveFailedError: 'Entra federation の保存に失敗しました。',
    addPageTitle: 'ドメインフェデレーションを追加',
    addPageDescription:
      'Microsoft 365 のドメインフェデレーション向けに、検証済みドメインごとに claim preset を作成します。',
    federationList: 'フェデレーション一覧',
    verifiedDomainLabel: '検証済み domain',
    sourceAnchorAttributeLabel: 'sourceAnchor 属性',
    issuerUriLabel: 'IssuerUri',
    issuerUriPlaceholder: '空なら自動生成',
    replyUrlLabel: 'wreply URL',
    save: '保存',
    hybridJoinNotProvidedNotice:
      'Hybrid Azure AD Join のデバイス登録は未提供です。必要な場合は managed/PHS への切替または AD FS 併存を検討してください。',
    backToFederationList: 'フェデレーション一覧へ戻る',
  },
  {
    federationMetadataLabel: 'Federation metadata',
    wsTrustMexLabel: 'WS-Trust MEX',
    wsTrustUsernamemixedLabel: 'WS-Trust usernamemixed',
    federationDeletedNotice: 'Deleted the federation for {domain}.',
    deleteFailedError: 'Failed to delete.',
    pageTitle: 'Entra domain federation',
    pageDescription:
      'Federate verified domains from Microsoft Entra to this IdP. Creates a relying party per domain with a UPN / ImmutableID / persistent NameID claim preset.',
    addDomainFederation: 'Add domain federation',
    noFederationsNotice:
      'There are no federated domains yet. Create one from "Add domain federation."',
    sourceAnchorPrefix: 'sourceAnchor: ',
    deleteAriaLabel: 'Delete {wtrealm}',
    federationSavedNotice: 'Saved the domain federation for {domain}.',
    federationSaveFailedError: 'Failed to save the Entra federation.',
    addPageTitle: 'Add domain federation',
    addPageDescription:
      'Creates a claim preset per verified domain for Microsoft 365 domain federation.',
    federationList: 'Federation list',
    verifiedDomainLabel: 'Verified domain',
    sourceAnchorAttributeLabel: 'sourceAnchor attribute',
    issuerUriLabel: 'IssuerUri',
    issuerUriPlaceholder: 'Auto-generated if blank',
    replyUrlLabel: 'wreply URL',
    save: 'Save',
    hybridJoinNotProvidedNotice:
      'Hybrid Azure AD Join device registration is not provided. If needed, consider switching to managed/PHS or keeping AD FS alongside.',
    backToFederationList: 'Back to federation list',
  },
)

export type AdminEntraFederationDictionary = (typeof adminEntraFederationDictionary)['ja']
