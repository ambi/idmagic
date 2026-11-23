import { defineDictionary } from '../lib/i18n/dictionary'

export const permissionImpactDictionary = defineDictionary(
  {
    changes: '変更の有無',
    none: '変更なし',
    reversible: '取り消せる変更',
    irreversible: '取り消せない変更',
    unknown: '影響度未登録',
    personal: '個人データを扱う',
    noPersonal: '個人データを扱わない',
    personalUnknown: '個人データの扱いは不明',
    warning: '取り消せない変更と同じ注意が必要です。',
    disclaimer: '登録者の申告です。IdMagic は業務上の影響が事実かを保証しません。',
    personalLabel: '個人データ',
    yes: '扱う',
    no: '扱わない',
    impact: '権限の影響度',
  },
  {
    changes: 'Changes',
    none: 'No changes',
    reversible: 'Reversible changes',
    irreversible: 'Irreversible changes',
    unknown: 'Impact not registered',
    personal: 'Handles personal data',
    noPersonal: 'Does not handle personal data',
    personalUnknown: 'Personal data handling unknown',
    warning: 'Review as an irreversible change.',
    disclaimer:
      'These are claims made by the registrant. IdMagic does not verify their business effects.',
    personalLabel: 'Personal data',
    yes: 'Yes',
    no: 'No',
    impact: 'Permission impact',
  },
)
