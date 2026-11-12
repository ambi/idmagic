import type { ReactNode } from 'react'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import type { ResourceAccessCondition, ResourceAccessLimit, ResourceAccessRule } from '../../types'
import {
  type AccessRuleProblem,
  accessRuleProblems,
  constraintsRepresentable,
  hasProblems,
  LIMIT_UNITS,
  MAX_EXCHANGE_RESOURCES,
  MAX_LIMITS,
} from './accessRuleProblems'
import type { adminMcpResourceServersDictionary } from './AdminMcpResourceServersPage.i18n'

type Dictionary = (typeof adminMcpResourceServersDictionary)['en']

const CONSTRAINT_KEYS = [
  'requires_approved_operation',
  'authorization_detail_type',
  'condition',
  'limits',
  'token_exchange_resources',
] as const

const textareaClass =
  'rounded-md border border-slate-300 bg-white p-2 font-mono text-sm text-slate-900 focus:border-blue-500 focus:outline-none'

// 空の入力は項目を除く。JSON.stringify は undefined の項目を書かない。
const orUndefined = (value: string) => (value === '' ? undefined : value)

function nextCondition(
  condition: ResourceAccessCondition | undefined,
  change: Partial<ResourceAccessCondition>,
): ResourceAccessCondition | undefined {
  const next = { ...condition, ...change }
  // 式と明細の種類がすべて空なら、条件そのものを除く (REQ-OAUTH2-090)。
  return next.deny_when || next.require_approval_when || next.approval_detail_type
    ? next
    : undefined
}

// described は欄の問題を表示する間だけ、エラー文への参照と aria-invalid を付ける。
// fieldset には aria-invalid を使えないので、群では aria-describedby だけを使う。
function described(id: string, problem: AccessRuleProblem | undefined, help?: string) {
  const ids = [help, problem ? `${id}-error` : undefined].filter(Boolean).join(' ')
  return {
    'aria-invalid': problem ? (true as const) : undefined,
    'aria-describedby': ids === '' ? undefined : ids,
  }
}

function RowError({ id, problem, t }: { id: string; problem?: AccessRuleProblem; t: Dictionary }) {
  if (!problem) return null
  return (
    <p id={`${id}-error`} className="text-xs leading-5 text-destructive">
      {t[problem]}
    </p>
  )
}

function Field({ children }: { children: ReactNode }) {
  return <div className="flex flex-col gap-1">{children}</div>
}

/**
 * AccessRuleConstraints は規則一件の承認、上限、交換先、条件を入力欄で編集する (REQ-OAUTH2-090)。
 * 入力欄で表せない型の制約は読み取り専用に示し、JSON の入力で直させる。
 * `showProblems` はフォームが accessRules の欄の問題を表示する間だけ真にする。
 */
export function AccessRuleConstraints({
  rule,
  index,
  showProblems,
  onChange,
  t,
}: {
  rule: ResourceAccessRule
  index: number
  showProblems: boolean
  onChange: (change: Partial<ResourceAccessRule>) => void
  t: Dictionary
}) {
  const configured = CONSTRAINT_KEYS.some((key) => rule[key] !== undefined)
  if (!constraintsRepresentable(rule)) {
    const constraints = Object.fromEntries(
      CONSTRAINT_KEYS.filter((key) => rule[key] !== undefined).map((key) => [key, rule[key]]),
    )
    return (
      <details className="w-full text-sm" open>
        <summary className="cursor-pointer text-slate-600">{t.ruleConstraints}</summary>
        <p className="mt-2 text-sm text-slate-600">{t.ruleConstraintsUnreadable}</p>
        <pre className="mt-1 overflow-x-auto rounded bg-slate-50 p-2 text-xs">
          {JSON.stringify(constraints, null, 2)}
        </pre>
      </details>
    )
  }
  const all = accessRuleProblems(rule)
  const problems = showProblems ? all : { tokenExchangeEntries: [], limits: [] }
  const approval = rule.requires_approved_operation === true
  const condition = rule.condition
  const limits = rule.limits ?? []
  const exchange = rule.token_exchange_resources ?? []
  const id = (name: string) => `rule-${name}-${index}`
  const setCondition = (change: Partial<ResourceAccessCondition>) =>
    onChange({ condition: nextCondition(condition, change) })
  const setLimit = (position: number, change: Partial<ResourceAccessLimit>) =>
    onChange({
      limits: limits.map((limit, at) => (at === position ? { ...limit, ...change } : limit)),
    })
  const setExchange = (next: string[]) =>
    onChange({ token_exchange_resources: next.length > 0 ? next : undefined })
  const unitLabel = {
    agent: t.ruleLimitByAgent,
    user: t.ruleLimitByUser,
    resource: t.ruleLimitByResource,
  }
  return (
    <details className="w-full text-sm" open={configured || (showProblems && hasProblems(all))}>
      <summary className="cursor-pointer text-slate-600">{t.ruleConstraints}</summary>
      <div className="mt-2 flex flex-col gap-3">
        <div className="flex flex-wrap items-end gap-3">
          <label className="flex items-center gap-2 text-sm">
            <input
              id={id('approval')}
              type="checkbox"
              checked={approval}
              onChange={(event) =>
                onChange({ requires_approved_operation: event.target.checked || undefined })
              }
            />
            {t.ruleApproval}
          </label>
          <Field>
            <Label htmlFor={id('detail-type')} required={approval}>
              {t.ruleDetailType}
            </Label>
            <Input
              id={id('detail-type')}
              value={rule.authorization_detail_type ?? ''}
              required={approval}
              onChange={(event) =>
                onChange({ authorization_detail_type: orUndefined(event.target.value) })
              }
              {...described(
                id('detail-type'),
                problems.authorizationDetailType,
                id('detail-type-help'),
              )}
            />
            <p id={id('detail-type-help')} className="text-xs leading-5 text-slate-500">
              {t.ruleDetailTypeHelp}
            </p>
            <RowError id={id('detail-type')} problem={problems.authorizationDetailType} t={t} />
          </Field>
        </div>
        <fieldset className="flex flex-col gap-2" aria-describedby={id('condition-help')}>
          <legend className="text-sm font-medium">{t.ruleCondition}</legend>
          <p id={id('condition-help')} className="text-xs leading-5 text-slate-500">
            {t.ruleConditionHelp}
          </p>
          <Field>
            <Label htmlFor={id('deny-when')}>{t.ruleDenyWhen}</Label>
            <textarea
              id={id('deny-when')}
              rows={2}
              className={textareaClass}
              value={condition?.deny_when ?? ''}
              onChange={(event) => setCondition({ deny_when: orUndefined(event.target.value) })}
              {...described(id('deny-when'), problems.denyWhen)}
            />
            <RowError id={id('deny-when')} problem={problems.denyWhen} t={t} />
          </Field>
          <Field>
            <Label htmlFor={id('require-approval-when')}>{t.ruleRequireApprovalWhen}</Label>
            <textarea
              id={id('require-approval-when')}
              rows={2}
              className={textareaClass}
              value={condition?.require_approval_when ?? ''}
              onChange={(event) =>
                setCondition({ require_approval_when: orUndefined(event.target.value) })
              }
              {...described(id('require-approval-when'), problems.requireApprovalWhen)}
            />
            <RowError
              id={id('require-approval-when')}
              problem={problems.requireApprovalWhen}
              t={t}
            />
          </Field>
          <Field>
            <Label
              htmlFor={id('approval-detail-type')}
              required={Boolean(condition?.require_approval_when)}
            >
              {t.ruleApprovalDetailType}
            </Label>
            <Input
              id={id('approval-detail-type')}
              value={condition?.approval_detail_type ?? ''}
              required={Boolean(condition?.require_approval_when)}
              onChange={(event) =>
                setCondition({ approval_detail_type: orUndefined(event.target.value) })
              }
              {...described(id('approval-detail-type'), problems.approvalDetailType)}
            />
            <RowError id={id('approval-detail-type')} problem={problems.approvalDetailType} t={t} />
          </Field>
        </fieldset>
        <fieldset
          className="flex flex-col gap-2"
          aria-describedby={
            described(id('limits'), problems.limitsGroup, id('limits-help'))['aria-describedby']
          }
        >
          <legend className="text-sm font-medium">{t.ruleLimits}</legend>
          <p id={id('limits-help')} className="text-xs leading-5 text-slate-500">
            {t.ruleLimitsHelp}
          </p>
          <RowError id={id('limits')} problem={problems.limitsGroup} t={t} />
          {limits.map((limit, position) => {
            const limitID = (name: string) => `rule-limit-${name}-${index}-${position}`
            const problem = problems.limits[position] ?? {}
            return (
              // 上限は内容で識別できず、並べ替えもない。
              // biome-ignore lint/suspicious/noArrayIndexKey: 位置だけが上限を識別する。
              <div key={position} className="flex flex-wrap items-end gap-3">
                <Field>
                  <Label htmlFor={limitID('max')} required>
                    {t.ruleLimitMaxRequests}
                  </Label>
                  <Input
                    id={limitID('max')}
                    type="number"
                    inputMode="numeric"
                    className="w-40"
                    value={String(limit.max_requests)}
                    required
                    onChange={(event) =>
                      setLimit(position, { max_requests: Number(event.target.value) })
                    }
                    {...described(limitID('max'), problem.maxRequests)}
                  />
                  <RowError id={limitID('max')} problem={problem.maxRequests} t={t} />
                </Field>
                <Field>
                  <Label htmlFor={limitID('window')} required>
                    {t.ruleLimitWindowSeconds}
                  </Label>
                  <Input
                    id={limitID('window')}
                    type="number"
                    inputMode="numeric"
                    className="w-40"
                    value={String(limit.window_seconds)}
                    required
                    onChange={(event) =>
                      setLimit(position, { window_seconds: Number(event.target.value) })
                    }
                    {...described(limitID('window'), problem.windowSeconds)}
                  />
                  <RowError id={limitID('window')} problem={problem.windowSeconds} t={t} />
                </Field>
                <fieldset className="flex flex-wrap items-center gap-3">
                  <legend className="text-sm">{t.ruleLimitBy}</legend>
                  {LIMIT_UNITS.map((unit) => (
                    <label key={unit} className="flex items-center gap-2 text-sm">
                      <input
                        type="checkbox"
                        checked={limit.by?.includes(unit) ?? false}
                        onChange={(event) => {
                          const by = (limit.by ?? []).filter((item) => item !== unit)
                          if (event.target.checked) by.push(unit)
                          setLimit(position, { by: by.length > 0 ? by : undefined })
                        }}
                      />
                      {unitLabel[unit]}
                    </label>
                  ))}
                </fieldset>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => {
                    const next = limits.filter((_, at) => at !== position)
                    onChange({ limits: next.length > 0 ? next : undefined })
                  }}
                >
                  {t.ruleLimitRemove}
                </Button>
              </div>
            )
          })}
          {limits.length < MAX_LIMITS ? (
            <div>
              <Button
                type="button"
                variant="outline"
                onClick={() =>
                  onChange({ limits: [...limits, { max_requests: 100, window_seconds: 3600 }] })
                }
              >
                {t.ruleLimitAdd}
              </Button>
            </div>
          ) : null}
        </fieldset>
        <fieldset
          className="flex flex-col gap-2"
          aria-describedby={
            described(id('exchange'), problems.tokenExchange, id('exchange-help'))[
              'aria-describedby'
            ]
          }
        >
          <legend className="text-sm font-medium">{t.ruleExchange}</legend>
          <p id={id('exchange-help')} className="text-xs leading-5 text-slate-500">
            {t.ruleExchangeHelp}
          </p>
          <RowError id={id('exchange')} problem={problems.tokenExchange} t={t} />
          {exchange.map((resource, position) => {
            const entryID = `rule-exchange-${index}-${position}`
            const problem = problems.tokenExchangeEntries[position]
            return (
              // biome-ignore lint/suspicious/noArrayIndexKey: 宛先は重複し得るので位置で識別する。
              <div key={position} className="flex flex-wrap items-end gap-3">
                <Field>
                  <Label htmlFor={entryID}>{t.ruleExchangeResource}</Label>
                  <Input
                    id={entryID}
                    type="url"
                    className="w-96 max-w-full"
                    value={resource}
                    placeholder="https://api.example.com"
                    onChange={(event) =>
                      setExchange(
                        exchange.map((item, at) => (at === position ? event.target.value : item)),
                      )
                    }
                    {...described(entryID, problem)}
                  />
                  <RowError id={entryID} problem={problem} t={t} />
                </Field>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setExchange(exchange.filter((_, at) => at !== position))}
                >
                  {t.ruleExchangeRemove}
                </Button>
              </div>
            )
          })}
          {exchange.length < MAX_EXCHANGE_RESOURCES ? (
            <div>
              <Button
                type="button"
                variant="outline"
                onClick={() => setExchange([...exchange, ''])}
              >
                {t.ruleExchangeAdd}
              </Button>
            </div>
          ) : null}
        </fieldset>
      </div>
    </details>
  )
}
