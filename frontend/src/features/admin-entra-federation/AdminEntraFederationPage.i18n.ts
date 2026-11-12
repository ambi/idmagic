import { defineDictionary } from '../../lib/i18n'

export const adminEntraFederationDictionary = defineDictionary(
  {
    federationMetadataLabel: 'Federation metadata',
    wsTrustMexLabel: 'WS-Trust MEX',
    wsTrustUsernamemixedLabel: 'WS-Trust usernamemixed',
    pageTitle: 'Entra ドメインフェデレーション',
    pageDescription:
      '検証済みドメインを Microsoft Entra から本 IdP へフェデレーションします。UPN / ImmutableID / persistent NameID の claim preset を持つ Entra の relying party を一つ作り、ドメインごとに IssuerUri を持ちます。',
    addDomainFederation: 'ドメインフェデレーションを追加',
    rolloverSigningKey: '署名鍵を更新',
    noFederationsNotice:
      'まだ構成したドメインがありません。「ドメインフェデレーションを追加」から作成してください。',
    sourceAnchorPrefix: 'sourceAnchor: ',
    issuerUriPrefix: 'IssuerUri: ',
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
    sourceAnchorAttributeLabel: 'IdMagic の保存先の属性名',
    sourceAnchorAdvancedSettings: 'ImmutableID の詳細設定',
    sourceAnchorAdvancedHint:
      '既存の別属性を使う構成だけで変更してください。AD の属性名やユーザーごとの識別子の値を入力する欄ではありません。一意で、本人が編集できない属性が必要です。',
    immutableIdStorageLabel: 'IdMagic の ImmutableID の保存先',
    immutableIdHeading: 'Entra のユーザー識別子（ImmutableID）',
    immutableIdHint:
      'Entra Connect が同期した値と同じ値が必要です。通常は組み込みの「オンプレミスの不変 ID」（on_premises_immutable_id）を使います。',
    immutableIdConnectStep:
      'Entra Connect の担当者が「現在の構成を表示」（View current configuration）の SOURCE ANCHOR を確認し、objectGUID と mS-DS-ConsistencyGuid のどちらかを確かめてください。新しい導入でも以前の設定を引き継ぐため、分からない場合は推測せず、担当者に確認してください。',
    immutableIdMappingStep:
      'Connector の ldap.attributes.on_premises_immutable_id を同じ AD 属性に合わせてください。objectGUID は既定値です。mS-DS-ConsistencyGuid は明示的に指定してください。',
    immutableIdResyncStep:
      'Connector の設定を確認して再起動し、完全な再同期の完了を待ってください。IdMagic のユーザー詳細で、保存先の属性の実値を確認してください。',
    immutableIdCompareStep:
      'Microsoft Graph PowerShell に User.Read.All で対象テナントへ接続し、Get-MgUser -UserId <対象の UPN> -Property userPrincipalName,onPremisesImmutableId で取得した OnPremisesImmutableId と、同じユーザーの保存値を大文字と小文字を含めて比較してください。',
    immutableIdEncodingHint:
      'GUID は生バイトの Base64 を使います。GUID の表示文字列や、その文字列の Base64、Base64 の二重変換とは異なります。二つの AD 属性が同じ値の場合もあるため、属性名だけで一致と判断しないでください。',
    immutableIdConnectReference: 'Entra Connect の sourceAnchor の確認資料',
    sourceAnchorRepairHint:
      '対象ユーザーの IdMagic の保存先「{attribute}」を確認してください。取り込んだユーザーは Connector の ldap.attributes の対応付けと AD の元の値を確認し、完全な再同期の後に Entra の実値と比較してください。',
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
    removalHeading: 'ドメインのフェデレーションの解除',
    removalNotice:
      '解除しても Entra の設定は変わりません。先に Entra 側でこのドメインを次のどちらかの方法で切り戻してから解除してください。解除するとき、IdMagic は Microsoft に問い合わせ、Entra がまだこのドメインを IdMagic へ送っていれば解除しません。',
    removalRemainingDomains: '解除した後も {domains} のサインインは続きます。',
    removalLastDomain:
      'このドメインは最後のドメインです。解除すると Entra の relying party も削除します。署名鍵は残ります。',
    removalTargetManaged: 'Managed（Microsoft Entra の認証）に切り替える',
    removalTargetPreviousIdp: '変更前に保存した以前の IdP の構成に戻す',
    removalManagedNotice:
      'Managed にする前に、パスワードハッシュ同期かパススルー認証を準備し、クラウドでサインインできる管理者を確保してください。変換には最大 60 分かかります。',
    removalPreviousIdpNotice:
      '「2. 変更前の構成を保存」が書き出したファイルにある以前の IdP の値で、「1. 接続」の後に Update-MgDomainFederationConfiguration を実行してください。IdMagic を適用した後に保存したファイルは IdMagic の値を持つので使えません。',
    procedureReleaseToManaged: 'Managed に切り替えるコマンド',
    removalMigrationGuide: 'Microsoft の移行の手順（federation から cloud authentication へ）',
    removeDomain: 'ドメインを解除',
    removalConfirmTitle: '{domain} のフェデレーションを解除',
    removalConfirmDescription:
      'IdMagic は Microsoft に問い合わせ、Entra がもうこのドメインを IdMagic へ送っていないことを確かめてから解除します。',
    removalRemovedNotice: '{domain} のフェデレーションを解除しました。',
    removalStillActiveError:
      'Microsoft によると、Entra はまだこのドメインの利用者を IdMagic へ送っています。Entra 側を切り戻してから解除してください。Managed への変換は反映に最大 60 分かかるので、切り戻した直後なら時間をおいて再実行してください。',
    removalUnconfirmedError:
      'Microsoft に問い合わせて切り戻しを確かめられませんでした。時間をおいて再実行するか、「Entra への反映の確認」で切り戻した後の読み戻しを取り込むと、その結果で解除できます。',
    removalFailedError: '解除に失敗しました。',
  },
  {
    federationMetadataLabel: 'Federation metadata',
    wsTrustMexLabel: 'WS-Trust MEX',
    wsTrustUsernamemixedLabel: 'WS-Trust usernamemixed',
    pageTitle: 'Entra domain federation',
    pageDescription:
      'Federate verified domains from Microsoft Entra to this IdP. Creates one Entra relying party with a UPN / ImmutableID / persistent NameID claim preset that holds an IssuerUri per domain.',
    addDomainFederation: 'Add domain federation',
    rolloverSigningKey: 'Update the signing key',
    noFederationsNotice: 'No domains are configured yet. Create one from "Add domain federation."',
    sourceAnchorPrefix: 'sourceAnchor: ',
    issuerUriPrefix: 'IssuerUri: ',
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
    sourceAnchorAttributeLabel: 'IdMagic storage attribute name',
    sourceAnchorAdvancedSettings: 'Advanced ImmutableID settings',
    sourceAnchorAdvancedHint:
      'Change this only for an existing configuration that uses another storage attribute. This field takes neither an AD attribute name nor an individual user identifier. The attribute must be unique and not editable by users.',
    immutableIdStorageLabel: 'IdMagic ImmutableID storage attribute',
    immutableIdHeading: 'Entra user identifier (ImmutableID)',
    immutableIdHint:
      'The value must match the value synced by Entra Connect. Normally, use the built-in On-premises immutable ID (on_premises_immutable_id).',
    immutableIdConnectStep:
      'Ask the Entra Connect administrator to open View current configuration and check SOURCE ANCHOR for objectGUID or mS-DS-ConsistencyGuid. A new installation can inherit an earlier configuration. If unknown, confirm with the administrator before choosing.',
    immutableIdMappingStep:
      'Map Connector ldap.attributes.on_premises_immutable_id to the same AD attribute. objectGUID is the default; specify mS-DS-ConsistencyGuid explicitly when used.',
    immutableIdResyncStep:
      'Check the Connector configuration, restart it, and wait for a full sync to finish. Check the stored value in the IdMagic user detail.',
    immutableIdCompareStep:
      'Connect Microsoft Graph PowerShell to the target tenant with User.Read.All. Run Get-MgUser -UserId <user UPN> -Property userPrincipalName,onPremisesImmutableId and compare OnPremisesImmutableId with the same user’s stored value, including case.',
    immutableIdEncodingHint:
      'Use Base64 of the raw GUID bytes, not the displayed GUID string, Base64 of that string, or a second Base64 encoding. The two AD attributes can hold the same value, so attribute names alone do not prove a match.',
    immutableIdConnectReference: 'Entra Connect sourceAnchor reference',
    sourceAnchorRepairHint:
      'Check the affected users’ IdMagic storage attribute “{attribute}”. For imported users, check the Connector ldap.attributes mapping and original AD values, run a full sync, and compare with the actual Entra values.',
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
    removalHeading: 'Remove the federation of this domain',
    removalNotice:
      'Removing does not change Entra. Roll the domain back in Entra with one of the methods below first. When you remove the domain, IdMagic asks Microsoft and does not remove it while Entra still sends the domain to IdMagic.',
    removalRemainingDomains: 'Sign-in keeps working for {domains} after the removal.',
    removalLastDomain:
      'This is the last domain. Removing it also deletes the Entra relying party. The signing key is kept.',
    removalTargetManaged: 'Switch to Managed (Microsoft Entra authentication)',
    removalTargetPreviousIdp: 'Restore the previous IdP configuration saved before the change',
    removalManagedNotice:
      'Before switching to Managed, prepare password hash sync or pass-through authentication and keep an administrator who can sign in to the cloud. The conversion can take up to 60 minutes.',
    removalPreviousIdpNotice:
      'After "1. Connect", run Update-MgDomainFederationConfiguration with the previous IdP values in the file written by "2. Save the current configuration". A file saved after IdMagic was applied holds the IdMagic values and cannot be used.',
    procedureReleaseToManaged: 'Command to switch to Managed',
    removalMigrationGuide: 'Microsoft migration guide (from federation to cloud authentication)',
    removeDomain: 'Remove domain',
    removalConfirmTitle: 'Remove the federation of {domain}',
    removalConfirmDescription:
      'IdMagic asks Microsoft and removes the domain only after confirming that Entra no longer sends it to IdMagic.',
    removalRemovedNotice: 'Removed the federation of {domain}.',
    removalStillActiveError:
      'According to Microsoft, Entra still sends the users of this domain to IdMagic. Roll the domain back in Entra before removing it. The conversion to Managed can take up to 60 minutes, so retry later if you have just rolled it back.',
    removalUnconfirmedError:
      'The rollback could not be confirmed with Microsoft. Retry later, or import a read-back taken after the rollback in "Verification in Entra" to remove the domain on its basis.',
    removalFailedError: 'Failed to remove the domain.',
  },
)

export type AdminEntraFederationDictionary = (typeof adminEntraFederationDictionary)['ja']
