import { IconPlus, IconServer2, IconTrash, IconX } from '@tabler/icons-react'
import { type FormEvent, useState } from 'react'
import {
  AuthenticationAPIError,
  createMcpResourceServer,
  deleteMcpResourceServer,
  updateMcpResourceServer,
} from '../../api'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Toast } from '../../components/ui/toast'
import { useDictionary } from '../../lib/i18n'
import type { McpResourceServer } from '../../types'
import { adminMcpResourceServersDictionary } from './AdminMcpResourceServersPage.i18n'

type FormState = {
  resource: string
  name: string
  scopes: string
  state: McpResourceServer['state']
}

const emptyForm: FormState = {
  resource: '',
  name: '',
  scopes: '',
  state: 'Active',
}

function toForm(resourceServer: McpResourceServer): FormState {
  return {
    resource: resourceServer.resource,
    name: resourceServer.name,
    scopes: resourceServer.scopes.join(' '),
    state: resourceServer.state,
  }
}

function parseScopes(scopes: string): string[] {
  return scopes.split(/[\s,]+/).filter(Boolean)
}

export function AdminMcpResourceServersPage({
  csrfToken,
  actorUsername,
  resourceServers,
}: {
  csrfToken: string
  actorUsername?: string
  resourceServers: McpResourceServer[]
}) {
  const [items, setItems] = useState(resourceServers)
  const [editing, setEditing] = useState<string | null>(null)
  const [form, setForm] = useState<FormState>(emptyForm)
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const dict = useDictionary(adminMcpResourceServersDictionary)

  function reset() {
    setEditing(null)
    setCreating(false)
    setForm(emptyForm)
    setError('')
  }

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setError('')
    const input = {
      name: form.name,
      scopes: parseScopes(form.scopes),
      state: form.state,
    }
    try {
      if (editing) {
        const updated = await updateMcpResourceServer(csrfToken, editing, input)
        setItems((previous) =>
          previous.map((item) => (item.resource_server_id === editing ? updated : item)),
        )
        setNotice(dict.updatedNotice.replace('{resource}', updated.resource))
      } else {
        const created = await createMcpResourceServer(csrfToken, {
          resource: form.resource,
          ...input,
        })
        setItems((previous) =>
          [...previous, created].sort((left, right) => left.resource.localeCompare(right.resource)),
        )
        setNotice(dict.registeredNotice.replace('{resource}', created.resource))
      }
      reset()
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : dict.saveFailedError)
    }
  }

  async function handleDelete(resourceServer: McpResourceServer) {
    setError('')
    try {
      await deleteMcpResourceServer(csrfToken, resourceServer.resource_server_id)
      setItems((previous) =>
        previous.filter((item) => item.resource_server_id !== resourceServer.resource_server_id),
      )
      setNotice(dict.deletedNotice.replace('{resource}', resourceServer.resource))
      if (editing === resourceServer.resource_server_id) reset()
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : dict.deleteFailedError)
    }
  }

  const showForm = creating || editing !== null

  return (
    <AdminShell
      active="mcp-resource-servers"
      actorUsername={actorUsername}
      title={dict.pageTitle}
      description={dict.pageDescription}
      actions={
        showForm ? undefined : (
          <Button
            type="button"
            onClick={() => {
              setCreating(true)
              setForm(emptyForm)
            }}
          >
            <IconPlus size={17} aria-hidden="true" />
            {dict.registerResourceServer}
          </Button>
        )
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Toast message={notice} onDismiss={() => setNotice('')} />

      {showForm ? (
        <Card className="p-4">
          <form className="flex flex-col gap-4" onSubmit={handleSubmit}>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="resource">{dict.resourceLabel}</Label>
              <Input
                id="resource"
                type="url"
                value={form.resource}
                disabled={editing !== null}
                placeholder={dict.resourcePlaceholder}
                onChange={(event) => setForm({ ...form, resource: event.target.value })}
                required
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="name">{dict.nameLabel}</Label>
              <Input
                id="name"
                value={form.name}
                onChange={(event) => setForm({ ...form, name: event.target.value })}
                required
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="scopes">{dict.scopesLabel}</Label>
              <Input
                id="scopes"
                value={form.scopes}
                placeholder={dict.scopesPlaceholder}
                onChange={(event) => setForm({ ...form, scopes: event.target.value })}
                required
              />
              <p className="text-xs leading-5 text-slate-500">{dict.scopesHelp}</p>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="state">{dict.stateLabel}</Label>
              <select
                id="state"
                value={form.state}
                onChange={(event) =>
                  setForm({
                    ...form,
                    state: event.target.value as McpResourceServer['state'],
                  })
                }
                className="w-40 rounded-md border border-slate-300 bg-white p-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none"
              >
                <option value="Active">Active</option>
                <option value="Disabled">Disabled</option>
              </select>
            </div>
            <div className="flex gap-2.5">
              <Button type="submit">{editing ? dict.update : dict.register}</Button>
              <Button type="button" variant="ghost" onClick={reset}>
                <IconX size={17} aria-hidden="true" />
                {dict.cancel}
              </Button>
            </div>
          </form>
        </Card>
      ) : null}

      {items.length === 0 ? (
        <Card className="p-8 text-center text-sm text-slate-500">{dict.emptyNotice}</Card>
      ) : (
        <div className="flex flex-col gap-3">
          {items.map((item) => (
            <Card key={item.resource_server_id} className="flex flex-col gap-3 p-4">
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <p className="break-all font-mono text-sm font-semibold text-slate-900">
                      {item.resource}
                    </p>
                    <span
                      className={
                        item.state === 'Active'
                          ? 'rounded-full bg-emerald-50 px-2 py-0.5 text-[0.68rem] font-bold text-emerald-700'
                          : 'rounded-full bg-slate-100 px-2 py-0.5 text-[0.68rem] font-bold text-slate-500'
                      }
                    >
                      {item.state}
                    </span>
                  </div>
                  <p className="mt-0.5 text-xs leading-5 text-slate-500">{item.name}</p>
                </div>
                <div className="flex shrink-0 gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => {
                      setEditing(item.resource_server_id)
                      setCreating(false)
                      setForm(toForm(item))
                    }}
                  >
                    {dict.edit}
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    aria-label={`${dict.delete}: ${item.resource}`}
                    onClick={() => handleDelete(item)}
                  >
                    <IconTrash size={16} aria-hidden="true" />
                  </Button>
                </div>
              </div>
              <div className="flex items-start gap-2 rounded-lg bg-slate-50 p-2.5 text-xs leading-5 text-slate-600">
                <IconServer2
                  size={15}
                  className="mt-0.5 shrink-0 text-blue-600"
                  aria-hidden="true"
                />
                <div className="flex flex-wrap gap-1.5">
                  {item.scopes.map((scope) => (
                    <span
                      key={scope}
                      className="rounded-full bg-white px-2 py-0.5 font-mono text-slate-700"
                    >
                      {scope}
                    </span>
                  ))}
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}
    </AdminShell>
  )
}
