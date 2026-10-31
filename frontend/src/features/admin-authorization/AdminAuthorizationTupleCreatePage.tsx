import { IconArrowLeft } from '@tabler/icons-react'
import { Link } from '@tanstack/react-router'
import { type FormEvent, useState } from 'react'
import { writeRelationTuples } from '../../api/admin'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import type { RelationDefinition, RelationTuple, ResourceTypeDefinition } from '../../types'
import { adminAuthorizationDictionary } from './AdminAuthorizationPage.i18n'
import { authorizationErrorMessage, selectClassName, TextField } from './AdminAuthorizationShared'
import { parseSubjectForm } from './authorizationModel'

function directForms(relation: RelationDefinition | undefined): string[] {
  return (relation?.rewrites ?? []).flatMap((rewrite) =>
    rewrite.kind === 'direct' ? (rewrite.direct_subject_types ?? []) : [],
  )
}

function FormSelect({
  id,
  label,
  chooseLabel,
  options,
  value,
  onChange,
}: {
  id: string
  label: string
  chooseLabel: string
  options: string[]
  value: string
  onChange: (value: string) => void
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor={id}>{label}</Label>
      <select
        id={id}
        className={selectClassName}
        required
        value={value}
        onChange={(event) => onChange(event.target.value)}
      >
        <option value="">{chooseLabel}</option>
        {options.map((option) => (
          <option key={option} value={option}>
            {option}
          </option>
        ))}
      </select>
    </div>
  )
}

export function AdminAuthorizationTupleCreatePage({
  csrfToken,
  actorUsername,
  resourceTypes,
  onCreated,
}: {
  csrfToken: string
  actorUsername?: string
  resourceTypes: ResourceTypeDefinition[] | null
  onCreated: () => void
}) {
  const t = useDictionary(adminAuthorizationDictionary)
  const [resourceType, setResourceType] = useState('')
  const [resourceId, setResourceId] = useState('')
  const [relationName, setRelationName] = useState('')
  const [subjectForm, setSubjectForm] = useState('')
  const [subjectId, setSubjectId] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  const relations = (resourceTypes ?? [])
    .find((type) => type.name === resourceType)
    ?.relations.filter((relation) => directForms(relation).length > 0)
  const forms = directForms(relations?.find((relation) => relation.name === relationName))
  const parsedForm = subjectForm ? parseSubjectForm(subjectForm) : null
  const needsSubjectId = parsedForm?.kind !== 'wildcard'

  function selectResourceType(next: string) {
    setResourceType(next)
    setRelationName('')
    setSubjectForm('')
  }

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (!parsedForm) return
    const tuple: RelationTuple = {
      resource_type: resourceType,
      resource_id: resourceId,
      relation: relationName,
      subject_type: parsedForm.type,
      subject_id: parsedForm.kind === 'wildcard' ? '*' : subjectId,
      ...(parsedForm.kind === 'subject_set' ? { subject_relation: parsedForm.relation } : {}),
    }
    setBusy(true)
    setError('')
    try {
      await writeRelationTuples(csrfToken, { writes: [tuple] })
      onCreated()
    } catch (cause) {
      setError(authorizationErrorMessage(cause, t))
    } finally {
      setBusy(false)
    }
  }

  return (
    <AdminShell
      active="authorization"
      actorUsername={actorUsername}
      title={t.addTuple}
      description={t.pageDescription}
      actions={
        <Button
          variant="outline"
          nativeButton={false}
          render={<Link to="/admin/authorization/relation-tuples" />}
        >
          <IconArrowLeft size={16} aria-hidden="true" />
          {t.backToTuples}
        </Button>
      }
    >
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      {!resourceTypes ? (
        <Card className="flex flex-col items-start gap-3 p-6 text-sm text-slate-600">
          {t.noModelForTuples}
          <Button nativeButton={false} render={<Link to="/admin/authorization/model/edit" />}>
            {t.editModel}
          </Button>
        </Card>
      ) : (
        <Card className="w-full max-w-2xl p-4">
          <form className="flex flex-col gap-4" onSubmit={submit}>
            <FormSelect
              id="tuple-resource-type"
              label={t.resourceType}
              chooseLabel={t.choose}
              options={resourceTypes.map((type) => type.name)}
              value={resourceType}
              onChange={selectResourceType}
            />
            {resourceType && relations?.length === 0 ? <Alert>{t.noDirectRelation}</Alert> : null}
            <TextField
              id="tuple-resource-id"
              label={t.resourceId}
              required
              value={resourceId}
              onChange={setResourceId}
            />
            <FormSelect
              id="tuple-relation"
              label={t.relation}
              chooseLabel={t.choose}
              options={relations?.map((relation) => relation.name) ?? []}
              value={relationName}
              onChange={(next) => {
                setRelationName(next)
                setSubjectForm('')
              }}
            />
            <FormSelect
              id="tuple-subject-form"
              label={t.subjectForm}
              chooseLabel={t.choose}
              options={forms}
              value={subjectForm}
              onChange={setSubjectForm}
            />
            {needsSubjectId ? (
              <TextField
                id="tuple-subject-id"
                label={t.subjectId}
                required
                value={subjectId}
                onChange={setSubjectId}
              />
            ) : null}
            <div>
              <Button type="submit" disabled={busy}>
                {t.addTuple}
              </Button>
            </div>
          </form>
        </Card>
      )}
    </AdminShell>
  )
}
