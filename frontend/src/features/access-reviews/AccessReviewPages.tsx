import { useState } from 'react'
import {
  createAccessReviewPlan,
  updateAccessReviewPlan,
  listAccessReviewPlans,
  listAccessReviews,
  listMyAccessReviews,
  decideAccessReview,
  type AccessReview,
  type AccessReviewPlan,
  type AccessReviewPlanInput,
  type ReviewTargetKind,
  type ReviewDecision,
} from '../../api/accessReviews'
import { AuthenticationAPIError, tenantURL } from '../../api/core'
import { AdminShell } from '../../components/AdminShell'
import { AccountShell } from '../../components/AccountShell'
import { ConfirmActionDialog } from '../../components/ConfirmActionDialog'
import { FieldError } from '../../components/FieldError'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Toast } from '../../components/ui/toast'
import {
  formSchema,
  requiredText,
  requiredList,
  integerInRange,
  rule,
  useFormValidation,
} from '../../lib/form'
import { useDictionary, useFormatters } from '../../lib/i18n'
import type { AdminUser, AdminGroup, AdminAgent, AdminApplication } from '../../types'
import { accessReviewsDictionary } from './accessReviews.i18n'

type IdentityProps = { csrfToken: string; actorUsername?: string }
type Catalogs = {
  users: AdminUser[]
  groups: AdminGroup[]
  agents: AdminAgent[]
  applications: AdminApplication[]
}

function planURL(id: string) {
  return tenantURL(`/admin/access-reviews/${encodeURIComponent(id)}`)
}

export function AccessReviewPlansPage({
  actorUsername,
  plans: initial,
}: IdentityProps & { plans: AccessReviewPlan[] }) {
  const t = useDictionary(accessReviewsDictionary)
  const { formatDateTime } = useFormatters()
  const [plans, setPlans] = useState(initial)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  async function refresh() {
    setBusy(true)
    try {
      setPlans(await listAccessReviewPlans())
      setError('')
    } catch {
      setError(t.failed)
    } finally {
      setBusy(false)
    }
  }
  return (
    <AdminShell
      active="access-reviews"
      actorUsername={actorUsername}
      title={t.title}
      description={t.description}
    >
      {error && <Alert variant="destructive">{error}</Alert>}
      <div className="mb-5 flex gap-3">
        <Button nativeButton={false} render={<a href={tenantURL('/admin/access-reviews/new')} />}>
          {t.create}
        </Button>
        <Button variant="outline" disabled={busy} onClick={refresh}>
          {busy ? t.working : t.refresh}
        </Button>
      </div>
      {plans.length === 0 ? (
        <p>{t.emptyPlans}</p>
      ) : (
        <div className="space-y-3">
          {plans.map((plan) => (
            <Card className="flex flex-wrap items-center justify-between gap-3 p-5" key={plan.id}>
              <div>
                <h2 className="font-semibold">{plan.name}</h2>
                <p>
                  {t[plan.target_kind]} · {plan.enabled ? t.enabledYes : t.enabledNo}
                </p>
                <p>
                  {t.nextStart}:{' '}
                  {plan.enabled && plan.next_start_at
                    ? formatDateTime(plan.next_start_at)
                    : t.noNext}
                </p>
                {plan.last_error && <Alert variant="destructive">{t.startFailed}</Alert>}
              </div>
              <Button variant="outline" nativeButton={false} render={<a href={planURL(plan.id)} />}>
                {t.detail}
              </Button>
            </Card>
          ))}
        </div>
      )}
    </AdminShell>
  )
}

function localDateTime(value: string) {
  const date = new Date(value)
  return new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 16)
}

export function AccessReviewPlanEditor({
  csrfToken,
  actorUsername,
  initial,
  users,
  groups,
  agents,
  applications,
}: IdentityProps & Catalogs & { initial?: AccessReviewPlan }) {
  const t = useDictionary(accessReviewsDictionary)
  const [name, setName] = useState(initial?.name ?? '')
  const [kind, setKind] = useState<ReviewTargetKind>(initial?.target_kind ?? 'group_membership')
  const [resources, setResources] = useState(initial?.resource_ids ?? [])
  const [reviewer, setReviewer] = useState(initial?.reviewer_id ?? '')
  const [fallback, setFallback] = useState(initial?.fallback_reviewer_id ?? '')
  const [decision, setDecision] = useState<ReviewDecision>(initial?.default_decision ?? 'retain')
  const [starts, setStarts] = useState(
    localDateTime(initial?.starts_at ?? new Date().toISOString()),
  )
  const [duration, setDuration] = useState(String(initial?.duration_days ?? 14))
  const [recurrence, setRecurrence] = useState(String(initial?.recurrence_days ?? 0))
  const [enabled, setEnabled] = useState(initial?.enabled ?? true)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [confirming, setConfirming] = useState(false)
  const schema = formSchema({
    name: requiredText(200),
    reviewer: requiredText(),
    fallback: requiredText(),
    resources: rule(
      requiredList((value) => value.split('\n').filter(Boolean)),
      (ids) => (ids.length > 50 ? { kind: 'tooMany', max: 50 } : undefined),
    ),
    starts: rule(requiredText(), (value) =>
      Number.isNaN(Date.parse(value)) ? { kind: 'dateTime' } : undefined,
    ),
    duration: integerInRange(1, 90, { required: true }),
    recurrence: rule(integerInRange(0, 365, { required: true }), (value) =>
      value !== undefined && value > 0 && value < Number(duration)
        ? { kind: 'message', text: t.repeatError }
        : undefined,
    ),
  })
  const validation = useFormValidation(schema, {
    name,
    reviewer: kind === 'agent' ? fallback : reviewer,
    fallback,
    resources: resources.join('\n'),
    starts,
    duration,
    recurrence,
  })
  const activeUsers = users.filter((user) => !user.status || user.status === 'active')
  const choices =
    kind === 'group_membership'
      ? groups.filter((group) => group.membership_type !== 'dynamic' && !group.upstream_managed)
      : kind === 'agent'
        ? agents
        : applications
  async function persist() {
    const data = validation.data
    if (!data) {
      validation.revealAll()
      return
    }
    setBusy(true)
    setError('')
    const input: AccessReviewPlanInput = {
      name: data.name,
      target_kind: kind,
      resource_ids: data.resources,
      reviewer_id: kind === 'agent' ? undefined : reviewer,
      fallback_reviewer_id: fallback,
      default_decision: decision,
      starts_at:
        initial && starts === localDateTime(initial.starts_at)
          ? initial.starts_at
          : new Date(starts).toISOString(),
      duration_days: data.duration ?? 14,
      recurrence_days: data.recurrence ?? 0,
      enabled,
    }
    try {
      const plan = initial
        ? await updateAccessReviewPlan(csrfToken, initial, input)
        : await createAccessReviewPlan(csrfToken, input)
      window.location.assign(`${planURL(plan.id)}?saved=1`)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.failed)
      setConfirming(false)
      setBusy(false)
    }
  }
  return (
    <AdminShell
      active="access-reviews"
      actorUsername={actorUsername}
      title={initial ? t.edit : t.create}
      description={t.description}
    >
      <form
        className="grid max-w-4xl gap-5 sm:grid-cols-2"
        onSubmit={(event) => {
          event.preventDefault()
          if (!validation.data) {
            validation.revealAll()
            return
          }
          if (decision === 'revoke') setConfirming(true)
          else void persist()
        }}
      >
        {error && (
          <Alert className="sm:col-span-2" variant="destructive">
            {error}
          </Alert>
        )}
        <div>
          <Label htmlFor="review-name" required>
            {t.name}
          </Label>
          <Input
            id="review-name"
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
            {...validation.fieldProps('name')}
          />
          <FieldError form={validation} name="name" />
        </div>
        <div>
          <Label htmlFor="review-kind" required>
            {t.kind}
          </Label>
          <select
            id="review-kind"
            className="w-full rounded border p-2"
            value={kind}
            onChange={(e) => {
              setKind(e.target.value as ReviewTargetKind)
              setResources([])
            }}
          >
            {(['group_membership', 'application_assignment', 'agent'] as const).map((value) => (
              <option value={value} key={value}>
                {t[value]}
              </option>
            ))}
          </select>
        </div>
        <fieldset
          className="sm:col-span-2"
          {...validation.fieldProps('resources')}
          aria-describedby={[
            'review-target-help',
            validation.problem('resources') ? validation.errorId('resources') : '',
          ]
            .filter(Boolean)
            .join(' ')}
        >
          <legend className="font-medium">{t.resources}</legend>
          <p id="review-target-help" className="my-2 text-sm text-slate-600">
            {kind === 'agent' ? t.ownerHelp : kind === 'group_membership' ? t.groupHelp : t.appHelp}
          </p>
          {choices.length === 0 && <p>{t.noResources}</p>}
          <div className="grid max-h-64 gap-2 overflow-auto sm:grid-cols-2">
            {choices.map((choice) => (
              <label className="flex items-center gap-2" key={choice.id}>
                <input
                  type="checkbox"
                  value={choice.id}
                  checked={resources.includes(choice.id)}
                  onChange={(e) =>
                    setResources((current) =>
                      e.target.checked
                        ? [...current, choice.id]
                        : current.filter((id) => id !== choice.id),
                    )
                  }
                />
                {choice.name}
              </label>
            ))}
          </div>
          <FieldError form={validation} name="resources" />
        </fieldset>
        {kind !== 'agent' && (
          <div>
            <Label htmlFor="review-reviewer" required>
              {t.reviewer}
            </Label>
            <select
              id="review-reviewer"
              className="w-full rounded border p-2"
              required
              value={reviewer}
              onChange={(e) => setReviewer(e.target.value)}
              {...validation.fieldProps('reviewer')}
            >
              <option value="">{t.reviewer}</option>
              {activeUsers.map((user) => (
                <option key={user.id} value={user.id}>
                  {user.name ?? user.preferred_username}
                </option>
              ))}
            </select>
            <FieldError form={validation} name="reviewer" />
          </div>
        )}
        <div>
          <Label htmlFor="review-fallback" required>
            {t.fallback}
          </Label>
          <select
            id="review-fallback"
            className="w-full rounded border p-2"
            required
            value={fallback}
            onChange={(e) => setFallback(e.target.value)}
            {...validation.fieldProps('fallback')}
          >
            <option value="">{t.fallback}</option>
            {activeUsers.map((user) => (
              <option key={user.id} value={user.id}>
                {user.name ?? user.preferred_username}
              </option>
            ))}
          </select>
          <FieldError form={validation} name="fallback" />
          {activeUsers.length === 0 && <p>{t.noReviewers}</p>}
        </div>
        <div>
          <Label htmlFor="review-start" required>
            {t.starts}
          </Label>
          <Input
            id="review-start"
            type="datetime-local"
            required
            value={starts}
            onChange={(e) => setStarts(e.target.value)}
            {...validation.fieldProps('starts')}
          />
          <FieldError form={validation} name="starts" />
        </div>
        <div>
          <Label htmlFor="review-duration" required>
            {t.duration}
          </Label>
          <Input
            id="review-duration"
            type="number"
            min={1}
            max={90}
            required
            value={duration}
            onChange={(e) => setDuration(e.target.value)}
            {...validation.fieldProps('duration')}
          />
          <FieldError form={validation} name="duration" />
        </div>
        <div>
          <Label htmlFor="review-recurrence" required>
            {t.recurrence}
          </Label>
          <Input
            id="review-recurrence"
            type="number"
            min={0}
            max={365}
            required
            value={recurrence}
            onChange={(e) => setRecurrence(e.target.value)}
            {...validation.fieldProps('recurrence')}
          />
          <FieldError form={validation} name="recurrence" />
        </div>
        <div>
          <Label htmlFor="review-default" required>
            {t.defaultDecision}
          </Label>
          <select
            id="review-default"
            className="w-full rounded border p-2"
            value={decision}
            onChange={(e) => setDecision(e.target.value as ReviewDecision)}
          >
            <option value="retain">{t.retain}</option>
            <option value="revoke">{t.revoke}</option>
          </select>
        </div>
        <label className="flex items-center gap-2">
          <input type="checkbox" checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />
          {t.enabled}
        </label>
        <div className="flex gap-3 sm:col-span-2">
          <Button type="submit" disabled={busy || !validation.data}>
            {busy ? t.saving : t.save}
          </Button>
          <Button
            variant="outline"
            nativeButton={false}
            render={<a href={initial ? planURL(initial.id) : tenantURL('/admin/access-reviews')} />}
          >
            {t.cancel}
          </Button>
        </div>
      </form>
      {confirming && (
        <ConfirmActionDialog
          title={t.confirmCreateTitle}
          target={name}
          description={t.confirmCreate}
          confirmLabel={t.confirm}
          busy={busy}
          onClose={() => setConfirming(false)}
          onConfirm={() => void persist()}
        />
      )}
    </AdminShell>
  )
}

export function AccessReviewPlanPage({
  csrfToken,
  actorUsername,
  plan: initial,
  reviews: initialReviews,
  saved = false,
  catalogs,
}: IdentityProps & {
  plan: AccessReviewPlan
  reviews: AccessReview[]
  saved?: boolean
  catalogs?: Catalogs
}) {
  const t = useDictionary(accessReviewsDictionary)
  const { formatDateTime } = useFormatters()
  const [plan, setPlan] = useState(initial)
  const [reviews, setReviews] = useState(initialReviews)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState(saved ? t.saved : '')
  const [busy, setBusy] = useState(false)
  const [confirming, setConfirming] = useState(false)
  const reviewerName = (id: string) =>
    catalogs?.users.find((user) => user.id === id)?.preferred_username ?? id
  const resources =
    plan.target_kind === 'agent'
      ? catalogs?.agents
      : plan.target_kind === 'application_assignment'
        ? catalogs?.applications
        : catalogs?.groups
  async function toggle() {
    setBusy(true)
    try {
      setPlan(await updateAccessReviewPlan(csrfToken, plan, { ...plan, enabled: !plan.enabled }))
      setError('')
      setNotice(t.saved)
      setConfirming(false)
    } catch {
      setError(t.failed)
    } finally {
      setBusy(false)
    }
  }
  async function refresh() {
    setBusy(true)
    try {
      const [plans, results] = await Promise.all([
        listAccessReviewPlans(),
        listAccessReviews(plan.id),
      ])
      const next = plans.find((item) => item.id === plan.id)
      if (next) setPlan(next)
      setReviews(results)
      setError('')
    } catch {
      setError(t.failed)
    } finally {
      setBusy(false)
    }
  }
  return (
    <AdminShell
      active="access-reviews"
      actorUsername={actorUsername}
      title={plan.name}
      description={t.description}
    >
      <Toast message={notice} onDismiss={() => setNotice('')} />
      {error && <Alert variant="destructive">{error}</Alert>}
      {plan.last_error && <Alert variant="destructive">{t.startFailed}</Alert>}
      <div className="mb-5 flex flex-wrap gap-3">
        <Button
          variant="outline"
          nativeButton={false}
          render={<a href={tenantURL('/admin/access-reviews')} />}
        >
          {t.back}
        </Button>
        <Button
          variant="outline"
          nativeButton={false}
          render={<a href={`${planURL(plan.id)}/edit`} />}
        >
          {t.edit}
        </Button>
        <Button
          variant={plan.enabled ? 'destructive' : 'outline'}
          disabled={busy}
          onClick={() => (plan.enabled ? setConfirming(true) : void toggle())}
        >
          {plan.enabled ? t.disable : t.enable}
        </Button>
        <Button variant="outline" disabled={busy} onClick={refresh}>
          {busy ? t.working : t.refresh}
        </Button>
      </div>
      <Card className="mb-6 space-y-2 p-5">
        <p>
          {t.resources}:{' '}
          {plan.resource_ids
            .map((id) => resources?.find((resource) => resource.id === id)?.name ?? id)
            .join(', ')}
        </p>
        <p>
          {t.kind}: {t[plan.target_kind]}
        </p>
        <p>
          {t.defaultDecision}: {t[plan.default_decision]}
        </p>
        <p>
          {t.duration}: {plan.duration_days}
        </p>
        <p>
          {t.recurrence}: {plan.recurrence_days}
        </p>
        <p>
          {t.reviewer}:{' '}
          {plan.target_kind === 'agent' ? t.ownerHelp : reviewerName(plan.reviewer_id ?? '')}
        </p>
        <p>
          {t.fallback}: {reviewerName(plan.fallback_reviewer_id)}
        </p>
        <p>
          {t.nextStart}:{' '}
          {plan.enabled && plan.next_start_at ? formatDateTime(plan.next_start_at) : t.noNext}
        </p>
      </Card>
      <h2 className="mb-3 text-lg font-semibold">{t.results}</h2>
      {reviews.length === 0 && <p>{t.emptyReviews}</p>}
      <ReviewResults reviews={reviews} reviewerName={reviewerName} />
      {confirming && (
        <ConfirmActionDialog
          title={t.disable}
          target={plan.name}
          description={t.confirmDisable}
          confirmLabel={t.confirm}
          busy={busy}
          error={error}
          onClose={() => setConfirming(false)}
          onConfirm={() => void toggle()}
        />
      )}
    </AdminShell>
  )
}

function ReviewResults({
  reviews,
  reviewerName,
}: {
  reviews: AccessReview[]
  reviewerName: (id: string) => string
}) {
  const t = useDictionary(accessReviewsDictionary)
  const { formatDateTime } = useFormatters()
  return (
    <div className="space-y-4">
      {reviews.map((review) => (
        <Card className="overflow-x-auto p-5" key={review.id}>
          <h3 className="font-semibold">
            {review.name} · {t[review.status]}
          </h3>
          <p>
            {t.due}: {formatDateTime(review.due_at)}
          </p>
          {review.notification_failed && <Alert variant="destructive">{t.notifyFailed}</Alert>}
          {review.items.length === 0 ? (
            <p className="mt-3">{t.emptyItems}</p>
          ) : (
            <table className="mt-3 w-full text-left text-sm">
              <thead>
                <tr>
                  <th>{t.resources}</th>
                  <th>{t.subject}</th>
                  <th>{t.reviewer}</th>
                  <th>{t.decision}</th>
                  <th>{t.outcome}</th>
                </tr>
              </thead>
              <tbody>
                {review.items.map((item) => (
                  <tr key={item.id} className="border-t">
                    <td className="p-2">{item.resource_name}</td>
                    <td>{item.subject_name}</td>
                    <td>{reviewerName(item.reviewer_id)}</td>
                    <td>
                      {item.decision ? t[item.decision] : t.unanswered}
                      {item.defaulted ? ` (${t.defaulted})` : ''}
                    </td>
                    <td>
                      {item.error_code
                        ? t.applyFailed
                        : item.outcome === 'revoked'
                          ? t.revoked
                          : item.outcome === 'retained'
                            ? t.retained
                            : t.pending}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </Card>
      ))}
    </div>
  )
}

export function MyAccessReviewsPage({
  csrfToken,
  username,
  isAdmin,
  reviews: initial,
}: {
  csrfToken: string
  username: string
  isAdmin: boolean
  reviews: AccessReview[]
}) {
  const t = useDictionary(accessReviewsDictionary)
  const { formatDateTime } = useFormatters()
  const [reviews, setReviews] = useState(initial)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [busy, setBusy] = useState(false)
  const [confirming, setConfirming] = useState<{
    review: AccessReview
    itemID: string
    name: string
  } | null>(null)
  async function answer(review: AccessReview, itemID: string, decision: ReviewDecision) {
    setBusy(true)
    try {
      const updated = await decideAccessReview(csrfToken, review.id, itemID, decision)
      setReviews((current) => current.map((item) => (item.id === updated.id ? updated : item)))
      setError('')
      setNotice(t.answered)
      setConfirming(null)
    } catch (cause) {
      setError(cause instanceof AuthenticationAPIError ? cause.message : t.failed)
    } finally {
      setBusy(false)
    }
  }
  async function refresh() {
    setBusy(true)
    try {
      setReviews(await listMyAccessReviews())
      setError('')
    } catch {
      setError(t.failed)
    } finally {
      setBusy(false)
    }
  }
  return (
    <AccountShell
      active="access-reviews"
      username={username}
      isAdmin={isAdmin}
      title={t.title}
      description={t.finalHelp}
    >
      <Toast message={notice} onDismiss={() => setNotice('')} />
      {error && <Alert variant="destructive">{error}</Alert>}
      <Button variant="outline" disabled={busy} onClick={refresh}>
        {busy ? t.working : t.refresh}
      </Button>
      {reviews.length === 0 && <p className="mt-5">{t.emptyTasks}</p>}
      <div className="mt-5 space-y-4">
        {reviews.map((review) => (
          <Card className="space-y-3 p-5" key={review.id}>
            <h2 className="font-semibold">{review.name}</h2>
            <p>
              {t.due}: {formatDateTime(review.due_at)}
            </p>
            <p>
              {t.defaultDecision}: {t[review.default_decision]}
            </p>
            {review.target_kind === 'application_assignment' && (
              <p className="text-sm">{t.appHelp}</p>
            )}
            {review.items.map((item) => (
              <div
                key={item.id}
                className="flex flex-wrap items-center justify-between gap-3 border-t py-3"
              >
                <div>
                  <p>
                    {item.resource_name} / {item.subject_name}
                  </p>
                  <p>
                    {t.decision}: {item.decision ? t[item.decision] : t.unanswered}
                  </p>
                  <p>
                    {t.outcome}:{' '}
                    {item.error_code
                      ? t.applyFailed
                      : item.outcome === 'revoked'
                        ? t.revoked
                        : item.outcome === 'retained'
                          ? t.retained
                          : t.pending}
                  </p>
                </div>
                {review.status === 'open' &&
                  !item.decision &&
                  Date.now() < Date.parse(review.due_at) && (
                    <div className="flex gap-2">
                      <Button
                        variant="outline"
                        disabled={busy}
                        onClick={() => void answer(review, item.id, 'retain')}
                      >
                        {t.chooseRetain}
                      </Button>
                      <Button
                        variant="destructive"
                        disabled={busy}
                        onClick={() => {
                          setError('')
                          setConfirming({
                            review,
                            itemID: item.id,
                            name: `${item.resource_name} / ${item.subject_name}`,
                          })
                        }}
                      >
                        {t.chooseRevoke}
                      </Button>
                    </div>
                  )}
              </div>
            ))}
          </Card>
        ))}
      </div>
      {confirming && (
        <ConfirmActionDialog
          title={t.confirmTitle}
          target={confirming.name}
          description={t.confirmRevoke}
          confirmLabel={t.confirm}
          busy={busy}
          error={error}
          onClose={() => setConfirming(null)}
          onConfirm={() => void answer(confirming.review, confirming.itemID, 'revoke')}
        />
      )}
    </AccountShell>
  )
}
