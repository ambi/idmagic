import {
  findAuthorizationModel,
  getAdminAgent,
  listAdminAgents,
  listMcpResourceServers,
  listRelationTuples,
} from '../../../api/admin'
import { requirePortalAccount } from '../../-guards'

export function validateAccessSearch(search: Record<string, unknown>) {
  return {
    server: typeof search.server === 'string' ? search.server : '',
    agent: typeof search.agent === 'string' ? search.agent : '',
  }
}

export async function loadAgentAccess(pathname: string, searchStr: string, selectedAgent: string) {
  const account = await requirePortalAccount('admin', pathname, searchStr)
  const [model, resourceServers, listed, agents, selected] = await Promise.all([
    findAuthorizationModel(),
    listMcpResourceServers(),
    listRelationTuples({ subject_type: 'agent' }),
    listAdminAgents(),
    selectedAgent ? getAdminAgent(selectedAgent) : null,
  ])
  return {
    csrfToken: account.csrf_token,
    actorUsername: account.preferred_username,
    resourceTypes: model?.authorization_model.resource_types ?? [],
    resourceServers,
    tuples: listed.tuples,
    agents:
      selected && !agents.some((agent) => agent.id === selected.id)
        ? [...agents, selected]
        : agents,
    consistency: listed.consistency,
  }
}
