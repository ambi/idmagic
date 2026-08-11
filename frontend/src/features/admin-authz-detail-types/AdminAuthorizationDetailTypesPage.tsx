import { IconPlus, IconShieldCheck, IconTrash } from '@tabler/icons-react'
import { useState } from 'react'
import { AuthenticationAPIError, deleteAuthorizationDetailType, tenantURL } from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Toast } from '../../components/ui/toast'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import type { AuthorizationDetailType } from '../../types'
import { adminAuthorizationDetailTypesDictionary } from './AdminAuthorizationDetailTypesPage.i18n'
import { editURL, newURL } from './AdminAuthorizationDetailTypesShared'

// AdminAuthorizationDetailTypesPage は認可詳細タイプの参照専用一覧。作成/編集は専用ルート
// (new / $type/edit) に一本化されている (policy #2, wi-314 T014)。
export function AdminAuthorizationDetailTypesPage({
  csrfToken,
  actorUsername,
  types,
}: {
  csrfToken: string
  actorUsername?: string
  types: AuthorizationDetailType[]
}) {
  const [items, setItems] = useState(types)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const dict = useDictionary(adminAuthorizationDetailTypesDictionary)

  async function handleDelete(detailType: string) {
    setError('')
    try {
      await deleteAuthorizationDetailType(csrfToken, detailType)
      setItems((prev) => prev.filter((item) => item.type !== detailType))
      setNotice(dict.deletedNotice.replace('{type}', detailType))
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : dict.deleteFailedError)
    }
  }

  return (
    <AdminShell
      active="authz-detail-types"
      actorUsername={actorUsername}
      title={dict.pageTitle}
      description={dict.pageDescription}
      actions={
        <Button nativeButton={false} render={<a href={newURL()} />}>
          <IconPlus size={17} aria-hidden="true" />
          {dict.registerType}
        </Button>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Toast message={notice} onDismiss={() => setNotice('')} />

      {items.length === 0 ? (
        <Card className="p-8 text-center text-sm text-slate-500">{dict.emptyNotice}</Card>
      ) : (
        <div className="flex flex-col gap-3">
          {items.map((item) => (
            <Card key={item.type} className="flex flex-col gap-3 p-4">
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <p className="font-mono text-sm font-semibold text-slate-900">{item.type}</p>
                    <span
                      className={
                        item.state === 'Enabled'
                          ? 'rounded-full bg-emerald-50 px-2 py-0.5 text-[0.68rem] font-bold text-emerald-700'
                          : 'rounded-full bg-slate-100 px-2 py-0.5 text-[0.68rem] font-bold text-slate-500'
                      }
                    >
                      {item.state}
                    </span>
                  </div>
                  {item.description ? (
                    <p className="mt-0.5 text-xs leading-5 text-slate-500">{item.description}</p>
                  ) : null}
                </div>
                <div className="flex shrink-0 gap-2">
                  <Button
                    variant="outline"
                    nativeButton={false}
                    render={<a href={editURL(item.type)} />}
                  >
                    {dict.edit}
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    onClick={() => void handleDelete(item.type)}
                  >
                    <IconTrash size={16} aria-hidden="true" />
                  </Button>
                </div>
              </div>
              <div className="flex items-start gap-2 rounded-lg bg-slate-50 p-2.5 text-xs leading-5 text-slate-600">
                <IconShieldCheck
                  size={15}
                  className="mt-0.5 shrink-0 text-blue-600"
                  aria-hidden="true"
                />
                <div className="flex flex-wrap gap-1.5">
                  {item.schema.rules.map((rule) => (
                    <span key={rule.name} className="font-mono">
                      {rule.name}:{rule.semantics}
                      {rule.required ? '*' : ''}
                    </span>
                  ))}
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}

      <p className="text-xs text-slate-400">
        <a className="underline" href={tenantURL('/admin/applications')}>
          {dict.footerLinkLabel}
        </a>{' '}
        {dict.footerText}
      </p>
    </AdminShell>
  )
}
