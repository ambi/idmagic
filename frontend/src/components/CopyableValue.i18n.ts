import { defineDictionary } from '../lib/i18n'

export const copyableValueDictionary = defineDictionary(
  {
    copied: 'コピーしました。',
    failed: 'コピーできませんでした。値を選択してコピーしてください。',
  },
  { copied: 'Copied.', failed: 'Copy failed. Select and copy the value manually.' },
)
