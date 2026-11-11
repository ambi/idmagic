import { useDictionary } from '../../lib/i18n'
import { adminEntraFederationDictionary } from './AdminEntraFederationPage.i18n'

export function EntraImmutableIdGuide() {
  const t = useDictionary(adminEntraFederationDictionary)
  return (
    <section className="grid gap-2 text-sm">
      <h2 className="font-semibold">{t.immutableIdHeading}</h2>
      <p>{t.immutableIdHint}</p>
      <ol className="list-decimal space-y-2 pl-5 text-xs text-slate-600">
        <li>{t.immutableIdConnectStep}</li>
        <li>{t.immutableIdMappingStep}</li>
        <li>{t.immutableIdResyncStep}</li>
        <li>{t.immutableIdCompareStep}</li>
      </ol>
      <p className="text-xs text-slate-600">{t.immutableIdEncodingHint}</p>
      <a
        className="text-xs text-blue-700 underline"
        href="https://learn.microsoft.com/en-us/entra/identity/hybrid/connect/plan-connect-design-concepts#sourceanchor"
      >
        {t.immutableIdConnectReference}
      </a>
    </section>
  )
}
