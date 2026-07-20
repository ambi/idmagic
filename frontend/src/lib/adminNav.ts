import {
  IconActivity,
  IconApps,
  IconCheckupList,
  IconCloudUpload,
  IconForms,
  IconLayoutDashboard,
  IconPlugConnected,
  IconRobot,
  IconSettings,
  IconShieldLock,
  IconUsersGroup,
  IconUserShield,
  IconUsers,
  IconWorldShare,
} from '@tabler/icons-react'
import { tenantURL } from '../api'
import type { Locale } from './i18n'
import { shellDictionary } from '../components/shell.i18n'

export type AdminNavKey =
  | 'dashboard'
  | 'users'
  | 'groups'
  | 'agents'
  | 'workflows'
  | 'roles'
  | 'applications'
  | 'provisioning'
  | 'sign-in-policy'
  | 'entra-federation'
  | 'authz-detail-types'
  | 'mcp-resource-servers'
  | 'consents'
  | 'audit-events'
  | 'keys'
  | 'tenant-attributes'
  | 'settings'

export type AdminNavItem = {
  key: AdminNavKey
  label: string
  icon: typeof IconUsers
  href: string
  active: boolean
}

export function adminNavItems(active: AdminNavKey, locale: Locale = 'ja'): AdminNavItem[] {
  const t = shellDictionary[locale]
  const items: AdminNavItem[] = [
    {
      key: 'dashboard',
      label: t.dashboard,
      icon: IconLayoutDashboard,
      href: tenantURL('/admin'),
      active: active === 'dashboard',
    },
    {
      key: 'users',
      label: t.users,
      icon: IconUsers,
      href: tenantURL('/admin/users'),
      active: active === 'users',
    },
    {
      key: 'groups',
      label: t.groups,
      icon: IconUsersGroup,
      href: tenantURL('/admin/groups'),
      active: active === 'groups',
    },
    {
      key: 'agents',
      label: t.agents,
      icon: IconRobot,
      href: tenantURL('/admin/agents'),
      active: active === 'agents',
    },
    {
      key: 'workflows',
      label: 'ワークフロー',
      icon: IconActivity,
      href: tenantURL('/admin/lifecycle-workflows'),
      active: active === 'workflows',
    },
    {
      key: 'roles',
      label: t.roles,
      icon: IconUserShield,
      href: tenantURL('/admin/roles'),
      active: active === 'roles',
    },
    {
      key: 'applications',
      label: t.applications,
      icon: IconApps,
      href: tenantURL('/admin/applications'),
      active: active === 'applications',
    },
    {
      key: 'provisioning',
      label: t.provisioning,
      icon: IconCloudUpload,
      href: tenantURL('/admin/provisioning'),
      active: active === 'provisioning',
    },
    {
      key: 'sign-in-policy',
      label: t.signInPolicy,
      icon: IconShieldLock,
      href: tenantURL('/admin/sign-in-policy'),
      active: active === 'sign-in-policy',
    },
    {
      key: 'authz-detail-types',
      label: t.authorizationDetailTypes,
      icon: IconForms,
      href: tenantURL('/admin/authorization-detail-types'),
      active: active === 'authz-detail-types',
    },
    {
      key: 'mcp-resource-servers',
      label: t.mcpResourceServers,
      icon: IconPlugConnected,
      href: tenantURL('/admin/mcp-resource-servers'),
      active: active === 'mcp-resource-servers',
    },
    {
      key: 'consents',
      label: t.consents,
      icon: IconCheckupList,
      href: tenantURL('/admin/consents'),
      active: active === 'consents',
    },
    {
      key: 'audit-events',
      label: t.auditEvents,
      icon: IconActivity,
      href: tenantURL('/admin/audit_events'),
      active: active === 'audit-events',
    },
    {
      key: 'keys',
      label: t.signingKeys,
      icon: IconShieldLock,
      href: tenantURL('/admin/keys'),
      active: active === 'keys',
    },
    // OAuth2 クライアント / WS-Federation RP の設定は「アプリケーション」に一本化した。
    // 高度な OIDC 設定 (grant / response 種別・PAR・DPoP、作成時の認証方式) もアプリ編集画面に
    // 畳んだため、専用の低レベル client 画面は撤去した。
  ]
  items.push({
    key: 'tenant-attributes',
    label: t.userAttributes,
    icon: IconForms,
    href: tenantURL('/admin/tenant/attributes'),
    active: active === 'tenant-attributes',
  })
  items.push({
    key: 'entra-federation',
    label: t.entraFederation,
    icon: IconWorldShare,
    href: tenantURL('/admin/federation/entra'),
    active: active === 'entra-federation',
  })
  items.push({
    key: 'settings',
    label: t.settings,
    icon: IconSettings,
    href: tenantURL('/admin/settings'),
    active: active === 'settings',
  })
  return items
}
