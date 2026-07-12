import { defineDictionary } from '../../lib/i18n'

export const emailVerifyPageDictionary = defineDictionary(
  {
    confirmationFailed: '確認に失敗しました。',
    title: 'メールアドレスの確認',
    description: 'このアドレスをアカウントのメールアドレスとして確定します。',
    confirmed: 'メールアドレスを確認しました。',
    returnToAccount: 'アカウントへ戻る',
    invalidLink: '確認リンクが正しくありません。メール内のリンクをもう一度開いてください。',
    verifying: '確認中…',
    confirmEmail: 'メールアドレスを確認する',
  },
  {
    confirmationFailed: 'Could not confirm the email address.',
    title: 'Confirm your email address',
    description: 'Confirm this address as the email address for your account.',
    confirmed: 'Your email address has been confirmed.',
    returnToAccount: 'Return to account',
    invalidLink: 'The confirmation link is invalid. Open the link in your email again.',
    verifying: 'Confirming…',
    confirmEmail: 'Confirm email address',
  },
)
