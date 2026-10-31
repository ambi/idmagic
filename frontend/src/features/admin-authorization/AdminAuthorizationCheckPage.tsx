import { IconPlus, IconX } from '@tabler/icons-react'
import { useState } from 'react'
import { checkAccess, listAccessibleResources } from '../../api/admin'
import { AdminShell } from '../../components/AdminShell'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { useDictionary } from '../../lib/i18n'
import type {
  FgaActor,
  FgaCheckResult,
  FgaListAccessibleResourcesResult,
  ListAccessibleResourcesInput,
  ResourceTypeDefinition,
} from '../../types'
import { adminAuthorizationDictionary } from './AdminAuthorizationPage.i18n'
import {
  authorizationErrorMessage,
  AuthorizationSectionNav,
  TextField,
} from './AdminAuthorizationShared'

type Dictionary = (typeof adminAuthorizationDictionary)['en']

// 判定理由は HTTP のエラーコードとは別の語彙なので、共通のエラー辞書に置かない。
const REASON_KEYS: Record<string, keyof Dictionary> = {
  relationship_facts_present: 'reasonRelationshipFactsPresent',
  relationship_permits_subject: 'reasonRelationshipPermitsSubject',
  relationship_permits_actor_chain: 'reasonRelationshipPermitsActorChain',
  actor_chain_principals_active: 'reasonActorChainPrincipalsActive',
  scope_subset_of_client_scope: 'reasonScopeSubsetOfClientScope',
  actor_and_resource_share_tenant: 'reasonActorAndResourceShareTenant',
  no_relation_path: 'reasonNoRelationPath',
  unknown_relation: 'reasonUnknownRelation',
  evaluation_depth_exceeded: 'reasonEvaluationDepthExceeded',
  relation_cycle_detected: 'reasonRelationCycleDetected',
  subject_form_not_declared: 'reasonSubjectFormNotDeclared',
}

function reasonDescription(code: string, t: Dictionary): string | null {
  const key = Object.hasOwn(REASON_KEYS, code) ? REASON_KEYS[code] : undefined
  return key ? t[key] : null
}

function parseScopes(value: string): string[] {
  return value.split(/\s+/).filter(Boolean)
}

function CheckResultCard({ result }: { result: FgaCheckResult }) {
  const t = useDictionary(adminAuthorizationDictionary)
  return (
    <Card className="flex flex-col gap-3 p-4">
      <div className="flex items-center gap-3">
        <h2
          className={
            result.permitted ? 'font-semibold text-emerald-700' : 'font-semibold text-red-700'
          }
        >
          {result.permitted ? t.permitted : t.denied}
        </h2>
        <span className="text-sm text-slate-500">
          {t.resultModelVersion.replace('{version}', String(result.model_version))}
        </span>
      </div>
      <div>
        <h3 id="relation-path-heading" className="text-sm font-medium">
          {t.relationPath}
        </h3>
        {result.relation_path.length === 0 ? (
          <p className="text-sm text-slate-600">{t.noRelationPathWalked}</p>
        ) : (
          <ol aria-label={t.relationPath} className="mt-1 list-decimal pl-5 font-mono text-sm">
            {result.relation_path.map((step) => (
              <li key={step}>{step}</li>
            ))}
          </ol>
        )}
      </div>
      {result.permitted ? null : (
        <div>
          <h3 className="text-sm font-medium">{t.reasons}</h3>
          {result.reasons.length === 0 ? (
            <p className="text-sm text-slate-600">{t.noReasons}</p>
          ) : (
            <ul className="mt-1 flex flex-col gap-1 text-sm">
              {result.reasons.map((code) => {
                const description = reasonDescription(code, t)
                return (
                  <li key={code}>
                    {description ? <p>{description}</p> : null}
                    <code className="text-xs text-slate-500">{code}</code>
                  </li>
                )
              })}
            </ul>
          )}
        </div>
      )}
    </Card>
  )
}

function ListResultCard({ result }: { result: FgaListAccessibleResourcesResult }) {
  const t = useDictionary(adminAuthorizationDictionary)
  return (
    <Card className="flex flex-col gap-3 p-4">
      <div className="flex items-center gap-3">
        <h2 className="font-semibold">{t.listResult}</h2>
        <span className="text-sm text-slate-500">
          {t.resultModelVersion.replace('{version}', String(result.model_version))}
        </span>
      </div>
      {result.truncated ? <Alert>{t.listTruncated}</Alert> : null}
      {result.resource_ids.length === 0 ? (
        <p className="text-sm text-slate-600">{t.listEmpty}</p>
      ) : (
        <ul className="list-disc pl-5 font-mono text-sm">
          {result.resource_ids.map((id) => (
            <li key={id}>{id}</li>
          ))}
        </ul>
      )}
    </Card>
  )
}

export function AdminAuthorizationCheckPage({
  csrfToken,
  actorUsername,
  resourceTypes,
  consistency: initialConsistency,
}: {
  csrfToken: string
  actorUsername?: string
  resourceTypes: ResourceTypeDefinition[]
  consistency: string
}) {
  const t = useDictionary(adminAuthorizationDictionary)
  const [subjectType, setSubjectType] = useState('')
  const [subjectId, setSubjectId] = useState('')
  const [resourceType, setResourceType] = useState('')
  const [resourceId, setResourceId] = useState('')
  const [relation, setRelation] = useState('')
  const [actors, setActors] = useState<FgaActor[]>([])
  const [grantedScopes, setGrantedScopes] = useState('')
  const [requiredScopes, setRequiredScopes] = useState('')
  const [consistency, setConsistency] = useState(initialConsistency)
  const [checkResult, setCheckResult] = useState<FgaCheckResult | null>(null)
  const [listResult, setListResult] = useState<FgaListAccessibleResourcesResult | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  const typeNames = resourceTypes.map((type) => type.name)
  const relationNames = [
    ...new Set(
      resourceTypes
        .filter((type) => !resourceType || type.name === resourceType)
        .flatMap((type) => type.relations.map((candidate) => candidate.name)),
    ),
  ]
  const actorsFilled = actors.every((actor) => actor.type && actor.id)
  const canList = Boolean(subjectType && subjectId && resourceType && relation && actorsFilled)

  function commonInput(): ListAccessibleResourcesInput {
    const granted = parseScopes(grantedScopes)
    const required = parseScopes(requiredScopes)
    return {
      resource_type: resourceType,
      relation,
      subject_type: subjectType,
      subject_id: subjectId,
      ...(actors.length > 0 ? { actor_chain: actors } : {}),
      ...(consistency ? { minimum_consistency: consistency } : {}),
      ...(granted.length > 0 ? { granted_scopes: granted } : {}),
      ...(required.length > 0 ? { required_scopes: required } : {}),
    }
  }

  async function run(action: () => Promise<void>) {
    setBusy(true)
    setError('')
    setCheckResult(null)
    setListResult(null)
    try {
      await action()
    } catch (cause) {
      setError(authorizationErrorMessage(cause, t))
    } finally {
      setBusy(false)
    }
  }

  function updateActor(index: number, actor: FgaActor) {
    setActors(actors.map((current, position) => (position === index ? actor : current)))
  }

  return (
    <AdminShell
      active="authorization"
      actorUsername={actorUsername}
      title={t.pageTitle}
      description={t.pageDescription}
    >
      <AuthorizationSectionNav active="check" consistency={consistency} />
      <Card className="flex flex-col gap-4 p-4">
        <p className="text-sm text-slate-600">{t.checkFormDescription}</p>
        <datalist id="authorization-type-names">
          {typeNames.map((name) => (
            <option key={name} value={name} />
          ))}
        </datalist>
        <datalist id="authorization-relation-names">
          {relationNames.map((name) => (
            <option key={name} value={name} />
          ))}
        </datalist>
        <div className="grid gap-3 md:grid-cols-2">
          <TextField
            id="check-subject-type"
            label={t.subjectType}
            list="authorization-type-names"
            value={subjectType}
            onChange={setSubjectType}
          />
          <TextField
            id="check-subject-id"
            label={t.subjectId}
            value={subjectId}
            onChange={setSubjectId}
          />
          <TextField
            id="check-resource-type"
            label={t.resourceType}
            list="authorization-type-names"
            value={resourceType}
            onChange={setResourceType}
          />
          <TextField
            id="check-resource-id"
            label={t.resourceId}
            value={resourceId}
            onChange={setResourceId}
          />
          <TextField
            id="check-relation"
            label={t.relation}
            list="authorization-relation-names"
            value={relation}
            onChange={setRelation}
          />
        </div>
        <div className="flex flex-col gap-2">
          <h2 className="text-sm font-medium">{t.actorChain}</h2>
          {actors.map((actor, index) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: rows are positional and have no identity until submitted
            <div key={index} className="flex flex-wrap items-end gap-2">
              <TextField
                id={`check-actor-${index}-type`}
                label={t.actorType}
                list="authorization-type-names"
                value={actor.type}
                onChange={(type) => updateActor(index, { ...actor, type })}
              />
              <TextField
                id={`check-actor-${index}-id`}
                label={t.actorId}
                value={actor.id}
                onChange={(id) => updateActor(index, { ...actor, id })}
              />
              <Button
                type="button"
                variant="ghost"
                onClick={() => setActors(actors.filter((_, position) => position !== index))}
              >
                <IconX size={16} aria-hidden="true" />
                {t.removeActor}
              </Button>
            </div>
          ))}
          <div>
            <Button
              type="button"
              variant="outline"
              onClick={() => setActors([...actors, { type: '', id: '' }])}
            >
              <IconPlus size={16} aria-hidden="true" />
              {t.addActor}
            </Button>
          </div>
        </div>
        <div className="grid gap-3 md:grid-cols-2">
          <TextField
            id="check-granted-scopes"
            label={t.grantedScopes}
            help={t.scopesHelp}
            value={grantedScopes}
            onChange={setGrantedScopes}
          />
          <TextField
            id="check-required-scopes"
            label={t.requiredScopes}
            value={requiredScopes}
            onChange={setRequiredScopes}
          />
        </div>
        <TextField
          id="check-minimum-consistency"
          label={t.minimumConsistency}
          help={t.minimumConsistencyHelp}
          value={consistency}
          onChange={setConsistency}
        />
        <div className="flex flex-wrap gap-2.5">
          <Button
            disabled={busy || !canList || !resourceId}
            onClick={() =>
              void run(async () => {
                setCheckResult(
                  await checkAccess(csrfToken, { ...commonInput(), resource_id: resourceId }),
                )
              })
            }
          >
            {t.runCheck}
          </Button>
          <Button
            variant="outline"
            disabled={busy || !canList}
            onClick={() =>
              void run(async () => {
                setListResult(await listAccessibleResources(csrfToken, commonInput()))
              })
            }
          >
            {t.runList}
          </Button>
        </div>
      </Card>
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      {checkResult ? <CheckResultCard result={checkResult} /> : null}
      {listResult ? <ListResultCard result={listResult} /> : null}
    </AdminShell>
  )
}
