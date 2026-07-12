import { defineDictionary } from './dictionary'

// common.i18n は特定 feature に属さない横断文言 (言語切替 UI、通信不能時の既定文言) を持つ。
export const commonDictionary = defineDictionary(
  {
    languageSwitcherLabel: '表示言語',
    japanese: '日本語',
    english: 'English',
    networkError: '認証サービスに接続できませんでした。',
    invalidCredentials: '認証情報が正しくありません。',
    passwordPolicy: 'パスワードがポリシーを満たしていません。',
    passwordReuse: '以前使用したパスワードは使用できません。',
    invalidRequest: '入力内容を確認してください。',
    accessDenied: 'この操作は許可されていません。',
    csrfToken: 'ページを再読み込みして、もう一度お試しください。',
    sessionNotFound: 'セッションが見つかりません。もう一度サインインしてください。',
  },
  {
    languageSwitcherLabel: 'Display language',
    japanese: '日本語',
    english: 'English',
    networkError: 'Could not connect to the authentication service.',
    invalidCredentials: 'The credentials are invalid.',
    passwordPolicy: 'The password does not meet the policy.',
    passwordReuse: 'You cannot reuse a previous password.',
    invalidRequest: 'Check the submitted information.',
    accessDenied: 'This action is not permitted.',
    csrfToken: 'Reload the page and try again.',
    sessionNotFound: 'The session was not found. Sign in again.',
  },
)
