import { MCP_OPERATION_TYPE } from '../../lib/mcpAccess'
import type { RelationTuple, ResourceAccessRule } from '../../types'

// 対象ごとの権限を持つ操作の範囲 (REQ-AUTHORIZATION-017)。none は範囲を選んでいない状態で、
// どの対象も許可しない。
export type TargetRange = { mode: 'none' | 'any' | 'specific'; targets: string }

export const ANY_TARGET_RELATION = 'invoke_any_target'

// isTargetScoped は、組み込みの型で対象引数を持ち、割当の範囲で対象を限る操作かを返す。
export function isTargetScoped(rule: ResourceAccessRule): boolean {
  return rule.resource_type === MCP_OPERATION_TYPE && Boolean(rule.tool?.resource_id_argument)
}

function ownTuple(tuple: RelationTuple, agentID: string): boolean {
  return (
    tuple.resource_type === MCP_OPERATION_TYPE &&
    tuple.subject_type === 'agent' &&
    tuple.subject_id === agentID &&
    !tuple.subject_relation
  )
}

// savedRange は保存済みの関係から、操作の範囲を読み取る。
export function savedRange(
  tuples: RelationTuple[],
  operationID: string,
  agentID: string,
): TargetRange {
  const own = tuples.filter((tuple) => ownTuple(tuple, agentID))
  if (
    own.some((tuple) => tuple.relation === ANY_TARGET_RELATION && tuple.resource_id === operationID)
  ) {
    return { mode: 'any', targets: '' }
  }
  const targets = own
    .filter(
      (tuple) => tuple.relation === 'invoke' && tuple.resource_id.startsWith(`${operationID}/`),
    )
    .map((tuple) => tuple.resource_id.slice(operationID.length + 1))
  return targets.length > 0
    ? { mode: 'specific', targets: targets.join('\n') }
    : { mode: 'none', targets: '' }
}

// parseTargets は一行に一つの対象 ID を読む。空行と重複は除く。
export function parseTargets(text: string): string[] {
  return [
    ...new Set(
      text
        .split('\n')
        .map((line) => line.trim())
        .filter(Boolean),
    ),
  ]
}

// invalidTargets は関係の ID に使えない対象を返す。
export function invalidTargets(text: string): string[] {
  return parseTargets(text).filter((target) => /[:#]/.test(target))
}

// rangeTuples は範囲を表す関係を返す。操作の割当そのもの (invoke) は含めない。
export function rangeTuples(
  operationID: string,
  agentID: string,
  range: TargetRange,
): RelationTuple[] {
  const tuple = (resourceID: string, relation: string): RelationTuple => ({
    resource_type: MCP_OPERATION_TYPE,
    resource_id: resourceID,
    relation,
    subject_type: 'agent',
    subject_id: agentID,
  })
  if (range.mode === 'any') return [tuple(operationID, ANY_TARGET_RELATION)]
  if (range.mode === 'specific') {
    return parseTargets(range.targets).map((target) => tuple(`${operationID}/${target}`, 'invoke'))
  }
  return []
}

export function tupleKey(tuple: RelationTuple): string {
  return `${tuple.resource_id}#${tuple.relation}`
}
