import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import type { CheckOutcome } from './runner.ts'
import { documentKind } from './specification-doc.ts'
import {
  collectErrorCodes,
  collectEventTypes,
  collectModelNames,
  compareWithDebt,
  findUnspecified,
  ruleTexts,
  sharedEventOwners,
  type SpecifiedTexts,
  specifiedTexts,
  type UnspecifiedTerm,
  type VocabularyDebt,
} from './unspecified-vocabulary.ts'

/**
 * 検査するモジュール。誤検出の割合を確かめてから広げるので、要件の書き直しを終えたモジュールだけとする。
 * `backend` は、そのモジュールのドメインイベントを宣言する Go の木である。
 */
const MODULES = [
  { module: 'identity-management', backend: 'backend/idmanagement' },
  { module: 'tenancy', backend: 'backend/tenancy' },
  { module: 'api-tokens', backend: 'backend/apitoken' },
  { module: 'application', backend: 'backend/application' },
  { module: 'audit', backend: 'backend/audit' },
  { module: 'authentication', backend: 'backend/authentication' },
  { module: 'authorization', backend: 'backend/authorization' },
  { module: 'claim-mapping', backend: 'backend/claimmapping' },
  { module: 'data-keys', backend: 'backend/datakeys' },
  { module: 'identity-governance', backend: 'backend/idgovernance' },
  { module: 'jobs', backend: 'backend/jobs' },
  { module: 'oauth2', backend: 'backend/oauth2' },
  { module: 'provisioning', backend: 'backend/provisioning' },
  { module: 'saml', backend: 'backend/saml' },
  { module: 'seeding', backend: 'backend/seeding' },
  { module: 'sharedsignals', backend: 'backend/sharedsignals' },
  { module: 'signing-keys', backend: 'backend/signingkeys' },
  { module: 'sourcing', backend: 'backend/sourcing' },
  { module: 'system', backend: 'backend/shared/spec' },
  { module: 'workloadidentity', backend: 'backend/workloadidentity' },
  { module: 'ws-federation', backend: 'backend/wsfederation' },
]

/** 導入時点の違反の一覧。モジュールごとに持ち、減る方向にしか動かさない。 */
const DEBT = 'tools/check/unspecified-vocabulary-debt.json'

/**
 * どのモジュールにも属さない共通の宣言を持つモジュール。その Go の木 `backend/shared/spec` の
 * イベントは複数のモジュールが発行するので、同じ名前のモデルを TypeSpec で宣言するモジュールの
 * 要件で照合する。共通のエラーは、各モジュールの経路へ同じ形で適用する保証なので、横断的な
 * シナリオの Rule でも照合する。
 */
const COMMON = 'system'
const CROSS_MODULE_SCENARIOS = 'docs/modules/scenarios.feature.md'

export async function checkUnspecifiedVocabulary(
  snapshot: WorkspaceSnapshot,
): Promise<CheckOutcome> {
  const debts: Record<string, VocabularyDebt> = snapshot.exists(DEBT)
    ? JSON.parse(await snapshot.read(DEBT))
    : {}
  const typespecs = new Map<string, string[]>()
  for (const { module } of MODULES) {
    const sources: string[] = []
    for (const path of await snapshot.files(`spec/modules/${module}`, [])) {
      if (path.endsWith('.tsp')) sources.push(await snapshot.read(path))
    }
    typespecs.set(module, sources)
  }
  const goEvents = new Map<string, string[]>()
  for (const { module, backend } of MODULES) {
    const events = new Set<string>()
    for (const path of await snapshot.files(backend, ['vendor'])) {
      if (!path.endsWith('.go') || path.endsWith('_test.go')) continue
      for (const event of collectEventTypes(await snapshot.read(path))) events.add(event)
    }
    goEvents.set(module, [...events])
  }
  const sharedEvents = sharedEventOwners(
    goEvents.get(COMMON) ?? [],
    new Map(
      [...typespecs].map(([module, sources]) => [
        module,
        new Set(sources.flatMap(collectModelNames)),
      ]),
    ),
    COMMON,
  )
  const crossModuleRules = snapshot.exists(CROSS_MODULE_SCENARIOS)
    ? ruleTexts(await snapshot.read(CROSS_MODULE_SCENARIOS))
    : []
  const lines: string[] = []
  let terms = 0
  for (const { module } of MODULES) {
    const errors = new Map<string, string[]>()
    for (const source of typespecs.get(module) ?? []) {
      for (const [code, models] of collectErrorCodes(source)) {
        errors.set(code, [...(errors.get(code) ?? []), ...models])
      }
    }
    const events = new Set([
      ...(module === COMMON ? [] : (goEvents.get(module) ?? [])),
      ...(sharedEvents.get(module) ?? []),
    ])
    const texts: SpecifiedTexts = { requirements: [], transitions: [] }
    for (const path of await snapshot.files(`docs/modules/${module}`, [])) {
      if (documentKind(path) !== 'specification') continue
      const read = specifiedTexts(await snapshot.read(path))
      texts.requirements.push(...read.requirements)
      texts.transitions.push(...read.transitions)
    }
    if (module === COMMON) texts.requirements.push(...crossModuleRules)
    terms += errors.size + events.size
    const found = findUnspecified({ errors, events: [...events] }, texts)
    const { fresh, stale } = compareWithDebt(found, debts[module] ?? { errors: [], events: [] })
    lines.push(
      ...fresh.map(
        (term) => `fail  ${module}: ${describe(term)} appears in no requirement of the module`,
      ),
      ...stale.map(
        (term) =>
          `fail  ${module}: ${describe(term)} appears in a requirement now; remove it from ${DEBT}`,
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
