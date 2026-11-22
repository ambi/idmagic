export function createFormatters(locale: string) {
  const dateFormatter = new Intl.DateTimeFormat(locale, { dateStyle: 'medium' })
  const dateTimeFormatter = new Intl.DateTimeFormat(locale, {
    dateStyle: 'medium',
    timeStyle: 'short',
  })
  const dateTimeWithSecondsFormatter = new Intl.DateTimeFormat(locale, {
    dateStyle: 'medium',
    timeStyle: 'medium',
  })
  const numberFormatter = new Intl.NumberFormat(locale)
  return {
    formatDate: (value: Date | string | number) => dateFormatter.format(new Date(value)),
    formatDateTime: (value: Date | string | number) => dateTimeFormatter.format(new Date(value)),
    formatDateTimeWithSeconds: (value: Date | string | number) =>
      dateTimeWithSecondsFormatter.format(new Date(value)),
    formatNumber: (value: number) => numberFormatter.format(value),
  }
}
