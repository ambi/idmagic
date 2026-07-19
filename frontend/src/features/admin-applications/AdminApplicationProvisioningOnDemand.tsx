import { type FormEvent, useState } from 'react'
import { provisionOnDemand, startAdminApplicationProvisioningFullResync } from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Select } from '../../components/ui/select'
import { useDictionary } from '../../lib/i18n'
import { messageOf, SectionTitle } from './AdminApplicationsShared'
import { provisioningDictionary } from './AdminApplicationProvisioning.i18n'
import type { ProvisioningSourceType } from '../../types'

export function OnDemandAndResyncPanel({
  csrfToken,
  applicationID,
}: {
  csrfToken: string
  applicationID: string
}) {
  const t = useDictionary(provisioningDictionary)
  const [subjectType, setSubjectType] = useState<ProvisioningSourceType>('user')
  const [subjectID, setSubjectID] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [resyncing, setResyncing] = useState(false)

  async function submitOnDemand(e: FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setError('')
    setNotice('')
    try {
      await provisionOnDemand(csrfToken, applicationID, subjectType, subjectID)
      setNotice(t.onDemandEnqueuedNotice)
      setSubjectID('')
    } catch (cause) {
      setError(messageOf(cause, t.onDemandFailedError))
    } finally {
      setSubmitting(false)
    }
  }

  async function fullResync() {
    setResyncing(true)
    setError('')
    setNotice('')
    try {
      const res = await startAdminApplicationProvisioningFullResync(csrfToken, applicationID)
      setNotice(t.fullResyncEnqueuedNotice.replace('{count}', String(res.enqueued_count)))
    } catch (cause) {
      setError(messageOf(cause, t.fullResyncFailedError))
    } finally {
      setResyncing(false)
    }
  }

  return (
    <Card className="grid gap-6 p-5">
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      {notice ? <Alert variant="success">{notice}</Alert> : null}
      <section className="grid gap-3">
        <SectionTitle>{t.onDemandHeading}</SectionTitle>
        <p className="text-xs text-slate-500">{t.onDemandHelp}</p>
        <form className="flex flex-wrap items-end gap-3" onSubmit={(e) => void submitOnDemand(e)}>
          <div className="grid gap-1.5">
            <Label>{t.onDemandSubjectTypeFieldLabel}</Label>
            <Select
              value={subjectType}
              onValueChange={(v) => setSubjectType(v as ProvisioningSourceType)}
              options={[
                { value: 'user', label: t.onDemandSubjectTypeUser },
                { value: 'group', label: t.onDemandSubjectTypeGroup },
              ]}
            />
          </div>
          <div className="grid gap-1.5">
            <Label>{t.onDemandSubjectIdFieldLabel}</Label>
            <Input required value={subjectID} onChange={(e) => setSubjectID(e.target.value)} />
          </div>
          <Button type="submit" disabled={submitting || subjectID.trim() === ''}>
            {submitting ? t.onDemandSubmitting : t.onDemandButton}
          </Button>
        </form>
      </section>
      <section className="grid gap-3 border-t border-slate-100 pt-5">
        <SectionTitle>{t.fullResyncHeading}</SectionTitle>
        <p className="text-xs text-slate-500">{t.fullResyncHelp}</p>
        <div>
          <Button
            type="button"
            variant="outline"
            disabled={resyncing}
            onClick={() => void fullResync()}
          >
            {resyncing ? t.fullResyncSubmitting : t.fullResyncButton}
          </Button>
        </div>
      </section>
    </Card>
  )
}
