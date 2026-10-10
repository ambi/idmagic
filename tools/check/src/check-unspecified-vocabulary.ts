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
 * 検査するモジュール。誤検出の割合を確かめてから広げるので、要件の書き直しを終えたモジュールだけとする。
 * `backend` は、そのモジュールのドメインイベントを宣言する Go の木である。
 */
const CONTEXTS = [
  { context: 'identity-management', backend: 'backend/idmanagement' },
  { context: 'tenancy', backend: 'backend/tenancy' },
  { context: 'api-tokens', backend: 'backend/apitoken' },
  { context: 'application', backend: 'backend/application' },
  { context: 'audit', backend: 'backend/audit' },
  { context: 'authentication', backend: 'backend/authentication' },
  { context: 'authorization', backend: 'backend/authorization' },
  { context: 'claim-mapping', backend: 'backend/claimmapping' },
  { context: 'data-keys', backend: 'backend/datakeys' },
  { context: 'identity-governance', backend: 'backend/idgovernance' },
  { context: 'jobs', backend: 'backend/jobs' },
  { context: 'oauth2', backend: 'backend/oauth2' },
  { context: 'provisioning', backend: 'backend/provisioning' },
  { context: 'saml', backend: 'backend/saml' },
  { context: 'seeding', backend: 'backend/seeding' },
  { context: 'sharedsignals', backend: 'backend/sharedsignals' },
  { context: 'signing-keys', backend: 'backend/signingkeys' },
  { context: 'sourcing', backend: 'backend/sourcing' },
  { context: 'system', backend: 'backend/shared/spec' },
  { context: 'workloadidentity', backend: 'backend/workloadidentity' },
  { context: 'ws-federation', backend: 'backend/wsfederation' },
]

/** 導入時点の違反の一覧。モジュールごとに持ち、減る方向にしか動かさない。 */
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
    for (const path of await snapshot.files(`docs/modules/${context}`, [])) {
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
