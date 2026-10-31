import { useState, type FormEvent } from 'react'
import { AuthenticationAPIError } from '../../api'
import {
  createTokenVaultConnector,
  deleteTokenVaultConnector,
  listTokenVaultConnectors,
  setTokenVaultConnectorStatus,
} from '../../api/admin'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Toast } from '../../components/ui/toast'
import { useDictionary } from '../../lib/i18n'
import type { TokenVaultConnector, TokenVaultExecutionRule } from '../../types'
import { adminTokenVaultDictionary } from './AdminTokenVaultPage.i18n'

const emptyForm = {
  name: '',
  authorization_endpoint: '',
  token_endpoint: '',
  revocation_endpoint: '',
  client_id: '',
  client_secret: '',
  allowed_scopes: '',
  rules: '[]',
}

export function parseRules(raw: string): TokenVaultExecutionRule[] | null {
  try {
    const parsed: unknown = JSON.parse(raw)
    return Array.isArray(parsed) ? (parsed as TokenVaultExecutionRule[]) : null
  } catch {
    return null
  }
}

export function AdminTokenVaultPage({
  csrfToken,
  actorUsername,
  connectors: initial,
  confirmDelete = (message: string) => window.confirm(message),
}: {
  csrfToken: string
  actorUsername?: string
  connectors: TokenVaultConnector[]
  confirmDelete?: (message: string) => boolean
}) {
  const t = useDictionary(adminTokenVaultDictionary)
  const [connectors, setConnectors] = useState(initial)
  const [form, setForm] = useState(emptyForm)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  async function run(action: () => Promise<void>, success: string) {
    setBusy(true)
    setError('')
    setNotice('')
    try {
      await action()
      setConnectors(await listTokenVaultConnectors())
      setNotice(success)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.genericError)
    } finally {
      setBusy(false)
    }
  }

  function submit(event: FormEvent) {
    event.preventDefault()
    const rules = parseRules(form.rules)
    if (!rules) {
      setError(t.rulesInvalid)
      return
    }
    void run(async () => {
      await createTokenVaultConnector(csrfToken, {
        name: form.name,
        authorization_endpoint: form.authorization_endpoint,
        token_endpoint: form.token_endpoint,
        ...(form.revocation_endpoint ? { revocation_endpoint: form.revocation_endpoint } : {}),
        client_id: form.client_id,
        client_secret: form.client_secret,
        allowed_scopes: form.allowed_scopes.split(/\s+/).filter(Boolean),
        rules,
      })
      setForm(emptyForm)
    }, t.createdNotice)
  }

  const field = (key: keyof typeof emptyForm, label: string, type = 'text', required = true) => (
    <div className="flex flex-col gap-1">
      <Label htmlFor={`token-vault-${key}`}>{label}</Label>
      <Input
        id={`token-vault-${key}`}
        type={type}
        required={required}
        autoComplete="off"
        value={form[key]}
        onChange={(event) => setForm({ ...form, [key]: event.target.value })}
      />
    </div>
  )

  return (
    <AdminShell
      active="token-vault"
      actorUsername={actorUsername}
      title={t.title}
      description={t.description}
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Toast message={notice} onDismiss={() => setNotice('')} />
      <div className="flex flex-col gap-3">
        {connectors.length === 0 ? (
          <Card className="p-6 text-sm text-slate-500">{t.empty}</Card>
        ) : null}
        {connectors.map((connector) => (
          <Card
            key={connector.id}
            className="flex flex-col gap-3 p-4 md:flex-row md:items-center md:justify-between"
          >
            <div className="min-w-0">
              <h2 className="font-semibold">{connector.name}</h2>
              <p className="text-sm text-slate-600">
                {connector.status === 'active' ? t.statusActive : t.statusDisabled} ·{' '}
                {t.rulesCount.replace('{count}', String(connector.rules.length))}
              </p>
              <p className="break-all text-xs text-slate-500">{connector.token_endpoint}</p>
            </div>
            <div className="flex shrink-0 gap-2">
              <Button
                variant="outline"
                disabled={busy}
                onClick={() =>
                  run(async () => {
                    await setTokenVaultConnectorStatus(
                      csrfToken,
                      connector.id,
                      connector.status === 'active' ? 'disabled' : 'active',
                    )
                  }, t.updatedNotice)
                }
              >
                {connector.status === 'active' ? t.disable : t.enable}
              </Button>
              <Button
                variant="outline"
                disabled={busy}
                onClick={() => {
                  if (!confirmDelete(t.deleteConfirm.replace('{name}', connector.name))) return
                  void run(
                    () => deleteTokenVaultConnector(csrfToken, connector.id),
                    t.deletedNotice,
                  )
                }}
              >
                {t.delete}
              </Button>
            </div>
          </Card>
        ))}
      </div>
      <Card className="p-4">
        <form className="grid gap-3 md:grid-cols-2" onSubmit={submit}>
          {field('name', t.name)}
          {field('client_id', t.clientId)}
          {field('authorization_endpoint', t.authorizationEndpoint, 'url')}
          {field('token_endpoint', t.tokenEndpoint, 'url')}
          {field('revocation_endpoint', t.revocationEndpoint, 'url', false)}
          {field('client_secret', t.clientSecret, 'password')}
          {field('allowed_scopes', t.allowedScopes)}
          <div className="flex flex-col gap-1 md:col-span-2">
            <label className="flex flex-col gap-1 text-sm font-medium">
              {t.rules}
              <textarea
                className="min-h-28 rounded-md border border-slate-300 p-2 font-mono text-xs"
                value={form.rules}
                onChange={(event) => setForm({ ...form, rules: event.target.value })}
              />
            </label>
            <span className="text-xs text-slate-500">{t.rulesHelp}</span>
          </div>
          <div className="md:col-span-2">
            <Button type="submit" disabled={busy}>
              {t.add}
            </Button>
          </div>
        </form>
      </Card>
    </AdminShell>
  )
}
