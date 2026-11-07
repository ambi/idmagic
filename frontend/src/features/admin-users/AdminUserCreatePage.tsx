import { IconArrowLeft, IconUserPlus } from '@tabler/icons-react'
import { type ChangeEvent, type FormEvent, useState } from 'react'
import { AuthenticationAPIError, createAdminUser, tenantURL } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { FieldError } from '../../components/FieldError'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import {
  formSchema,
  optionalText,
  requiredSecret,
  requiredText,
  useFormValidation,
} from '../../lib/form'
import { useDictionary, useLocale } from '../../lib/i18n'
import { localizedErrorMessage } from '../../lib/i18n/errorMessage'
import { LENGTH } from '../../lib/lengthLimits'
import { adminUsersDictionary } from './AdminUsersPage.i18n'
import { Field, parseRoles } from './AdminUsersPrimitives'

// パスワードの長さの規則はテナントのポリシーが決め、API が検証する。
const createUserSchema = formSchema({
  preferred_username: requiredText(LENGTH.name),
  name: optionalText(LENGTH.displayName),
  email: optionalText(LENGTH.email),
  password: requiredSecret(),
})

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
  const [values, setValues] = useState({
    preferred_username: '',
    name: '',
    email: '',
    password: '',
  })
  const form = useFormValidation(createUserSchema, values)
  const t = useDictionary(adminUsersDictionary)
  const { locale } = useLocale()

  const change = (name: keyof typeof values) => (event: ChangeEvent<HTMLInputElement>) =>
    setValues((current) => ({ ...current, [name]: event.target.value }))

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const formData = new FormData(event.currentTarget)
    if (!form.data) {
      form.revealAll()
      return
    }

    setBusy(true)
    setError('')

    try {
      const created = await createAdminUser(csrfToken, {
        ...form.data,
        email_verified: formData.get('email_verified') === 'on',
        roles: parseRoles(String(formData.get('roles') ?? '')),
      })
      window.location.assign(tenantURL(`/admin/users/${encodeURIComponent(created.id)}`))
    } catch (cause) {
      setError(
        cause instanceof AuthenticationAPIError
          ? localizedErrorMessage(locale, cause.code, cause.message)
          : t.createUserFailedError,
      )
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
      </div>

      <div className="mt-6 admin-form">
        <Card className="shadow-[0_1px_2px_rgb(15_23_42/4%)]">
          <form onSubmit={handleSubmit}>
            <div className="grid gap-6 p-6 md:grid-cols-2">
              {error && (
                <Alert className="md:col-span-2" variant="destructive">
                  {error}
                </Alert>
              )}

              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 md:col-span-2">
                <Field
                  id="preferred_username"
                  label={t.username}
                  required
                  value={values.preferred_username}
                  onChange={change('preferred_username')}
                  {...form.fieldProps('preferred_username')}
                  error={<FieldError form={form} name="preferred_username" />}
                />
                <Field
                  id="name"
                  label={t.displayName}
                  value={values.name}
                  onChange={change('name')}
                  {...form.fieldProps('name')}
                  error={<FieldError form={form} name="name" />}
                />
              </div>

              <Field
                id="email"
                label={t.emailFieldLabel}
                type="email"
                value={values.email}
                onChange={change('email')}
                {...form.fieldProps('email')}
                error={<FieldError form={form} name="email" />}
              />

              <Field
                id="password"
                label={t.initialPasswordLabel}
                type="password"
                required
                minLength={12}
                description={t.initialPasswordDescription}
                value={values.password}
                onChange={change('password')}
                {...form.fieldProps('password')}
                error={<FieldError form={form} name="password" />}
              />

              <Field
                id="roles"
                label={t.initialRolesLabel}
                placeholder="support, admin"
                description={t.initialRolesDescription}
              />

              <label className="flex items-start gap-3 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm text-slate-700 cursor-pointer md:col-span-2">
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
              <Button type="submit" disabled={busy || !form.data}>
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
