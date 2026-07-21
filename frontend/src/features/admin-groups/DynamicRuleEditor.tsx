import { IconPlus, IconTrash } from '@tabler/icons-react'
import { useEffect, useState } from 'react'
import {
  AuthenticationAPIError,
  listAdminUsers,
  previewDynamicGroupRule,
  setDynamicGroupRuleEnabled,
  updateDynamicGroupRule,
} from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Select } from '../../components/ui/select'
import { useDictionary } from '../../lib/i18n'
import type {
  AdminUser,
  DynamicGroupPreview,
  DynamicGroupRule,
  UserAttributeDef,
} from '../../types'
import { adminGroupsDictionary } from './AdminGroupsPage.i18n'
import {
  type BuilderCondition,
  builderAttributeOptions,
  buildDynamicRuleExpression,
  operatorIdsForType,
  operatorNeedsValue,
} from './dynamicRuleCel'

const BUILTIN_ATTRIBUTE_LABEL_KEYS: Record<string, keyof typeof adminGroupsDictionary.en> = {
  id: 'builtinAttrId',
  preferred_username: 'builtinAttrPreferredUsername',
  name: 'builtinAttrName',
  given_name: 'builtinAttrGivenName',
  family_name: 'builtinAttrFamilyName',
  email: 'builtinAttrEmail',
  email_verified: 'builtinAttrEmailVerified',
}

const OPERATOR_LABEL_KEYS: Record<string, keyof typeof adminGroupsDictionary.en> = {
  equals: 'opEquals',
  notEquals: 'opNotEquals',
  contains: 'opContains',
  startsWith: 'opStartsWith',
  endsWith: 'opEndsWith',
  isTrue: 'opIsTrue',
  isFalse: 'opIsFalse',
  before: 'opBefore',
  after: 'opAfter',
  on: 'opOn',
}

const emptyCondition = (): BuilderCondition => ({ attribute: '', operator: '', value: '' })

export function DynamicRuleEditor({
  csrfToken,
  groupId,
  initialRule,
  customAttributes,
}: {
  csrfToken: string
  groupId: string
  initialRule?: DynamicGroupRule
  customAttributes: UserAttributeDef[]
}) {
  const t = useDictionary(adminGroupsDictionary)
  const attributeOptions = builderAttributeOptions(customAttributes)
  const hadInitialExpression = !!initialRule?.expression

  const [rule, setRule] = useState(initialRule)
  // 既存式はビルダーへ逆変換しない (wi-216 Out of Scope) ため、保存済みルールは
  // 詳細設定 (生 CEL) で開き、新規ルールだけビルダーで開く。
  const [mode, setMode] = useState<'builder' | 'advanced'>(
    hadInitialExpression ? 'advanced' : 'builder',
  )
  const [conditions, setConditions] = useState<BuilderCondition[]>([emptyCondition()])
  const [advancedExpression, setAdvancedExpression] = useState(initialRule?.expression ?? '')
  const [allUsers, setAllUsers] = useState<AdminUser[]>([])
  const [previewUserIDs, setPreviewUserIDs] = useState<string[]>([])
  const [preview, setPreview] = useState<DynamicGroupPreview[]>([])
  const [ruleBusy, setRuleBusy] = useState(false)
  const [ruleError, setRuleError] = useState('')

  useEffect(() => {
    let cancelled = false
    void listAdminUsers().then((users) => {
      if (!cancelled) setAllUsers(users)
    })
    return () => {
      cancelled = true
    }
  }, [])

  const expression =
    mode === 'builder' ? buildDynamicRuleExpression(conditions) : advancedExpression

  function attributeLabel(key: string): string {
    const builtinKey = BUILTIN_ATTRIBUTE_LABEL_KEYS[key]
    if (builtinKey) return t[builtinKey]
    const custom = customAttributes.find((def) => def.key === key)
    return custom?.label ?? key
  }

  async function withRule(action: () => Promise<void>) {
    setRuleBusy(true)
    setRuleError('')
    try {
      await action()
    } catch (cause) {
      setRuleError(cause instanceof AuthenticationAPIError ? cause.message : t.genericActionError)
    } finally {
      setRuleBusy(false)
    }
  }

  function updateCondition(index: number, patch: Partial<BuilderCondition>) {
    setConditions((current) =>
      current.map((condition, i) => (i === index ? { ...condition, ...patch } : condition)),
    )
  }

  function switchToAdvanced() {
    if (mode === 'builder') setAdvancedExpression(expression)
    setMode('advanced')
  }

  return (
    <Card className="shadow-[0_1px_2px_rgb(15_23_42/4%)]">
      <div className="grid gap-4 p-6">
        {ruleError && <Alert variant="destructive">{ruleError}</Alert>}
        <div className="flex items-center justify-between gap-3">
          <h3 className="text-xs font-bold uppercase tracking-normal text-slate-400">
            {t.dynamicRuleHeading}
          </h3>
          <span
            className={`rounded-md px-2 py-1 text-xs font-semibold ${rule?.enabled ? 'bg-emerald-100 text-emerald-800' : 'bg-slate-100 text-slate-600'}`}
          >
            {rule?.enabled ? t.ruleEnabled : t.ruleDisabled}
          </span>
        </div>

        <div className="inline-flex w-fit rounded-lg border border-slate-200 bg-slate-50 p-0.5">
          <button
            type="button"
            onClick={() => setMode('builder')}
            className={`rounded-md px-3 py-1 text-sm font-medium transition ${mode === 'builder' ? 'bg-white text-slate-900 shadow-sm' : 'text-slate-500 hover:text-slate-700'}`}
          >
            {t.ruleModeBuilder}
          </button>
          <button
            type="button"
            onClick={switchToAdvanced}
            className={`rounded-md px-3 py-1 text-sm font-medium transition ${mode === 'advanced' ? 'bg-white text-slate-900 shadow-sm' : 'text-slate-500 hover:text-slate-700'}`}
          >
            {t.ruleModeAdvanced}
          </button>
        </div>

        {mode === 'builder' ? (
          <div className="grid gap-3">
            <p className="text-xs text-slate-500">{t.ruleBuilderHelp}</p>
            {conditions.map((condition, index) => {
              const type = attributeOptions.find((o) => o.key === condition.attribute)?.type
              const operatorIds = type ? operatorIdsForType(type) : []
              const showValue = !!condition.operator && operatorNeedsValue(condition.operator)
              return (
                <div
                  // biome-ignore lint/suspicious/noArrayIndexKey: rows are positional and reorder-free
                  key={index}
                  className="flex flex-wrap items-end gap-2 rounded-lg border border-slate-200 p-3"
                >
                  <div className="grid gap-1.5">
                    <Label htmlFor={`rule-attr-${index}`}>{t.ruleAttributeLabel}</Label>
                    <Select
                      id={`rule-attr-${index}`}
                      value={condition.attribute}
                      onValueChange={(value) =>
                        updateCondition(index, { attribute: value, operator: '', value: '' })
                      }
                      placeholder={t.ruleAttributePlaceholder}
                      aria-label={t.ruleAttributeLabel}
                      options={attributeOptions.map((option) => ({
                        value: option.key,
                        label: attributeLabel(option.key),
                      }))}
                    />
                  </div>
                  <div className="grid gap-1.5">
                    <Label htmlFor={`rule-op-${index}`}>{t.ruleOperatorLabel}</Label>
                    <Select
                      id={`rule-op-${index}`}
                      value={condition.operator}
                      disabled={!condition.attribute}
                      onValueChange={(value) => updateCondition(index, { operator: value })}
                      placeholder={t.ruleOperatorPlaceholder}
                      aria-label={t.ruleOperatorLabel}
                      options={operatorIds.map((id) => ({
                        value: id,
                        label: t[OPERATOR_LABEL_KEYS[id]],
                      }))}
                    />
                  </div>
                  {showValue ? (
                    <div className="grid gap-1.5">
                      <Label htmlFor={`rule-value-${index}`}>{t.ruleValueLabel}</Label>
                      <Input
                        id={`rule-value-${index}`}
                        type={type === 'date' ? 'date' : 'text'}
                        value={condition.value}
                        onChange={(event) => updateCondition(index, { value: event.target.value })}
                        aria-label={t.ruleValueLabel}
                      />
                    </div>
                  ) : null}
                  <Button
                    type="button"
                    variant="outline"
                    className="size-10 p-0"
                    disabled={conditions.length === 1}
                    aria-label={t.removeCondition}
                    onClick={() =>
                      setConditions((current) =>
                        current.length === 1 ? current : current.filter((_, i) => i !== index),
                      )
                    }
                  >
                    <IconTrash size={16} aria-hidden="true" />
                  </Button>
                </div>
              )
            })}
            <Button
              type="button"
              variant="outline"
              className="w-fit"
              onClick={() => setConditions((current) => [...current, emptyCondition()])}
            >
              <IconPlus size={16} aria-hidden="true" className="mr-1" />
              {t.addCondition}
            </Button>
            <div className="grid gap-1.5">
              <Label>{t.ruleGeneratedExpression}</Label>
              <pre className="overflow-x-auto rounded-md border border-slate-200 bg-slate-50 p-3 font-mono text-sm text-slate-700">
                {expression || '—'}
              </pre>
            </div>
          </div>
        ) : (
          <div className="grid gap-2">
            {hadInitialExpression && (
              <p className="text-xs text-amber-700">{t.ruleBuilderFallbackNotice}</p>
            )}
            <p className="text-xs text-slate-500">{t.dynamicRuleHelp}</p>
            <textarea
              value={advancedExpression}
              onChange={(event) => setAdvancedExpression(event.target.value)}
              aria-label={t.dynamicRuleExpression}
              className="min-h-28 w-full rounded-md border border-slate-300 bg-white p-3 font-mono text-sm"
              placeholder={'user.department == "Engineering"'}
            />
          </div>
        )}

        <div className="flex flex-wrap gap-2">
          <Button
            type="button"
            disabled={ruleBusy || !expression.trim()}
            onClick={() =>
              void withRule(async () => {
                const saved = await updateDynamicGroupRule(csrfToken, groupId, expression)
                setRule(saved)
              })
            }
          >
            {t.saveRule}
          </Button>
          <Button
            type="button"
            variant="outline"
            disabled={ruleBusy || !expression.trim() || previewUserIDs.length === 0}
            onClick={() =>
              void withRule(async () => {
                const result = await previewDynamicGroupRule(
                  csrfToken,
                  groupId,
                  expression,
                  previewUserIDs,
                )
                setPreview(result.results)
              })
            }
          >
            {t.previewRule}
          </Button>
          <Button
            type="button"
            variant="outline"
            disabled={ruleBusy || !rule}
            onClick={() =>
              void withRule(async () => {
                const saved = await setDynamicGroupRuleEnabled(csrfToken, groupId, !rule?.enabled)
                setRule(saved)
              })
            }
          >
            {rule?.enabled ? t.disableRule : t.enableRule}
          </Button>
        </div>

        <Label htmlFor="group-editor-dynamic-preview">{t.previewUsers}</Label>
        <select
          id="group-editor-dynamic-preview"
          multiple
          value={previewUserIDs}
          onChange={(event) =>
            setPreviewUserIDs(
              Array.from(event.currentTarget.selectedOptions, (option) => option.value),
            )
          }
          className="min-h-24 w-full rounded-md border border-slate-300 bg-white p-2 text-sm"
        >
          {allUsers.map((user) => (
            <option key={user.id} value={user.id}>
              {user.preferred_username}
            </option>
          ))}
        </select>
        {preview.length > 0 ? (
          <ul className="grid gap-1 text-sm">
            {preview.map((item) => (
              <li key={item.user_id} className="rounded bg-slate-50 px-2 py-1 font-mono">
                {item.user_id}: {item.matched ? t.matches : t.doesNotMatch} ({item.change})
              </li>
            ))}
          </ul>
        ) : null}
      </div>
    </Card>
  )
}
