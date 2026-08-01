import { IconKey } from '@tabler/icons-react'
import { useState } from 'react'
import { issueApplicationClientSecret, revokeApplicationClientSecret } from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Label } from '../../components/ui/label'
import { useDictionary, useLocale } from '../../lib/i18n'
import type { ClientSecretCredentialMetadata } from '../../types'
import { adminApplicationsDictionary } from './AdminApplicationsPage.i18n'
import { CopyableField, messageOf } from './AdminApplicationsShared'

const EXPIRY_DAY_OPTIONS = [30, 90, 180, 365] as const

export function ClientSecretRotationPanel({
  applicationID,
  csrfToken,
  initialCredentials,
  onError,
}: {
  applicationID: string
  csrfToken: string
  initialCredentials: ClientSecretCredentialMetadata[]
  onError: (message: string) => void
}) {
  const [credentials, setCredentials] = useState(initialCredentials)
  const [expiresInDays, setExpiresInDays] = useState(90)
  const [issuing, setIssuing] = useState(false)
  const [revokingID, setRevokingID] = useState('')
  const [confirmingID, setConfirmingID] = useState('')
  const [secret, setSecret] = useState('')
  const [copied, setCopied] = useState(false)
  const t = useDictionary(adminApplicationsDictionary)
  const { locale } = useLocale()
  const activeCount = credentials.filter((credential) => credential.status === 'Active').length
  const atLimit = activeCount >= 2

  function formatDate(value: string): string {
    return new Intl.DateTimeFormat(locale === 'ja' ? 'ja-JP' : 'en-US', {
      dateStyle: 'medium',
      timeStyle: 'short',
    }).format(new Date(value))
  }

  function statusLabel(status: ClientSecretCredentialMetadata['status']): string {
    if (status === 'Revoked') return t.secretStatusRevoked
    if (status === 'Expired') return t.secretStatusExpired
    return t.secretStatusActive
  }

  function statusClass(status: ClientSecretCredentialMetadata['status']): string {
    if (status === 'Active') return 'bg-emerald-100 text-emerald-800'
    if (status === 'Expired') return 'bg-amber-100 text-amber-800'
    return 'bg-slate-200 text-slate-700'
  }

  async function issue() {
    setIssuing(true)
    setSecret('')
    setCopied(false)
    onError('')
    try {
      const result = await issueApplicationClientSecret(csrfToken, applicationID, expiresInDays)
      setCredentials(result.credentials)
      setSecret(result.client_secret)
    } catch (cause) {
      onError(messageOf(cause, t.secretIssueError))
    } finally {
      setIssuing(false)
    }
  }

  async function revoke(credentialID: string) {
    setRevokingID(credentialID)
    onError('')
    try {
      const result = await revokeApplicationClientSecret(csrfToken, applicationID, credentialID)
      setCredentials(result.credentials)
      setConfirmingID('')
    } catch (cause) {
      onError(messageOf(cause, t.secretRevokeError))
    } finally {
      setRevokingID('')
    }
  }

  return (
    <div>
      <header className="flex items-start gap-3">
        <span className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-blue-50 text-blue-700">
          <IconKey size={18} aria-hidden="true" />
        </span>
        <div>
          <h2
            id="client-secret-management-heading"
            className="text-base font-semibold text-slate-950"
          >
            {t.secretManagementHeading}
          </h2>
          <p className="mt-1 text-sm leading-6 text-slate-600">{t.secretManagementDescription}</p>
        </div>
      </header>

      <div className="mt-6 flex flex-wrap items-end gap-3 rounded-lg border border-slate-200 bg-slate-50 p-4">
        <div className="grid gap-1.5">
          <Label htmlFor="client-secret-expiry">{t.secretExpiryLabel}</Label>
          <select
            id="client-secret-expiry"
            value={expiresInDays}
            onChange={(event) => setExpiresInDays(Number(event.target.value))}
            disabled={issuing || atLimit}
            className="h-10 rounded-lg border border-slate-300 bg-white px-3 text-sm text-slate-800"
          >
            {EXPIRY_DAY_OPTIONS.map((days) => (
              <option key={days} value={days}>
                {t.secretExpiryDays.replace('{days}', String(days))}
              </option>
            ))}
          </select>
        </div>
        <Button type="button" disabled={issuing || atLimit} onClick={() => void issue()}>
          {t.secretIssueButton}
        </Button>
        {atLimit ? <p className="w-full text-xs text-amber-700">{t.secretLimitNotice}</p> : null}
      </div>

      {secret ? (
        <div className="mt-4 rounded-lg border border-emerald-200 bg-emerald-50 p-4">
          <CopyableField label={t.secretNewSecretLabel} value={secret} />
          <label className="mt-3 flex items-center gap-2 text-sm text-emerald-900">
            <input
              type="checkbox"
              checked={copied}
              onChange={(event) => setCopied(event.target.checked)}
            />
            {t.secretCopiedLabel}
          </label>
          {copied ? (
            <Button type="button" variant="outline" className="mt-3" onClick={() => setSecret('')}>
              {t.secretCloseButton}
            </Button>
          ) : null}
        </div>
      ) : null}

      {credentials.length === 0 ? (
        <p className="mt-6 text-sm text-slate-500">{t.secretNoCredentialsNotice}</p>
      ) : (
        <div className="mt-6 overflow-x-auto rounded-lg border border-slate-200">
          <table className="min-w-full divide-y divide-slate-200 text-left text-sm text-slate-700">
            <thead className="bg-slate-50 text-xs font-semibold text-slate-700">
              <tr>
                <th className="px-4 py-3">{t.secretCredentialIdHeader}</th>
                <th className="px-4 py-3">{t.secretCreatedAtHeader}</th>
                <th className="px-4 py-3">{t.secretExpiresAtHeader}</th>
                <th className="px-4 py-3">{t.secretStatusHeader}</th>
                <th className="px-4 py-3 text-right">{t.secretActionsHeader}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200 bg-white">
              {credentials.map((credential) => (
                <tr key={credential.credential_id}>
                  <td className="px-4 py-3 font-mono text-xs text-slate-800">
                    {credential.credential_id}
                  </td>
                  <td className="whitespace-nowrap px-4 py-3">
                    {formatDate(credential.created_at)}
                  </td>
                  <td className="whitespace-nowrap px-4 py-3">
                    {credential.expires_at
                      ? formatDate(credential.expires_at)
                      : t.secretNeverExpires}
                  </td>
                  <td className="px-4 py-3">
                    <span
                      className={`inline-flex rounded-full px-2 py-1 text-xs font-semibold ${statusClass(credential.status)}`}
                    >
                      {statusLabel(credential.status)}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-right">
                    {credential.status === 'Active' ? (
                      <Button
                        type="button"
                        variant="outline"
                        disabled={revokingID !== ''}
                        onClick={() => setConfirmingID(credential.credential_id)}
                      >
                        {t.secretRevokeButton}
                      </Button>
                    ) : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {confirmingID ? (
        <Alert variant="destructive" className="mt-4 grid gap-3">
          <p>{t.secretRevokeConfirmMessage}</p>
          <div className="flex gap-2">
            <Button
              type="button"
              variant="outline"
              disabled={revokingID !== ''}
              onClick={() => setConfirmingID('')}
            >
              {t.secretRevokeCancelButton}
            </Button>
            <Button
              type="button"
              variant="destructive"
              disabled={revokingID !== ''}
              onClick={() => void revoke(confirmingID)}
            >
              {t.secretRevokeConfirmButton}
            </Button>
          </div>
        </Alert>
      ) : null}
    </div>
  )
}
