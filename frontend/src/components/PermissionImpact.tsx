import { useDictionary } from '../lib/i18n'
import type { PermissionImpact as Impact } from '../types'
import { permissionImpactDictionary } from './PermissionImpact.i18n'

export function ImpactDisclaimer() {
  const t = useDictionary(permissionImpactDictionary)
  return <p className="text-xs text-slate-600">{t.disclaimer}</p>
}

export function PermissionImpact({ impact }: { impact?: Impact | null }) {
  const t = useDictionary(permissionImpactDictionary)
  const valid =
    impact &&
    ['none', 'reversible', 'irreversible'].includes(impact.changes) &&
    typeof impact.personal_data === 'boolean'
  const attention = !valid || impact.changes === 'irreversible'
  return (
    <div
      className={`mt-1 rounded-md p-2 text-xs ${attention ? 'border border-amber-300 bg-amber-50 text-amber-950' : 'bg-slate-50 text-slate-700'}`}
    >
      <p>{valid ? t[impact.changes] : t.unknown}</p>
      <p>{valid ? (impact.personal_data ? t.personal : t.noPersonal) : t.personalUnknown}</p>
      {attention ? <p>{t.warning}</p> : null}
    </div>
  )
}
