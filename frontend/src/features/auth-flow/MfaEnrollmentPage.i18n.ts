import { defineDictionary } from '../../lib/i18n'

export const mfaEnrollmentPageDictionary = defineDictionary(
  {
    startFailed: '登録を開始できませんでした。',
    completeFailed: '登録を完了できませんでした。',
    eyebrow: 'MFA 登録が必要です',
    title: '認証アプリを登録',
    description:
      '管理者が承認した登録手続きです。完了するまでアプリやマイページにはアクセスできません。',
    setupInstruction: '認証アプリに次のキーを登録してください。',
    account: 'アカウント: {account}',
    verificationCode: '確認コード',
    enrolling: '登録中…',
    submit: '登録してログインを続行',
    securityNote: 'この登録は一度だけ利用できる管理者承認に基づいています。',
  },
  {
    startFailed: 'Could not start enrollment.',
    completeFailed: 'Could not complete enrollment.',
    eyebrow: 'MFA enrollment required',
    title: 'Enroll an authenticator app',
    description:
      'An administrator approved this enrollment. You cannot access applications or the account portal until it is complete.',
    setupInstruction: 'Add the following key to your authenticator app.',
    account: 'Account: {account}',
    verificationCode: 'Verification code',
    enrolling: 'Enrolling…',
    submit: 'Enroll and continue signing in',
    securityNote: 'This enrollment is based on a single-use administrator approval.',
  },
)
