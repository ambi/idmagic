import { IconChevronRight, IconMenu2 } from '@tabler/icons-react'
import { Link } from '@tanstack/react-router'
import { tenantRouterPath } from '../api'
import { adminNavGroups, type AdminNavItem, type AdminNavKey } from '../lib/adminNav'
import { useDictionary, useLocale } from '../lib/i18n'
import { cn } from '../lib/utils'
import { shellDictionary } from './shell.i18n'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from './ui/dropdown-menu'

export function AdminNavigation({
  active,
  mobile = false,
}: {
  active: AdminNavKey
  mobile?: boolean
}) {
  const { locale } = useLocale()
  const t = useDictionary(shellDictionary)
  const groups = adminNavGroups(active, locale)
  if (mobile) {
    return (
      <DropdownMenu>
        <DropdownMenuTrigger
          aria-label={t.adminNavigation}
          className="flex size-9 items-center justify-center rounded-md hover:bg-slate-100 lg:hidden"
        >
          <IconMenu2 size={20} aria-hidden="true" />
        </DropdownMenuTrigger>
        <DropdownMenuContent className="w-72" align="start">
          {groups.map((group) => (
            <div key={group.key}>
              <DropdownMenuLabel>{group.label}</DropdownMenuLabel>
              {group.items.map((item) => (
                <NavigationMenuItem key={item.key} item={item} />
              ))}
            </div>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
    )
  }
  return (
    <nav className="flex flex-col gap-1.5 p-4" aria-label={t.adminNavigation}>
      {groups.map((group) => {
        const first = group.items[0]
        if (!first) return null
        if (group.items.length === 1) {
          return <NavigationLink key={group.key} item={first} />
        }
        const current = group.items.some((item) => item.active)
        return (
          <section key={group.key} aria-label={group.label}>
            {current ? (
              <>
                <h2 className="flex h-9 items-center gap-3 px-3 text-xs font-semibold text-slate-500">
                  <first.icon size={18} stroke={1.8} aria-hidden="true" />
                  {group.label}
                </h2>
                <div className="ml-2 border-l border-slate-200 pl-1">
                  {group.items.map((item) => (
                    <NavigationLink key={item.key} item={item} />
                  ))}
                </div>
              </>
            ) : (
              <DropdownMenu modal={false}>
                <DropdownMenuTrigger
                  openOnHover
                  delay={100}
                  className="flex h-10 w-full items-center gap-3 rounded-md px-3 text-left text-sm font-medium text-slate-600 hover:bg-slate-100 hover:text-slate-950 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/30 data-popup-open:bg-slate-100 data-popup-open:text-slate-950"
                >
                  <first.icon size={18} stroke={1.8} aria-hidden="true" />
                  <span className="flex-1">{group.label}</span>
                  <IconChevronRight size={15} aria-hidden="true" />
                </DropdownMenuTrigger>
                <DropdownMenuContent side="right" align="start" className="w-64" sideOffset={8}>
                  <DropdownMenuLabel>{group.label}</DropdownMenuLabel>
                  {group.items.map((item) => (
                    <NavigationMenuItem key={item.key} item={item} />
                  ))}
                </DropdownMenuContent>
              </DropdownMenu>
            )}
          </section>
        )
      })}
    </nav>
  )
}

function NavigationLink({ item }: { item: AdminNavItem }) {
  return (
    <Link
      to={tenantRouterPath(item.href)}
      activeOptions={{ exact: true }}
      title={item.label}
      className={cn(
        'flex min-h-9 min-w-0 flex-1 items-center gap-3 rounded-md px-3 py-2 text-left text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/30',
        item.active
          ? 'bg-accent/10 font-semibold text-slate-950'
          : 'font-medium text-slate-600 hover:bg-slate-100 hover:text-slate-950',
      )}
      aria-current={item.active ? 'page' : undefined}
    >
      <item.icon size={18} stroke={1.8} className="shrink-0" aria-hidden="true" />
      <span className="truncate">{item.label}</span>
    </Link>
  )
}

function NavigationMenuItem({ item }: { item: AdminNavItem }) {
  return (
    <DropdownMenuItem
      render={
        <Link
          to={tenantRouterPath(item.href)}
          activeOptions={{ exact: true }}
          aria-current={item.active ? 'page' : undefined}
        />
      }
      className="min-h-9 cursor-pointer gap-3 px-3"
    >
      <item.icon size={18} stroke={1.8} aria-hidden="true" />
      {item.label}
    </DropdownMenuItem>
  )
}
