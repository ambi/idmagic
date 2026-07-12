import { IconApps, IconTrash } from '@tabler/icons-react'
import { useState } from 'react'
import { AuthenticationAPIError, revokeAccountConsent } from '../../api'
import { AccountShell } from '../../components/AccountShell'
import { Alert } from '../../components/ui/alert'
import { Toast } from '../../components/ui/toast'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import type { AccountConsent } from '../../types'
import { useDictionary, useFormatters } from '../../lib/i18n'
import { accountApplicationsDictionary } from './AccountApplicationsPage.i18n'

export function formatAccountConsentDate(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return date.toLocaleDateString('ja-JP', { year: 'numeric', month: 'long', day: 'numeric' })
}

export function AccountApplicationsPage({
  csrfToken,
  username,
  consents: initial,
  isAdmin,
}: {
  csrfToken: string
  username: string
  consents: AccountConsent[]
  isAdmin: boolean
}) {
  const t = useDictionary(accountApplicationsDictionary)
  const [consents, setConsents] = useState<AccountConsent[]>(initial)
  const [pending, setPending] = useState('')
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  async function handleRevoke(consent: AccountConsent) {
    const clientId = consent.client_id
    setPending(clientId)
    setError('')
    setNotice('')
    try {
      await revokeAccountConsent(csrfToken, clientId)
      setConsents((current) => current.filter((c) => c.client_id !== clientId))
      setNotice(t.revokeNotice.replace('{name}', consent.client_name))
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.revokeFailed)
    } finally {
      setPending('')
    }
  }

  return (
    <AccountApplicationsPresentation
      username={username}
      consents={consents}
      isAdmin={isAdmin}
      pending={pending}
      error={error}
      notice={notice}
      onDismissNotice={() => setNotice('')}
      onRevoke={handleRevoke}
    />
  )
}

export function AccountApplicationsPresentation({
  username,
  consents,
  isAdmin,
  pending,
  error,
  notice,
  onDismissNotice,
  onRevoke,
}: {
  username: string
  consents: AccountConsent[]
  isAdmin: boolean
  pending: string
  error: string
  notice: string
  onDismissNotice: () => void
  onRevoke: (consent: AccountConsent) => void
}) {
  const t = useDictionary(accountApplicationsDictionary)
  const { formatDate } = useFormatters()
  return (
    <AccountShell
      active="applications"
      username={username}
      isAdmin={isAdmin}
      title={t.title}
      description={t.description}
    >
      <Toast message={notice} onDismiss={onDismissNotice} />
      {error ? <Alert variant="destructive">{error}</Alert> : null}

      {consents.length === 0 ? (
        <Card className="flex flex-col items-center gap-2 p-10 text-center">
          <IconApps size={28} className="text-slate-300" aria-hidden="true" />
          <p className="text-sm text-slate-500">{t.empty}</p>
        </Card>
      ) : (
        <div className="grid gap-3">
          {consents.map((consent) => (
            <Card key={consent.client_id} className="flex flex-wrap items-start gap-4 p-5">
              <span className="flex size-11 shrink-0 items-center justify-center rounded-xl border border-blue-100 bg-blue-50 text-sm font-bold text-blue-700">
                {consent.client_name.slice(0, 2).toUpperCase()}
              </span>
              <div className="min-w-0 flex-1">
                <p
                  className="truncate text-sm font-semibold text-slate-900"
                  title={consent.client_id}
                >
                  {consent.client_name}
                </p>
                <p className="mt-0.5 text-xs text-slate-500">
                  {t.granted.replace('{date}', formatDate(consent.granted_at))}
                </p>
                <div className="mt-2 flex flex-wrap gap-1.5">
                  {consent.scopes.map((scope) => (
                    <span
                      key={scope}
                      className="rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs text-slate-600"
                    >
                      {scope}
                    </span>
                  ))}
                </div>
              </div>
              <Button
                type="button"
                variant="outline"
                className="text-red-700 hover:bg-red-50"
                disabled={pending === consent.client_id}
                onClick={() => onRevoke(consent)}
              >
                <IconTrash size={16} aria-hidden="true" />
                {pending === consent.client_id ? t.revoking : t.revoke}
              </Button>
            </Card>
          ))}
        </div>
      )}
    </AccountShell>
  )
}
