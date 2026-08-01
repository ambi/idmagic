import { defineDictionary } from '../../lib/i18n'

export const informationalPagesDictionary = defineDictionary(
  {
    homeEyebrow: 'IDプロバイダー',
    homeTitle: 'このページでは何も操作できません',
    homeDescription:
      'ここは各アプリケーションのサインインを仲介する基盤のトップページです。ログイン画面は、利用するアプリケーションを経由してアクセスしたときに表示されます。',
    homeDirectLogin:
      'このアドレスへ直接アクセスした場合は、利用したいアプリケーションの画面に戻ってログインを始めてください。',
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
    homeTitle: 'There is nothing to do on this page',
    homeDescription:
      'This is the landing page for the sign-in service that connected applications use. The sign-in screen appears only when you reach it through the application you are using.',
    homeDirectLogin:
      'If you reached this address directly, go back to the application you want to use and start signing in from there.',
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
