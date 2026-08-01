import { defineDictionary } from '../../lib/i18n'

export const informationalPagesDictionary = defineDictionary(
  {
    homeEyebrow: 'IDプロバイダー',
    homeTitle: 'サインインを開始',
    homeDescription:
      'このテナントのアプリケーションへのサインインとアクセス管理を行う ID プロバイダーです。ご利用のアプリケーションを経由してアクセスした場合は、そのままサインイン画面が表示されます。',
    homeDirectLogin:
      'このページへ直接アクセスした場合は、下記からアカウントまたは管理コンソールを開いてサインインするか、利用したいアプリケーションの画面に戻ってサインインを始めてください。',
    startDemo: 'ローカルデモ認証を開始',
    startFromApplication: '利用するアプリケーションからログインを開始してください。',
    demoUser: 'デモユーザー:',
    callbackComplete: 'ローカルデモ認証が完了しました',
    callbackFailed: '認証を完了できませんでした',
    callbackCompleteText: '認可コードが発行され、ブラウザ認証フローが正常に完了しました。',
    invalidAuthorizationResponse: '認可レスポンスが不正です。',
    openAdmin: '管理コンソールを開く',
    tryAgain: 'もう一度試す',
  },
  {
    homeEyebrow: 'Identity provider',
    homeTitle: 'Sign in to get started',
    homeDescription:
      'This is the identity provider that handles sign-in and access management for this tenant. If you reached this page through an application, the sign-in screen will appear automatically.',
    homeDirectLogin:
      'If you reached this page directly, open your account or the admin console below to sign in, or go back to the application you want to use and start signing in from there.',
    startDemo: 'Start local demo authorization',
    startFromApplication: 'Start signing in from the application you use.',
    demoUser: 'Demo user:',
    callbackComplete: 'Local demo authorization is complete',
    callbackFailed: 'Could not complete authentication',
    callbackCompleteText:
      'An authorization code was issued and the browser authentication flow completed successfully.',
    invalidAuthorizationResponse: 'The authorization response is invalid.',
    openAdmin: 'Open admin console',
    tryAgain: 'Try again',
  },
)
