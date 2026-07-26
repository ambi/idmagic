import { IconArrowLeft, IconPlus } from '@tabler/icons-react'
import { tenantURL } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import type { AdminSamlIDPProfile } from '../../types'
import { adminSamlIDPProfilesDictionary } from './AdminSamlIDPProfilesPage.i18n'

export function AdminSamlIDPProfilesListPage({
  actorUsername,
  profiles,
}: {
  actorUsername?: string
  profiles: AdminSamlIDPProfile[]
}) {
  const t = useDictionary(adminSamlIDPProfilesDictionary)

  return (
    <AdminShell
      active="settings"
      actorUsername={actorUsername}
      title={t.listTitle}
      description={t.listDescription}
      actions={
        <>
          <Button asChild variant="outline">
            <a href={tenantURL('/admin/settings?tab=integration-endpoints')}>
              <IconArrowLeft size={16} aria-hidden="true" />
              {t.backToSettings}
            </a>
          </Button>
          <Button asChild>
            <a href={tenantURL('/admin/settings/saml-idp-profiles/new')}>
              <IconPlus size={16} aria-hidden="true" />
              {t.createProfile}
            </a>
          </Button>
        </>
      }
    >
      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
            <tr>
              <th className="px-4 py-3">{t.profileName}</th>
              <th className="px-4 py-3">{t.sharingMode}</th>
              <th className="px-4 py-3">{t.assignedServiceProvidersHeader}</th>
              <th className="px-4 py-3 text-right" aria-label={t.viewDetails} />
            </tr>
          </thead>
          <tbody>
            {profiles.map((entry) => (
              <tr key={entry.profile.profile_id} className="border-t border-slate-100">
                <td className="px-4 py-3">
                  <p className="font-semibold text-slate-900">{entry.profile.name}</p>
                  <p className="mt-0.5 font-mono text-xs text-slate-500">
                    {entry.profile.profile_id}
                  </p>
                </td>
                <td className="px-4 py-3 text-slate-600">
                  {entry.profile.mode === 'shared' ? t.shared : t.dedicated}
                  {entry.profile.is_default ? (
                    <span className="ml-2 rounded-full bg-blue-50 px-2 py-0.5 text-xs font-medium text-blue-700">
                      {t.defaultProfile}
                    </span>
                  ) : null}
                </td>
                <td className="px-4 py-3 text-slate-600">{entry.service_provider_count}</td>
                <td className="px-4 py-3 text-right">
                  <Button asChild variant="outline">
                    <a
                      href={tenantURL(
                        `/admin/settings/saml-idp-profiles/${encodeURIComponent(entry.profile.profile_id)}`,
                      )}
                    >
                      {t.viewDetails}
                    </a>
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {profiles.length === 0 ? (
          <p className="px-6 py-10 text-center text-sm text-slate-500">{t.emptyProfiles}</p>
        ) : null}
      </Card>
    </AdminShell>
  )
}
