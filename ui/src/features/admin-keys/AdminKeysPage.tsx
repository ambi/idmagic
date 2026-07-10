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
import type { AdminKey } from '../../types'

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
      setError(
        cause instanceof AuthenticationAPIError
          ? cause.message
          : '署名鍵の操作を完了できませんでした。',
      )
    } finally {
      setBusy(false)
    }
  }

  async function handleRotate() {
    await run(async () => {
      const result = await rotateTenantSigningKey(csrfToken)
      await refresh(result.next.kid)
    }, '新しい署名鍵に切り替えました。旧鍵は JWKS の verifying に残ります。')
    setConfirm(false)
  }

  async function handleDisable() {
    const target = disableTarget
    if (!target) return
    await run(async () => {
      await disableTenantKey(csrfToken, target.kid)
      await refresh()
    }, `鍵 ${target.kid} を無効化しました。JWKS から除外されます。`)
    setDisableTarget(null)
  }

  return (
    <AdminShell
      active="keys"
      actorUsername={actorUsername}
      title="署名鍵 (Signing Keys)"
      description="ID Token / Access Token の署名に使う JWKS の鍵集合。鍵はテナントごとに分離され、KeyProvider (Local / Postgres / VaultTransit) が実体を保持する。"
      actions={
        <>
          <Button
            variant="outline"
            className="size-9 px-0"
            aria-label="一覧を再読み込み"
            onClick={() => run(() => refresh(selected?.kid), '一覧を更新しました。')}
            disabled={busy}
          >
            <IconRefresh size={16} aria-hidden="true" />
          </Button>
          {canManage ? (
            <Button onClick={() => setConfirm(true)} disabled={busy}>
              <IconRotateClockwise size={16} aria-hidden="true" />
              ローテート
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
            <h2 className="text-base font-semibold text-slate-900">署名鍵をローテートします</h2>
            <p className="mt-3 text-sm text-slate-600">
              新しい active 鍵が生成され、旧 active 鍵は JWKS に verifying として残ります。 JWKS
              キャッシュが更新されるまで一時的な検証遅延が起きる可能性があります。
            </p>
            <div className="mt-5 flex justify-end gap-2">
              <Button variant="outline" onClick={() => setConfirm(false)} disabled={busy}>
                キャンセル
              </Button>
              <Button onClick={handleRotate} disabled={busy}>
                ローテート実行
              </Button>
            </div>
          </Card>
        </div>
      ) : null}

      {disableTarget ? (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-slate-900/40 px-4">
          <Card className="w-full max-w-md p-6">
            <h2 className="text-base font-semibold text-slate-900">署名鍵を無効化します</h2>
            <p className="mt-3 text-sm text-slate-600">
              鍵 <span className="font-mono text-xs">{disableTarget.kid}</span> を JWKS
              から除外します。この鍵で署名された既存トークンは検証できなくなります。鍵漏洩など緊急時のみ使用してください。
            </p>
            <div className="mt-5 flex justify-end gap-2">
              <Button variant="outline" onClick={() => setDisableTarget(null)} disabled={busy}>
                キャンセル
              </Button>
              <Button variant="destructive" onClick={handleDisable} disabled={busy}>
                無効化実行
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
  return (
    <Card className="overflow-hidden">
      <table className="w-full text-sm">
        <thead className="bg-slate-50 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
          <tr>
            <th className="px-4 py-3">Kid</th>
            <th className="px-4 py-3">Provider</th>
            <th className="px-4 py-3">Alg</th>
            <th className="px-4 py-3">状態</th>
            <th className="px-4 py-3">生成</th>
            {canManage ? <th className="px-4 py-3 text-right">操作</th> : null}
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
                  {key.active ? 'active' : 'verifying'}
                </span>
              </td>
              <td className="px-4 py-3 text-xs text-slate-500">{formatDate(key.created_at)}</td>
              {canManage ? (
                <td className="px-4 py-3 text-right">
                  <Button
                    variant="outline"
                    className="h-8 px-2 text-xs text-red-600"
                    aria-label={`鍵 ${key.kid} を無効化`}
                    onClick={(event) => {
                      event.stopPropagation()
                      onDisable(key)
                    }}
                    disabled={busy}
                  >
                    <IconTrash size={14} aria-hidden="true" />
                    無効化
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
  return (
    <Card className="p-5">
      <div className="flex items-center gap-2">
        <IconKey size={16} aria-hidden="true" className="text-slate-500" />
        <h2 className="text-sm font-semibold text-slate-700">公開鍵 JWK</h2>
      </div>
      {keyItem ? (
        <>
          <dl className="mt-4 grid grid-cols-[80px_minmax(0,1fr)] gap-y-2 text-xs">
            <dt className="text-slate-500">Kid</dt>
            <dd className="break-all font-mono">{keyItem.kid}</dd>
            <dt className="text-slate-500">Provider</dt>
            <dd>{keyItem.provider}</dd>
            <dt className="text-slate-500">Alg</dt>
            <dd>{keyItem.alg}</dd>
            <dt className="text-slate-500">状態</dt>
            <dd>{keyItem.active ? 'yes' : 'no'}</dd>
            <dt className="text-slate-500">生成</dt>
            <dd>{formatDate(keyItem.created_at)}</dd>
          </dl>
          <pre className="mt-4 max-h-[360px] overflow-auto rounded-md bg-slate-950 p-3 text-xs text-slate-50">
            {JSON.stringify(keyItem.public_jwk, null, 2)}
          </pre>
        </>
      ) : (
        <p className="mt-4 text-sm text-slate-500">署名鍵がありません。</p>
      )}
    </Card>
  )
}

export function formatDate(value: string): string {
  try {
    return new Date(value).toLocaleString()
  } catch {
    return value
  }
}
