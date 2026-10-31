import { IconPencil } from '@tabler/icons-react'
import { Link } from '@tanstack/react-router'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary, useFormatters } from '../../lib/i18n'
import type { AuthorizationModel } from '../../types'
import { adminAuthorizationDictionary } from './AdminAuthorizationPage.i18n'
import { AuthorizationSectionNav } from './AdminAuthorizationShared'
import { formatSubjectForm, toModelDraft, type RewriteDraft } from './authorizationModel'

function useRewriteSentence() {
  const t = useDictionary(adminAuthorizationDictionary)
  return (rewrite: RewriteDraft): string => {
    switch (rewrite.kind) {
      case 'direct':
        return t.ruleDirect.replace('{forms}', rewrite.subjects.map(formatSubjectForm).join(', '))
      case 'computed_userset':
        return t.ruleComputed.replace('{relation}', rewrite.computedRelation)
      case 'tuple_to_userset':
        return t.ruleTupleToUserset
          .replace('{tupleset}', rewrite.tuplesetRelation)
          .replace('{relation}', rewrite.computedRelation)
    }
  }
}

export function AdminAuthorizationModelPage({
  actorUsername,
  model,
  requestedVersion,
}: {
  actorUsername?: string
  model: AuthorizationModel | null
  requestedVersion?: number
}) {
  const t = useDictionary(adminAuthorizationDictionary)
  const { formatDateTime } = useFormatters()
  const rewriteSentence = useRewriteSentence()
  const draft = model && toModelDraft(model.resource_types)
  const missingVersion = !model && requestedVersion !== undefined

  return (
    <AdminShell
      active="authorization"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
      actions={
        <Button nativeButton={false} render={<Link to="/admin/authorization/model/edit" />}>
          <IconPencil size={16} aria-hidden="true" />
          {t.editModel}
        </Button>
      }
    >
      <AuthorizationSectionNav active="model" />
      {missingVersion ? (
        <Alert variant="destructive">
          {t.versionNotFound.replace('{version}', String(requestedVersion))}
        </Alert>
      ) : null}
      {!model && !missingVersion ? (
        <Card className="p-6 text-sm text-slate-600">{t.noModel}</Card>
      ) : null}
      {model ? (
        <div className="flex flex-wrap items-center gap-3 text-sm text-slate-600">
          <span className="font-semibold text-slate-900">
            {t.modelVersion.replace('{version}', String(model.version))}
          </span>
          <span>{formatDateTime(model.created_at)}</span>
          {model.version > 1 ? (
            <Link
              to="/admin/authorization/model"
              search={{ version: model.version - 1 }}
              className="underline"
            >
              {t.previousVersion}
            </Link>
          ) : null}
          {requestedVersion !== undefined ? (
            <Link to="/admin/authorization/model" className="underline">
              {t.latestVersion}
            </Link>
          ) : null}
        </div>
      ) : null}
      {model && requestedVersion !== undefined ? (
        <Alert>{t.pastVersion.replace('{version}', String(model.version))}</Alert>
      ) : null}
      {model && !draft ? (
        <Card className="p-4">
          <pre className="overflow-x-auto text-xs">
            {JSON.stringify(model.resource_types, null, 2)}
          </pre>
        </Card>
      ) : null}
      {draft?.resourceTypes.map((type) => (
        <Card key={type.name} className="flex flex-col gap-3 p-4">
          <h2 className="font-mono font-semibold">{type.name}</h2>
          {type.relations.length === 0 ? (
            <p className="text-sm text-slate-500">{t.noRelations}</p>
          ) : null}
          {type.relations.map((relation) => (
            <div key={relation.name}>
              <h3 className="font-mono text-sm font-medium">{relation.name}</h3>
              <ul className="mt-1 list-disc pl-5 text-sm text-slate-600">
                {relation.rewrites.map((rewrite) => {
                  const sentence = rewriteSentence(rewrite)
                  return <li key={sentence}>{sentence}</li>
                })}
              </ul>
            </div>
          ))}
        </Card>
      ))}
    </AdminShell>
  )
}
