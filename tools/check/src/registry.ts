import { checkAdminScopes } from './check-admin-scopes.ts'
import { checkAgentGuidance } from './check-agent-guidance.ts'
import { checkApiCompat, checkPublishedApiCompat } from './check-api-compat.ts'
import { checkBoundaries } from './check-boundaries.ts'
import { checkBoundaryDebtRatchet } from './boundary-debt-ratchet.ts'
import { checkDocumentLayout } from './check-document-layout.ts'
import { checkContractDrift } from './check-contract-drift.ts'
import { checkOperationReferences } from './check-operation-references.ts'
import { checkDocuments } from './check-documents.ts'
import { checkExecutableExamples } from './check-executable-examples.ts'
import { checkEventContract } from './check-event-contract.ts'
import { checkLinks } from './check-links.ts'
import { checkModuleDependencies } from './check-module-dependencies.ts'
import { checkRawSql } from './check-raw-sql.ts'
import { checkSchemaTables } from './check-schema-tables.ts'
import { checkSecurityControls } from './check-security-controls.ts'
import { checkSpecificationRules } from './check-specification-rules.ts'
import { checkStateMachines } from './check-state-machines.ts'
import { checkSpecImpact } from './check-spec-impact.ts'
import { checkStatusDrift } from './check-status-drift.ts'
import { checkTerminology } from './check-terminology.ts'
import { checkTestDocumentReads } from './test-document-reads.ts'
import { checkToolTestInputs } from './tool-test-inputs.ts'
import { checkUnspecifiedVocabulary } from './check-unspecified-vocabulary.ts'
import { checkVulnerabilitySuppressions } from './check-vulnerability-suppressions.ts'
import { checkWorkItems } from './check-work-items.ts'
import type { RepositoryCheck } from './runner.ts'

export const repositoryChecks: readonly RepositoryCheck[] = [
  { name: 'executable-examples', groups: ['all', 'spec'], run: checkExecutableExamples },
  { name: 'state-machines', groups: ['all', 'spec'], run: checkStateMachines },
  { name: 'documents', groups: ['all', 'spec'], run: checkDocuments },
  { name: 'specification-rules', groups: ['all', 'spec'], run: checkSpecificationRules },
  { name: 'boundary-debt-ratchet', groups: ['all'], run: checkBoundaryDebtRatchet },
  { name: 'document-layout', groups: ['all', 'spec'], run: checkDocumentLayout },
  { name: 'work-items', groups: ['all'], run: checkWorkItems },
  { name: 'spec-impact', groups: ['all'], run: checkSpecImpact },
  { name: 'links', groups: ['all'], run: checkLinks },
  { name: 'terminology', groups: ['all'], run: checkTerminology },
  { name: 'boundaries', groups: ['all'], run: checkBoundaries },
  { name: 'module-dependencies', groups: ['all'], run: checkModuleDependencies },
  { name: 'raw-sql', groups: ['all'], run: checkRawSql },
  { name: 'test-document-reads', groups: ['all'], run: checkTestDocumentReads },
  { name: 'tool-test-inputs', groups: ['all'], run: checkToolTestInputs },
  { name: 'agent-guidance', groups: ['all'], run: checkAgentGuidance },
  { name: 'admin-scopes', groups: ['all'], run: checkAdminScopes },
  { name: 'contract-drift', groups: ['all'], run: checkContractDrift },
  { name: 'operation-references', groups: ['all'], run: checkOperationReferences },
  { name: 'status-drift', groups: ['all'], run: checkStatusDrift },
  { name: 'event-contract', groups: ['all'], run: checkEventContract },
  { name: 'security-controls', groups: ['all'], run: checkSecurityControls },
  { name: 'unspecified-vocabulary', groups: ['all'], run: checkUnspecifiedVocabulary },
  { name: 'schema-tables', groups: ['all'], run: checkSchemaTables },
  {
    name: 'vulnerability-suppressions',
    groups: ['all'],
    run: checkVulnerabilitySuppressions,
  },
  { name: 'api-compat', groups: [], run: checkApiCompat },
  { name: 'published-api-compat', groups: ['all'], run: checkPublishedApiCompat },
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
