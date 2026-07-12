import { defineDictionary } from './dictionary'

// common.i18n は特定 feature に属さない横断文言 (言語切替 UI、通信不能時の既定文言) を持つ。
export const commonDictionary = defineDictionary(
  {
    languageSwitcherLabel: '表示言語',
    japanese: '日本語',
    english: 'English',
    networkError: '認証サービスに接続できませんでした。',
  },
  {
    languageSwitcherLabel: 'Display language',
    japanese: '日本語',
    english: 'English',
    networkError: 'Could not connect to the authentication service.',
  },
)
