import { checkAdminScopes } from './check-admin-scopes.ts'
import { checkAgentGuidance } from './check-agent-guidance.ts'
import { checkApiCompat } from './check-api-compat.ts'
import { checkBoundaries } from './check-boundaries.ts'
import { checkCommandMap } from './check-command-map.ts'
import { checkContractDrift } from './check-contract-drift.ts'
import { checkCoverageDebtRatchet } from './coverage-debt-ratchet.ts'
import { checkDocuments } from './check-documents.ts'
import { checkEventContract } from './check-event-contract.ts'
import { checkLinks } from './check-links.ts'
import { checkSecurityControls } from './check-security-controls.ts'
import { checkSloReferences } from './check-slo-references.ts'
import { checkStatusDrift } from './check-status-drift.ts'
import { checkVulnerabilitySuppressions } from './check-vulnerability-suppressions.ts'
import { checkWorkItems } from './check-work-items.ts'
import type { RepositoryCheck } from './runner.ts'

export const repositoryChecks: readonly RepositoryCheck[] = [
  { name: 'documents', groups: ['all'], run: checkDocuments },
  { name: 'coverage-debt-ratchet', groups: ['all'], run: checkCoverageDebtRatchet },
  { name: 'work-items', groups: ['all'], run: checkWorkItems },
  { name: 'links', groups: ['all'], run: checkLinks },
  { name: 'boundaries', groups: ['all'], run: checkBoundaries },
  { name: 'command-map', groups: ['all'], run: checkCommandMap },
  { name: 'agent-guidance', groups: ['all'], run: checkAgentGuidance },
  { name: 'admin-scopes', groups: ['all'], run: checkAdminScopes },
  { name: 'contract-drift', groups: ['all'], run: checkContractDrift },
  { name: 'status-drift', groups: ['all'], run: checkStatusDrift },
  { name: 'event-contract', groups: ['all'], run: checkEventContract },
  { name: 'security-controls', groups: ['all'], run: checkSecurityControls },
  { name: 'slo-references', groups: ['all'], run: checkSloReferences },
  {
    name: 'vulnerability-suppressions',
    groups: ['all'],
    run: checkVulnerabilitySuppressions,
  },
  { name: 'api-compat', groups: ['all'], run: checkApiCompat },
]

export function selectChecks(selectors: readonly string[]): RepositoryCheck[] {
  const requested = selectors.length === 0 ? ['all'] : selectors
  const selected = repositoryChecks.filter(
    (check) =>
      requested.includes(check.name) || check.groups.some((group) => requested.includes(group)),
  )
  const known = new Set(repositoryChecks.flatMap((check) => [check.name, ...check.groups]))
  const unknown = requested.filter((selector) => !known.has(selector))
  if (unknown.length > 0) throw new Error(`unknown check selector(s): ${unknown.join(', ')}`)
  return selected
}
