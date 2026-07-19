import { useState } from 'react'
import { resumeAdminApplicationProvisioning, testAdminApplicationProvisioning } from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary, useLocale } from '../../lib/i18n'
import { messageOf, SectionTitle } from './AdminApplicationsShared'
import { provisioningDictionary } from './AdminApplicationProvisioning.i18n'
import { formatDate } from './AdminApplicationProvisioningShared'
import type { ProvisioningConnection } from '../../types'

export function ConnectionStatusPanel({
  csrfToken,
  applicationID,
  connection,
  onChanged,
}: {
  csrfToken: string
  applicationID: string
  connection: ProvisioningConnection
  onChanged: (conn: ProvisioningConnection) => void
}) {
  const t = useDictionary(provisioningDictionary)
  const { locale } = useLocale()
  const [resuming, setResuming] = useState(false)
  const [error, setError] = useState('')

  async function resume() {
    setResuming(true)
    setError('')
    try {
      onChanged(await resumeAdminApplicationProvisioning(csrfToken, applicationID))
    } catch (cause) {
      setError(messageOf(cause, t.resumeFailedError))
    } finally {
      setResuming(false)
    }
  }

  const healthLabel =
    connection.health === 'ok'
      ? t.healthOkLabel
      : connection.health === 'degraded'
        ? t.healthDegradedLabel
        : t.healthQuarantinedLabel
  const healthBadge =
    connection.health === 'ok'
      ? 'bg-emerald-50 text-emerald-700'
      : connection.health === 'degraded'
        ? 'bg-amber-50 text-amber-700'
        : 'bg-red-50 text-red-700'

  return (
    <Card className="grid gap-3 p-5">
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <div className="flex flex-wrap items-center gap-2">
        <span
          className={`rounded-md px-2 py-0.5 text-xs font-medium ${
            connection.status === 'active'
              ? 'bg-emerald-50 text-emerald-700'
              : 'bg-slate-100 text-slate-500'
          }`}
        >
          {connection.status === 'active' ? t.statusActiveLabel : t.statusDisabledLabel}
        </span>
        <span className={`rounded-md px-2 py-0.5 text-xs font-medium ${healthBadge}`}>
          {healthLabel}
        </span>
        <span className="text-xs text-slate-400">
          {t.lastUpdatedAtLabel}: {formatDate(connection.updated_at, locale, t.unknownDate)}
        </span>
      </div>
      {connection.health === 'quarantined' ? (
        <Alert variant="destructive" className="grid gap-2">
          <p>{t.quarantinedNotice}</p>
          {connection.quarantine_reason ? (
            <p className="text-xs">
              {t.quarantineReasonLabel}: {connection.quarantine_reason}
            </p>
          ) : null}
          <div>
            <Button
              type="button"
              variant="destructive"
              disabled={resuming}
              onClick={() => void resume()}
            >
              {resuming ? t.resuming : t.resumeButton}
            </Button>
          </div>
        </Alert>
      ) : null}
    </Card>
  )
}

export function TestConnectionPanel({
  csrfToken,
  applicationID,
  connection,
  onChanged,
}: {
  csrfToken: string
  applicationID: string
  connection: ProvisioningConnection
  onChanged: (conn: ProvisioningConnection) => void
}) {
  const t = useDictionary(provisioningDictionary)
  const { locale } = useLocale()
  const [testing, setTesting] = useState(false)
  const [result, setResult] = useState<{ reachable: boolean; error?: string } | null>(null)

  async function test() {
    setTesting(true)
    setResult(null)
    try {
      const res = await testAdminApplicationProvisioning(csrfToken, applicationID)
      setResult({ reachable: res.reachable, error: res.error })
      if (res.reachable) {
        onChanged({ ...connection, capabilities: res.capabilities ?? connection.capabilities })
      }
    } catch (cause) {
      setResult({ reachable: false, error: messageOf(cause, t.testUnreachableError) })
    } finally {
      setTesting(false)
    }
  }

  const caps = connection.capabilities

  return (
    <Card className="grid gap-3 p-5">
      <div className="flex items-center justify-between">
        <SectionTitle>{t.capabilitiesHeading}</SectionTitle>
        <Button type="button" variant="outline" disabled={testing} onClick={() => void test()}>
          {testing ? t.testing : t.testConnectionButton}
        </Button>
      </div>
      {result ? (
        <Alert variant={result.reachable ? 'success' : 'destructive'}>
          {result.reachable
            ? t.testReachableNotice
            : t.testUnreachableError.replace('{error}', result.error ?? '')}
        </Alert>
      ) : null}
      {caps ? (
        <div className="flex flex-wrap gap-2 text-xs">
          {(
            [
              ['supports_patch', t.capabilitySupportsPatch],
              ['supports_bulk', t.capabilitySupportsBulk],
              ['supports_filter', t.capabilitySupportsFilter],
              ['supports_etag', t.capabilitySupportsEtag],
              ['supports_sort', t.capabilitySupportsSort],
            ] as [keyof typeof caps, string][]
          ).map(([key, label]) => (
            <span
              key={key}
              className={`rounded px-2 py-0.5 ${caps[key] ? 'bg-blue-50 text-blue-700' : 'bg-slate-100 text-slate-400'}`}
            >
              {label}
            </span>
          ))}
          <span className="text-slate-400">
            {formatDate(caps.discovered_at, locale, t.unknownDate)}
          </span>
        </div>
      ) : (
        <p className="text-xs text-slate-400">{t.notDiscoveredYetNotice}</p>
      )}
    </Card>
  )
}
