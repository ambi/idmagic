import { commonDictionary } from '../lib/i18n/common.i18n'
import { useDictionary, useLocale } from '../lib/i18n'
import { cn } from '../lib/utils'

// LanguageSwitcher は DisplayLanguage を明示的に切り替える chrome 部品 (UX-LOCALE)。
// 選択は useLocale 経由で即時反映し、以後の起動では保存済み設定として優先される。
export function LanguageSwitcher({ className }: { className?: string }) {
  const { locale, setLocale } = useLocale()
  const t = useDictionary(commonDictionary)

  return (
    <fieldset
      className={cn(
        'flex items-center gap-0.5 rounded-lg border border-slate-200/80 bg-white/70 p-0.5 text-xs font-medium shadow-xs',
        className,
      )}
      aria-label={t.languageSwitcherLabel}
    >
      <button
        type="button"
        aria-pressed={locale === 'ja'}
        onClick={() => setLocale('ja')}
        className={cn(
          'rounded-md px-2 py-1 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/30',
          locale === 'ja' ? 'bg-slate-950 text-white' : 'text-slate-600 hover:bg-slate-100',
        )}
      >
        {t.japanese}
      </button>
      <button
        type="button"
        aria-pressed={locale === 'en'}
        onClick={() => setLocale('en')}
        className={cn(
          'rounded-md px-2 py-1 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-600/30',
          locale === 'en' ? 'bg-slate-950 text-white' : 'text-slate-600 hover:bg-slate-100',
        )}
      >
        {t.english}
      </button>
    </fieldset>
  )
}
