import { provisioningDictionary } from '../admin-applications/AdminApplicationProvisioning.i18n'
import { provisioningURL } from '../admin-applications/AdminApplicationsShared'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Card } from '../../components/ui/card'
import { useDictionary, useLocale } from '../../lib/i18n'
import type { AdminApplication, ProvisioningConnection } from '../../types'

function formatDate(value: string | null | undefined, locale: string, unknown: string): string {
  if (!value) return unknown
  return new Date(value).toLocaleString(locale === 'ja' ? 'ja-JP' : 'en-US')
}

export function AdminProvisioningOverviewPage({
  actorUsername,
  connections,
  applications,
}: {
  actorUsername?: string
  connections: ProvisioningConnection[]
  applications: AdminApplication[]
}) {
  const t = useDictionary(provisioningDictionary)
  const { locale } = useLocale()
  const appNames = new Map(applications.map((a) => [a.application_id, a.name]))

  return (
    <AdminShell
      active="provisioning"
      actorUsername={actorUsername}
      title={t.overviewHeading}
      description={t.overviewDescription}
    >
      <Card className="overflow-hidden">
        {connections.length === 0 ? (
          <Alert className="m-5">{t.overviewEmptyNotice}</Alert>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-slate-50 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
                <tr>
                  <th className="px-4 py-3">{t.tableHeaderApplication}</th>
                  <th className="px-4 py-3">{t.tableHeaderBaseUrl}</th>
                  <th className="px-4 py-3">{t.statusFieldLabel}</th>
                  <th className="px-4 py-3">{t.healthFieldLabel}</th>
                  <th className="px-4 py-3">{t.tableHeaderLastFullSync}</th>
                </tr>
              </thead>
              <tbody>
                {connections.map((c) => (
                  <tr
                    key={c.application_id}
                    className="border-t border-slate-100 hover:bg-slate-50"
                  >
                    <td className="px-4 py-3">
                      <a
                        href={provisioningURL(c.application_id)}
                        className="font-medium text-blue-700 hover:underline"
                      >
                        {appNames.get(c.application_id) ?? c.application_id}
                      </a>
                    </td>
                    <td className="max-w-xs truncate px-4 py-3 font-mono text-xs">{c.base_url}</td>
                    <td className="px-4 py-3">
                      <span
                        className={`rounded-md px-2 py-0.5 text-xs font-medium ${
                          c.status === 'active'
                            ? 'bg-emerald-50 text-emerald-700'
                            : 'bg-slate-100 text-slate-500'
                        }`}
                      >
                        {c.status === 'active' ? t.statusActiveLabel : t.statusDisabledLabel}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <span
                        className={`rounded-md px-2 py-0.5 text-xs font-medium ${
                          c.health === 'ok'
                            ? 'bg-emerald-50 text-emerald-700'
                            : c.health === 'degraded'
                              ? 'bg-amber-50 text-amber-700'
                              : 'bg-red-50 text-red-700'
                        }`}
                      >
                        {c.health === 'ok'
                          ? t.healthOkLabel
                          : c.health === 'degraded'
                            ? t.healthDegradedLabel
                            : t.healthQuarantinedLabel}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-xs text-slate-500">
                      {formatDate(c.last_full_sync_at, locale, t.neverSyncedLabel)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </AdminShell>
  )
}
