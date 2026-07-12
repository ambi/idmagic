import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react'
import { setCurrentLocale } from './currentLocale'
import { FALLBACK_LOCALE, type Locale } from './locale'
import { resolveLocale } from './resolveLocale'

const STORAGE_KEY = 'idmagic.displayLocale'

function readSavedLocale(): string | null {
  try {
    return window.localStorage.getItem(STORAGE_KEY)
  } catch {
    return null
  }
}

function writeSavedLocale(locale: Locale): void {
  try {
    window.localStorage.setItem(STORAGE_KEY, locale)
  } catch {
    // プライベートブラウジング等で localStorage が使えない場合は保存を諦める。
  }
}

function readInitialLocale(): Locale {
  if (typeof window === 'undefined') return FALLBACK_LOCALE
  const params = new URLSearchParams(window.location.search)
  return resolveLocale({
    uiLocalesHint: params.get('ui_locales'),
    saved: readSavedLocale(),
    browserLanguages: window.navigator?.languages,
  })
}

type LocaleContextValue = {
  locale: Locale
  setLocale: (next: Locale) => void
}

// Provider が無い純粋な presentation unit test では従来の日本語を返す。本体と統合
// テストは必ず LocaleProvider を通り、FallbackLocale=en を使用する。
const defaultLocaleContextValue: LocaleContextValue = {
  locale: 'ja',
  setLocale: () => {},
}

const LocaleContext = createContext<LocaleContextValue>(defaultLocaleContextValue)

export function LocaleProvider({
  children,
  initialLocale,
}: {
  children: ReactNode
  initialLocale?: Locale
}) {
  const [locale, setLocaleState] = useState<Locale>(() => initialLocale ?? readInitialLocale())

  useEffect(() => {
    document.documentElement.lang = locale
    setCurrentLocale(locale)
  }, [locale])

  const setLocale = useCallback((next: Locale) => {
    setLocaleState(next)
    writeSavedLocale(next)
  }, [])

  const value = useMemo(() => ({ locale, setLocale }), [locale, setLocale])

  return <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>
}

export function useLocale(): LocaleContextValue {
  return useContext(LocaleContext)
}

// useDictionary は defineDictionary で作った辞書から現在 locale の文言オブジェクトを返す。
// 呼び出し側は t.someKey の形でそのまま参照でき、key は呼び出し元で型検査される。
export function useDictionary<T extends Record<string, string>>(dictionary: Record<Locale, T>): T {
  const { locale } = useLocale()
  return dictionary[locale]
}

export function useFormatters() {
  const { locale } = useLocale()
  return useMemo(() => {
    const dateFormatter = new Intl.DateTimeFormat(locale, { dateStyle: 'medium' })
    const dateTimeFormatter = new Intl.DateTimeFormat(locale, {
      dateStyle: 'medium',
      timeStyle: 'short',
    })
    const numberFormatter = new Intl.NumberFormat(locale)
    return {
      formatDate: (value: Date | string | number) => dateFormatter.format(new Date(value)),
      formatDateTime: (value: Date | string | number) => dateTimeFormatter.format(new Date(value)),
      formatNumber: (value: number) => numberFormatter.format(value),
    }
  }, [locale])
}
