import { defineDictionary } from '../../lib/i18n'

export const accountApplicationsDictionary = defineDictionary(
  {
    revokeNotice:
      '「{name}」へのアクセスを取り消しました。次回このアプリを使うときは、改めて許可を求められます。',
    revokeFailed: 'アクセスを取り消せませんでした。',
    title: '接続済みアプリ',
    description:
      'あなたのアカウントへのアクセスを許可したアプリケーションです。不要なものは取り消せます。',
    empty: 'アクセスを許可したアプリはありません。',
    granted: '{date} に許可',
    revoking: '取り消し中…',
    revoke: 'アクセスを取り消す',
  },
  {
    revokeNotice:
      'Access for “{name}” has been revoked. You will be asked to grant access again the next time you use this app.',
    revokeFailed: 'Could not revoke access.',
    title: 'Connected apps',
    description:
      'These applications have access to your account. You can revoke access you no longer need.',
    empty: 'No applications have been granted access.',
    granted: 'Granted {date}',
    revoking: 'Revoking…',
    revoke: 'Revoke access',
  },
)
