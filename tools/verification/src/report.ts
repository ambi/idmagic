import { appendFile } from 'node:fs/promises'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { verificationInput } from './input.ts'
import type { DBVerification, ReferenceResult, TestExecution } from './evidence.ts'
import type { CheckResult } from '../../check/src/runner.ts'

export type VerificationReport = {
  input: { revision: string; fingerprint: string }
  tests: TestExecution[]
  references: ReferenceResult[]
  databases: DBVerification[]
  checks: CheckResult[]
  failures: string[]
  changeResistance: { state: string }
  command?: string[]
  requireDB?: boolean
  scope?: string
}

export function renderVerificationReport(
  report: VerificationReport,
  currentFingerprint: string,
): string {
  const current = report.input.fingerprint === currentFingerprint
  const states = ['passed', 'failed', 'skipped', 'not-run'] as const
  const rows = states.map(
    (state) =>
      `| ${state} | ${report.tests.filter((test) => test.name && test.state === state).length} | ${report.references.filter((test) => test.normativeIds.length && test.state === state).length} |`,
  )
  return [
    `Verification revision: \`${report.input.revision}\``,
    `Input identity: **${current ? 'current' : 'stale; unverified for the current tree'}** (\`${report.input.fingerprint}\`)`,
    `Command: \`${report.command?.join(' ') ?? 'not recorded'}\`; scope=${report.scope ?? 'not recorded'}, required DB=${report.requireDB ?? 'not recorded'}`,
    '',
    '| Execution state | Tests | Tests citing normative IDs |',
    '| --- | ---: | ---: |',
    ...rows,
    '',
    ...report.checks.flatMap((check) => check.lines),
    ...report.databases.map(
      (db) =>
        `DB ${db.state}: ${db.package}${db.reasons.length ? ` (${db.reasons.join(', ')})` : ''}`,
    ),
    '',
    `Change-resistance evidence: **${report.changeResistance.state}**; passing tests do not measure detection ability.`,
    `Gate failures: ${report.failures.length}`,
    ...report.failures.map((failure) => `- ${failure}`),
  ].join('\n')
}

if (import.meta.main) {
  const snapshot = createWorkspaceSnapshot(process.cwd())
  const path = process.argv[2] ?? '.verification/test-go.json'
  const report = (await Bun.file(snapshot.path(path)).json()) as VerificationReport
  const current = await verificationInput(snapshot)
  const text = renderVerificationReport(report, current.fingerprint)
  console.log(text)
  if (process.env.GITHUB_STEP_SUMMARY)
    await appendFile(process.env.GITHUB_STEP_SUMMARY, `${text}\n`)
}
