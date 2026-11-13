import { type FormEvent, useMemo, useState } from 'react'
import { AuthenticationAPIError, createAdminTenant } from '../../api'
import { FieldError } from '../../components/FieldError'
import { SystemShell } from '../../components/SystemShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { formSchema, requiredText, rule, useForm } from '../../lib/form'
import { useDictionary } from '../../lib/i18n'
import { LENGTH } from '../../lib/lengthLimits'
import { systemTenantsDictionary } from './SystemTenantsPage.i18n'
import { SYSTEM_TENANTS_PATH, tenantDetailURL } from './SystemTenantsShared'

// realm は Tenant の @pattern と同じく、英小文字か数字で始まり終わる 63 文字以内の DNS ラベルである。
const REALM = /^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$/

function createTenantSchema(realmInvalid: string) {
  return formSchema({
    realm: rule(requiredText(), (value) =>
      REALM.test(value) ? undefined : { kind: 'message', text: realmInvalid },
    ),
    displayName: requiredText(LENGTH.displayName),
  })
}

export function SystemTenantCreatePage({
  csrfToken,
  actorUsername,
}: {
  csrfToken: string
  actorUsername?: string
}) {
  const t = useDictionary(systemTenantsDictionary)
  const schema = useMemo(() => createTenantSchema(t.realmInvalidError), [t.realmInvalidError])
  const form = useForm(schema, { realm: '', displayName: '' })
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!form.data) {
      form.revealAll()
      return
    }
    setSaving(true)
    setError('')
    try {
      const created = await createAdminTenant(csrfToken, {
        realm: form.data.realm,
        display_name: form.data.displayName,
      })
      window.location.assign(tenantDetailURL(created.realm, 'created'))
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.tenantCreateFailedError)
      setSaving(false)
    }
  }

  return (
    <SystemShell
      active="tenants"
      actorUsername={actorUsername}
      title={t.newTenant}
      description={t.createDescription}
    >
      <Card className="max-w-2xl p-6">
        <form onSubmit={(event) => void submit(event)} className="grid gap-4">
          {error ? <Alert variant="destructive">{error}</Alert> : null}
          <div className="grid gap-1.5">
            <Label htmlFor="tenant-realm" required>
              {t.realmLabel}
            </Label>
            <Input id="tenant-realm" required placeholder="acme" {...form.input('realm')} />
            <FieldError form={form} name="realm" />
            <p className="text-xs text-slate-500">{t.realmHelp}</p>
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="tenant-display" required>
              {t.displayNameLabel}
            </Label>
            <Input
              id="tenant-display"
              required
              placeholder="Acme Inc."
              {...form.input('displayName')}
            />
            <FieldError form={form} name="displayName" />
          </div>
          <div className="flex justify-end gap-2">
            <Button
              variant="outline"
              nativeButton={false}
              render={<a href={SYSTEM_TENANTS_PATH} />}
            >
              {t.cancel}
            </Button>
            <Button type="submit" disabled={saving || !form.data}>
              {saving ? t.saving : t.create}
            </Button>
          </div>
        </form>
      </Card>
    </SystemShell>
  )
}
