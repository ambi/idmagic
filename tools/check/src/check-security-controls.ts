import {
  FEATURES_DIRECTORY,
  featureDocument,
  SHARED_SCENARIOS,
} from '../../workspace/src/document-layout.ts'
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
  const modulesDirectory = 'docs/modules'
  const contractDirectory = 'spec/modules'
  const platformRefusals = browserRefusalTypesNamedByPlatformScenario(
    await snapshot.read(SHARED_SCENARIOS),
  )
  const apiTokenRefusals = insufficientScopeTypeNamedByApiTokenScenario(
    await snapshot.read(`${modulesDirectory}/api-tokens/features/authentication.feature.md`),
  )
  const sharedRefusals = new Set([...platformRefusals, ...apiTokenRefusals])
  let declared = sharedRefusals.size
  let promised = 0
  for (const entry of await snapshot.list(modulesDirectory)) {
    if (!entry.isDirectory()) continue
    const sources = await moduleScenarioSources(snapshot, `${modulesDirectory}/${entry.name}`)
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
 * モジュールの機能スライスにある例の付録。拒否の宣言はモジュール単位で判定するので、
 * 例をどの機能スライスへ置いても同じモジュールの宣言として数える。
 * 例の付録は、モジュールの機能の段に `<feature>.feature.md` として並ぶ。
 */
async function moduleScenarioSources(
  snapshot: WorkspaceSnapshot,
  context: string,
): Promise<string[]> {
  const features = `${context}/${FEATURES_DIRECTORY}`
  if (!snapshot.exists(features)) return []
  const sources: string[] = []
  for (const entry of await snapshot.list(features)) {
    const path = `${features}/${entry.name}`
    if (entry.isFile() && featureDocument(path)?.part === 'examples') {
      sources.push(await snapshot.read(path))
    }
  }
  return sources
}
