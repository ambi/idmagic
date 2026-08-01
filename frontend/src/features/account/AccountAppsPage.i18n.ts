import { defineDictionary } from '../../lib/i18n'

export const accountAppsDictionary = defineDictionary(
  {
    saveFailed: '並び順を保存できませんでした。',
    title: 'アプリ',
    description: 'あなたが利用できるアプリケーションです。タイルから起動できます。',
    empty: '利用できるアプリはまだありません。',
    saving: '並び順を保存中...',
    unconfiguredLaunchURL: 'アプリケーション URL が未設定です',
    dragToReorder: '{name} をドラッグして並び替え',
    other: 'その他',
  },
  {
    saveFailed: 'Could not save the order.',
    title: 'Applications',
    description: 'These are the applications available to you. Launch one from its tile.',
    empty: 'There are no applications available yet.',
    saving: 'Saving order...',
    unconfiguredLaunchURL: 'Application URL is not configured',
    dragToReorder: 'Drag {name} to reorder',
    other: 'Other',
  },
)
