import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { alertReferences, checkAlerts, declaredObjectives } from './slo-references.ts'
import type { CheckOutcome } from './runner.ts'

const MONITORING_ASSETS = [
  'infra/docker/prometheus-rules.yml',
  'infra/k8s/monitoring/prometheus-rule.yaml',
]

export async function checkSloReferences(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  const declared = declaredObjectives(await snapshot.read('docs/requirements/quality.md'))
  if (declared.size === 0) {
    return {
      ok: false,
      lines: ['fail  docs/requirements/quality.md declares no SLO-* or CAP-* objective'],
    }
  }
  const findings = []
  for (const path of MONITORING_ASSETS) {
    findings.push(
      ...checkAlerts(path, alertReferences(await snapshot.read(path)), declared, (relative) =>
        snapshot.exists(relative),
      ),
    )
  }
  return {
    ok: findings.length === 0,
    lines:
      findings.length > 0
        ? findings.map((finding) => `fail  ${finding.path}: ${finding.message}`)
        : [`ok  ${declared.size} objective(s), ${MONITORING_ASSETS.length} monitoring asset(s)`],
  }
}
