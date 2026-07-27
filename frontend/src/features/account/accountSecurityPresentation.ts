import { accountSecurityDictionary } from './AccountSecurityPage.i18n'

export function formatAccountSecurityDateTime(
  value?: string,
  locale = 'en',
  noRecord = accountSecurityDictionary.en.noRecord,
): string {
  if (!value) return noRecord
  return new Date(value).toLocaleString(locale, { dateStyle: 'medium', timeStyle: 'short' })
}
