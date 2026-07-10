import { IconBuildingCommunity, IconShieldCheck } from '@tabler/icons-react'

// システムコンソールは system_admin 専用の、テナント横断 (control plane) 管理領域。
// テナント管理コンソール (/admin) とは別ルート・別シェルに隔離し、各ルートの
// loader でも system_admin ロールを必須化する (path ではなく role でゲート)。
export type SystemNavKey = 'tenants' | 'key-health'

export type SystemNavItem = {
  key: SystemNavKey
  label: string
  icon: typeof IconShieldCheck
  href: string
  active: boolean
}

export function systemNavItems(active: SystemNavKey): SystemNavItem[] {
  return [
    {
      key: 'tenants',
      label: 'テナント',
      icon: IconBuildingCommunity,
      href: '/system/tenants',
      active: active === 'tenants',
    },
    {
      key: 'key-health',
      label: '署名鍵の状態',
      icon: IconShieldCheck,
      href: '/system/keys',
      active: active === 'key-health',
    },
  ]
}
