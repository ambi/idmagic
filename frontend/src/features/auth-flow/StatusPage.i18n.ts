import { defineDictionary } from '../../lib/i18n'

export const statusPageDictionary = defineDictionary(
  {
    approvedEyebrow: '接続を承認しました',
    approvedTitle: 'デバイスを承認しました',
    approvedText: '認証が完了しました。このウィンドウを閉じて、接続したデバイスに戻ってください。',
    approvedNote: 'この操作に心当たりがない場合は、すぐに管理者へ連絡してください。',
    deniedEyebrow: '接続を拒否しました',
    deniedTitle: '接続を拒否しました',
    deniedText: 'デバイスへの接続を拒否しました。アカウント情報は共有されていません。',
    deniedNote: '同じ要求が繰り返される場合は、管理者へ連絡してください。',
    signedOutEyebrow: 'ログアウトしました',
    signedOutTitle: 'ログアウトしました',
    signedOutText: 'IdMagic のセッションを安全に終了しました。',
    signedOutNote: '共有端末では、このブラウザを閉じることをおすすめします。',
    authRequiredEyebrow: 'ログインが必要です',
    authRequiredTitle: 'ログインが必要です',
    authRequiredText: 'デバイスを承認するには、先に認証フローからログインしてください。',
    authRequiredNote:
      '元のアプリケーションまたはデバイスに戻り、接続操作を最初からやり直してください。',
    signInToAccount: 'マイページにログイン',
    signInToAdmin: '管理コンソールにログイン',
  },
  {
    approvedEyebrow: 'Connection approved',
    approvedTitle: 'Device approved',
    approvedText:
      'Authentication is complete. Close this window and return to the connected device.',
    approvedNote: 'If you did not initiate this action, contact your administrator immediately.',
    deniedEyebrow: 'Connection denied',
    deniedTitle: 'Connection denied',
    deniedText: 'The connection to the device was denied. No account information was shared.',
    deniedNote: 'Contact your administrator if the same request is repeated.',
    signedOutEyebrow: 'Signed out',
    signedOutTitle: 'You have signed out',
    signedOutText: 'Your IdMagic session was ended securely.',
    signedOutNote: 'On a shared device, we recommend closing this browser.',
    authRequiredEyebrow: 'Authentication required',
    authRequiredTitle: 'Sign-in required',
    authRequiredText: 'Sign in through the authentication flow before approving this device.',
    authRequiredNote:
      'Return to the original application or device and start the connection again.',
    signInToAccount: 'Sign in to account',
    signInToAdmin: 'Sign in to admin console',
  },
)
