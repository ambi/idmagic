import type { McpResourceServer, RelationTuple } from '../types'

export const MCP_OPERATION_TYPE = 'idmagic_mcp_operation'

export function operationResourceID(server: McpResourceServer, toolName: string) {
  return (
    server.access_rules?.find((rule) => rule.tool?.name === toolName)?.tool
      ?.assignment_resource_id ?? ''
  )
}

// assignedOperationIDs は、Agent 自身へ直接付いた操作の割当 (invoke) の資源 ID を返す。
// 集合を介した割当 (subject_relation 付き) と対象の範囲 (op/target など) は含めない。
export function assignedOperationIDs(tuples: RelationTuple[], agentID: string): Set<string> {
  return new Set(
    tuples
      .filter(
        (tuple) =>
          tuple.resource_type === MCP_OPERATION_TYPE &&
          tuple.relation === 'invoke' &&
          tuple.subject_type === 'agent' &&
          tuple.subject_id === agentID &&
          !tuple.subject_relation,
      )
      .map((tuple) => tuple.resource_id),
  )
}

export function operationTuple(
  server: McpResourceServer,
  toolName: string,
  agentID: string,
): RelationTuple {
  return {
    resource_type: MCP_OPERATION_TYPE,
    resource_id: operationResourceID(server, toolName),
    relation: 'invoke',
    subject_type: 'agent',
    subject_id: agentID,
  }
}
