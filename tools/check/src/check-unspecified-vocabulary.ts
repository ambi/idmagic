import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import type { CheckOutcome } from './runner.ts'
import { documentKind } from './specification-doc.ts'
import {
  collectErrorCodes,
  collectEventTypes,
  compareWithDebt,
  findUnspecified,
  type SpecifiedTexts,
  specifiedTexts,
  type UnspecifiedTerm,
  type VocabularyDebt,
} from './unspecified-vocabulary.ts'

/**
 * 検査する Context。誤検出の割合を確かめてから広げるので、最初は IdManagement だけとする。
 * `backend` は、その Context のドメインイベントを宣言する Go の木である。
 */
const CONTEXTS = [{ context: 'identity-management', backend: 'backend/idmanagement' }]

/** 導入時点の違反の一覧。Context ごとに持ち、減る方向にしか動かさない。 */
const DEBT = 'tools/check/unspecified-vocabulary-debt.json'

export async function checkUnspecifiedVocabulary(
  snapshot: WorkspaceSnapshot,
): Promise<CheckOutcome> {
  const debts: Record<string, VocabularyDebt> = snapshot.exists(DEBT)
    ? JSON.parse(await snapshot.read(DEBT))
    : {}
  const lines: string[] = []
  let terms = 0
  for (const { context, backend } of CONTEXTS) {
    const typespecDirectory = `spec/contexts/${context}`
    const errors = new Map<string, string[]>()
    for (const path of await snapshot.files(typespecDirectory, [])) {
      if (!path.endsWith('.tsp')) continue
      for (const [code, models] of collectErrorCodes(await snapshot.read(path))) {
        errors.set(code, [...(errors.get(code) ?? []), ...models])
      }
    }
    const events = new Set<string>()
    for (const path of await snapshot.files(backend, ['vendor'])) {
      if (!path.endsWith('.go') || path.endsWith('_test.go')) continue
      for (const event of collectEventTypes(await snapshot.read(path))) events.add(event)
    }
    const texts: SpecifiedTexts = { requirements: [], transitions: [] }
    for (const path of await snapshot.files(`docs/domain/${context}`, [])) {
      if (documentKind(path) !== 'specification') continue
      const read = specifiedTexts(await snapshot.read(path))
      texts.requirements.push(...read.requirements)
      texts.transitions.push(...read.transitions)
    }
    terms += errors.size + events.size
    const found = findUnspecified({ errors, events: [...events] }, texts)
    const { fresh, stale } = compareWithDebt(found, debts[context] ?? { errors: [], events: [] })
    lines.push(
      ...fresh.map(
        (term) => `fail  ${context}: ${describe(term)} appears in no requirement of the context`,
      ),
      ...stale.map(
        (term) =>
          `fail  ${context}: ${describe(term)} appears in a requirement now; remove it from ${DEBT}`,
      ),
    )
  }
  return {
    ok: lines.length === 0,
    lines:
      lines.length > 0
        ? lines
        : [`ok  ${terms} error code(s) and domain event(s) checked against the requirements`],
  }
}

function describe(term: UnspecifiedTerm): string {
  return term.kind === 'error' ? `error code ${term.name}` : `domain event ${term.name}`
}
