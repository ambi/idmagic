import { useState, type FormEvent } from 'react'
import { Link } from '@tanstack/react-router'
import { AuthenticationAPIError } from '../../api'
import { createTokenVaultConnector } from '../../api/admin'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import type { TokenVaultExecutionRule } from '../../types'
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

export function AdminTokenVaultCreatePage({
  csrfToken,
  actorUsername,
  onCreated,
}: {
  csrfToken: string
  actorUsername?: string
  onCreated: () => void | Promise<void>
}) {
  const t = useDictionary(adminTokenVaultDictionary)
  const [form, setForm] = useState(emptyForm)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  async function submit(event: FormEvent) {
    event.preventDefault()
    const rules = parseRules(form.rules)
    if (!rules) {
      setError(t.rulesInvalid)
      return
    }
    setBusy(true)
    setError('')
    try {
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
      await onCreated()
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.genericError)
    } finally {
      setBusy(false)
    }
  }
  const field = (key: keyof typeof emptyForm, label: string, type = 'text', required = true) => (
    <div className="grid gap-1.5">
      <Label htmlFor={`service-connection-${key}`} required={required} optional={!required}>
        {label}
      </Label>
      <Input
        id={`service-connection-${key}`}
        name={`service-connection-${key}`}
        type={type}
        required={required}
        autoComplete={key === 'client_secret' ? 'new-password' : 'off'}
        value={form[key]}
        onChange={(event) => setForm({ ...form, [key]: event.target.value })}
      />
    </div>
  )
  return (
    <AdminShell
      active="token-vault"
      actorUsername={actorUsername}
      title={t.add}
      description={t.description}
      actions={
        <Button variant="outline" nativeButton={false} render={<Link to="/admin/token-vault" />}>
          {t.back}
        </Button>
      }
    >
      <Card className="admin-form p-6">
        <form
          className="grid gap-5 md:grid-cols-2"
          autoComplete="off"
          onSubmit={(event) => void submit(event)}
        >
          {error ? (
            <Alert className="md:col-span-2" variant="destructive">
              {error}
            </Alert>
          ) : null}
          {field('name', t.name)}
          {field('client_id', t.clientId)}
          {field('authorization_endpoint', t.authorizationEndpoint, 'url')}
          {field('token_endpoint', t.tokenEndpoint, 'url')}
          {field('revocation_endpoint', t.revocationEndpoint, 'url', false)}
          {field('client_secret', t.clientSecret, 'password')}
          <div className="md:col-span-2">{field('allowed_scopes', t.allowedScopes)}</div>
          <div className="grid gap-1.5 md:col-span-2">
            <Label htmlFor="service-connection-rules" required>
              {t.rules}
            </Label>
            <textarea
              id="service-connection-rules"
              name="service-connection-rules"
              required
              aria-describedby="service-connection-rules-help"
              className="min-h-40 rounded-md border border-slate-300 p-3 font-mono text-sm"
              value={form.rules}
              onChange={(event) => setForm({ ...form, rules: event.target.value })}
            />
            <p id="service-connection-rules-help" className="text-xs text-slate-500">
              {t.rulesHelp}
            </p>
          </div>
          <div className="flex gap-2 md:col-span-2">
            <Button type="submit" disabled={busy}>
              {t.add}
            </Button>
            <Button
              variant="outline"
              nativeButton={false}
              render={<Link to="/admin/token-vault" />}
            >
              {t.cancel}
            </Button>
          </div>
        </form>
      </Card>
    </AdminShell>
  )
}
