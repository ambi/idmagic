import { createFormatters } from '../../lib/i18n'
import { accountSecurityDictionary } from './AccountSecurityPage.i18n'

export function formatAccountSecurityDateTime(
  value?: string,
  locale = 'en',
  noRecord = accountSecurityDictionary.en.noRecord,
): string {
  if (!value) return noRecord
  return createFormatters(locale).formatDateTime(value)
}
