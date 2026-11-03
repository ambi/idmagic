import { getAdminAgent, listMcpResourceServers, listRelationTuples } from '../../../api/admin'
import { MCP_OPERATION_TYPE } from '../../../lib/mcpAccess'
import { requirePortalAccount } from '../../-guards'

export async function loadAgentResources(pathname: string, searchStr: string, agentID: string) {
  const account = await requirePortalAccount('admin', pathname, searchStr)
  const [agent, resourceServers, result] = await Promise.all([
    getAdminAgent(agentID),
    listMcpResourceServers(),
    listRelationTuples({
      resource_type: MCP_OPERATION_TYPE,
      subject_type: 'agent',
      subject_id: agentID,
    }),
  ])
  return {
    csrfToken: account.csrf_token,
    actorUsername: account.preferred_username,
    agent,
    resourceServers,
    tuples: result.tuples,
  }
}
