import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import {
  collectConsumedEventFields,
  collectDeclaredEventFields,
  diffEventFieldVocabulary,
} from './event-contract.ts'
import type { CheckOutcome } from './runner.ts'

const DECLARATION = 'spec/contexts/system/models.tsp'
const CONSUMERS = [
  'backend/audit/usecases/audit_search_extractor.go',
  'backend/authentication/securitynotification/domain/catalog.go',
  'backend/authentication/securitynotification/usecases/dispatch.go',
]

export async function checkEventContract(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  const declared = collectDeclaredEventFields(await snapshot.read(DECLARATION))
  const sources = new Map<string, string>()
  for (const path of CONSUMERS) sources.set(path, await snapshot.read(path))
  const { missing, undeclared } = diffEventFieldVocabulary(
    declared,
    collectConsumedEventFields(sources),
  )
  const lines = [
    ...missing.map(
      (field) =>
        `fail  ${DECLARATION}: DomainEventPayload does not declare the consumed field ${field}`,
    ),
    ...undeclared.map(
      (field) =>
        `fail  ${DECLARATION}: DomainEventPayload declares ${field}, which no consumer reads`,
    ),
  ]
  return {
    ok: lines.length === 0,
    lines:
      lines.length > 0
        ? lines
        : [
            `ok  ${declared.size} published event payload field(s), ${CONSUMERS.length} consumer(s)`,
          ],
  }
}
