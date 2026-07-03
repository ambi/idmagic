import { IconRefresh, IconShieldCheck } from '@tabler/icons-react'
import { useState } from 'react'
import { AuthenticationAPIError, listTenantKeyHealth } from '../../api'
import { SystemShell } from '../../components/SystemShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import type { TenantKeyHealth } from '../../types'

export function SystemKeyHealthPage({
  actorUsername,
  tenants: initial,
}: {
  actorUsername?: string
  tenants: TenantKeyHealth[]
}) {
  const [tenants, setTenants] = useState(initial)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  async function refresh() {
    setBusy(true)
    setError('')
    try {
      setTenants(await listTenantKeyHealth())
    } catch (cause) {
      setError(
        cause instanceof AuthenticationAPIError
          ? cause.message
          : 'テナント別の署名鍵の状態を取得できませんでした。',
      )
    } finally {
      setBusy(false)
    }
  }

  return (
    <SystemShell
      active="key-health"
      actorUsername={actorUsername}
      title="署名鍵の状態（全テナント）"
      description="全テナントの署名鍵プロバイダ（Local / Postgres / VaultTransit）の稼働状況と active kid を横断で確認します。"
      actions={
        <Button
          variant="outline"
          className="size-9 px-0"
          aria-label="一覧を再読み込み"
          onClick={refresh}
          disabled={busy}
        >
          <IconRefresh size={16} aria-hidden="true" />
        </Button>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}

      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
            <tr>
              <th className="px-4 py-3">テナント</th>
              <th className="px-4 py-3">プロバイダ</th>
              <th className="px-4 py-3">Active kid</th>
              <th className="px-4 py-3">JWKS 鍵数</th>
              <th className="px-4 py-3">プロバイダ状態</th>
            </tr>
          </thead>
          <tbody>
            {tenants.map((t) => (
              <tr key={t.tenant_id} className="border-t border-slate-100">
                <td className="px-4 py-3 font-medium text-slate-800">{t.tenant_id}</td>
                <td className="px-4 py-3 text-xs">{t.provider}</td>
                <td className="px-4 py-3 font-mono text-xs">{t.active_kid || '—'}</td>
                <td className="px-4 py-3 text-slate-600">{t.jwks_key_count}</td>
                <td className="px-4 py-3">
                  {t.provider_healthy ? (
                    <span className="inline-flex items-center gap-1 rounded-md bg-emerald-50 px-2 py-0.5 text-xs font-semibold text-emerald-700">
                      <IconShieldCheck size={13} aria-hidden="true" />
                      正常
                    </span>
                  ) : (
                    <span className="rounded-md bg-red-50 px-2 py-0.5 text-xs font-semibold text-red-700">
                      接続不可
                    </span>
                  )}
                </td>
              </tr>
            ))}
            {tenants.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-6 text-center text-sm text-slate-500">
                  テナントがありません。
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </Card>
    </SystemShell>
  )
}
