import { IconPlus, IconTrash } from '@tabler/icons-react'
import { Link } from '@tanstack/react-router'
import { type FormEvent, useState } from 'react'
import { listRelationTuples, RELATION_TUPLE_LIST_LIMIT, writeRelationTuples } from '../../api/admin'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Toast } from '../../components/ui/toast'
import { useDictionary } from '../../lib/i18n'
import type { RelationTuple, RelationTupleFilters, RelationTupleWriteInput } from '../../types'
import { adminAuthorizationDictionary } from './AdminAuthorizationPage.i18n'
import {
  authorizationErrorMessage,
  AuthorizationSectionNav,
  DestructiveConfirmDialog,
  TextField,
} from './AdminAuthorizationShared'

type PendingDelete =
  | { kind: 'tuple'; tuple: RelationTuple }
  | { kind: 'object'; type: string; id: string }

function subjectLabel(tuple: RelationTuple): string {
  const base = `${tuple.subject_type}:${tuple.subject_id}`
  return tuple.subject_relation ? `${base}#${tuple.subject_relation}` : base
}

function tupleLabel(tuple: RelationTuple): string {
  return `${tuple.resource_type}:${tuple.resource_id}#${tuple.relation} @ ${subjectLabel(tuple)}`
}

function filledFilters(filters: RelationTupleFilters): RelationTupleFilters {
  return Object.fromEntries(Object.entries(filters).filter(([, value]) => value))
}

export function AdminAuthorizationTuplesPage({
  csrfToken,
  actorUsername,
  tuples: initialTuples,
  consistency: initialConsistency,
  filters,
  onFiltersChange,
}: {
  csrfToken: string
  actorUsername?: string
  tuples: RelationTuple[]
  consistency: string
  filters: RelationTupleFilters
  onFiltersChange: (filters: RelationTupleFilters) => void
}) {
  const t = useDictionary(adminAuthorizationDictionary)
  const [tuples, setTuples] = useState(initialTuples)
  const [consistency, setConsistency] = useState(initialConsistency)
  const [form, setForm] = useState(filters)
  const [objectType, setObjectType] = useState('')
  const [objectId, setObjectId] = useState('')
  const [pending, setPending] = useState<PendingDelete | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  async function confirmDelete() {
    if (!pending) return
    const input: RelationTupleWriteInput =
      pending.kind === 'tuple'
        ? { deletes: [pending.tuple] }
        : { delete_objects: [{ type: pending.type, id: pending.id }] }
    setBusy(true)
    setError('')
    setNotice('')
    try {
      const written = await writeRelationTuples(csrfToken, input)
      setConsistency(written.consistency)
      setNotice(t.deletedNotice.replace('{count}', String(written.deleted_count)))
      setTuples((await listRelationTuples(filters)).tuples)
      if (pending.kind === 'object') {
        setObjectType('')
        setObjectId('')
      }
    } catch (cause) {
      setError(authorizationErrorMessage(cause, t))
    } finally {
      setPending(null)
      setBusy(false)
    }
  }

  function submitFilters(event: FormEvent) {
    event.preventDefault()
    onFiltersChange(filledFilters(form))
  }

  const filterField = (key: keyof RelationTupleFilters, label: string) => (
    <TextField
      id={`tuple-filter-${key}`}
      label={label}
      value={form[key] ?? ''}
      onChange={(value) => setForm({ ...form, [key]: value })}
    />
  )

  return (
    <AdminShell
      active="authorization"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
      actions={
        <Button
          nativeButton={false}
          render={<Link to="/admin/authorization/relation-tuples/new" />}
        >
          <IconPlus size={16} aria-hidden="true" />
          {t.addTuple}
        </Button>
      }
    >
      <AuthorizationSectionNav active="tuples" consistency={consistency} />
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      <Toast message={notice} onDismiss={() => setNotice('')} />
      <Card className="p-4">
        <form className="grid items-end gap-3 md:grid-cols-3" onSubmit={submitFilters}>
          {filterField('resource_type', t.resourceType)}
          {filterField('resource_id', t.resourceId)}
          {filterField('relation', t.relation)}
          {filterField('subject_type', t.subjectType)}
          {filterField('subject_id', t.subjectId)}
          <div>
            <Button type="submit" variant="outline">
              {t.search}
            </Button>
          </div>
        </form>
      </Card>
      {tuples.length >= RELATION_TUPLE_LIST_LIMIT ? <Alert>{t.tuplesLimitReached}</Alert> : null}
      {tuples.length === 0 ? (
        <Card className="p-6 text-sm text-slate-600">{t.tuplesEmpty}</Card>
      ) : (
        <Card className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-slate-200 text-slate-500">
              <tr>
                <th className="px-4 py-2 font-medium">{t.resource}</th>
                <th className="px-4 py-2 font-medium">{t.relation}</th>
                <th className="px-4 py-2 font-medium">{t.subject}</th>
                <th className="px-4 py-2 font-medium">{t.actions}</th>
              </tr>
            </thead>
            <tbody>
              {tuples.map((tuple) => (
                <tr key={tupleLabel(tuple)} className="border-b border-slate-100 font-mono">
                  <td className="break-all px-4 py-2">
                    {tuple.resource_type}:{tuple.resource_id}
                  </td>
                  <td className="px-4 py-2">{tuple.relation}</td>
                  <td className="break-all px-4 py-2">{subjectLabel(tuple)}</td>
                  <td className="px-4 py-2">
                    <Button
                      variant="destructive"
                      disabled={busy}
                      onClick={() => setPending({ kind: 'tuple', tuple })}
                    >
                      <IconTrash size={16} aria-hidden="true" />
                      {t.delete}
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </Card>
      )}
      <Card className="flex flex-col gap-3 p-4">
        <h2 className="font-semibold">{t.deleteObjectTitle}</h2>
        <p className="text-sm text-slate-600">{t.deleteObjectDescription}</p>
        <div className="grid items-end gap-3 md:grid-cols-3">
          <TextField
            id="object-type"
            label={t.objectType}
            value={objectType}
            onChange={setObjectType}
          />
          <TextField id="object-id" label={t.objectId} value={objectId} onChange={setObjectId} />
          <div>
            <Button
              variant="destructive"
              disabled={busy || !objectType || !objectId}
              onClick={() => setPending({ kind: 'object', type: objectType, id: objectId })}
            >
              {t.deleteObject}
            </Button>
          </div>
        </div>
      </Card>
      {pending ? (
        <DestructiveConfirmDialog
          title={pending.kind === 'tuple' ? t.deleteTupleTitle : t.deleteObjectConfirmTitle}
          target={
            pending.kind === 'tuple' ? tupleLabel(pending.tuple) : `${pending.type}:${pending.id}`
          }
          result={pending.kind === 'tuple' ? t.deleteTupleResult : t.deleteObjectResult}
          busy={busy}
          onClose={() => setPending(null)}
          onConfirm={() => void confirmDelete()}
        />
      ) : null}
    </AdminShell>
  )
}
