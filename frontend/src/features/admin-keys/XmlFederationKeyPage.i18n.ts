import { defineDictionary } from '../../lib/i18n'

export const xmlFederationKeyDictionary = defineDictionary(
  {
    pageTitle: '共有 XML 署名鍵の更新',
    pageDescription:
      'WS-Federation、Entra のドメインフェデレーション、デフォルトの SAML IdP が共有する XML 署名鍵を、次の証明書の準備、信頼先への登録、確認付きの切り替えの順に更新します。',
    signingKeysLink: '署名鍵',
    entraLink: 'Entra ドメインフェデレーション',
    certificatesHeading: '証明書',
    kidLabel: 'Kid',
    stateLabel: '状態',
    fingerprintLabel: 'SHA-256 フィンガープリント',
    createdAtLabel: '生成',
    expiresAtLabel: '公開の期限',
    stateActive: '現在の署名鍵',
    statePrepared: '準備済み（署名に未使用）',
    stateVerifying: '旧鍵（検証用に公開中）',
    noKeysNotice: 'XML 署名鍵はまだありません。最初の署名のときに作られます。',
    impactHeading: '同じ鍵を信頼する連携先',
    impactNotice:
      '切り替えると、次のすべての連携先へ新しい鍵で署名します。Entra の照合が一致しても、ほかの連携先の更新が済んだことにはなりません。',
    impactEntra: 'Entra のドメイン',
    impactWsFed: 'WS-Federation の Relying Party',
    impactSaml: 'SAML のサービスプロバイダー（デフォルトの IdP プロファイル）',
    impactNone: 'ありません',
    prepareHeading: '1. 次の証明書を準備',
    prepareNotice:
      '準備した証明書はメタデータに公開されますが、署名には現在の鍵を使い続けます。この画面を離れても準備済みの鍵は残り、再び開くと同じ鍵から続けられます。',
    prepare: '次の証明書を準備',
    preparing: '準備しています…',
    preparedNotice: '次の証明書を準備しました。署名には現在の鍵を使い続けます。',
    preparedKeyLabel: '準備済みの鍵',
    cancelPrepared: '準備を取り消す',
    cancelConfirmTitle: '準備済みの鍵を取り消します',
    cancelConfirmDescription:
      'この鍵をメタデータから外します。署名に使う鍵は変わりません。取り消した鍵は元に戻せません。Entra に NextSigningCertificate として登録した場合は、取り消した後に表示される更新のコマンドで外してください。',
    cancelledNotice: '準備済みの鍵 {kid} を取り消しました。',
    registerHeading: '2. 連携先へ次の証明書を登録',
    registerEntraNotice:
      'ドメインごとに更新のコマンドを実行し、ドメインの詳細で読み戻しを取り込んで照合してください。照合は、準備済みの証明書を NextSigningCertificate として確認したときだけ一致になります。',
    openDomain: 'ドメインの詳細で照合する',
    copyUpdateCommand: '{domain} の更新のコマンドをコピー',
    entraReady: '準備済みの証明書を確認済み',
    entraNotReady: '準備済みの証明書を未確認',
    registerTrustNotice:
      'ほかの連携先は IdMagic から登録を確かめられません。それぞれの連携先の管理画面に次の証明書を登録し、登録したものを選んでください。',
    nextCertificateLabel: '次の証明書（Base64、公開情報）',
    copyNextCertificate: '次の証明書をコピー',
    trustRegistered: '{name} に次の証明書を登録した',
    activateHeading: '3. 確認して切り替え',
    activateNotice:
      '切り替えると新しい鍵で署名し、旧鍵は 7 日間だけ検証用に公開します。確認はこの画面を開くたびにやり直します。',
    overallConfirm: 'この鍵を信頼するすべての連携先で、次の証明書の登録を確認した',
    blockersHeading: '切り替えの前に必要なこと',
    blockerNoPrepared: '次の証明書を準備してください。',
    blockerEntra: '{domain} の照合で、準備済みの証明書が一致していません。',
    blockerTrust: '{name} への登録を確認してください。',
    blockerOverall: 'すべての連携先の確認を選んでください。',
    activate: '新しい鍵に切り替える',
    activating: '切り替えています…',
    activateConfirmTitle: '共有 XML 署名鍵を切り替えます',
    activateConfirmDescription:
      '以後の Assertion とトークンは新しい鍵で署名します。旧鍵での署名へ戻す操作はありません。旧鍵の証明書は 7 日間だけメタデータに残ります。',
    activatedNotice: '新しい鍵 {kid} に切り替えました。',
    changedSinceConfirmation:
      '確認の後に、準備済みの鍵、Entra の照合、または連携先が変わりました。最新の状態で確認し直してください。',
    activationRejected:
      '準備済みの鍵が変わったため、切り替えられませんでした。最新の状態で確認し直してください。',
    actionFailed: '署名鍵の操作を完了できませんでした。',
    afterHeading: '切り替えの後の確認',
    afterNotice: '切り替えの後は、旧鍵の公開の期限までに次を済ませてください。',
    afterEntra:
      'Entra のドメインごとに更新のコマンドを再び実行し、SigningCertificate を新しい証明書にして、読み戻しを取り込む。',
    afterSignIn: '各ドメインと連携先で、新しくサインインできることを確かめる。',
    afterTrusts: 'Entra 以外の連携先で、現在の署名証明書を新しい証明書に置き換える。',
    recoveryHeading: '失敗したときの復旧',
    recoveryBefore:
      '切り替えの前：準備を取り消せば、どの連携先にも影響しません。Entra に登録した NextSigningCertificate は、取り消した後の更新のコマンドで外します。',
    recoveryAfter:
      '切り替えの後：旧鍵で署名し直す操作はありません。サインインに失敗する連携先には、新しい証明書を登録して直してください。旧鍵の証明書は公開の期限まではメタデータに残ります。',
    recoveryDeadline: '旧鍵 {kid} の公開の期限：{date}',
  },
  {
    pageTitle: 'Update the shared XML signing key',
    pageDescription:
      'Update the XML signing key that WS-Federation, Entra domain federation, and the default SAML IdP share: prepare the next certificate, register it with every trust, and switch after confirming.',
    signingKeysLink: 'Signing keys',
    entraLink: 'Entra domain federation',
    certificatesHeading: 'Certificates',
    kidLabel: 'Kid',
    stateLabel: 'State',
    fingerprintLabel: 'SHA-256 fingerprint',
    createdAtLabel: 'Created',
    expiresAtLabel: 'Published until',
    stateActive: 'Current signing key',
    statePrepared: 'Prepared (not signing yet)',
    stateVerifying: 'Previous key (published for verification)',
    noKeysNotice: 'There is no XML signing key yet. It is created on the first signature.',
    impactHeading: 'Trusts that rely on this key',
    impactNotice:
      'After the switch, every trust below receives signatures from the new key. A matched Entra check does not mean that the other trusts are updated.',
    impactEntra: 'Entra domains',
    impactWsFed: 'WS-Federation relying parties',
    impactSaml: 'SAML service providers (default IdP profile)',
    impactNone: 'None',
    prepareHeading: '1. Prepare the next certificate',
    prepareNotice:
      'The prepared certificate is published in the metadata, but the current key keeps signing. The prepared key stays when you leave this page, and you can continue from it later.',
    prepare: 'Prepare the next certificate',
    preparing: 'Preparing…',
    preparedNotice: 'Prepared the next certificate. The current key keeps signing.',
    preparedKeyLabel: 'Prepared key',
    cancelPrepared: 'Cancel the preparation',
    cancelConfirmTitle: 'Cancel the prepared key',
    cancelConfirmDescription:
      'The key is removed from the metadata. The signing key does not change. A cancelled key cannot be restored. If you registered it in Entra as NextSigningCertificate, remove it with the update command shown after the cancellation.',
    cancelledNotice: 'Cancelled the prepared key {kid}.',
    registerHeading: '2. Register the next certificate with every trust',
    registerEntraNotice:
      'Run the update command for each domain, then import the read-back on the domain page. The check matches only when it confirms the prepared certificate as NextSigningCertificate.',
    openDomain: 'Check on the domain page',
    copyUpdateCommand: 'Copy the update command for {domain}',
    entraReady: 'Prepared certificate confirmed',
    entraNotReady: 'Prepared certificate not confirmed',
    registerTrustNotice:
      'IdMagic cannot check the other trusts. Register the next certificate in each of them, then select the ones you registered.',
    nextCertificateLabel: 'Next certificate (Base64, public)',
    copyNextCertificate: 'Copy the next certificate',
    trustRegistered: 'Registered the next certificate with {name}',
    activateHeading: '3. Confirm and switch',
    activateNotice:
      'After the switch, the new key signs and the previous key stays published for verification for 7 days. You confirm again each time you open this page.',
    overallConfirm: 'I confirmed that every trust that relies on this key has the next certificate',
    blockersHeading: 'Before you switch',
    blockerNoPrepared: 'Prepare the next certificate.',
    blockerEntra: 'The check for {domain} does not match the prepared certificate.',
    blockerTrust: 'Confirm the registration with {name}.',
    blockerOverall: 'Confirm every trust.',
    activate: 'Switch to the new key',
    activating: 'Switching…',
    activateConfirmTitle: 'Switch the shared XML signing key',
    activateConfirmDescription:
      'New assertions and tokens are signed with the new key. There is no action to sign with the previous key again. The previous certificate stays in the metadata for 7 days.',
    activatedNotice: 'Switched to the new key {kid}.',
    changedSinceConfirmation:
      'The prepared key, an Entra check, or a trust changed after you confirmed. Confirm again with the current state.',
    activationRejected:
      'The prepared key changed, so the switch was refused. Confirm again with the current state.',
    actionFailed: 'Could not complete the signing key action.',
    afterHeading: 'After the switch',
    afterNotice:
      'After the switch, finish the following before the previous key stops being published.',
    afterEntra:
      'For each Entra domain, run the update command again to set SigningCertificate to the new certificate, and import the read-back.',
    afterSignIn: 'Confirm that new sign-ins succeed for every domain and trust.',
    afterTrusts: 'Replace the signing certificate in the trusts other than Entra with the new one.',
    recoveryHeading: 'Recovery',
    recoveryBefore:
      'Before the switch: cancelling the preparation affects no trust. Remove the NextSigningCertificate registered in Entra with the update command shown after the cancellation.',
    recoveryAfter:
      'After the switch: there is no action to sign with the previous key again. Fix a trust that fails sign-in by registering the new certificate. The previous certificate stays in the metadata until it stops being published.',
    recoveryDeadline: 'The previous key {kid} is published until {date}',
  },
)

export type XmlFederationKeyDictionary = (typeof xmlFederationKeyDictionary)['ja']
