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
      'まだ構成したドメインがありません。「ドメインフェデレーションを追加」から作成してください。',
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
    unfederatedSourceUsersNotice:
      '取り込んだユーザーのうち {count} 人は、UPN がフェデレーションしたドメインに属さないため、Entra にサインインできません。',
    addPageTitle: 'ドメインフェデレーションを追加',
    addPageDescription:
      'Microsoft 365 のドメインフェデレーションに、検証済みドメインとその IssuerUri を追加します。構成済みのドメインを指定すると IssuerUri を置き換えます。',
    federationList: 'フェデレーション一覧',
    verifiedDomainLabel: '検証済み domain',
    domainNotUrlError:
      'contoso.com のように、https:// を付けずにドメイン名だけを入力してください。',
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
      'Microsoft Entra hybrid join の即時の登録は提供しません。Windows 10 1803 以降の端末は、Microsoft Entra Connect がコンピューターを同期した後に登録されます。',
    backToFederationList: 'フェデレーション一覧へ戻る',
    entraTenantIdLabel: 'Entra テナント ID',
    entraTenantIdHint:
      '指定すると、手順はこのテナントにだけ接続し、取り込んだ登録値のテナントも照合します。',
    entraTenantIdInvalidError:
      '00000000-0000-0000-0000-000000000000 の形の GUID を入力してください。',
    cancel: 'キャンセル',
    edit: '編集',
    editPageTitle: '{domain} のフェデレーションを編集',
    editPageDescription:
      '保存すると IdMagic の設定だけが変わります。Entra への反映は、詳細の手順で適用して確認してください。',
    domainDetailDescription:
      'IdMagic に保存した設定と、Entra に登録すべき値、適用と確認の手順です。この画面は設定を変えません。',
    viewDomainDetail: '{domain} の詳細',
    statusUnverified: '未確認',
    statusMatched: '一致',
    statusMismatched: '不一致',
    statusStale: '再確認が必要',
    statusUnverifiedDescription: 'IdMagic に保存しただけで、Entra への反映はまだ確かめていません。',
    statusMatchedDescription: '取り込んだ Entra の登録値が、現在の期待値とすべて一致しました。',
    statusMismatchedDescription: '取り込んだ Entra の登録値に、期待値と異なる項目があります。',
    statusStaleDescription:
      '前回の確認の後に IdMagic の設定か署名鍵が変わりました。もう一度読み戻して取り込んでください。',
    verificationHeading: 'Entra への反映の確認',
    verificationManualNotice:
      '管理者が取り込んだ時点の登録値との照合です。その後の Entra の変更は監視しません。',
    retrievedAtLabel: 'Graph から取得',
    checkedAtLabel: '取り込んで照合',
    observedTenantLabel: '確認したテナント',
    differenceFieldLabel: '項目',
    differenceExpectedLabel: '期待値',
    differenceObservedLabel: '登録値',
    observationLabel: '読み戻しの結果（JSON）',
    observationFileLabel: 'ファイルから読み込む',
    observationInvalidJsonError:
      'JSON として読めません。読み戻しの手順が書き出したファイルの内容を貼り付けてください。',
    observationOutdatedError:
      '保存済みの確認より前に取得した結果です。最新の読み戻しの結果を取り込んでください。',
    observationRejectedError: '取り込めませんでした: {detail}',
    importObservation: '取り込んで照合',
    importingObservation: '照合しています…',
    observationRecordedNotice: 'Entra の登録値を取り込んで照合しました。',
    savedSettingsHeading: '保存した設定',
    expectedSettingsHeading: 'Entra に登録する値',
    certificatesHeading: '署名証明書',
    currentCertificateLabel: '現在の証明書',
    nextCertificateLabel: '次の証明書',
    thumbprintLabel: 'SHA-256 拇印',
    notAfterLabel: '有効期限',
    notSpecified: '指定なし',
    procedureHeading: 'Graph PowerShell の手順',
    procedureNotice:
      '上から順に実行します。適用は Entra の設定を変えるので、変更前の保存と差分の確認を先に行ってください。',
    procedureConnect: '1. 接続（適用用の権限）',
    procedureConnectReadOnly: '確認だけの接続（読み取りの権限）',
    procedureBackup: '2. 変更前の構成を保存',
    procedureCompare: '3. 差分を表示',
    procedureApply: '4. 作成または更新',
    procedureReadBack: '5. 読み戻して保存',
    copy: 'コピー',
    commandsHeading: '個別のコマンド',
    prerequisitesHeading: '導入の前提',
    prerequisiteAdDelegation:
      'AD への認証の委譲: Directory Connector がパスワードの照合を AD に委ねている。',
    prerequisiteUpnImmutableId:
      'UPN と ImmutableID: 利用者の UPN がこのドメインに属し、sourceAnchor が Entra の onPremisesImmutableId と同じ値である。',
    prerequisiteConnectComputers:
      'Entra Connect のコンピューターの同期: 対象の OU のコンピューターとデバイスの属性を同期している。',
    prerequisiteScp: 'SCP: AD のサービス接続ポイントが Entra のテナントを指している。',
    prerequisiteSystemNetwork:
      'SYSTEM の通信: 端末の SYSTEM アカウントがプロキシ越しに Microsoft Entra の登録とサインインの URL へ届く。',
    postChecksHeading: '構成の後の確認',
    postCheckSyncJoin:
      'sync join の同期待ち: Entra Connect がコンピューターの userCertificate を同期するまで、最初の登録は失敗する。',
    postCheckDeviceJoin: '端末の参加: dsregcmd /status で AzureAdJoined が YES になる。',
    postCheckPrt: 'PRT: dsregcmd /status で AzureAdPrt が YES になる。',
    postCheckAppSso:
      'アプリの SSO: 参加した端末から Microsoft 365 のアプリへ追加のサインインなしで入れる。',
    postChecksNotice: 'これらは端末と Entra の状態であり、この画面は自動では確かめません。',
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
    noFederationsNotice: 'No domains are configured yet. Create one from "Add domain federation."',
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
    unfederatedSourceUsersNotice:
      '{count} ingested users cannot sign in to Entra because their UPN is not in a federated domain.',
    addPageTitle: 'Add domain federation',
    addPageDescription:
      'Adds a verified domain and its IssuerUri to the Microsoft 365 domain federation. A configured domain has its IssuerUri replaced.',
    federationList: 'Federation list',
    verifiedDomainLabel: 'Verified domain',
    domainNotUrlError: 'Enter only the domain name without https://, such as contoso.com.',
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
      'Immediate Microsoft Entra hybrid join registration is not provided. Devices running Windows 10 1803 or later register after Microsoft Entra Connect syncs their computer objects.',
    backToFederationList: 'Back to federation list',
    entraTenantIdLabel: 'Entra tenant ID',
    entraTenantIdHint:
      'When set, the procedure connects only to this tenant, and imported values are checked against it.',
    entraTenantIdInvalidError: 'Enter a GUID such as 00000000-0000-0000-0000-000000000000.',
    cancel: 'Cancel',
    edit: 'Edit',
    editPageTitle: 'Edit the federation for {domain}',
    editPageDescription:
      'Saving changes only the IdMagic settings. Apply and verify them in Entra with the procedure on the detail page.',
    domainDetailDescription:
      'The settings saved in IdMagic, the values to register in Entra, and the procedure to apply and verify them. This page does not change settings.',
    viewDomainDetail: 'Details of {domain}',
    statusUnverified: 'Unverified',
    statusMatched: 'Matched',
    statusMismatched: 'Mismatched',
    statusStale: 'Needs re-verification',
    statusUnverifiedDescription: 'Saved in IdMagic only. Entra has not been checked yet.',
    statusMatchedDescription: 'The imported Entra values match all current expected values.',
    statusMismatchedDescription: 'Some imported Entra values differ from the expected values.',
    statusStaleDescription:
      'The IdMagic settings or signing key changed after the last check. Read the values back and import them again.',
    verificationHeading: 'Verification in Entra',
    verificationManualNotice:
      'This compares the values an administrator imported at one point in time. Later changes in Entra are not monitored.',
    retrievedAtLabel: 'Retrieved from Graph',
    checkedAtLabel: 'Imported and checked',
    observedTenantLabel: 'Checked tenant',
    differenceFieldLabel: 'Setting',
    differenceExpectedLabel: 'Expected',
    differenceObservedLabel: 'Registered',
    observationLabel: 'Read-back result (JSON)',
    observationFileLabel: 'Load from file',
    observationInvalidJsonError:
      'This is not valid JSON. Paste the contents of the file that the read-back step wrote.',
    observationOutdatedError:
      'This result was retrieved before the stored check. Import the latest read-back result.',
    observationRejectedError: 'Could not import: {detail}',
    importObservation: 'Import and check',
    importingObservation: 'Checking…',
    observationRecordedNotice: 'Imported and checked the Entra values.',
    savedSettingsHeading: 'Saved settings',
    expectedSettingsHeading: 'Values to register in Entra',
    certificatesHeading: 'Signing certificates',
    currentCertificateLabel: 'Current certificate',
    nextCertificateLabel: 'Next certificate',
    thumbprintLabel: 'SHA-256 thumbprint',
    notAfterLabel: 'Expires',
    notSpecified: 'Not specified',
    procedureHeading: 'Graph PowerShell procedure',
    procedureNotice:
      'Run the steps in order. Applying changes Entra, so save the current configuration and review the differences first.',
    procedureConnect: '1. Connect (permissions to apply)',
    procedureConnectReadOnly: 'Connect to check only (read permissions)',
    procedureBackup: '2. Save the current configuration',
    procedureCompare: '3. Show the differences',
    procedureApply: '4. Create or update',
    procedureReadBack: '5. Read back and save',
    copy: 'Copy',
    commandsHeading: 'Individual commands',
    prerequisitesHeading: 'Prerequisites',
    prerequisiteAdDelegation:
      'Authentication delegated to AD: Directory Connector delegates password verification to AD.',
    prerequisiteUpnImmutableId:
      "UPN and ImmutableID: users' UPNs belong to this domain, and the sourceAnchor equals Entra's onPremisesImmutableId.",
    prerequisiteConnectComputers:
      'Computer sync in Entra Connect: computers in the target OUs and their device attributes are synced.',
    prerequisiteScp: 'SCP: the service connection point in AD points to the Entra tenant.',
    prerequisiteSystemNetwork:
      "SYSTEM connectivity: the device's SYSTEM account reaches the Microsoft Entra registration and sign-in URLs through the proxy.",
    postChecksHeading: 'Checks after configuration',
    postCheckSyncJoin:
      "Wait for sync join: the first registration fails until Entra Connect syncs the computer's userCertificate.",
    postCheckDeviceJoin: 'Device join: dsregcmd /status shows AzureAdJoined : YES.',
    postCheckPrt: 'PRT: dsregcmd /status shows AzureAdPrt : YES.',
    postCheckAppSso: 'App SSO: Microsoft 365 apps open on a joined device without another sign-in.',
    postChecksNotice:
      'These are device and Entra states. This page does not check them automatically.',
  },
)

export type AdminEntraFederationDictionary = (typeof adminEntraFederationDictionary)['ja']
