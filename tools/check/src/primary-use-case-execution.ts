import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { verificationInput } from '../../verification/src/input.ts'
import type { VerificationReport } from '../../verification/src/report.ts'

/** 実行器の引用と実行イベントを照合する。参照や親テストの名前だけでは成功にしない。 */
export function passedPrimaryUseCase(
  report: VerificationReport,
  fingerprint: string,
  path: string,
  name: string,
  requirement: string,
): boolean {
  return (
    report.input?.fingerprint === fingerprint &&
    Array.isArray(report.failures) &&
    report.failures.length === 0 &&
    Array.isArray(report.references) &&
    Array.isArray(report.tests) &&
    report.references.some(
      (reference) =>
        reference.path === path &&
        reference.name === name &&
        reference.state === 'passed' &&
        Array.isArray(reference.normativeIds) &&
        reference.normativeIds.includes(requirement) &&
        report.tests.some(
          (test) =>
            test.package === reference.package && test.name === name && test.state === 'passed',
        ) &&
        !report.tests.some(
          (test) =>
            test.package === reference.package &&
            test.name.startsWith(`${name}/`) &&
            test.state !== 'passed',
        ),
    )
  )
}

/** 既存の生成済みの報告を再利用し、ここからテストを起動しない。 */
export async function primaryUseCaseExecution(snapshot: WorkspaceSnapshot) {
  const { fingerprint } = await verificationInput(snapshot)
  const reports: VerificationReport[] = []
  if (snapshot.exists('.verification')) {
    const results = await Promise.allSettled(
      (await snapshot.list('.verification'))
        .filter((entry) => entry.isFile() && entry.name.endsWith('.json'))
        .map(
          async (entry) =>
            JSON.parse(await snapshot.read(`.verification/${entry.name}`)) as VerificationReport,
        ),
    )
    for (const result of results)
      if (result.status === 'fulfilled' && result.value !== null) reports.push(result.value)
  }
  return (path: string, name: string, requirement: string) =>
    reports.some((report) => passedPrimaryUseCase(report, fingerprint, path, name, requirement))
}
