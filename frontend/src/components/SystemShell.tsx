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
      <header className="app-header border-b-2 border-amber-400/70">
        <div className="flex h-16 items-center justify-between px-5 lg:px-7">
          <div className="flex items-center gap-5">
            <Link
              to="/system/keys"
              aria-label={t.systemConsole}
              className="rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500/30"
            >
              <Brand compact />
            </Link>
            <div className="hidden h-6 w-px bg-slate-200/80 sm:block" />
            <div className="hidden items-center gap-2 rounded-lg border border-amber-300 bg-amber-50 px-2.5 py-1.5 text-sm font-semibold text-amber-800 sm:flex">
              <span className="flex size-7 items-center justify-center rounded-md bg-amber-500 text-xs font-bold text-white">
                SYS
              </span>
              {t.systemConsole} ({t.allTenants})
            </div>
          </div>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                type="button"
                className="flex items-center gap-3 rounded-lg px-2 py-1.5 text-left transition-colors hover:bg-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500/30"
                aria-label={t.accountMenu}
              >
                <div className="hidden text-right sm:block">
                  <p className="text-sm font-semibold text-slate-800">
                    {actorUsername ?? 'system administrator'}
                  </p>
                  <p className="text-xs text-slate-500">{t.systemAdministrator}</p>
                </div>
                <span className="flex size-9 items-center justify-center rounded-lg bg-amber-500 text-sm font-semibold text-white shadow-sm">
                  {(actorUsername ?? 'S').slice(0, 1).toUpperCase()}
                </span>
                <IconChevronDown size={15} className="text-slate-400" aria-hidden="true" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuLabel>
                <p className="text-xs font-medium text-slate-500">{t.signedInAs}</p>
                <p className="mt-0.5 text-sm font-semibold text-slate-900">
                  {actorUsername ?? 'system administrator'}
                </p>
              </DropdownMenuLabel>
              <DropdownMenuSeparator className="my-1 h-px bg-slate-200" />
              <DropdownMenuItem asChild>
                <Link to="/admin" preload={false}>
                  <IconArrowLeft size={17} aria-hidden="true" />
                  {t.returnToAdminConsole}
                </Link>
              </DropdownMenuItem>
              <DropdownMenuSeparator className="my-1 h-px bg-slate-200" />
              <DropdownMenuItem asChild>
                <button
                  type="button"
                  onClick={() => {
                    void logout('admin')
                  }}
                  className="w-full text-left text-red-700"
                >
                  <IconLogout size={17} aria-hidden="true" />
                  {t.signOut}
                </button>
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
              className="mb-2 flex h-10 w-full items-center gap-3 rounded-lg px-3 text-left text-sm font-medium text-slate-500 transition-colors hover:bg-white hover:text-slate-950"
            >
              <IconArrowLeft size={18} stroke={1.8} aria-hidden="true" />
              {t.adminConsole}
            </Link>
            {items.map((item) => (
              <Link
                key={item.key}
                to={item.href}
                className={cn(
                  'flex h-10 w-full items-center gap-3 rounded-lg px-3 text-left text-sm font-medium transition-[background-color,color,box-shadow]',
                  item.active
                    ? 'bg-amber-500 text-white shadow-sm'
                    : 'text-slate-600 hover:bg-white hover:text-slate-950 hover:shadow-xs',
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
