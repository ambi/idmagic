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
  const contextsDirectory = 'docs/contexts'
  const contractDirectory = 'spec/contexts'
  const platformRefusals = browserRefusalTypesNamedByPlatformScenario(
    await snapshot.read('docs/scenarios.feature.md'),
  )
  const apiTokenRefusals = insufficientScopeTypeNamedByApiTokenScenario(
    await snapshot.read(`${contextsDirectory}/api-tokens/scenarios.feature.md`),
  )
  const sharedRefusals = new Set([...platformRefusals, ...apiTokenRefusals])
  let declared = sharedRefusals.size
  let promised = 0
  for (const entry of await snapshot.list(contextsDirectory)) {
    if (!entry.isDirectory()) continue
    const scenarioPath = `${contextsDirectory}/${entry.name}/scenarios.feature.md`
    let source: string
    try {
      source = await snapshot.read(scenarioPath)
    } catch {
      continue
    }
    const local = errorTypesNamedByScenarios(source)
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
