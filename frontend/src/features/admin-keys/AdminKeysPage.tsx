import { IconKey, IconRefresh, IconRotateClockwise, IconTrash } from '@tabler/icons-react'
import { useState } from 'react'
import {
  AuthenticationAPIError,
  disableTenantKey,
  listAdminKeys,
  rotateTenantSigningKey,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Toast } from '../../components/ui/toast'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary, useLocale } from '../../lib/i18n'
import type { AdminKey } from '../../types'
import { adminKeysDictionary } from './AdminKeysPage.i18n'

export function AdminKeysPage({
  csrfToken,
  actorUsername,
  actorRoles,
  keys: initial,
}: {
  csrfToken: string
  actorUsername?: string
  actorRoles: string[]
  actorRealm: string
  keys: AdminKey[]
}) {
  const [keys, setKeys] = useState(initial)
  const [selected, setSelected] = useState<AdminKey | null>(
    initial.find((k) => k.active) ?? initial[0] ?? null,
  )
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [confirm, setConfirm] = useState(false)
  const [disableTarget, setDisableTarget] = useState<AdminKey | null>(null)
  const t = useDictionary(adminKeysDictionary)

  // per-tenant 鍵になったため、自テナントの管理者 (admin / system_admin) は
  // 自テナントの鍵をローテート / 無効化できる。全テナント横断の状態確認は
  // システムコンソール (/system) に分離した。
  const canManage = actorRoles.includes('admin') || actorRoles.includes('system_admin')

  async function refresh(preferred?: string) {
    const next = await listAdminKeys()
    setKeys(next)
    const match = next.find((k) => k.kid === preferred) ?? next.find((k) => k.active) ?? next[0]
    setSelected(match ?? null)
  }

  async function run(action: () => Promise<void>, success: string) {
    setBusy(true)
    setError('')
    setNotice('')
    try {
      await action()
      setNotice(success)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.genericActionError)
    } finally {
      setBusy(false)
    }
  }

  async function handleRotate() {
    await run(async () => {
      const result = await rotateTenantSigningKey(csrfToken)
      await refresh(result.next.kid)
    }, t.rotatedNotice)
    setConfirm(false)
  }

  async function handleDisable() {
    const target = disableTarget
    if (!target) return
    await run(
      async () => {
        await disableTenantKey(csrfToken, target.kid)
        await refresh()
      },
      t.disabledNotice.replace('{kid}', target.kid),
    )
    setDisableTarget(null)
  }

  return (
    <AdminShell
      active="keys"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
      actions={
        <>
          <Button
            variant="outline"
            className="size-9 px-0"
            aria-label={t.reloadAriaLabel}
            onClick={() => run(() => refresh(selected?.kid), t.listRefreshedNotice)}
            disabled={busy}
          >
            <IconRefresh size={16} aria-hidden="true" />
          </Button>
          {canManage ? (
            <Button onClick={() => setConfirm(true)} disabled={busy}>
              <IconRotateClockwise size={16} aria-hidden="true" />
              {t.rotate}
            </Button>
          ) : null}
        </>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Toast message={notice} onDismiss={() => setNotice('')} />

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_420px]">
        <SigningKeyTable
          keys={keys}
          selectedKid={selected?.kid}
          canManage={canManage}
          busy={busy}
          onSelect={setSelected}
          onDisable={setDisableTarget}
        />
        <SigningKeyDetail keyItem={selected} />
      </div>

      {confirm ? (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-slate-900/40 px-4">
          <Card className="w-full max-w-md p-6">
            <h2 className="text-base font-semibold text-slate-900">{t.rotateConfirmTitle}</h2>
            <p className="mt-3 text-sm text-slate-600">{t.rotateConfirmMessage}</p>
            <div className="mt-5 flex justify-end gap-2">
              <Button variant="outline" onClick={() => setConfirm(false)} disabled={busy}>
                {t.cancel}
              </Button>
              <Button onClick={handleRotate} disabled={busy}>
                {t.executeRotate}
              </Button>
            </div>
          </Card>
        </div>
      ) : null}

      {disableTarget ? (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-slate-900/40 px-4">
          <Card className="w-full max-w-md p-6">
            <h2 className="text-base font-semibold text-slate-900">{t.disableConfirmTitle}</h2>
            <p className="mt-3 text-sm text-slate-600">
              {t.disableConfirmMessagePrefix}{' '}
              <span className="font-mono text-xs">{disableTarget.kid}</span>{' '}
              {t.disableConfirmMessageSuffix}
            </p>
            <div className="mt-5 flex justify-end gap-2">
              <Button variant="outline" onClick={() => setDisableTarget(null)} disabled={busy}>
                {t.cancel}
              </Button>
              <Button variant="destructive" onClick={handleDisable} disabled={busy}>
                {t.executeDisable}
              </Button>
            </div>
          </Card>
        </div>
      ) : null}
    </AdminShell>
  )
}

export function SigningKeyTable({
  keys,
  selectedKid,
  canManage,
  busy,
  onSelect,
  onDisable,
}: {
  keys: AdminKey[]
  selectedKid?: string
  canManage: boolean
  busy: boolean
  onSelect: (key: AdminKey) => void
  onDisable: (key: AdminKey) => void
}) {
  const t = useDictionary(adminKeysDictionary)
  const { locale } = useLocale()
  return (
    <Card className="overflow-hidden">
      <table className="w-full text-sm">
        <thead className="bg-slate-50 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
          <tr>
            <th className="px-4 py-3">{t.tableHeaderKid}</th>
            <th className="px-4 py-3">{t.tableHeaderProvider}</th>
            <th className="px-4 py-3">{t.tableHeaderAlg}</th>
            <th className="px-4 py-3">{t.tableHeaderStatus}</th>
            <th className="px-4 py-3">{t.tableHeaderCreatedAt}</th>
            {canManage ? <th className="px-4 py-3 text-right">{t.tableHeaderActions}</th> : null}
          </tr>
        </thead>
        <tbody>
          {keys.map((key) => (
            <tr
              key={key.kid}
              onClick={() => onSelect(key)}
              className={`cursor-pointer border-t border-slate-100 hover:bg-slate-50 ${selectedKid === key.kid ? 'bg-blue-50/60' : ''}`}
            >
              <td className="px-4 py-3 font-mono text-xs">{key.kid}</td>
              <td className="px-4 py-3 text-xs">{key.provider}</td>
              <td className="px-4 py-3">{key.alg}</td>
              <td className="px-4 py-3">
                <span
                  className={
                    key.active
                      ? 'rounded-md bg-emerald-50 px-2 py-0.5 text-xs font-semibold text-emerald-700'
                      : 'rounded-md bg-slate-100 px-2 py-0.5 text-xs font-semibold text-slate-600'
                  }
                >
                  {key.active ? t.statusActive : t.statusVerifying}
                </span>
              </td>
              <td className="px-4 py-3 text-xs text-slate-500">
                {formatDate(key.created_at, locale)}
              </td>
              {canManage ? (
                <td className="px-4 py-3 text-right">
                  <Button
                    variant="outline"
                    className="h-8 px-2 text-xs text-red-600"
                    aria-label={t.disableKeyAria.replace('{kid}', key.kid)}
                    onClick={(event) => {
                      event.stopPropagation()
                      onDisable(key)
                    }}
                    disabled={busy}
                  >
                    <IconTrash size={14} aria-hidden="true" />
                    {t.disable}
                  </Button>
                </td>
              ) : null}
            </tr>
          ))}
        </tbody>
      </table>
    </Card>
  )
}

export function SigningKeyDetail({ keyItem }: { keyItem: AdminKey | null }) {
  const t = useDictionary(adminKeysDictionary)
  const { locale } = useLocale()
  return (
    <Card className="p-5">
      <div className="flex items-center gap-2">
        <IconKey size={16} aria-hidden="true" className="text-slate-500" />
        <h2 className="text-sm font-semibold text-slate-700">{t.publicJwkHeading}</h2>
      </div>
      {keyItem ? (
        <>
          <dl className="mt-4 grid grid-cols-[80px_minmax(0,1fr)] gap-y-2 text-xs">
            <dt className="text-slate-500">{t.tableHeaderKid}</dt>
            <dd className="break-all font-mono">{keyItem.kid}</dd>
            <dt className="text-slate-500">{t.tableHeaderProvider}</dt>
            <dd>{keyItem.provider}</dd>
            <dt className="text-slate-500">{t.tableHeaderAlg}</dt>
            <dd>{keyItem.alg}</dd>
            <dt className="text-slate-500">{t.activeFieldLabel}</dt>
            <dd>{keyItem.active ? t.activeYes : t.activeNo}</dd>
            <dt className="text-slate-500">{t.tableHeaderCreatedAt}</dt>
            <dd>{formatDate(keyItem.created_at, locale)}</dd>
          </dl>
          <pre className="mt-4 max-h-[360px] overflow-auto rounded-md bg-slate-950 p-3 text-xs text-slate-50">
            {JSON.stringify(keyItem.public_jwk, null, 2)}
          </pre>
        </>
      ) : (
        <p className="mt-4 text-sm text-slate-500">{t.noSigningKeysNotice}</p>
      )}
    </Card>
  )
}

export function formatDate(value: string, locale: 'ja' | 'en' = 'en'): string {
  try {
    return new Date(value).toLocaleString(locale === 'ja' ? 'ja-JP' : 'en-US')
  } catch {
    return value
  }
}
