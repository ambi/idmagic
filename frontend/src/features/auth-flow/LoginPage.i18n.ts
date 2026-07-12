import { defineDictionary } from '../../lib/i18n'

export const loginPageDictionary = defineDictionary(
  {
    eyebrow: 'サインイン',
    title: 'アカウントにログイン',
    description: '組織から発行された認証情報を入力して、安全に続行してください。',
    errorTitle: 'ログインできません',
    usernameLabel: 'ユーザー名',
    usernamePlaceholder: '例: your.name',
    passwordLabel: 'パスワード',
    passwordPlaceholder: 'パスワードを入力',
    showPassword: 'パスワードを表示',
    hidePassword: 'パスワードを隠す',
    submitting: '確認しています…',
    submit: 'ログインして続行',
    forgotPassword: 'パスワードを忘れた場合',
    securityNote:
      '認証情報は保護された接続で送信されます。共有端末では、利用後に必ずログアウトしてください。',
  },
  {
    eyebrow: 'Sign in',
    title: 'Sign in to your account',
    description: 'Enter the credentials issued by your organization to continue securely.',
    errorTitle: "Can't sign in",
    usernameLabel: 'Username',
    usernamePlaceholder: 'e.g. your.name',
    passwordLabel: 'Password',
    passwordPlaceholder: 'Enter your password',
    showPassword: 'Show password',
    hidePassword: 'Hide password',
    submitting: 'Checking…',
    submit: 'Sign in',
    forgotPassword: 'Forgot your password?',
    securityNote:
      'Your credentials are sent over a protected connection. Always sign out after using a shared device.',
  },
)
