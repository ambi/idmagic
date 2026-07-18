import { IconArrowLeft, IconUserPlus } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import { AuthenticationAPIError, createAdminUser, tenantURL } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import { adminUsersDictionary } from './AdminUsersPage.i18n'
import { Field, optionalValue, parseRoles } from './AdminUsersPrimitives'

export function AdminUserCreatePage({
  csrfToken,
  actorUsername,
}: {
  csrfToken: string
  actorUsername?: string
}) {
  const listPath = tenantURL('/admin/users')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const t = useDictionary(adminUsersDictionary)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const form = event.currentTarget
    const data = new FormData(form)
    const username = String(data.get('preferred_username') ?? '').trim()
    const password = String(data.get('password') ?? '')

    if (!username || !password) return

    setBusy(true)
    setError('')

    try {
      const created = await createAdminUser(csrfToken, {
        preferred_username: username,
        password: password,
        name: optionalValue(data.get('name')),
        email: optionalValue(data.get('email')),
        email_verified: data.get('email_verified') === 'on',
        roles: parseRoles(String(data.get('roles') ?? '')),
      })
      window.location.assign(tenantURL(`/admin/users/${encodeURIComponent(created.id)}`))
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.createUserFailedError)
      setBusy(false)
    }
  }

  return (
    <AdminShell
      active="users"
      actorUsername={actorUsername}
      title={t.addUser}
      description={t.createUserDescription}
    >
      <div className="flex items-center gap-3">
        <a
          href={listPath}
          className="inline-flex size-9 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-700 transition hover:bg-slate-50 hover:text-slate-900"
          aria-label={t.backToUserListAria}
        >
          <IconArrowLeft size={18} aria-hidden="true" />
        </a>
        <h1 className="text-2xl font-bold tracking-tight text-slate-900">{t.addUser}</h1>
      </div>

      <div className="mt-6 max-w-2xl">
        <Card className="shadow-[0_1px_2px_rgb(15_23_42/4%)]">
          <form onSubmit={handleSubmit}>
            <div className="grid gap-6 p-6">
              {error && <Alert>{error}</Alert>}

              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <Field id="preferred_username" label={t.username} required />
                <Field id="name" label={t.displayName} />
              </div>

              <Field id="email" label={t.emailFieldLabel} type="email" />

              <Field
                id="password"
                label={t.initialPasswordLabel}
                type="password"
                required
                minLength={12}
                description={t.initialPasswordDescription}
              />

              <Field
                id="roles"
                label={t.initialRolesLabel}
                placeholder="support, admin"
                description={t.initialRolesDescription}
              />

              <label className="flex items-start gap-3 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm text-slate-700 cursor-pointer">
                <input
                  name="email_verified"
                  type="checkbox"
                  className="mt-0.5 size-4 rounded border-slate-300"
                />
                <span>
                  <span className="block font-semibold text-slate-900">{t.createAsVerified}</span>
                  <span className="mt-0.5 block text-xs leading-5 text-slate-500">
                    {t.verifiedOwnershipNotice}
                  </span>
                </span>
              </label>
            </div>

            <div className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50 px-6 py-4">
              <a
                href={listPath}
                className="inline-flex h-9 items-center justify-center rounded-lg border border-slate-200 bg-white px-4 text-sm font-medium text-slate-700 shadow-sm transition hover:bg-slate-50 hover:text-slate-900"
              >
                {t.cancel}
              </a>
              <Button type="submit" disabled={busy}>
                <IconUserPlus size={16} aria-hidden="true" />
                {t.create}
              </Button>
            </div>
          </form>
        </Card>
      </div>
    </AdminShell>
  )
}
