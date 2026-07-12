import { defineDictionary } from '../../lib/i18n'

export const informationalPagesDictionary = defineDictionary(
  {
    homeEyebrow: 'IDプロバイダー',
    homeTitle: 'IdMagic は起動しています',
    homeDescription:
      'ログイン画面は、接続するアプリケーションから認証要求を受けたときに表示されます。',
    homeDirectLogin:
      '/login を直接開くことはできません。OAuth 2.0 / OpenID Connect クライアントから /authorize を開始してください。',
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
    homeTitle: 'IdMagic is running',
    homeDescription:
      'The sign-in page appears when a connected application starts an authorization request.',
    homeDirectLogin:
      'You cannot open /login directly. Start /authorize from an OAuth 2.0 or OpenID Connect client.',
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
