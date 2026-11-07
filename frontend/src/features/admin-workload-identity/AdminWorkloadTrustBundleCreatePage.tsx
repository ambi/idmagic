import { IconArrowLeft } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, registerWorkloadTrustBundle } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import { adminWorkloadIdentityDictionary } from './AdminWorkloadIdentityPage.i18n'
import {
  emptyBundleForm,
  listURL,
  TrustBundleFormFields,
  useTrustBundleValidation,
} from './AdminWorkloadIdentityShared'
import { parseInlineJwks } from './presentation'

export function AdminWorkloadTrustBundleCreatePage({
  csrfToken,
  actorUsername,
}: {
  csrfToken: string
  actorUsername?: string
}) {
  const [form, setForm] = useState(emptyBundleForm)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const t = useDictionary(adminWorkloadIdentityDictionary)
  const validation = useTrustBundleValidation(form, t, true)

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setError('')
    const data = validation.data
    if (!data) {
      validation.revealAll()
      return
    }
    const jwks = parseInlineJwks(data.jwks)
    setSaving(true)
    try {
      await registerWorkloadTrustBundle(csrfToken, {
        name: data.name,
        profile: form.profile,
        trust_domain: data.trustDomain ?? '',
        ...(data.issuer ? { issuer: data.issuer } : {}),
        ...(data.jwksUri ? { jwks_uri: data.jwksUri } : {}),
        ...(jwks.ok && jwks.value ? { jwks: jwks.value } : {}),
        accepted_audiences: data.acceptedAudiences,
        max_subject_token_ttl_seconds: data.maxTtlSeconds,
      })
      window.location.assign(listURL())
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.saveFailedError)
      setSaving(false)
    }
  }

  return (
    <AdminShell
      active="workload-identity"
      actorUsername={actorUsername}
      title={t.registerTrustBundle}
      description={t.pageDescription}
      actions={
        <Button variant="outline" nativeButton={false} render={<a href={listURL()} />}>
          <IconArrowLeft size={16} aria-hidden="true" />
          {t.backToList}
        </Button>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Card className="w-full admin-form p-4">
        <form className="grid gap-4 md:grid-cols-2" onSubmit={handleSubmit}>
          <TrustBundleFormFields
            form={form}
            validation={validation}
            onChange={setForm}
            locked={false}
            t={t}
          />
          <div className="flex gap-2.5 md:col-span-2">
            <Button type="submit" disabled={saving || !validation.data}>
              {t.register}
            </Button>
            <Button variant="outline" nativeButton={false} render={<a href={listURL()} />}>
              {t.cancel}
            </Button>
          </div>
        </form>
      </Card>
    </AdminShell>
  )
}
