import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { repositoryGoFiles } from './repository-inputs.ts'
import type { CheckOutcome } from './runner.ts'
import {
  browserRefusalTypesNamedByPlatformScenario,
  checkContractRefusalsAreDeclared,
  checkSecurityGuards,
  contractRefusalsOfStateChanges,
  errorTypesNamedByScenarios,
  insufficientScopeTypeNamedByApiTokenScenario,
  type Finding,
} from './security-controls.ts'

export async function checkSecurityControls(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  const findings: Finding[] = [...checkSecurityGuards(await repositoryGoFiles(snapshot, true))]
  const contextsDirectory = 'docs/domain'
  const contractDirectory = 'spec/contexts'
  const platformRefusals = browserRefusalTypesNamedByPlatformScenario(
    await snapshot.read('docs/domain/scenarios.feature.md'),
  )
  const apiTokenRefusals = insufficientScopeTypeNamedByApiTokenScenario(
    await snapshot.read(`${contextsDirectory}/api-tokens/authentication/examples.feature.md`),
  )
  const sharedRefusals = new Set([...platformRefusals, ...apiTokenRefusals])
  let declared = sharedRefusals.size
  let promised = 0
  for (const entry of await snapshot.list(contextsDirectory)) {
    if (!entry.isDirectory()) continue
    const sources = await contextScenarioSources(snapshot, `${contextsDirectory}/${entry.name}`)
    if (sources.length === 0) continue
    const local = new Set(sources.flatMap((source) => [...errorTypesNamedByScenarios(source)]))
    const named = new Set([...local, ...sharedRefusals])
    declared += local.size
    const contract = new Map<string, string[]>()
    let contractFiles = []
    try {
      contractFiles = await snapshot.list(`${contractDirectory}/${entry.name}`)
    } catch {
      continue
    }
    for (const contractFile of contractFiles) {
      if (!contractFile.isFile() || !contractFile.name.endsWith('.tsp')) continue
      const typespec = await snapshot.read(
        `${contractDirectory}/${entry.name}/${contractFile.name}`,
      )
      for (const [type, operations] of contractRefusalsOfStateChanges(typespec)) {
        contract.set(type, [...(contract.get(type) ?? []), ...operations])
      }
    }
    promised += contract.size
    findings.push(...checkContractRefusalsAreDeclared(entry.name, contract, named))
  }
  return {
    ok: findings.length === 0,
    lines:
      findings.length > 0
        ? findings.map((finding) => `${finding.path}: [${finding.rule}] ${finding.message}`)
        : [
            `ok  security controls (${promised} refusal(s) promised by a 403 on a state change, ` +
              `${declared} error type(s) named by the scenarios)`,
          ],
  }
}

/**
 * Context のルートと、その下の機能ノードにあるシナリオ。拒否の宣言は Context 単位で
 * 判定するので、規則をどの機能ノードへ置いても同じ Context の宣言として数える。
 * 新しい形式の機能ノードは機能群の一段下にもあり、例は付録 `examples.feature.md` に置く。
 */
async function contextScenarioSources(
  snapshot: WorkspaceSnapshot,
  context: string,
): Promise<string[]> {
  const scenarioFiles = (directory: string) => [
    `${directory}/scenarios.feature.md`,
    `${directory}/examples.feature.md`,
  ]
  const paths = scenarioFiles(context)
  for (const entry of await snapshot.list(context)) {
    if (!entry.isDirectory()) continue
    const feature = `${context}/${entry.name}`
    paths.push(...scenarioFiles(feature))
    for (const child of await snapshot.list(feature)) {
      if (child.isDirectory()) paths.push(...scenarioFiles(`${feature}/${child.name}`))
    }
  }
  const sources: string[] = []
  for (const path of paths) {
    if (snapshot.exists(path)) sources.push(await snapshot.read(path))
  }
  return sources
}
