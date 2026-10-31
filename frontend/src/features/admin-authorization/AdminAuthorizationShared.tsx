import { IconAlertTriangle } from '@tabler/icons-react'
import { Link } from '@tanstack/react-router'
import { AuthenticationAPIError } from '../../api'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import { cn } from '../../lib/utils'
import { adminAuthorizationDictionary } from './AdminAuthorizationPage.i18n'

type Dictionary = (typeof adminAuthorizationDictionary)['en']

export const selectClassName = 'h-9 rounded-md border border-slate-300 bg-white px-3 text-sm'

export function authorizationErrorMessage(cause: unknown, t: Dictionary): string {
  if (!(cause instanceof AuthenticationAPIError)) return t.genericError
  if (cause.code === 'consistency_not_satisfied') return t.consistencyNotSatisfied
  if (cause.code === 'authorization_model_not_found') return t.modelNotFound
  return cause.message
}

export function AuthorizationSectionNav({
  active,
  consistency,
}: {
  active: 'model' | 'tuples' | 'check'
  consistency?: string
}) {
  const t = useDictionary(adminAuthorizationDictionary)
  const linkClass = (section: typeof active) =>
    cn(
      'rounded-md px-3 py-1.5 text-sm font-medium',
      active === section ? 'bg-slate-900 text-white' : 'text-slate-700 hover:bg-slate-100',
    )
  return (
    <nav aria-label={t.sectionNavLabel} className="flex flex-wrap gap-1">
      <Link
        to="/admin/authorization/model"
        aria-current={active === 'model' ? 'page' : undefined}
        className={linkClass('model')}
      >
        {t.sectionModel}
      </Link>
      <Link
        to="/admin/authorization/relation-tuples"
        aria-current={active === 'tuples' ? 'page' : undefined}
        className={linkClass('tuples')}
      >
        {t.sectionTuples}
      </Link>
      <Link
        to="/admin/authorization/check"
        search={consistency ? { consistency } : {}}
        aria-current={active === 'check' ? 'page' : undefined}
        className={linkClass('check')}
      >
        {t.sectionCheck}
      </Link>
    </nav>
  )
}

export function TextField({
  id,
  label,
  help,
  required = false,
  list,
  value,
  onChange,
}: {
  id: string
  label: string
  help?: string
  required?: boolean
  list?: string
  value: string
  onChange: (value: string) => void
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        required={required}
        list={list}
        autoComplete="off"
        value={value}
        onChange={(event) => onChange(event.target.value)}
      />
      {help ? <p className="text-xs leading-5 text-slate-500">{help}</p> : null}
    </div>
  )
}

export function DestructiveConfirmDialog({
  title,
  target,
  result,
  busy,
  onClose,
  onConfirm,
}: {
  title: string
  target: string
  result: string
  busy: boolean
  onClose: () => void
  onConfirm: () => void
}) {
  const t = useDictionary(adminAuthorizationDictionary)
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/35 p-5 backdrop-blur-[2px]"
      role="dialog"
      aria-modal="true"
      aria-labelledby="authorization-confirm-title"
    >
      <Card className="relative flex w-full max-w-lg flex-col gap-3 p-6 shadow-2xl">
        <h2 id="authorization-confirm-title" className="flex items-center gap-2 font-semibold">
          <IconAlertTriangle size={18} aria-hidden="true" className="text-red-600" />
          {title}
        </h2>
        <p className="break-all font-mono text-sm">{target}</p>
        <p className="text-sm text-slate-700">{result}</p>
        <p className="text-sm font-medium text-red-700">{t.irreversibleWarning}</p>
        <div className="flex justify-end gap-2.5">
          <Button variant="outline" disabled={busy} onClick={onClose}>
            {t.cancel}
          </Button>
          <Button variant="destructive" disabled={busy} onClick={onConfirm}>
            {busy ? t.deleting : t.deleteConfirm}
          </Button>
        </div>
      </Card>
    </div>
  )
}
