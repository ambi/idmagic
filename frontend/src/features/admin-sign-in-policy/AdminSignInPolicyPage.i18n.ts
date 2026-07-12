import { defineDictionary } from '../../lib/i18n'

export const adminSignInPolicyDictionary = defineDictionary(
  {
    strengthPasswordLabel: 'パスワードのみ',
    strengthMfaLabel: 'MFA 必須',
    noAdditionalRequirementsNotice: '追加要件なし',
    reauthSuffix: '再認証まで {seconds} 秒',
    allowedNetworkPrefix: '許可ネットワーク {cidrs}',
    pageTitle: 'サインインポリシー',
    pageDescription:
      '全アプリ共通のデフォルトポリシーと、アプリごとの実効ポリシーを一元的に管理します。',
    defaultPolicyHeading: 'デフォルトサインインポリシー',
    defaultPolicyDescription:
      '独自ポリシーを設定していない全アプリケーションに適用される baseline です。各アプリはこれを上書きできます。',
    edit: '編集',
    requiredAuthnStrengthFieldLabel: '要求する認証強度',
    reauthTimeFieldLabel: '再認証を求めるまでの時間',
    reauthSecondsValue: '{seconds} 秒',
    unlimitedNoReauthNotice: '無期限（再認証を求めない）',
    allowedNetworksFieldLabel: '許可するネットワーク',
    noRestrictionNotice: '制限なし',
    mfaStepUpHelp:
      '「MFA 必須」の場合、単要素セッションはサインイン時に再認証 (step-up) へ誘導されます。',
    reauthSecondsFieldLabel: '再認証を求めるまでの時間（秒）',
    reauthSecondsPlaceholder: '例: 3600',
    reauthSecondsHelp: 'この秒数を超えた認証は再認証（再ログイン）を求めます。空欄なら無期限です。',
    allowedNetworksCidrFieldLabel: '許可するネットワーク (CIDR)',
    allowedNetworksHelp:
      '1 行に 1 つの CIDR を入力します。指定するとリスト外の IP からのサインインは拒否されます。空欄なら制限しません。',
    saving: '保存中…',
    save: '保存',
    cancel: 'キャンセル',
    defaultPolicyUpdateFailedError: 'デフォルトサインインポリシーを更新できませんでした。',
    reauthMaxAgeInvalidError: '再認証を求めるまでの時間には 1 以上の秒数を入力してください。',
    applicationPolicyHeading: 'アプリケーション別ポリシー',
    applicationPolicyDescription:
      'アプリ独自のポリシーはデフォルトを上書きします。「最終的に適用されるポリシー」で実効値を確認でき、個別設定はアプリ編集画面から変更できます。',
    noAppsNotice: 'サインインポリシー対象のアプリケーションがありません。',
    tableHeaderApplication: 'アプリケーション',
    tableHeaderOverride: '個別設定',
    tableHeaderEffectivePolicy: '最終的に適用されるポリシー',
    overrideBadge: '上書き',
    weakerThanDefaultBadge: 'デフォルトより弱い',
    defaultAppliedBadge: 'デフォルトを適用',
  },
  {
    strengthPasswordLabel: 'Password only',
    strengthMfaLabel: 'MFA required',
    noAdditionalRequirementsNotice: 'No additional requirements',
    reauthSuffix: 'Re-authenticate after {seconds}s',
    allowedNetworkPrefix: 'Allowed network {cidrs}',
    pageTitle: 'Sign-in policy',
    pageDescription:
      "Manage the default policy shared by all apps and each app's effective policy in one place.",
    defaultPolicyHeading: 'Default sign-in policy',
    defaultPolicyDescription:
      'The baseline applied to all applications that have not set their own policy. Each app can override this.',
    edit: 'Edit',
    requiredAuthnStrengthFieldLabel: 'Required authentication strength',
    reauthTimeFieldLabel: 'Time before re-authentication',
    reauthSecondsValue: '{seconds}s',
    unlimitedNoReauthNotice: 'Unlimited (no re-authentication required)',
    allowedNetworksFieldLabel: 'Allowed networks',
    noRestrictionNotice: 'No restriction',
    mfaStepUpHelp:
      'When "MFA required" is set, single-factor sessions are prompted to re-authenticate (step-up) at sign-in.',
    reauthSecondsFieldLabel: 'Time before re-authentication is required (seconds)',
    reauthSecondsPlaceholder: 'e.g., 3600',
    reauthSecondsHelp:
      'Authentications older than this many seconds require re-authentication (sign in again). Leave blank for no limit.',
    allowedNetworksCidrFieldLabel: 'Allowed networks (CIDR)',
    allowedNetworksHelp:
      'Enter one CIDR per line. When set, sign-ins from IPs outside the list are denied. Leave blank for no restriction.',
    saving: 'Saving…',
    save: 'Save',
    cancel: 'Cancel',
    defaultPolicyUpdateFailedError: 'Could not update the default sign-in policy.',
    reauthMaxAgeInvalidError:
      'Enter a number of seconds of 1 or more for the re-authentication time.',
    applicationPolicyHeading: 'Per-application policy',
    applicationPolicyDescription:
      'An app\'s own policy overrides the default. Check the effective value under "Effective policy," and change individual settings from the app edit screen.',
    noAppsNotice: 'There are no applications subject to sign-in policy.',
    tableHeaderApplication: 'Application',
    tableHeaderOverride: 'Override',
    tableHeaderEffectivePolicy: 'Effective policy',
    overrideBadge: 'Override',
    weakerThanDefaultBadge: 'Weaker than default',
    defaultAppliedBadge: 'Default applied',
  },
)

export type AdminSignInPolicyDictionary = (typeof adminSignInPolicyDictionary)['ja']
