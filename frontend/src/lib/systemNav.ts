import {
  IconBuildingCommunity,
  IconClipboardText,
  IconKey,
  IconListCheck,
  IconShieldCheck,
} from '@tabler/icons-react'
import type { Locale } from './i18n'
import { shellDictionary } from '../components/shell.i18n'

// システムコンソールは system_admin 専用の、テナント横断 (control plane) 管理領域。
// テナント管理コンソール (/admin) とは別ルート・別シェルに隔離し、各ルートの
// loader でも system_admin ロールを必須化する (path ではなく role でゲート)。
export type SystemNavKey = 'tenants' | 'audit-events' | 'jobs' | 'key-health' | 'data-key-health'

export type SystemNavItem = {
  key: SystemNavKey
  label: string
  icon: typeof IconShieldCheck
  href: string
  active: boolean
}

export function systemNavItems(active: SystemNavKey, locale: Locale = 'ja'): SystemNavItem[] {
  const t = shellDictionary[locale]
  return [
    {
      key: 'tenants',
      label: t.tenants,
      icon: IconBuildingCommunity,
      href: '/system/tenants',
      active: active === 'tenants',
    },
    {
      key: 'audit-events',
      label: t.auditEvents,
      icon: IconClipboardText,
      href: '/system/audit-events',
      active: active === 'audit-events',
    },
    {
      key: 'jobs',
      label: t.jobs,
      icon: IconListCheck,
      href: '/system/jobs',
      active: active === 'jobs',
    },
    {
      key: 'key-health',
      label: t.signingKeyHealth,
      icon: IconShieldCheck,
      href: '/system/keys',
      active: active === 'key-health',
    },
    {
      key: 'data-key-health',
      label: t.dataKeyHealth,
      icon: IconKey,
      href: '/system/data-keys',
      active: active === 'data-key-health',
    },
  ]
}
