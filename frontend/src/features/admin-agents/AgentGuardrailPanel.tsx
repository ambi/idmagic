import { type FormEvent, useEffect, useState } from 'react'
import {
  AuthenticationAPIError,
  deleteAgentGuardrailPolicy,
  getAgentGuardrailPolicy,
  putAgentGuardrailPolicy,
} from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import type {
  AgentGuardrailAction,
  AgentGuardrailLimits,
  AgentGuardrailPolicy,
  AgentGuardrailPolicySet,
} from '../../types'
import { agentGuardrailDictionary } from './AgentGuardrailPanel.i18n'

type FormState = {
  resources: string
  detailTypes: string
  maxIssuances: string
  windowSeconds: string
  ttl: string
  expiresAt: string
  depth: string
  requireApproval: boolean
  resourcesAction: AgentGuardrailAction
  detailTypesAction: AgentGuardrailAction
  rateAction: AgentGuardrailAction
  expiresAction: AgentGuardrailAction
}

function toForm(policy?: AgentGuardrailPolicy): FormState {
  return {
    resources: policy?.allowed_resources?.join('\n') ?? '',
    detailTypes: policy?.allowed_authorization_detail_types?.join('\n') ?? '',
    maxIssuances: policy?.issuance_rate?.max_issuances.toString() ?? '',
    windowSeconds: policy?.issuance_rate?.window_seconds.toString() ?? '',
    ttl: policy?.max_token_ttl_seconds?.toString() ?? '',
    expiresAt: policy?.expires_at ?? '',
    depth: policy?.max_delegation_depth?.toString() ?? '',
    requireApproval: policy?.require_approval ?? false,
    resourcesAction: policy?.on_violation?.allowed_resources ?? 'deny',
    detailTypesAction: policy?.on_violation?.allowed_authorization_detail_types ?? 'deny',
    rateAction: policy?.on_violation?.issuance_rate ?? 'deny',
    expiresAction: policy?.on_violation?.expires_at ?? 'deny',
  }
}

function lines(value: string): string[] | undefined {
  const items = value
    .split('\n')
    .map((item) => item.trim())
    .filter(Boolean)
  return items.length > 0 ? items : undefined
}

function optionalNumber(value: string): number | undefined {
  return value.trim() === '' ? undefined : Number(value)
}

// 空の項目は省く。省いた項目は制限しないので、誤って空の許可（何も許さない）を送らない。
function toLimits(form: FormState): AgentGuardrailLimits {
  const maxIssuances = optionalNumber(form.maxIssuances)
  const windowSeconds = optionalNumber(form.windowSeconds)
  const resources = lines(form.resources)
  const detailTypes = lines(form.detailTypes)
  const rate = maxIssuances !== undefined && windowSeconds !== undefined
  const expiresAt = form.expiresAt.trim() || undefined
  return {
    allowed_resources: resources,
    allowed_authorization_detail_types: detailTypes,
    issuance_rate: rate
      ? { max_issuances: maxIssuances, window_seconds: windowSeconds }
      : undefined,
    max_token_ttl_seconds: optionalNumber(form.ttl),
    expires_at: expiresAt,
    max_delegation_depth: optionalNumber(form.depth),
    require_approval: form.requireApproval || undefined,
    on_violation: {
      allowed_resources: resources ? form.resourcesAction : undefined,
      allowed_authorization_detail_types: detailTypes ? form.detailTypesAction : undefined,
      issuance_rate: rate ? form.rateAction : undefined,
      expires_at: expiresAt ? form.expiresAction : undefined,
    },
  }
}

function PolicySummary({ policy }: { policy: AgentGuardrailPolicy }) {
  const t = useDictionary(agentGuardrailDictionary)
  const rows: [string, string | undefined][] = [
    [t.resourcesLabel, policy.allowed_resources?.join(', ')],
    [t.detailTypesLabel, policy.allowed_authorization_detail_types?.join(', ')],
    [t.maxIssuancesLabel, policy.issuance_rate?.max_issuances.toString()],
    [t.windowSecondsLabel, policy.issuance_rate?.window_seconds.toString()],
    [t.ttlLabel, policy.max_token_ttl_seconds?.toString()],
    [t.expiresAtLabel, policy.expires_at],
    [t.depthLabel, policy.max_delegation_depth?.toString()],
    [t.requireApprovalLabel, policy.require_approval ? '✓' : undefined],
    [t.version, policy.version.slice(0, 12)],
  ]
  return (
    <dl className="grid grid-cols-1 gap-x-4 gap-y-1 text-sm sm:grid-cols-2">
      {rows
        .filter(([, value]) => value !== undefined)
        .map(([label, value]) => (
          <div key={label}>
            <dt className="text-slate-500">{label}</dt>
            <dd className="break-all">{value}</dd>
          </div>
        ))}
    </dl>
  )
}

function ActionSelect({
  id,
  value,
  onChange,
}: {
  id: string
  value: AgentGuardrailAction
  onChange: (value: AgentGuardrailAction) => void
}) {
  const t = useDictionary(agentGuardrailDictionary)
  return (
    <div className="flex items-center gap-2 text-sm">
      <label htmlFor={id}>{t.actionLabel}</label>
      <select
        id={id}
        className="rounded border border-slate-300 bg-white px-2 py-1"
        value={value}
        onChange={(event) => onChange(event.target.value as AgentGuardrailAction)}
      >
        <option value="deny">{t.actionDeny}</option>
        <option value="require_approval">{t.actionApproval}</option>
      </select>
    </div>
  )
}

// AgentGuardrailPanel はエージェントの発行の上限を示す (REQ-OAUTH2-077)。csrfToken がなければ読み取り
// 専用である。読んだバージョンを保存の expected_version に渡し、同時の編集を上書きしない。
export function AgentGuardrailPanel({
  agentID,
  csrfToken,
}: {
  agentID: string
  csrfToken?: string
}) {
  const t = useDictionary(agentGuardrailDictionary)
  const [policies, setPolicies] = useState<AgentGuardrailPolicySet>()
  const [form, setForm] = useState<FormState>(toForm())
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    let active = true
    getAgentGuardrailPolicy(agentID)
      .then((loaded) => {
        if (!active) return
        setPolicies(loaded)
        setForm(toForm(loaded.agent))
      })
      .catch(() => {
        if (active) setError(t.loadFailed)
      })
    return () => {
      active = false
    }
  }, [agentID, t.loadFailed])

  const failure = (cause: unknown) => {
    if (cause instanceof AuthenticationAPIError && cause.code === 'agent_guardrail_conflict')
      return t.conflict
    if (cause instanceof AuthenticationAPIError && cause.code === 'invalid_agent_guardrail')
      return t.invalid
    return t.failed
  }

  const save = async (event: FormEvent) => {
    event.preventDefault()
    if (!csrfToken) return
    setBusy(true)
    setError('')
    setNotice('')
    try {
      const saved = await putAgentGuardrailPolicy(
        csrfToken,
        agentID,
        toLimits(form),
        policies?.agent?.version ?? '',
      )
      setPolicies({ ...policies, agent: saved })
      setNotice(t.saved)
    } catch (cause) {
      setError(failure(cause))
    } finally {
      setBusy(false)
    }
  }

  const remove = async () => {
    if (!csrfToken) return
    setBusy(true)
    setError('')
    setNotice('')
    try {
      await deleteAgentGuardrailPolicy(csrfToken, agentID)
      setPolicies({ ...policies, agent: undefined })
      setForm(toForm())
    } catch (cause) {
      setError(failure(cause))
    } finally {
      setBusy(false)
    }
  }

  const set = <K extends keyof FormState>(key: K, value: FormState[K]) =>
    setForm({ ...form, [key]: value })
  const text = (
    key: 'maxIssuances' | 'windowSeconds' | 'ttl' | 'expiresAt' | 'depth',
    label: string,
  ) => (
    <div className="space-y-1">
      <Label htmlFor={`guardrail-${key}`}>{label}</Label>
      <Input
        id={`guardrail-${key}`}
        value={form[key]}
        onChange={(event) => set(key, event.target.value)}
      />
    </div>
  )

  return (
    <Card className="w-full max-w-2xl space-y-4 p-6">
      <div>
        <h2 className="text-base font-semibold">{t.title}</h2>
        <p className="text-sm text-slate-600">{t.description}</p>
      </div>
      {error ? <Alert variant="destructive">{error}</Alert> : null}
      {notice ? <Alert>{notice}</Alert> : null}
      <section className="space-y-2">
        <h3 className="text-sm font-medium">{t.tenantDefaultTitle}</h3>
        {policies?.tenant_default ? (
          <PolicySummary policy={policies.tenant_default} />
        ) : (
          <p className="text-sm text-slate-600">{t.tenantDefaultNone}</p>
        )}
      </section>
      {!csrfToken ? (
        policies?.agent ? (
          <PolicySummary policy={policies.agent} />
        ) : (
          <p className="text-sm text-slate-600">{t.agentNone}</p>
        )
      ) : (
        <form className="space-y-4" onSubmit={save}>
          <div className="space-y-1">
            <Label htmlFor="guardrail-resources">{t.resourcesLabel}</Label>
            <textarea
              id="guardrail-resources"
              className="w-full rounded border border-slate-300 p-2 text-sm"
              value={form.resources}
              onChange={(event) => set('resources', event.target.value)}
            />
            <ActionSelect
              id="guardrail-resources-action"
              value={form.resourcesAction}
              onChange={(v) => set('resourcesAction', v)}
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="guardrail-detail-types">{t.detailTypesLabel}</Label>
            <textarea
              id="guardrail-detail-types"
              className="w-full rounded border border-slate-300 p-2 text-sm"
              value={form.detailTypes}
              onChange={(event) => set('detailTypes', event.target.value)}
            />
            <ActionSelect
              id="guardrail-detail-types-action"
              value={form.detailTypesAction}
              onChange={(v) => set('detailTypesAction', v)}
            />
          </div>
          <div className="grid gap-3 sm:grid-cols-2">
            {text('maxIssuances', t.maxIssuancesLabel)}
            {text('windowSeconds', t.windowSecondsLabel)}
          </div>
          <ActionSelect
            id="guardrail-rate-action"
            value={form.rateAction}
            onChange={(v) => set('rateAction', v)}
          />
          {text('expiresAt', t.expiresAtLabel)}
          <ActionSelect
            id="guardrail-expires-action"
            value={form.expiresAction}
            onChange={(v) => set('expiresAction', v)}
          />
          <div className="grid gap-3 sm:grid-cols-2">
            {text('ttl', t.ttlLabel)}
            {text('depth', t.depthLabel)}
          </div>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={form.requireApproval}
              onChange={(event) => set('requireApproval', event.target.checked)}
            />
            {t.requireApprovalLabel}
          </label>
          <div className="flex justify-end gap-2">
            {policies?.agent ? (
              <Button type="button" variant="outline" disabled={busy} onClick={remove}>
                {t.remove}
              </Button>
            ) : null}
            <Button type="submit" disabled={busy}>
              {t.save}
            </Button>
          </div>
        </form>
      )}
    </Card>
  )
}
