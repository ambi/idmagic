type Mutation = {
  mutatorName?: unknown
  status?: unknown
}

type MutationFile = {
  mutants?: unknown
}

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null && !Array.isArray(value)

export function verifyMutationCanary(report: unknown): string[] {
  if (!isRecord(report) || !isRecord(report.files)) {
    return ['mutation report does not contain a files object']
  }

  const mutations = Object.values(report.files).flatMap((file: unknown): Mutation[] => {
    if (!isRecord(file)) return []
    const { mutants } = file as MutationFile
    return Array.isArray(mutants) ? (mutants.filter(isRecord) as Mutation[]) : []
  })

  const findings: string[] = []
  if (
    !mutations.some(
      (mutation) =>
        mutation.mutatorName === 'CONDITIONALS_BOUNDARY' && mutation.status === 'Survived',
    )
  ) {
    findings.push('expected a CONDITIONALS_BOUNDARY mutant with status Survived')
  }
  if (
    !mutations.some(
      (mutation) =>
        mutation.mutatorName === 'ARITHMETIC_BASE' && mutation.status === 'CompileError',
    )
  ) {
    findings.push('expected an ARITHMETIC_BASE mutant with status CompileError')
  }
  return findings
}

if (import.meta.main) {
  const reportPath = process.argv[2]
  if (!reportPath) {
    console.error('usage: bun run check/src/mutation-canary.ts <report.json>')
    process.exit(2)
  }

  try {
    const report = await Bun.file(reportPath).json()
    const findings = verifyMutationCanary(report)
    if (findings.length > 0) {
      for (const finding of findings) console.error(`fail  mutation canary: ${finding}`)
      process.exit(1)
    }
    console.log('ok  mutation canary verdicts')
  } catch (error) {
    console.error(`fail  mutation canary: ${String(error)}`)
    process.exit(1)
  }
}
