import { defineDictionary } from '../../lib/i18n'

export const accountDataDictionary = defineDictionary(
  {
    exportFailed: 'データをエクスポートできませんでした。',
    title: 'データとプライバシー',
    description: 'アカウントに保存されている個人データをダウンロードできます。',
    exportTitle: '個人データのエクスポート',
    exportDescription:
      'プロフィール (表示名・属性・メール・ライフサイクル) と、アクセスを許可したアプリ (接続済みアプリ) を JSON ファイルとしてダウンロードします。サインイン履歴やセッションの同梱は今後対応します。',
    creating: '生成中…',
    download: 'データをダウンロード (JSON)',
  },
  {
    exportFailed: 'Could not export your data.',
    title: 'Data and privacy',
    description: 'Download the personal data stored in your account.',
    exportTitle: 'Export personal data',
    exportDescription:
      'Download your profile (name, attributes, email, and lifecycle) and the applications you have authorized as a JSON file. Sign-in history and sessions are not included yet.',
    creating: 'Preparing…',
    download: 'Download data (JSON)',
  },
)
