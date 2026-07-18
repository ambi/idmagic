import { IconMail, IconPalette, IconShieldLock, IconTag, IconUsers } from '@tabler/icons-react'
import { useState } from 'react'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import { cn } from '../../lib/utils'
import type { AdminSettings } from '../../types'
import { adminSettingsDictionary, type AdminSettingsDictionary } from './AdminSettingsPage.i18n'
import { BrandingTab } from './BrandingTab'
import { GeneralTab } from './GeneralTab'
import { PasswordPolicyTab } from './PasswordPolicyTab'
import { ScimTab } from './ScimTab'

const DEFAULT_REALM = 'default'

type TabKey = 'general' | 'password-policy' | 'branding' | 'email' | 'scim'

type Tab = {
  key: TabKey
  label: string
  description: string
  icon: typeof IconTag
  disabled?: boolean
}

function tabs(t: AdminSettingsDictionary): Tab[] {
  return [
    {
      key: 'general',
      label: t.tabGeneralLabel,
      description: t.tabGeneralDescription,
      icon: IconTag,
    },
    {
      key: 'password-policy',
      label: t.tabPasswordPolicyLabel,
      description: t.tabPasswordPolicyDescription,
      icon: IconShieldLock,
    },
    {
      key: 'branding',
      label: t.tabBrandingLabel,
      description: t.tabBrandingDescription,
      icon: IconPalette,
    },
    {
      key: 'scim',
      label: t.tabScimLabel,
      description: t.tabScimDescription,
      icon: IconUsers,
    },
    {
      key: 'email',
      label: t.tabEmailLabel,
      description: t.tabEmailDescription,
      icon: IconMail,
      disabled: true,
    },
  ]
}

export function AdminSettingsPage({
  csrfToken,
  actorUsername,
  actorRoles,
  actorRealm,
  settings: initial,
}: {
  csrfToken: string
  actorUsername?: string
  actorRoles: string[]
  actorRealm: string
  settings: AdminSettings
}) {
  const [settings, setSettings] = useState(initial)
  const [active, setActive] = useState<TabKey>('general')
  const isSystemAdminOnDefault = actorRoles.includes('system_admin') && actorRealm === DEFAULT_REALM
  const t = useDictionary(adminSettingsDictionary)
  const tabList = tabs(t)

  return (
    <AdminShell
      active="settings"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
    >
      {isSystemAdminOnDefault ? (
        <Alert>
          <p className="text-sm text-slate-700">
            {t.crossTenantNoticePrefix}
            <a href="/system/tenants" className="ml-1 font-medium text-blue-700 hover:underline">
              {t.crossTenantNoticeLinkText}
            </a>
            {t.crossTenantNoticeSuffix}
          </p>
        </Alert>
      ) : null}

      <div className="grid gap-6 lg:grid-cols-[220px_minmax(0,1fr)]">
        <nav className="flex flex-col gap-1" aria-label={t.tabsAriaLabel}>
          {tabList.map((tab) => (
            <button
              key={tab.key}
              type="button"
              onClick={() => !tab.disabled && setActive(tab.key)}
              disabled={tab.disabled}
              aria-current={active === tab.key ? 'page' : undefined}
              className={cn(
                'flex items-center gap-3 rounded-lg px-3 py-2 text-left text-sm font-medium',
                tab.disabled
                  ? 'cursor-not-allowed text-slate-400'
                  : active === tab.key
                    ? 'bg-slate-950 text-white shadow-sm'
                    : 'text-slate-600 hover:bg-white hover:text-slate-950 hover:shadow-xs',
              )}
            >
              <tab.icon size={18} stroke={1.8} aria-hidden="true" />
              <span className="flex-1">{tab.label}</span>
              {tab.disabled ? (
                <span className="rounded-md bg-slate-100 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-slate-500">
                  {t.comingSoonBadge}
                </span>
              ) : null}
            </button>
          ))}
        </nav>

        <div className="min-w-0">
          {active === 'general' ? (
            <GeneralTab
              csrfToken={csrfToken}
              settings={settings}
              onSaved={(next) => setSettings(next)}
            />
          ) : null}
          {active === 'password-policy' ? (
            <PasswordPolicyTab
              csrfToken={csrfToken}
              settings={settings}
              onSaved={(next) => setSettings(next)}
            />
          ) : null}
          {active === 'branding' ? <BrandingTab csrfToken={csrfToken} /> : null}
          {active === 'scim' ? <ScimTab csrfToken={csrfToken} tenantID={settings.realm} /> : null}
          {active === 'email' ? (
            <Card className="p-6">
              <h2 className="text-base font-semibold text-slate-900">{t.emailTabHeading}</h2>
              <p className="mt-2 text-sm text-slate-600">{t.emailTabDescription}</p>
            </Card>
          ) : null}
        </div>
      </div>
    </AdminShell>
  )
}
