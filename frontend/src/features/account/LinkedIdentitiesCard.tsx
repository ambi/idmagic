import { IconLink, IconTrash } from '@tabler/icons-react'
import { useState } from 'react'
import { AuthenticationAPIError, unlinkIdentity, type LinkedIdentity } from '../../api'
import { StepUpCancelledError, useStepUpGuard } from '../../components/StepUpDialog'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Toast } from '../../components/ui/toast'
import { useDictionary, useLocale } from '../../lib/i18n'
import { accountSecurityDictionary } from './AccountSecurityPage.i18n'

function formatLinkedAt(value: string, locale: string, noRecord: string): string {
  if (!value) return noRecord
  return new Date(value).toLocaleString(locale, { dateStyle: 'medium', timeStyle: 'short' })
}

export function LinkedIdentitiesCard({
  csrfToken,
  initialIdentities,
}: {
  csrfToken: string
  initialIdentities: LinkedIdentity[]
}) {
  const t = useDictionary(accountSecurityDictionary)
  const { locale } = useLocale()
  const [identities, setIdentities] = useState(initialIdentities)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const { guard, dialog } = useStepUpGuard(csrfToken)

  async function handleUnlink(providerID: string) {
    setBusy(true)
    setError('')
    setNotice('')
    try {
      await guard(() => unlinkIdentity(csrfToken, providerID))
      setIdentities((current) => current.filter((identity) => identity.provider_id !== providerID))
      setNotice(t.identityUnlinked)
    } catch (cause) {
      if (cause instanceof StepUpCancelledError) return
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.identityUnlinkFailed)
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <Toast message={notice} onDismiss={() => setNotice('')} />
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Card className="flex flex-col gap-4 p-5">
        <div className="flex items-start gap-3">
          <span className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-slate-600">
            <IconLink size={20} aria-hidden="true" />
          </span>
          <div className="min-w-0">
            <p className="text-sm font-semibold text-slate-900">{t.linkedIdentities}</p>
            <p className="mt-1 text-sm text-slate-600">{t.linkedIdentitiesDescription}</p>
          </div>
        </div>
        {identities.length === 0 ? (
          <p className="border-t border-slate-100 pt-4 text-sm text-slate-500">
            {t.noLinkedIdentities}
          </p>
        ) : (
          <ul className="flex flex-col gap-2 border-t border-slate-100 pt-4">
            {identities.map((identity) => (
              <li
                key={identity.provider_id}
                className="flex items-center justify-between gap-3 rounded-lg border border-slate-200 px-3.5 py-2.5"
              >
                <div>
                  <p className="text-sm font-medium text-slate-800">{identity.provider_id}</p>
                  <p className="mt-0.5 text-xs text-slate-500">
                    {t.linkedAt.replace(
                      '{date}',
                      formatLinkedAt(identity.linked_at, locale, t.noRecord),
                    )}
                  </p>
                </div>
                <Button
                  type="button"
                  variant="ghost"
                  className="h-8 shrink-0 px-2 text-red-600 hover:bg-red-50"
                  disabled={busy}
                  aria-label={t.unlinkIdentity.replace('{provider}', identity.provider_id)}
                  onClick={() => void handleUnlink(identity.provider_id)}
                >
                  <IconTrash size={16} aria-hidden="true" />
                  {t.remove}
                </Button>
              </li>
            ))}
          </ul>
        )}
      </Card>
      {dialog}
    </>
  )
}
