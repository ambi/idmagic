import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import {
  collectConstructedTypes,
  collectConsumedEventFields,
  collectDeclaredEventFields,
  collectDeclaredEvents,
  type DeclaredEvent,
  diffEventFieldVocabulary,
  findEventsWithoutEmission,
} from './event-contract.ts'
import type { CheckOutcome } from './runner.ts'

const DECLARATION = 'spec/contexts/system/models.tsp'
const CONSUMERS = [
  'backend/audit/usecases/audit_search_extractor.go',
  'backend/authentication/securitynotification/domain/catalog.go',
  'backend/authentication/securitynotification/usecases/dispatch.go',
]

/** 発行経路を持たないイベントの導入時点の一覧。減る方向にしか動かさない。 */
const EMISSION_DEBT = 'tools/check/event-emission-debt.json'

export async function checkEventContract(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  const declared = collectDeclaredEventFields(await snapshot.read(DECLARATION))
  const sources = new Map<string, string>()
  for (const path of CONSUMERS) sources.set(path, await snapshot.read(path))
  const { missing, undeclared } = diffEventFieldVocabulary(
    declared,
    collectConsumedEventFields(sources),
  )
  const emission = await checkEmission(snapshot)
  const lines = [
    ...missing.map(
      (field) =>
        `fail  ${DECLARATION}: DomainEventPayload does not declare the consumed field ${field}`,
    ),
    ...undeclared.map(
      (field) =>
        `fail  ${DECLARATION}: DomainEventPayload declares ${field}, which no consumer reads`,
    ),
    ...emission.lines,
  ]
  return {
    ok: lines.length === 0,
    lines:
      lines.length > 0
        ? lines
        : [
            `ok  ${declared.size} published event payload field(s), ${CONSUMERS.length} consumer(s), ${emission.events} event type(s) with an emission path`,
          ],
  }
}

/**
 * 本番の Go コードがイベント型の値を一度も作らないとき、そのイベントは宣言だけがあって
 * 発行されない。負債に載っていないものと、もう発行されるのに負債に残っているものを落とす。
 */
async function checkEmission(
  snapshot: WorkspaceSnapshot,
): Promise<{ lines: string[]; events: number }> {
  const declared: DeclaredEvent[] = []
  const constructed = new Set<string>()
  for (const path of await snapshot.files('backend', ['vendor'])) {
    if (!path.endsWith('.go') || path.endsWith('_test.go')) continue
    const source = await snapshot.read(path)
    declared.push(...collectDeclaredEvents(source))
    for (const name of collectConstructedTypes(source)) constructed.add(name)
  }
  const debt: string[] = snapshot.exists(EMISSION_DEBT)
    ? JSON.parse(await snapshot.read(EMISSION_DEBT))
    : []
  const unemitted = findEventsWithoutEmission(declared, constructed)
  return {
    events: new Set(declared.map((event) => event.eventType)).size - unemitted.length,
    lines: [
      ...unemitted
        .filter((event) => !debt.includes(event))
        .map(
          (event) => `fail  backend: event ${event} is declared but no production code emits it`,
        ),
      ...debt
        .filter((event) => !unemitted.includes(event))
        .map(
          (event) =>
            `fail  ${EMISSION_DEBT}: ${event} now has an emission path or is gone; remove it from the debt`,
        ),
    ],
  }
}
