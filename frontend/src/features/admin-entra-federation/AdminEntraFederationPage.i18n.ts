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
      '検証済みドメインを Microsoft Entra から本 IdP へフェデレーションします。UPN / ImmutableID / persistent NameID の claim preset を持つ Entra の relying party を一つ作り、ドメインごとに IssuerUri を持ちます。',
    addDomainFederation: 'ドメインフェデレーションを追加',
    noFederationsNotice:
      'まだフェデレーション済みのドメインがありません。「ドメインフェデレーションを追加」から作成してください。',
    sourceAnchorPrefix: 'sourceAnchor: ',
    issuerUriPrefix: 'IssuerUri: ',
    deleteAriaLabel: '{wtrealm} を削除',
    federationSavedNotice: '{domain} のドメインフェデレーションを保存しました。',
    federationSaveFailedError: 'Entra federation の保存に失敗しました。',
    sourceAnchorNotUniqueError: 'sourceAnchor 属性を一意の属性に設定してください。',
    sourceAnchorEditableError: 'sourceAnchor 属性を本人が編集できない設定にしてください。',
    sourceAnchorMissingError: 'sourceAnchor の値がないユーザーが {count} 人います。',
    sourceAnchorInvalidError:
      'sourceAnchor を ImmutableID として使えないユーザーが {count} 人います。空白だけの値や @ を含む値を修正してください。',
    sourceAnchorSampleUsers: '対象のユーザー（先頭の 5 人まで）: {users}',
    addPageTitle: 'ドメインフェデレーションを追加',
    addPageDescription:
      'Microsoft 365 のドメインフェデレーションに、検証済みドメインとその IssuerUri を追加します。構成済みのドメインを指定すると IssuerUri を置き換えます。',
    federationList: 'フェデレーション一覧',
    verifiedDomainLabel: '検証済み domain',
    sourceAnchorAttributeLabel: 'sourceAnchor 属性',
    issuerUriLabel: 'IssuerUri',
    issuerUriPlaceholder: '空なら自動生成',
    replyUrlLabel: 'wreply URL',
    save: '保存',
    graphCommandsNotice:
      'Microsoft Graph PowerShell に接続し、新規の構成には作成コマンド、既存の構成には更新コマンドを実行してください。',
    createCommandLabel: '作成コマンド',
    updateCommandLabel: '更新コマンド',
    copyCreateCommand: '作成コマンドをコピー',
    copyUpdateCommand: '更新コマンドをコピー',
    commandCopiedNotice: 'コマンドをコピーしました。',
    commandCopyFailedError:
      'コピーできませんでした。表示したコマンドを選択してコピーしてください。',
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
      'Federate verified domains from Microsoft Entra to this IdP. Creates one Entra relying party with a UPN / ImmutableID / persistent NameID claim preset that holds an IssuerUri per domain.',
    addDomainFederation: 'Add domain federation',
    noFederationsNotice:
      'There are no federated domains yet. Create one from "Add domain federation."',
    sourceAnchorPrefix: 'sourceAnchor: ',
    issuerUriPrefix: 'IssuerUri: ',
    deleteAriaLabel: 'Delete {wtrealm}',
    federationSavedNotice: 'Saved the domain federation for {domain}.',
    federationSaveFailedError: 'Failed to save the Entra federation.',
    sourceAnchorNotUniqueError: 'Configure the sourceAnchor attribute as unique.',
    sourceAnchorEditableError: 'Configure the sourceAnchor attribute so users cannot edit it.',
    sourceAnchorMissingError: '{count} users have no sourceAnchor value.',
    sourceAnchorInvalidError:
      '{count} users have a sourceAnchor that cannot be an ImmutableID. Correct values that contain only whitespace or include @.',
    sourceAnchorSampleUsers: 'Affected users (first 5): {users}',
    addPageTitle: 'Add domain federation',
    addPageDescription:
      'Adds a verified domain and its IssuerUri to the Microsoft 365 domain federation. A configured domain has its IssuerUri replaced.',
    federationList: 'Federation list',
    verifiedDomainLabel: 'Verified domain',
    sourceAnchorAttributeLabel: 'sourceAnchor attribute',
    issuerUriLabel: 'IssuerUri',
    issuerUriPlaceholder: 'Auto-generated if blank',
    replyUrlLabel: 'wreply URL',
    save: 'Save',
    graphCommandsNotice:
      'Connect to Microsoft Graph PowerShell, then run the create command for a new configuration or the update command for an existing configuration.',
    createCommandLabel: 'Create command',
    updateCommandLabel: 'Update command',
    copyCreateCommand: 'Copy create command',
    copyUpdateCommand: 'Copy update command',
    commandCopiedNotice: 'Copied the command.',
    commandCopyFailedError: 'Could not copy. Select the displayed command and copy it manually.',
    hybridJoinNotProvidedNotice:
      'Hybrid Azure AD Join device registration is not provided. If needed, consider switching to managed/PHS or keeping AD FS alongside.',
    backToFederationList: 'Back to federation list',
  },
)

export type AdminEntraFederationDictionary = (typeof adminEntraFederationDictionary)['ja']
