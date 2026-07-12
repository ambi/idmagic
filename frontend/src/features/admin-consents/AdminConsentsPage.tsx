import { IconBan, IconRefresh } from '@tabler/icons-react'
import { useMemo, useState } from 'react'
import { AuthenticationAPIError, listAdminConsents, revokeAdminConsent } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Toast } from '../../components/ui/toast'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { useDictionary, useLocale } from '../../lib/i18n'
import type { AdminConsent } from '../../types'
import { adminConsentsDictionary } from './AdminConsentsPage.i18n'

export function filterAdminConsents(consents: AdminConsent[], query: string): AdminConsent[] {
  const needle = query.trim().toLowerCase()
  if (!needle) return consents
  return consents.filter((consent) =>
    [
      consent.user_id,
      consent.preferred_username ?? '',
      consent.client_id,
      consent.client_name,
      consent.state,
      ...consent.scopes,
    ].some((value) => value.toLowerCase().includes(needle)),
  )
}

export function AdminConsentsPage({
  csrfToken,
  actorUsername,
  consents: initial,
}: {
  csrfToken: string
  actorUsername?: string
  consents: AdminConsent[]
}) {
  const [consents, setConsents] = useState(initial)
  const [query, setQuery] = useState('')
  const [selected, setSelected] = useState<AdminConsent | null>(initial[0] ?? null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [confirmTarget, setConfirmTarget] = useState<AdminConsent | null>(null)
  const t = useDictionary(adminConsentsDictionary)
  const { locale } = useLocale()

  const filtered = useMemo(() => filterAdminConsents(consents, query), [consents, query])

  async function refresh(preferred?: AdminConsent) {
    const next = await listAdminConsents()
    setConsents(next)
    const match = preferred
      ? next.find((c) => c.user_id === preferred.user_id && c.client_id === preferred.client_id)
      : null
    setSelected(match ?? next[0] ?? null)
  }

  async function run(action: () => Promise<void>, success: string) {
    setBusy(true)
    setError('')
    setNotice('')
    try {
      await action()
      setNotice(success)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.genericActionError)
    } finally {
      setBusy(false)
    }
  }

  async function handleRevoke(target: AdminConsent) {
    await run(async () => {
      await revokeAdminConsent(csrfToken, target.user_id, target.client_id)
      await refresh(target)
    }, t.revokedNotice)
    setConfirmTarget(null)
  }

  return (
    <AdminShell
      active="consents"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Toast message={notice} onDismiss={() => setNotice('')} />

      <Card className="flex flex-col gap-3 p-4 md:flex-row md:items-center md:justify-between">
        <Input
          placeholder={t.filterPlaceholder}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          className="max-w-md"
        />
        <Button
          variant="outline"
          className="size-9 shrink-0 px-0"
          aria-label={t.reloadAriaLabel}
          disabled={busy}
          onClick={() => run(() => refresh(selected ?? undefined), t.listRefreshedNotice)}
        >
          <IconRefresh size={16} aria-hidden="true" />
        </Button>
      </Card>

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_360px]">
        <Card className="overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
              <tr>
                <th className="px-4 py-3">{t.tableHeaderUser}</th>
                <th className="px-4 py-3">{t.tableHeaderApplication}</th>
                <th className="px-4 py-3">{t.tableHeaderStatus}</th>
                <th className="px-4 py-3">{t.tableHeaderGranted}</th>
                <th className="px-4 py-3" />
              </tr>
            </thead>
            <tbody>
              {filtered.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-4 py-12 text-center text-sm text-slate-500">
                    {t.noMatchingConsentsNotice}
                  </td>
                </tr>
              ) : null}
              {filtered.map((c) => (
                <tr
                  key={`${c.user_id}:${c.client_id}`}
                  onClick={() => setSelected(c)}
                  className={`cursor-pointer border-t border-slate-100 hover:bg-slate-50 ${
                    selected?.user_id === c.user_id && selected.client_id === c.client_id
                      ? 'bg-blue-50/60'
                      : ''
                  }`}
                >
                  <td className="px-4 py-3">
                    <NameWithId name={c.preferred_username ?? c.user_id} id={c.user_id} />
                  </td>
                  <td className="px-4 py-3">
                    <NameWithId name={c.client_name} id={c.client_id} />
                  </td>
                  <td className="px-4 py-3">
                    <ConsentStateBadge state={c.state} />
                  </td>
                  <td className="px-4 py-3 text-xs text-slate-500">
                    {formatConsentDate(c.granted_at, locale)}
                  </td>
                  <td className="px-4 py-3 text-right">
                    {c.state === 'granted' ? (
                      <Button
                        variant="ghost"
                        className="text-rose-700 hover:bg-rose-50"
                        disabled={busy}
                        onClick={(e) => {
                          e.stopPropagation()
                          setConfirmTarget(c)
                        }}
                      >
                        <IconBan size={16} aria-hidden="true" />
                        {t.revoke}
                      </Button>
                    ) : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </Card>

        <Card className="p-5">
          <h2 className="text-sm font-semibold text-slate-700">{t.detailsHeading}</h2>
          {selected ? (
            <dl className="mt-4 grid grid-cols-[110px_minmax(0,1fr)] gap-y-3 text-sm">
              <dt className="text-slate-500">{t.tableHeaderUser}</dt>
              <dd>
                <NameWithId
                  name={selected.preferred_username ?? selected.user_id}
                  id={selected.user_id}
                />
              </dd>
              <dt className="text-slate-500">{t.tableHeaderApplication}</dt>
              <dd>
                <NameWithId name={selected.client_name} id={selected.client_id} />
              </dd>
              <dt className="text-slate-500">{t.scopesLabel}</dt>
              <dd className="flex flex-wrap gap-1">
                {selected.scopes.length === 0 ? (
                  <span className="text-slate-400">{t.noneLabel}</span>
                ) : (
                  selected.scopes.map((scope) => (
                    <span
                      key={scope}
                      className="rounded-md bg-slate-100 px-1.5 py-0.5 font-mono text-[11px] text-slate-700"
                    >
                      {scope}
                    </span>
                  ))
                )}
              </dd>
              <dt className="text-slate-500">{t.tableHeaderStatus}</dt>
              <dd>
                <ConsentStateBadge state={selected.state} />
              </dd>
              <dt className="text-slate-500">{t.tableHeaderGranted}</dt>
              <dd>{formatConsentDate(selected.granted_at, locale)}</dd>
              <dt className="text-slate-500">{t.expiresLabel}</dt>
              <dd>{formatConsentDate(selected.expires_at, locale)}</dd>
              {selected.revoked_at ? (
                <>
                  <dt className="text-slate-500">{t.revokedLabel}</dt>
                  <dd>{formatConsentDate(selected.revoked_at, locale)}</dd>
                </>
              ) : null}
            </dl>
          ) : (
            <p className="mt-4 text-sm text-slate-500">{t.selectConsentPrompt}</p>
          )}
        </Card>
      </div>

      {confirmTarget ? (
        <ConfirmDialog
          title={t.revokeConsentTitle}
          message={t.revokeConsentMessage
            .replace('{user}', confirmTarget.preferred_username ?? confirmTarget.user_id)
            .replace('{client}', confirmTarget.client_name)}
          confirmLabel={t.confirmRevoke}
          onCancel={() => setConfirmTarget(null)}
          onConfirm={() => handleRevoke(confirmTarget)}
          busy={busy}
        />
      ) : null}
    </AdminShell>
  )
}

// 可読名を主表示にし、UUID (client_id / user_id) は補助表記に留める (wi-141)。
// name が id と一致する場合 (解決名なしのフォールバック) は id 行を重複表示しない。
export function NameWithId({ name, id }: { name: string; id: string }) {
  return (
    <div className="min-w-0">
      <span className="block truncate text-sm text-slate-800" title={id}>
        {name}
      </span>
      {name === id ? null : (
        <span className="block truncate font-mono text-[11px] text-slate-400" title={id}>
          {id}
        </span>
      )}
    </div>
  )
}

export function ConsentStateBadge({ state }: { state: AdminConsent['state'] }) {
  const variants: Record<AdminConsent['state'], string> = {
    granted: 'bg-emerald-50 text-emerald-700',
    revoked: 'bg-rose-50 text-rose-700',
    expired: 'bg-amber-50 text-amber-700',
  }
  return (
    <span className={`rounded-md px-2 py-0.5 text-xs font-semibold ${variants[state]}`}>
      {state}
    </span>
  )
}

export function formatConsentDate(value?: string, locale: 'ja' | 'en' = 'en'): string {
  if (!value) return '—'
  try {
    return new Date(value).toLocaleString(locale === 'ja' ? 'ja-JP' : 'en-US')
  } catch {
    return value
  }
}

function ConfirmDialog({
  title,
  message,
  confirmLabel,
  onCancel,
  onConfirm,
  busy,
}: {
  title: string
  message: string
  confirmLabel: string
  onCancel: () => void
  onConfirm: () => void
  busy: boolean
}) {
  const t = useDictionary(adminConsentsDictionary)
  return (
    <div className="fixed inset-0 z-40 flex items-center justify-center bg-slate-900/40 px-4">
      <Card className="w-full max-w-md p-6">
        <h2 className="text-base font-semibold text-slate-900">{title}</h2>
        <p className="mt-3 text-sm text-slate-600">{message}</p>
        <div className="mt-5 flex justify-end gap-2">
          <Button variant="outline" onClick={onCancel} disabled={busy}>
            {t.cancel}
          </Button>
          <Button onClick={onConfirm} disabled={busy} className="bg-rose-600 hover:bg-rose-700">
            {confirmLabel}
          </Button>
        </div>
      </Card>
    </div>
  )
}
