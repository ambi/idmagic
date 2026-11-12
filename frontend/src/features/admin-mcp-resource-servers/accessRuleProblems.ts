import { parseAbsoluteUrl } from '../../lib/form'
import type { ResourceAccessLimit, ResourceAccessRule } from '../../types'

// 上限は oauth2/domain の規則の検証と TypeSpec の ResourceAccessRule と同じ値である (REQ-OAUTH2-064)。
export const MAX_LIMITS = 8
export const MAX_EXCHANGE_RESOURCES = 8
const MAX_LIMIT_REQUESTS = 1_000_000_000
const MAX_LIMIT_WINDOW_SECONDS = 2_592_000
export const LIMIT_UNITS = ['agent', 'user', 'resource'] as const

/** 欄の問題の理由。文言は画面の辞書の同名の項目である。 */
export type AccessRuleProblem =
  | 'detailTypeRequired'
  | 'approvalConditionConflict'
  | 'approvalDetailTypeRequired'
  | 'approvalDetailTypeUnused'
  | 'conditionEmpty'
  | 'exchangeWithApproval'
  | 'exchangeTooMany'
  | 'exchangeHttps'
  | 'limitsTooMany'
  | 'limitRequestsRange'
  | 'limitWindowRange'
  | 'limitDuplicate'

export type LimitProblems = {
  maxRequests?: AccessRuleProblem
  windowSeconds?: AccessRuleProblem
}

/** 規則一件の、行の入力欄ごとの問題。配列は規則の配列と同じ位置に対応する。 */
export type AccessRuleProblems = {
  authorizationDetailType?: AccessRuleProblem
  denyWhen?: AccessRuleProblem
  requireApprovalWhen?: AccessRuleProblem
  approvalDetailType?: AccessRuleProblem
  tokenExchange?: AccessRuleProblem
  tokenExchangeEntries: (AccessRuleProblem | undefined)[]
  limitsGroup?: AccessRuleProblem
  limits: LimitProblems[]
}

const present = (value: string | undefined) => value !== undefined && value !== ''

function inRange(value: number, max: number) {
  return Number.isInteger(value) && value >= 1 && value <= max
}

// サーバーは url.Parse の結果のスキームが https でホストが空でないことだけを見る。
function exchangeProblem(resource: string): AccessRuleProblem | undefined {
  const url = parseAbsoluteUrl(resource)
  return url?.scheme === 'https' && url.hostport !== '' ? undefined : 'exchangeHttps'
}

function limitKey(limit: ResourceAccessLimit) {
  return `${[...(limit.by ?? [])].sort().join(',')}|${limit.window_seconds}`
}

/**
 * accessRuleProblems は、サーバーが拒否する制約の組み合わせを行の欄ごとに返す (REQ-OAUTH2-090)。
 * 検証の正はサーバーであり、ここではサーバーより狭く拒まない。式のコンパイルは保存時にサーバーが行う。
 * 型が契約と異なる規則には使わず、先に constraintsRepresentable で確かめる。
 */
export function accessRuleProblems(rule: ResourceAccessRule): AccessRuleProblems {
  const approval = rule.requires_approved_operation === true
  const condition = rule.condition
  const conditionalApproval = present(condition?.require_approval_when)
  const exchange = rule.token_exchange_resources ?? []
  const limits = rule.limits ?? []
  const problems: AccessRuleProblems = {
    tokenExchangeEntries: exchange.map(exchangeProblem),
    limits: [],
  }
  if (approval && !present(rule.authorization_detail_type)) {
    problems.authorizationDetailType = 'detailTypeRequired'
  }
  if (condition) {
    if (!present(condition.deny_when) && !conditionalApproval) {
      problems.denyWhen = 'conditionEmpty'
    }
    if (approval && conditionalApproval) problems.requireApprovalWhen = 'approvalConditionConflict'
    if (conditionalApproval && !present(condition.approval_detail_type)) {
      problems.approvalDetailType = 'approvalDetailTypeRequired'
    } else if (!conditionalApproval && present(condition.approval_detail_type)) {
      problems.approvalDetailType = 'approvalDetailTypeUnused'
    }
  }
  if (exchange.length > MAX_EXCHANGE_RESOURCES) problems.tokenExchange = 'exchangeTooMany'
  else if (exchange.length > 0 && (approval || conditionalApproval)) {
    problems.tokenExchange = 'exchangeWithApproval'
  }
  if (limits.length > MAX_LIMITS) problems.limitsGroup = 'limitsTooMany'
  const seen = new Set<string>()
  for (const limit of limits) {
    const problem: LimitProblems = {}
    if (!inRange(limit.max_requests, MAX_LIMIT_REQUESTS)) problem.maxRequests = 'limitRequestsRange'
    if (!inRange(limit.window_seconds, MAX_LIMIT_WINDOW_SECONDS)) {
      problem.windowSeconds = 'limitWindowRange'
    } else if (seen.has(limitKey(limit))) problem.windowSeconds = 'limitDuplicate'
    seen.add(limitKey(limit))
    problems.limits.push(problem)
  }
  return problems
}

export function hasProblems(problems: AccessRuleProblems): boolean {
  return (
    Object.entries(problems).some(
      ([key, value]) => key !== 'tokenExchangeEntries' && key !== 'limits' && value !== undefined,
    ) ||
    problems.tokenExchangeEntries.some((problem) => problem !== undefined) ||
    problems.limits.some((limit) => Object.keys(limit).length > 0)
  )
}

const optional = (value: unknown, type: 'string' | 'boolean') =>
  value === undefined || typeof value === type

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null && !Array.isArray(value)

function limitRepresentable(limit: unknown): boolean {
  if (!isRecord(limit)) return false
  const by = limit.by
  return (
    typeof limit.max_requests === 'number' &&
    typeof limit.window_seconds === 'number' &&
    (by === undefined ||
      (Array.isArray(by) &&
        new Set(by).size === by.length &&
        by.every((unit) => (LIMIT_UNITS as readonly unknown[]).includes(unit))))
  )
}

/**
 * constraintsRepresentable は、承認、上限、交換先、条件が行の入力欄で表せる型かを返す。
 * 偽の規則は、行で読み取り専用に示し、JSON の入力で編集させる (REQ-OAUTH2-090)。
 */
export function constraintsRepresentable(rule: ResourceAccessRule): boolean {
  const value = rule as unknown as Record<string, unknown>
  const condition = value.condition
  const limits = value.limits
  const exchange = value.token_exchange_resources
  return (
    optional(value.requires_approved_operation, 'boolean') &&
    optional(value.authorization_detail_type, 'string') &&
    (condition === undefined ||
      (isRecord(condition) &&
        optional(condition.deny_when, 'string') &&
        optional(condition.require_approval_when, 'string') &&
        optional(condition.approval_detail_type, 'string'))) &&
    (limits === undefined || (Array.isArray(limits) && limits.every(limitRepresentable))) &&
    (exchange === undefined ||
      (Array.isArray(exchange) && exchange.every((resource) => typeof resource === 'string')))
  )
}
