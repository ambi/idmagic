type Mutation = {
  type?: unknown
  status?: unknown
}

type MutationFile = {
  mutations?: unknown
}

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null

export function verifyMutationCanary(report: unknown): string[] {
  if (!isRecord(report) || !Array.isArray(report.files)) {
    return ['mutation report does not contain a files array']
  }

  const mutations = report.files.flatMap((file: unknown): Mutation[] => {
    if (!isRecord(file)) return []
    const { mutations } = file as MutationFile
    return Array.isArray(mutations) ? (mutations.filter(isRecord) as Mutation[]) : []
  })

  const findings: string[] = []
  if (
    !mutations.some(
      (mutation) => mutation.type === 'CONDITIONALS_BOUNDARY' && mutation.status === 'LIVED',
    )
  ) {
    findings.push('expected a CONDITIONALS_BOUNDARY mutant with status LIVED')
  }
  if (
    !mutations.some(
      (mutation) => mutation.type === 'ARITHMETIC_BASE' && mutation.status === 'NOT VIABLE',
    )
  ) {
    findings.push('expected an ARITHMETIC_BASE mutant with status NOT VIABLE')
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
