import { type FormEvent, useState } from 'react'
import {
  AuthenticationAPIError,
  previewAgentWorkloadBindings,
  type AgentWorkloadBindingCreateInput,
  type AgentWorkloadBindingPreview,
} from '../../api'
import { FieldError } from '../../components/FieldError'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Select } from '../../components/ui/select'
import { formSchema, requiredText, useFormValidation } from '../../lib/form'
import { useDictionary } from '../../lib/i18n'
import { LENGTH } from '../../lib/lengthLimits'
import type { AdminAgent, AgentWorkloadBinding } from '../../types'
import { adminWorkloadIdentityDictionary } from './AdminWorkloadIdentityPage.i18n'
import { displayNameForID } from './presentation'

export function BindingForm({
  csrfToken,
  trustBundleID,
  agents,
  binding,
  onSave,
  busy = false,
  saveLabel,
}: {
  csrfToken: string
  trustBundleID: string
  agents: AdminAgent[]
  binding?: AgentWorkloadBinding
  onSave: (input: AgentWorkloadBindingCreateInput) => Promise<void>
  busy?: boolean
  saveLabel: string
}) {
  const t = useDictionary(adminWorkloadIdentityDictionary)
  const [pattern, setPattern] = useState(binding?.subject_pattern ?? '')
  const [agentID, setAgentID] = useState(binding?.agent_id ?? agents[0]?.id ?? '')
  const [subject, setSubject] = useState('')
  const [preview, setPreview] = useState<AgentWorkloadBindingPreview | null>(null)
  const [checking, setChecking] = useState(false)
  const [error, setError] = useState('')
  const [reviewedPattern, setReviewedPattern] = useState('')
  const validation = useFormValidation(
    formSchema({ pattern: requiredText(LENGTH.description), agentID: requiredText() }),
    { pattern, agentID },
  )
  const disabled = busy || checking
  const selectedAgent = agents.find((agent) => agent.id === agentID)

  async function checkPattern() {
    const result = await previewAgentWorkloadBindings(csrfToken, trustBundleID, {
      ...(subject !== '' ? { subject } : {}),
      ...(pattern.trim() ? { subject_pattern: pattern.trim() } : {}),
      ...(binding ? { exclude_binding_id: binding.id } : {}),
    })
    setPreview(result)
    if (result.possible_overlaps.length > 0) setReviewedPattern(pattern.trim())
    return result
  }

  async function tryPattern() {
    setChecking(true)
    setError('')
    try {
      await checkPattern()
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.genericActionError)
    } finally {
      setChecking(false)
    }
  }

  async function submit(event: FormEvent) {
    event.preventDefault()
    const input = validation.data
    if (!input || disabled) {
      validation.revealAll()
      return
    }
    setChecking(true)
    setError('')
    try {
      const result = await checkPattern()
      if (result.possible_overlaps.length > 0 && reviewedPattern !== input.pattern) return
      await onSave({ subject_pattern: input.pattern, agent_id: input.agentID })
      if (!binding) {
        setPattern('')
        setPreview(null)
        setReviewedPattern('')
      }
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.genericActionError)
    } finally {
      setChecking(false)
    }
  }

  return (
    <form onSubmit={submit} className="grid gap-4 p-4 md:grid-cols-2">
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="subject_pattern" required>
          {t.subjectPatternLabel}
        </Label>
        <Input
          id="subject_pattern"
          value={pattern}
          disabled={disabled}
          required
          placeholder={t.subjectPatternPlaceholder}
          onChange={(event) => {
            setPattern(event.target.value)
            setPreview(null)
            setReviewedPattern('')
          }}
          {...validation.fieldProps('pattern')}
        />
        <FieldError form={validation} name="pattern" />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="agent_id" required>
          {t.agentLabel}
        </Label>
        <Select
          id="agent_id"
          value={agentID}
          onValueChange={setAgentID}
          aria-label={t.agentLabel}
          disabled={disabled || agents.length === 0}
          options={agents.map((agent) => ({
            value: agent.id,
            label: agent.client_ids.length ? agent.name : `${agent.name} (${t.unboundAgent})`,
          }))}
        />
        {selectedAgent?.client_ids.length === 0 ? (
          <p className="text-xs text-amber-700">{t.unboundAgentHelp}</p>
        ) : null}
      </div>
      <div className="flex flex-col gap-1.5 md:col-span-2">
        <Label htmlFor="sample_subject">{t.sampleSubject}</Label>
        <Input
          id="sample_subject"
          value={subject}
          disabled={disabled}
          onChange={(event) => {
            setSubject(event.target.value)
            setPreview(null)
          }}
        />
        <p className="text-xs text-slate-500">{t.overlapHelp}</p>
        <Button
          type="button"
          variant="outline"
          disabled={disabled || (!pattern.trim() && !subject)}
          onClick={() => void tryPattern()}
        >
          {checking ? t.previewing : t.previewPattern}
        </Button>
      </div>
      {preview ? (
        <div className="space-y-2 md:col-span-2" aria-live="polite">
          {subject !== '' ? (
            <>
              <p>
                {preview.matches.length === 0
                  ? t.previewNone
                  : preview.matches.length === 1
                    ? t.previewOne
                    : t.previewMany}
              </p>
              <ul>
                {preview.matches.map((match) => (
                  <li key={match.id}>
                    {match.subject_pattern} → {displayNameForID(match.agent_id, agents)}
                  </li>
                ))}
              </ul>
              {preview.candidate_matches !== undefined ? (
                <p>{preview.candidate_matches ? t.candidateMatches : t.candidateDoesNotMatch}</p>
              ) : null}
            </>
          ) : null}
          {preview.possible_overlaps.length > 0 ? (
            <div
              role="status"
              className="rounded-md border border-amber-200 bg-amber-50 p-3 text-amber-950"
            >
              <p>{t.overlapWarning}</p>
              <ul>
                {preview.possible_overlaps.map((match) => (
                  <li key={match.id}>
                    {match.subject_pattern} → {displayNameForID(match.agent_id, agents)}
                  </li>
                ))}
              </ul>
            </div>
          ) : null}
        </div>
      ) : null}
      {error ? (
        <Alert variant="destructive" className="md:col-span-2">
          {error}
        </Alert>
      ) : null}
      <div className="md:col-span-2">
        <Button type="submit" disabled={disabled || !validation.data || agents.length === 0}>
          {preview?.possible_overlaps.length ? t.saveWithOverlap : saveLabel}
        </Button>
        {agents.length === 0 ? <p className="text-xs text-slate-500">{t.noAgentsNotice}</p> : null}
      </div>
    </form>
  )
}
