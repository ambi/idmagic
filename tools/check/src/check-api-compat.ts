import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { compareOpenApi, type JsonSchema } from './api-compat.ts'
import type { CheckOutcome } from './runner.ts'

export async function checkApiCompat(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  const baselinePath = await snapshot.openApiBaseline()
  const findings = compareOpenApi(
    JSON.parse(await snapshot.read(baselinePath)) as JsonSchema,
    JSON.parse(await snapshot.read(await snapshot.generatedOpenApi())) as JsonSchema,
  )
  if (findings.length === 0) {
    return { ok: true, lines: [`ok  API compatibility (no breaking changes vs ${baselinePath})`] }
  }
  return {
    ok: false,
    lines: [
      `fail  API compatibility (${findings.length} breaking change(s) vs ${baselinePath})`,
      ...findings.map((finding) => `  ${finding.operation}: ${finding.message}`),
      'If this break is intentional, version the path or add a new interface instead of changing it in place.',
    ],
  }
}
