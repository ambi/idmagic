import { IconArrowLeft, IconChevronDown, IconLogout } from '@tabler/icons-react'
import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { logout } from '../api'
import { cn } from '../lib/utils'
import { useDictionary, useLocale } from '../lib/i18n'
import { systemNavItems, type SystemNavKey } from '../lib/systemNav'
import { Brand } from './Brand'
import { shellDictionary } from './shell.i18n'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from './ui/dropdown-menu'

type SystemShellProps = {
  active: SystemNavKey
  actorUsername?: string
  title: string
  description?: string
  actions?: ReactNode
  children: ReactNode
}

// SystemShell は system_admin 専用のシステムコンソール用シェル。テナント管理
// コンソール (AdminShell) とは配色・ブランド表記で明確に区別し、誤って通常の
// テナント管理と混同しないようにする。
export function SystemShell({
  active,
  actorUsername,
  title,
  description,
  actions,
  children,
}: SystemShellProps) {
  const t = useDictionary(shellDictionary)
  const { locale } = useLocale()
  const items = systemNavItems(active, locale)
  const currentItem = items.find((item) => item.active)
  return (
    <div className="app-surface">
      <header className="app-header border-t-2 border-t-amber-500">
        <div className="flex h-16 items-center justify-between px-5 lg:px-7">
          <div className="flex items-center gap-5">
            <Link
              to="/system/keys"
              aria-label={t.systemConsole}
              className="rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500/30"
            >
              <Brand compact />
            </Link>
            <div className="hidden h-4 w-px bg-slate-200 sm:block" />
            <span className="hidden items-center gap-1.5 text-sm font-semibold text-amber-800 sm:flex">
              {t.systemConsole} ({t.allTenants})
            </span>
          </div>
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <button
                  type="button"
                  className="flex items-center gap-3 rounded-lg px-2 py-1.5 text-left transition-colors hover:bg-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500/30"
                  aria-label={t.accountMenu}
                />
              }
            >
              <div className="hidden text-right sm:block">
                <p className="text-sm font-semibold text-slate-800">
                  {actorUsername ?? 'system administrator'}
                </p>
                <p className="text-xs text-slate-500">{t.systemAdministrator}</p>
              </div>
              <span className="flex size-9 items-center justify-center rounded-full bg-amber-500 text-sm font-semibold text-white">
                {(actorUsername ?? 'S').slice(0, 1).toUpperCase()}
              </span>
              <IconChevronDown size={15} className="text-slate-400" aria-hidden="true" />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuLabel>
                <p className="text-xs font-medium text-slate-500">{t.signedInAs}</p>
                <p className="mt-0.5 text-sm font-semibold text-slate-900">
                  {actorUsername ?? 'system administrator'}
                </p>
              </DropdownMenuLabel>
              <DropdownMenuSeparator className="my-1 h-px bg-slate-200" />
              <DropdownMenuItem render={<Link to="/admin" preload={false} />}>
                <IconArrowLeft size={17} aria-hidden="true" />
                {t.returnToAdminConsole}
              </DropdownMenuItem>
              <DropdownMenuSeparator className="my-1 h-px bg-slate-200" />
              <DropdownMenuItem
                nativeButton
                render={
                  <button
                    type="button"
                    onClick={() => {
                      void logout('admin')
                    }}
                    className="w-full text-left text-red-700"
                  />
                }
              >
                <IconLogout size={17} aria-hidden="true" />
                {t.signOut}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </header>

      <div className="grid min-h-[calc(100vh-4rem)] lg:grid-cols-[248px_minmax(0,1fr)]">
        <aside className="app-sidebar">
          <nav className="flex flex-1 flex-col gap-1 p-4" aria-label={t.systemNavigation}>
            <Link
              to="/admin"
              preload={false}
              className="mb-2 flex h-9 w-full items-center gap-3 rounded-md px-3 text-left text-sm font-medium text-slate-500 transition-colors hover:text-slate-950"
            >
              <IconArrowLeft size={18} stroke={1.8} aria-hidden="true" />
              {t.adminConsole}
            </Link>
            {items.map((item) => (
              <Link
                key={item.key}
                to={item.href}
                className={cn(
                  'flex h-9 w-full items-center gap-3 rounded-md border-l-2 px-3 text-left text-sm transition-colors',
                  item.active
                    ? 'border-amber-500 font-semibold text-slate-950'
                    : 'border-transparent font-medium text-slate-600 hover:text-slate-950',
                )}
                aria-current={item.active ? 'page' : undefined}
              >
                <item.icon size={18} stroke={1.8} aria-hidden="true" />
                {item.label}
              </Link>
            ))}
          </nav>
        </aside>

        <main className="app-main">
          <div className="app-content max-w-[1500px]">
            <div className="flex flex-wrap items-start justify-between gap-4">
              <div>
                <nav aria-label={t.breadcrumb}>
                  <ol className="flex items-center gap-2 text-xs font-semibold text-slate-500">
                    <li>{t.systemConsole}</li>
                    <li aria-hidden="true">/</li>
                    <li aria-current="page">{currentItem?.label ?? title}</li>
                  </ol>
                </nav>
                <h1 className="app-page-title mt-2">{title}</h1>
                {description ? (
                  <p className="mt-2 max-w-[70ch] text-sm text-slate-600">{description}</p>
                ) : null}
              </div>
              {actions ? <div className="flex items-center gap-2">{actions}</div> : null}
            </div>
            {children}
          </div>
        </main>
      </div>
    </div>
  )
}
