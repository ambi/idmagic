import { IconCircleCheckFilled, IconHelpCircle, IconLock, IconSparkles } from '@tabler/icons-react'
import type { ReactNode } from 'react'
import { tenantBrandStyle, useTenantBranding } from '../lib/useTenantBranding'
import { useDictionary } from '../lib/i18n'
import { Brand } from './Brand'
import { LanguageSwitcher } from './LanguageSwitcher'
import { shellDictionary } from './shell.i18n'

type AuthShellProps = {
  children: ReactNode
  asideTitle?: string
  asideText?: string
  // aside=false で左のプロモ枠を出さない (認証済み self-service 画面向け)。
  aside?: boolean
}

export function AuthShell({ children, asideTitle, asideText, aside = true }: AuthShellProps) {
  const t = useDictionary(shellDictionary)
  const assuranceItems = [t.secureSignIn, t.leastPrivilege, t.auditableIdentity]
  const branding = useTenantBranding()
  const footerLinks = [branding.footer_link_1, branding.footer_link_2].filter(
    (link): link is { label: string; url: string } => Boolean(link?.label && link.url),
  )

  return (
    <div className="auth-background" style={tenantBrandStyle(branding)}>
      <div className="auth-container">
        <div className={aside ? 'auth-frame' : 'auth-frame auth-frame--solo'}>
          {aside ? (
            <aside className="auth-aside">
              <Brand inverse productName={branding.product_name} logoURL={branding.logo_url} />

              <div className="auth-aside-copy">
                <div className="flex w-fit items-center gap-2 text-xs font-semibold uppercase tracking-normal text-teal-300">
                  <IconSparkles size={14} aria-hidden="true" />
                  {t.identityControl}
                </div>
                <div className="flex flex-col gap-4">
                  <h1 className="aside-title">{asideTitle ?? t.defaultAuthAsideTitle}</h1>
                  <p className="aside-text">{asideText ?? t.defaultAuthAsideText}</p>
                </div>
                <ul className="grid gap-2.5" aria-label={t.security}>
                  {assuranceItems.map((item) => (
                    <li key={item} className="flex items-center gap-2.5 text-sm text-slate-300">
                      <IconCircleCheckFilled
                        size={16}
                        className="shrink-0 text-teal-300"
                        aria-hidden="true"
                      />
                      {item}
                    </li>
                  ))}
                </ul>
              </div>

              <div className="flex items-center justify-between border-t border-white/10 pt-5 text-xs text-slate-400">
                <span className="flex items-center gap-2">
                  <IconLock size={14} aria-hidden="true" />
                  {t.protectedConnection}
                </span>
                <span>OpenID Connect / OAuth 2.0</span>
              </div>
            </aside>
          ) : null}

          <main className="auth-main">
            <div className="auth-main-body">
              <div className="mobile-brand text-slate-950">
                <Brand productName={branding.product_name} logoURL={branding.logo_url} />
              </div>
              <div className="mb-7 flex items-center justify-between">
                <span className="flex items-center gap-2 text-xs font-medium text-slate-500">
                  <IconLock size={14} aria-hidden="true" />
                  {t.secureAuthentication}
                </span>
                <LanguageSwitcher />
              </div>

              {children}

              <footer className="mt-9 flex flex-wrap items-center justify-center gap-x-3 gap-y-2 border-t border-slate-100 pt-5 text-xs text-slate-500">
                <span className="flex items-center gap-1.5">
                  <IconHelpCircle size={14} aria-hidden="true" />
                  {t.contactAdministrator}
                </span>
                {footerLinks.map((link) => (
                  <a
                    key={`${link.label}:${link.url}`}
                    href={link.url}
                    target="_blank"
                    rel="noreferrer noopener"
                    className="font-medium text-accent-foreground hover:underline"
                  >
                    {link.label}
                  </a>
                ))}
                {branding.footer_text ? <span>{branding.footer_text}</span> : null}
              </footer>
            </div>
          </main>
        </div>
      </div>
    </div>
  )
}
