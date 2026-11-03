import type { McpResourceServer, RelationTuple } from '../types'

export const MCP_OPERATION_TYPE = 'idmagic_mcp_operation'

export function operationResourceID(server: McpResourceServer, toolName: string) {
  return (
    server.access_rules?.find((rule) => rule.tool?.name === toolName)?.tool
      ?.assignment_resource_id ?? ''
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
